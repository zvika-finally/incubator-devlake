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

// Tests for business value score calculation

func TestGetRevenueImpactWeight_Direct(t *testing.T) {
	weight := tasks.GetRevenueImpactWeight(models.RevenueImpactDirect)
	assert.Equal(t, 30, weight)
}

func TestGetRevenueImpactWeight_Enabling(t *testing.T) {
	weight := tasks.GetRevenueImpactWeight(models.RevenueImpactEnabling)
	assert.Equal(t, 20, weight)
}

func TestGetRevenueImpactWeight_Supporting(t *testing.T) {
	weight := tasks.GetRevenueImpactWeight(models.RevenueImpactSupporting)
	assert.Equal(t, 10, weight)
}

func TestGetRevenueImpactWeight_CostCenter(t *testing.T) {
	weight := tasks.GetRevenueImpactWeight(models.RevenueImpactCostCenter)
	assert.Equal(t, 0, weight)
}

func TestGetRevenueImpactWeight_Unknown(t *testing.T) {
	weight := tasks.GetRevenueImpactWeight("unknown")
	assert.Equal(t, 0, weight)
}

func TestGetEfficiencyBonus_Excellent(t *testing.T) {
	// Ratio >= 5 = excellent bonus
	assert.Equal(t, 20, tasks.GetEfficiencyBonus(5.0))
	assert.Equal(t, 20, tasks.GetEfficiencyBonus(10.0))
}

func TestGetEfficiencyBonus_Good(t *testing.T) {
	// 2 <= ratio < 5 = good bonus
	assert.Equal(t, 15, tasks.GetEfficiencyBonus(2.0))
	assert.Equal(t, 15, tasks.GetEfficiencyBonus(4.9))
}

func TestGetEfficiencyBonus_Fair(t *testing.T) {
	// 1 <= ratio < 2 = fair bonus
	assert.Equal(t, 10, tasks.GetEfficiencyBonus(1.0))
	assert.Equal(t, 10, tasks.GetEfficiencyBonus(1.9))
}

func TestGetEfficiencyBonus_Positive(t *testing.T) {
	// 0 < ratio < 1 = small bonus
	assert.Equal(t, 5, tasks.GetEfficiencyBonus(0.1))
	assert.Equal(t, 5, tasks.GetEfficiencyBonus(0.9))
}

func TestGetEfficiencyBonus_Zero(t *testing.T) {
	assert.Equal(t, 0, tasks.GetEfficiencyBonus(0))
}

func TestGetEfficiencyBonus_Negative(t *testing.T) {
	// Negative ratio (shouldn't happen but handle gracefully)
	assert.Equal(t, 0, tasks.GetEfficiencyBonus(-1.0))
}

func TestCalculateBusinessValueScore_DirectWithHighEfficiency(t *testing.T) {
	// Direct revenue impact + excellent efficiency
	// Base(50) + Direct(30) + Excellent(20) = 100
	score := tasks.CalculateBusinessValueScore(models.RevenueImpactDirect, 5.0)
	assert.Equal(t, 100, score)
}

func TestCalculateBusinessValueScore_DirectNoEfficiency(t *testing.T) {
	// Direct revenue impact + no efficiency data
	// Base(50) + Direct(30) + Zero(0) = 80
	score := tasks.CalculateBusinessValueScore(models.RevenueImpactDirect, 0)
	assert.Equal(t, 80, score)
}

func TestCalculateBusinessValueScore_EnablingWithGoodEfficiency(t *testing.T) {
	// Enabling impact + good efficiency
	// Base(50) + Enabling(20) + Good(15) = 85
	score := tasks.CalculateBusinessValueScore(models.RevenueImpactEnabling, 3.0)
	assert.Equal(t, 85, score)
}

func TestCalculateBusinessValueScore_SupportingWithFairEfficiency(t *testing.T) {
	// Supporting impact + fair efficiency
	// Base(50) + Supporting(10) + Fair(10) = 70
	score := tasks.CalculateBusinessValueScore(models.RevenueImpactSupporting, 1.5)
	assert.Equal(t, 70, score)
}

func TestCalculateBusinessValueScore_CostCenterNoEfficiency(t *testing.T) {
	// Cost center + no efficiency (compliance, maintenance)
	// Base(50) + CostCenter(0) + Zero(0) = 50
	score := tasks.CalculateBusinessValueScore(models.RevenueImpactCostCenter, 0)
	assert.Equal(t, 50, score)
}

func TestCalculateBusinessValueScore_CappedAt100(t *testing.T) {
	// Even with impossible values, should cap at 100
	score := tasks.CalculateBusinessValueScore(models.RevenueImpactDirect, 100.0)
	assert.Equal(t, 100, score)
}

func TestCalculateBusinessValueScore_MinimumScore(t *testing.T) {
	// Lowest possible: cost center with no efficiency
	score := tasks.CalculateBusinessValueScore(models.RevenueImpactCostCenter, 0)
	assert.Equal(t, 50, score, "Minimum score should be base score of 50")
}

// Test valid enum values
func TestValidBusinessCapabilities(t *testing.T) {
	capabilities := models.ValidBusinessCapabilities()
	assert.Contains(t, capabilities, models.CapabilityCoreProduct)
	assert.Contains(t, capabilities, models.CapabilityGrowth)
	assert.Contains(t, capabilities, models.CapabilityMonetization)
	assert.Contains(t, capabilities, models.CapabilityPlatform)
	assert.Contains(t, capabilities, models.CapabilityInfrastructure)
	assert.Contains(t, capabilities, models.CapabilityInternalTools)
	assert.Contains(t, capabilities, models.CapabilityCompliance)
	assert.Equal(t, 7, len(capabilities))
}

func TestValidRevenueImpacts(t *testing.T) {
	impacts := models.ValidRevenueImpacts()
	assert.Contains(t, impacts, models.RevenueImpactDirect)
	assert.Contains(t, impacts, models.RevenueImpactEnabling)
	assert.Contains(t, impacts, models.RevenueImpactSupporting)
	assert.Contains(t, impacts, models.RevenueImpactCostCenter)
	assert.Equal(t, 4, len(impacts))
}

// Scenario tests
func TestScenario_NewPaymentFeature(t *testing.T) {
	// New payment feature: core product, direct revenue, 3x ROI
	score := tasks.CalculateBusinessValueScore(models.RevenueImpactDirect, 3.0)
	assert.Equal(t, 95, score) // 50 + 30 + 15
}

func TestScenario_PlatformRefactoring(t *testing.T) {
	// Platform refactoring: enables other features, 1.5x ROI
	score := tasks.CalculateBusinessValueScore(models.RevenueImpactEnabling, 1.5)
	assert.Equal(t, 80, score) // 50 + 20 + 10
}

func TestScenario_ComplianceProject(t *testing.T) {
	// Compliance project: required but no direct revenue
	score := tasks.CalculateBusinessValueScore(models.RevenueImpactCostCenter, 0)
	assert.Equal(t, 50, score) // Base only
}
