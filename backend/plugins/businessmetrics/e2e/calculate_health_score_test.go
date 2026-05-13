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

package e2e

import (
	"testing"

	"github.com/apache/incubator-devlake/plugins/businessmetrics/models"
	"github.com/apache/incubator-devlake/plugins/businessmetrics/tasks"
	"github.com/stretchr/testify/assert"
)

// Tests for DORA health score calculation

func TestCalculateDeployFreqScore_ElitePerformance(t *testing.T) {
	// 1 deploy/day = elite benchmark = max score
	score := tasks.CalculateDeployFreqScore(1.0)
	assert.Equal(t, 25, score)
}

func TestCalculateDeployFreqScore_AboveElite(t *testing.T) {
	// 2 deploys/day = above elite, capped at 25
	score := tasks.CalculateDeployFreqScore(2.0)
	assert.Equal(t, 25, score)
}

func TestCalculateDeployFreqScore_HalfElite(t *testing.T) {
	// 0.5 deploys/day = half of elite = 12-13 points
	score := tasks.CalculateDeployFreqScore(0.5)
	assert.Equal(t, 12, score) // 0.5/1.0 * 25 = 12
}

func TestCalculateDeployFreqScore_LowPerformance(t *testing.T) {
	// 0.1 deploys/day = low = ~2 points
	score := tasks.CalculateDeployFreqScore(0.1)
	assert.Equal(t, 2, score) // 0.1/1.0 * 25 = 2
}

func TestCalculateDeployFreqScore_Zero(t *testing.T) {
	score := tasks.CalculateDeployFreqScore(0)
	assert.Equal(t, 0, score)
}

func TestCalculateLeadTimeScore_ElitePerformance(t *testing.T) {
	// 24 hours = elite benchmark = max score
	score := tasks.CalculateLeadTimeScore(24.0)
	assert.Equal(t, 25, score)
}

func TestCalculateLeadTimeScore_BetterThanElite(t *testing.T) {
	// 12 hours = better than elite, capped at 25
	score := tasks.CalculateLeadTimeScore(12.0)
	assert.Equal(t, 25, score) // 24/12 * 25 = 50, capped at 25
}

func TestCalculateLeadTimeScore_DoubleElite(t *testing.T) {
	// 48 hours = twice elite = half points
	score := tasks.CalculateLeadTimeScore(48.0)
	assert.Equal(t, 12, score) // 24/48 * 25 = 12
}

func TestCalculateLeadTimeScore_PoorPerformance(t *testing.T) {
	// 240 hours (10 days) = very poor
	score := tasks.CalculateLeadTimeScore(240.0)
	assert.Equal(t, 2, score) // 24/240 * 25 = 2
}

func TestCalculateCFRScore_ElitePerformance(t *testing.T) {
	// 5% CFR = elite benchmark = max score
	score := tasks.CalculateCFRScore(5.0)
	assert.Equal(t, 25, score)
}

func TestCalculateCFRScore_PerfectScore(t *testing.T) {
	// 0% CFR = perfect, max score
	score := tasks.CalculateCFRScore(0)
	assert.Equal(t, 25, score)
}

func TestCalculateCFRScore_PoorPerformance(t *testing.T) {
	// 25% CFR = 5x worse than elite = 5 points
	score := tasks.CalculateCFRScore(25.0)
	assert.Equal(t, 5, score) // 5/25 * 25 = 5
}

func TestCalculateMTTRScore_ElitePerformance(t *testing.T) {
	// 1 hour = elite benchmark = max score
	score := tasks.CalculateMTTRScore(1.0)
	assert.Equal(t, 25, score)
}

func TestCalculateMTTRScore_PerfectScore(t *testing.T) {
	// 0 hours = instant recovery, max score
	score := tasks.CalculateMTTRScore(0)
	assert.Equal(t, 25, score)
}

func TestCalculateMTTRScore_FourHours(t *testing.T) {
	// 4 hours = 4x worse than elite = 6 points
	score := tasks.CalculateMTTRScore(4.0)
	assert.Equal(t, 6, score) // 1/4 * 25 = 6
}

func TestDetermineHealthLevel_Elite(t *testing.T) {
	assert.Equal(t, models.HealthLevelElite, tasks.DetermineHealthLevel(100))
	assert.Equal(t, models.HealthLevelElite, tasks.DetermineHealthLevel(80))
}

func TestDetermineHealthLevel_High(t *testing.T) {
	assert.Equal(t, models.HealthLevelHigh, tasks.DetermineHealthLevel(79))
	assert.Equal(t, models.HealthLevelHigh, tasks.DetermineHealthLevel(60))
}

func TestDetermineHealthLevel_Medium(t *testing.T) {
	assert.Equal(t, models.HealthLevelMedium, tasks.DetermineHealthLevel(59))
	assert.Equal(t, models.HealthLevelMedium, tasks.DetermineHealthLevel(40))
}

func TestDetermineHealthLevel_Low(t *testing.T) {
	assert.Equal(t, models.HealthLevelLow, tasks.DetermineHealthLevel(39))
	assert.Equal(t, models.HealthLevelLow, tasks.DetermineHealthLevel(0))
}

func TestEliteTeamScenario(t *testing.T) {
	// Elite team: 1 deploy/day, 24h lead time, 5% CFR, 1h MTTR
	deployScore := tasks.CalculateDeployFreqScore(1.0)
	leadTimeScore := tasks.CalculateLeadTimeScore(24.0)
	cfrScore := tasks.CalculateCFRScore(5.0)
	mttrScore := tasks.CalculateMTTRScore(1.0)

	total := deployScore + leadTimeScore + cfrScore + mttrScore
	assert.Equal(t, 100, total, "Elite team should score 100")
	assert.Equal(t, models.HealthLevelElite, tasks.DetermineHealthLevel(total))
}

func TestAverageTeamScenario(t *testing.T) {
	// Average team: 0.5 deploy/day, 48h lead time, 10% CFR, 4h MTTR
	deployScore := tasks.CalculateDeployFreqScore(0.5)  // 12
	leadTimeScore := tasks.CalculateLeadTimeScore(48.0) // 12
	cfrScore := tasks.CalculateCFRScore(10.0)           // 12
	mttrScore := tasks.CalculateMTTRScore(4.0)          // 6

	total := deployScore + leadTimeScore + cfrScore + mttrScore
	assert.InDelta(t, 42, total, 2, "Average team should score around 42")
	assert.Equal(t, models.HealthLevelMedium, tasks.DetermineHealthLevel(total))
}

func TestStruggleTeamScenario(t *testing.T) {
	// Struggling team: 0.1 deploy/day, 168h (1 week) lead time, 25% CFR, 24h MTTR
	deployScore := tasks.CalculateDeployFreqScore(0.1)   // 2
	leadTimeScore := tasks.CalculateLeadTimeScore(168.0) // 3
	cfrScore := tasks.CalculateCFRScore(25.0)            // 5
	mttrScore := tasks.CalculateMTTRScore(24.0)          // 1

	total := deployScore + leadTimeScore + cfrScore + mttrScore
	assert.InDelta(t, 11, total, 2, "Struggling team should score around 11")
	assert.Equal(t, models.HealthLevelLow, tasks.DetermineHealthLevel(total))
}
