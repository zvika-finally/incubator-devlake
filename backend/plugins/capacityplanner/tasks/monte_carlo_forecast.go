/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tasks

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/capacityplanner/models"
)

var MonteCarloForecastMeta = plugin.SubTaskMeta{
	Name:             "monteCarloForecast",
	EntryPoint:       MonteCarloForecast,
	EnabledByDefault: true,
	Description:      "Run Monte Carlo simulation to forecast initiative completion with probabilistic estimates",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

// Constants from eng-product-metrics
const (
	MonteCarloIterations    = 1000
	DefaultVelocityVariance = 0.25
	DefaultVelocity         = 20.0 // story points per week
	DaysPerWeek             = 7
)

func MonteCarloForecast(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*CapacityPlannerTaskData)
	logger := taskCtx.GetLogger()

	logger.Info("Starting monteCarloForecast for project: %s", data.Options.ProjectName)

	// Resolve simulation configuration from settings with safe defaults.
	iterations := MonteCarloIterations
	defaultVelocity := DefaultVelocity
	varianceFactor := DefaultVelocityVariance
	if data.Settings != nil {
		if data.Settings.MonteCarloIterations > 0 {
			iterations = data.Settings.MonteCarloIterations
		}
		if data.Settings.DefaultVelocity > 0 {
			defaultVelocity = data.Settings.DefaultVelocity
		}
		if data.Settings.VelocityVariance > 0 {
			varianceFactor = data.Settings.VelocityVariance
		}
	}

	initiatives, err := getProjectInitiatives(db, data.Options.ProjectName)
	if err != nil {
		return err
	}

	logger.Info("Running Monte Carlo simulation for %d initiatives", len(initiatives))

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for _, initiative := range initiatives {
		var forecast models.InitiativeForecast
		err := db.First(&forecast,
			dal.From(&models.InitiativeForecast{}),
			dal.Where("initiative_id = ? AND remaining_story_points > 0", initiative.Id),
			dal.Orderby("calculated_at DESC"),
			dal.Limit(1),
		)
		if err != nil {
			if db.IsErrorNotFound(err) {
				continue
			}
			logger.Error(err, "failed to query forecast for initiative %s", initiative.Id)
			continue
		}

		// Get historical velocity for this initiative (avg and stddev)
		avgVelocity := forecast.AvgVelocity
		velocityStdDev := forecast.VelocityStdDev

		// Use defaults if no historical data
		if avgVelocity <= 0 {
			avgVelocity = defaultVelocity
		}
		if velocityStdDev <= 0 {
			velocityStdDev = avgVelocity * varianceFactor
			if velocityStdDev <= 0 {
				velocityStdDev = avgVelocity * DefaultVelocityVariance
			}
		}

		// Run Monte Carlo simulation
		completionDays := runSimulation(
			rng,
			float64(forecast.RemainingStoryPoints),
			avgVelocity,
			velocityStdDev,
			varianceFactor,
			iterations,
		)

		// Calculate percentiles
		p50Days := NearestRankPercentile(completionDays, 50)
		p75Days := NearestRankPercentile(completionDays, 75)
		p90Days := NearestRankPercentile(completionDays, 90)
		p95Days := NearestRankPercentile(completionDays, 95)

		// Convert to sprints (assuming 2-week sprints)
		sprintDays := data.Options.SprintDurationWeeks * DaysPerWeek
		if sprintDays <= 0 {
			sprintDays = 14
		}

		// Calculate completion dates
		now := time.Now()
		p50Date := now.AddDate(0, 0, p50Days)
		p75Date := now.AddDate(0, 0, p75Days)
		p90Date := now.AddDate(0, 0, p90Days)
		p95Date := now.AddDate(0, 0, p95Days)

		// Find earliest and latest
		earliest := completionDays[0]
		latest := completionDays[len(completionDays)-1]

		// Create Monte Carlo forecast record
		mcForecast := models.MonteCarloForecast{
			Id:               fmt.Sprintf("%s:mc:%d", forecast.InitiativeId, time.Now().Unix()),
			InitiativeId:     forecast.InitiativeId,
			SimulationCount:  iterations,
			VelocityVariance: varianceFactor,

			P50Sprints: (p50Days + sprintDays - 1) / sprintDays,
			P75Sprints: (p75Days + sprintDays - 1) / sprintDays,
			P90Sprints: (p90Days + sprintDays - 1) / sprintDays,
			P95Sprints: (p95Days + sprintDays - 1) / sprintDays,

			P50Date: &p50Date,
			P75Date: &p75Date,
			P90Date: &p90Date,
			P95Date: &p95Date,

			EarliestDays: earliest,
			LatestDays:   latest,

			CalculatedAt: time.Now(),
		}

		if err := db.CreateOrUpdate(&mcForecast); err != nil {
			logger.Error(err, "failed to save Monte Carlo forecast for initiative %s", forecast.InitiativeId)
			continue
		}

		logger.Info("Monte Carlo forecast for %s: P50=%d days, P90=%d days, P95=%d days",
			forecast.InitiativeId, p50Days, p90Days, p95Days)
	}

	logger.Info("Completed Monte Carlo forecasting for %d initiatives", len(initiatives))
	return nil
}

// runSimulation runs Monte Carlo simulation and returns sorted completion days
func runSimulation(rng *rand.Rand, remainingPoints, avgVelocity, velocityStdDev, variance float64, iterations int) []int {
	completionDays := make([]int, iterations)

	for i := 0; i < iterations; i++ {
		remaining := remainingPoints
		days := 0

		for remaining > 0 {
			// Generate random velocity using Gaussian distribution
			weeklyVelocity := GaussianRandom(rng, avgVelocity, velocityStdDev*variance)

			// Ensure minimum velocity of 1 point per week
			if weeklyVelocity < 1 {
				weeklyVelocity = 1
			}

			remaining -= weeklyVelocity
			days += DaysPerWeek
		}

		completionDays[i] = days
	}

	// Sort for percentile calculation
	sort.Ints(completionDays)
	return completionDays
}

// GaussianRandom generates a random number from a Gaussian distribution
// using the Box-Muller transform
// Exported for testing
func GaussianRandom(rng *rand.Rand, mean, stddev float64) float64 {
	u1 := rng.Float64()
	u2 := rng.Float64()

	// Box-Muller transform
	z := math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
	return mean + z*stddev
}

// NearestRankPercentile calculates the percentile using nearest-rank method
// Exported for testing
func NearestRankPercentile(sortedData []int, percentile int) int {
	if len(sortedData) == 0 {
		return 0
	}

	// Nearest-rank formula: index = ceil(percentile/100 * len) - 1
	// But we use: index = min(len * percentile / 100, len - 1)
	index := len(sortedData) * percentile / 100
	if index >= len(sortedData) {
		index = len(sortedData) - 1
	}
	if index < 0 {
		index = 0
	}

	return sortedData[index]
}
