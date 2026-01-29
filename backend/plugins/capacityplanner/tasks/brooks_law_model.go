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
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/capacityplanner/models"
)

var BrooksLawModelMeta = plugin.SubTaskMeta{
	Name:             "brooksLawModel",
	EntryPoint:       BrooksLawModel,
	EnabledByDefault: true,
	Description:      "Model team size change impact using Brooks's Law communication overhead formula",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

// Constants from eng-product-metrics
const (
	DefaultRampUpWeeks    = 8
	NewHireProductivity   = 0.5  // 50% productivity during ramp-up
	ChannelOverheadFactor = 0.1  // 10% overhead per additional channel
)

func BrooksLawModel(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*CapacityPlannerTaskData)
	logger := taskCtx.GetLogger()

	logger.Info("Starting brooksLawModel for project: %s", data.Options.ProjectName)

	// Get current team velocity to extract team size
	var velocity models.TeamVelocity
	velocityClauses := []dal.Clause{
		dal.From(&models.TeamVelocity{}),
		dal.Where("project_name = ?", data.Options.ProjectName),
		dal.Orderby("calculated_at DESC"),
		dal.Limit(1),
	}

	err := db.First(&velocity, velocityClauses...)
	if err != nil {
		if db.IsErrorNotFound(err) {
			logger.Info("No velocity data found for project %s, skipping Brooks's Law modeling", data.Options.ProjectName)
			return nil
		}
		return errors.Default.Wrap(err, "failed to query team velocity")
	}

	currentTeamSize := velocity.TeamSize
	if currentTeamSize <= 0 {
		logger.Info("No team size data for project %s, using default of 5", data.Options.ProjectName)
		currentTeamSize = 5
	}

	// Model scenarios: +1, +2, +3, -1, -2
	scenarios := []struct {
		name  string
		delta int
	}{
		{"add_1_member", 1},
		{"add_2_members", 2},
		{"add_3_members", 3},
		{"remove_1_member", -1},
		{"remove_2_members", -2},
	}

	for _, scenario := range scenarios {
		newTeamSize := currentTeamSize + scenario.delta
		if newTeamSize <= 0 {
			continue
		}

		// Calculate communication channels: n*(n-1)/2
		currentChannels := CalculateCommunicationChannels(currentTeamSize)
		newChannels := CalculateCommunicationChannels(newTeamSize)

		// Calculate productivity factor and overhead
		productivityFactor, overheadFactor := CalculateBrooksLawImpact(
			currentTeamSize,
			scenario.delta,
			currentChannels,
			newChannels,
		)

		// Projected impact on metrics
		// Deploy frequency scales with capacity
		projectedDeployDelta := (productivityFactor*overheadFactor - 1) * 100

		// Lead time inversely scales (more capacity = potentially faster, but overhead can slow it)
		projectedLeadDelta := (1/(productivityFactor*overheadFactor) - 1) * 100

		capacityModel := models.CapacityModel{
			Id:              fmt.Sprintf("%s:%s:%d", data.Options.ProjectName, scenario.name, time.Now().Unix()),
			ProjectName:     data.Options.ProjectName,
			ScenarioName:    scenario.name,

			CurrentTeamSize: currentTeamSize,
			TeamSizeDelta:   scenario.delta,
			RampUpWeeks:     DefaultRampUpWeeks,

			CurrentChannels: currentChannels,
			NewChannels:     newChannels,
			OverheadFactor:  overheadFactor,

			ProductivityFactor:   productivityFactor,
			ProjectedDeployDelta: projectedDeployDelta,
			ProjectedLeadDelta:   projectedLeadDelta,

			CalculatedAt: time.Now(),
		}

		if err := db.CreateOrUpdate(&capacityModel); err != nil {
			logger.Error(err, "failed to save capacity model for scenario %s", scenario.name)
			continue
		}

		logger.Info("Brooks's Law model for %s (team %d -> %d): productivity=%.2f, overhead=%.2f, deploy delta=%.1f%%",
			scenario.name, currentTeamSize, newTeamSize, productivityFactor, overheadFactor, projectedDeployDelta)
	}

	logger.Info("Completed Brooks's Law modeling for project %s", data.Options.ProjectName)
	return nil
}

// CalculateCommunicationChannels calculates the number of communication channels
// using Brooks's Law formula: n*(n-1)/2
// Exported for testing
func CalculateCommunicationChannels(teamSize int) int {
	if teamSize <= 1 {
		return 0
	}
	return teamSize * (teamSize - 1) / 2
}

// CalculateBrooksLawImpact calculates productivity factor and overhead factor
// based on team size changes using Brooks's Law principles
// Exported for testing
func CalculateBrooksLawImpact(currentSize, delta, currentChannels, newChannels int) (productivityFactor, overheadFactor float64) {
	newSize := currentSize + delta

	if delta > 0 {
		// Adding team members - new hires have reduced productivity during ramp-up
		effectiveNewCapacity := float64(delta) * NewHireProductivity
		productivityFactor = (float64(currentSize) + effectiveNewCapacity) / float64(currentSize)
	} else if delta < 0 {
		// Reducing team members - linear capacity reduction
		productivityFactor = float64(newSize) / float64(currentSize)
	} else {
		productivityFactor = 1.0
	}

	// Communication overhead increases with more channels
	// Formula: 1 - (channel_delta / (current_channels + 1)) * overhead_factor
	channelDelta := newChannels - currentChannels
	if currentChannels > 0 || channelDelta != 0 {
		overheadFactor = 1 - (float64(channelDelta)/float64(currentChannels+1))*ChannelOverheadFactor
	} else {
		overheadFactor = 1.0
	}

	// Ensure overhead doesn't go negative
	if overheadFactor < 0.5 {
		overheadFactor = 0.5
	}

	return productivityFactor, overheadFactor
}
