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
	"github.com/apache/incubator-devlake/core/log"
	"github.com/apache/incubator-devlake/core/models/domainlayer/ticket"
	"github.com/apache/incubator-devlake/core/plugin"
	bmModels "github.com/apache/incubator-devlake/plugins/businessmetrics/models"
	"github.com/apache/incubator-devlake/plugins/capacityplanner/models"
)

var MonteCarloForecastKanbanMeta = plugin.SubTaskMeta{
	Name:             "monteCarloForecastKanban",
	EntryPoint:       MonteCarloForecastKanban,
	EnabledByDefault: true,
	Description:      "Run Monte Carlo simulation using Kanban throughput (issue-based)",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

func MonteCarloForecastKanban(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*CapacityPlannerTaskData)
	logger := taskCtx.GetLogger()

	logger.Info("Starting monteCarloForecastKanban for project: %s", data.Options.ProjectName)

	// Get throughput history
	throughputHistory := getThroughputHistory(db, data.Options.ProjectName, logger)
	if len(throughputHistory) < 3 {
		logger.Warn(nil, "Insufficient throughput data (need at least 3 periods), skipping Monte Carlo")
		return nil
	}

	avgThroughput := calculateMean(throughputHistory)
	stdDev := calculateStdDev(throughputHistory, avgThroughput)
	variance := stdDev / avgThroughput // Coefficient of variation

	logger.Info("Throughput stats: avg=%.1f, stdDev=%.1f, variance=%.1f%%", avgThroughput, stdDev, variance*100)

	// Get active initiatives
	var initiatives []bmModels.BusinessInitiative
	if err := db.All(&initiatives,
		dal.From(&bmModels.BusinessInitiative{}),
		dal.Where("status IN ('active', 'planned')"),
	); err != nil {
		return errors.Default.Wrap(err, "failed to query initiatives")
	}

	logger.Info("Running Monte Carlo simulation for %d initiatives", len(initiatives))

	for _, initiative := range initiatives {
		forecast := runMonteCarloSimulationKanban(db, initiative, avgThroughput, variance, data.Options.SprintDurationWeeks, logger)
		if forecast == nil {
			continue
		}

		if err := db.CreateOrUpdate(forecast); err != nil {
			logger.Error(err, "failed to save Monte Carlo forecast for initiative %s", initiative.Name)
		}
	}

	logger.Info("Completed monteCarloForecastKanban")
	return nil
}

func getThroughputHistory(db dal.Dal, projectName string, logger log.Logger) []float64 {
	var velocities []struct {
		IssuesCompleted int
	}

	err := db.All(&velocities,
		dal.Select("issues_completed"),
		dal.From("team_velocities"),
		dal.Where("project_name = ? AND issues_completed > 0", projectName),
		dal.Orderby("sprint_end_date DESC"),
		dal.Limit(12), // Last 12 periods for better simulation
	)

	if err != nil {
		logger.Warn(err, "Could not get throughput history")
		return []float64{}
	}

	history := make([]float64, len(velocities))
	for i, v := range velocities {
		history[i] = float64(v.IssuesCompleted)
	}

	return history
}

func calculateMean(data []float64) float64 {
	sum := 0.0
	for _, v := range data {
		sum += v
	}
	return sum / float64(len(data))
}

func calculateStdDev(data []float64, mean float64) float64 {
	sumSquares := 0.0
	for _, v := range data {
		diff := v - mean
		sumSquares += diff * diff
	}
	variance := sumSquares / float64(len(data))
	return math.Sqrt(variance)
}

func runMonteCarloSimulationKanban(db dal.Dal, initiative bmModels.BusinessInitiative, avgThroughput, variance float64, weeksPerPeriod int, logger log.Logger) *models.MonteCarloForecast {
	// Count remaining issues for this initiative
	var totalIssues, completedIssues int64

	err := db.First(&totalIssues,
		dal.Select("COUNT(DISTINCT i.id)"),
		dal.From(&ticket.Issue{}),
		dal.Where("parent_issue_id = ? OR epic_key = ?", initiative.Id, initiative.JiraEpicKey),
	)
	if err != nil || totalIssues == 0 {
		return nil
	}

	err = db.First(&completedIssues,
		dal.Select("COUNT(DISTINCT i.id)"),
		dal.From(&ticket.Issue{}),
		dal.Where("(parent_issue_id = ? OR epic_key = ?) AND status = 'DONE'", initiative.Id, initiative.JiraEpicKey),
	)
	if err != nil {
		completedIssues = 0
	}

	remainingIssues := totalIssues - completedIssues
	if remainingIssues <= 0 {
		return nil // Already complete
	}

	// Run 1000 Monte Carlo simulations
	simulationCount := 1000
	completionWeeks := make([]int, simulationCount)

	rand.Seed(time.Now().UnixNano())

	for i := 0; i < simulationCount; i++ {
		remaining := float64(remainingIssues)
		weeks := 0

		for remaining > 0 && weeks < 520 { // Max 10 years
			// Generate random throughput using normal distribution
			// Simple approximation: throughput = avg ± (random * stdDev)
			randomFactor := (rand.Float64() - 0.5) * 2 * variance // Range: -variance to +variance
			weeklyThroughput := avgThroughput * (1 + randomFactor)

			// Ensure positive throughput
			if weeklyThroughput < 1 {
				weeklyThroughput = 1
			}

			remaining -= weeklyThroughput
			weeks++
		}

		completionWeeks[i] = weeks
	}

	// Sort results to calculate percentiles
	sort.Ints(completionWeeks)

	// Calculate percentiles (nearest-rank method)
	p50Weeks := completionWeeks[simulationCount*50/100]
	p75Weeks := completionWeeks[simulationCount*75/100]
	p90Weeks := completionWeeks[simulationCount*90/100]
	p95Weeks := completionWeeks[simulationCount*95/100]

	// Convert weeks to periods
	p50Periods := p50Weeks / weeksPerPeriod
	p75Periods := p75Weeks / weeksPerPeriod
	p90Periods := p90Weeks / weeksPerPeriod
	p95Periods := p95Weeks / weeksPerPeriod

	// Calculate completion dates
	p50Date := time.Now().AddDate(0, 0, p50Weeks*7)
	p75Date := time.Now().AddDate(0, 0, p75Weeks*7)
	p90Date := time.Now().AddDate(0, 0, p90Weeks*7)
	p95Date := time.Now().AddDate(0, 0, p95Weeks*7)

	earliestDays := completionWeeks[0] * 7
	latestDays := completionWeeks[simulationCount-1] * 7

	forecast := &models.MonteCarloForecast{
		Id:              fmt.Sprintf("%s:mc:%s", initiative.Id, time.Now().Format("20060102")),
		InitiativeId:    initiative.Id,
		SimulationCount: int(simulationCount),
		VelocityVariance: variance * 100, // Convert to percentage
		P50Sprints:      int(p50Periods),
		P75Sprints:      int(p75Periods),
		P90Sprints:      int(p90Periods),
		P95Sprints:      int(p95Periods),
		P50Date:         &p50Date,
		P75Date:         &p75Date,
		P90Date:         &p90Date,
		P95Date:         &p95Date,
		EarliestDays:    int(earliestDays),
		LatestDays:      int(latestDays),
		CalculatedAt:    time.Now(),
	}

	logger.Info("Monte Carlo for '%s': P50=%d weeks, P90=%d weeks (%.0f remaining issues, %.1f/week)",
		initiative.Name, p50Weeks, p90Weeks, float64(remainingIssues), avgThroughput)

	return forecast
}
