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

	"github.com/apache/incubator-devlake/plugins/capacityplanner/tasks"
	"github.com/stretchr/testify/assert"
)

// Tests for ROI calculation logic

func TestCalculateROIFromParams_BasicScenario(t *testing.T) {
	params := tasks.ROIParameters{
		TeamSize:           5,
		HourlyCost:         75.0,
		AIAdoptionPercent:  80.0,
		ProductivityGain:   10.0,
		QualityImprovement: 5.0,
		HoursSavedPerWeek:  2.0,
	}

	upfrontCost := 0.0
	monthlyCost := 100.0 // $100/month total

	annualBenefit, paybackMonths, threeYearROI := tasks.CalculateROIFromParams(params, upfrontCost, monthlyCost)

	// Direct benefit = 2 hours/week * 5 people * 52 weeks * $75 = $39,000
	// Productivity = 5 * 40 * 52 * 0.10 * $75 = $78,000
	// Quality = 5 * 40 * 52 * 0.20 * 0.05 * $75 = $7,800
	// Total = ~$124,800

	assert.Greater(t, annualBenefit, 100000.0, "Annual benefit should be substantial")
	assert.Less(t, paybackMonths, 2.0, "Payback should be under 2 months with low cost")
	assert.Greater(t, threeYearROI, 1000.0, "3-year ROI should be very high with low monthly cost")
}

func TestCalculateROIFromParams_HighCostScenario(t *testing.T) {
	params := tasks.ROIParameters{
		TeamSize:           5,
		HourlyCost:         75.0,
		AIAdoptionPercent:  50.0,
		ProductivityGain:   5.0,
		QualityImprovement: 3.0,
		HoursSavedPerWeek:  1.0,
	}

	upfrontCost := 50000.0 // $50k upfront
	monthlyCost := 5000.0  // $5k/month

	annualBenefit, paybackMonths, _ := tasks.CalculateROIFromParams(params, upfrontCost, monthlyCost)

	// Annual cost = $50k + $60k = $110k
	// This should have longer payback and lower ROI
	assert.Greater(t, annualBenefit, 0.0, "Should still have positive benefit")
	assert.Greater(t, paybackMonths, 10.0, "Payback should be longer with high costs")
}

func TestCalculateROIFromParams_ZeroTeam(t *testing.T) {
	params := tasks.ROIParameters{
		TeamSize:           0,
		HourlyCost:         75.0,
		AIAdoptionPercent:  80.0,
		ProductivityGain:   10.0,
		QualityImprovement: 5.0,
		HoursSavedPerWeek:  2.0,
	}

	annualBenefit, _, _ := tasks.CalculateROIFromParams(params, 0, 100)

	assert.Equal(t, 0.0, annualBenefit, "Zero team should have zero benefit")
}

func TestCalculateROIFromParams_DirectBenefitOnly(t *testing.T) {
	params := tasks.ROIParameters{
		TeamSize:           10,
		HourlyCost:         100.0,
		AIAdoptionPercent:  0.0,
		ProductivityGain:   0.0,
		QualityImprovement: 0.0,
		HoursSavedPerWeek:  5.0, // Only direct hours saved
	}

	annualBenefit, _, _ := tasks.CalculateROIFromParams(params, 0, 100)

	// Expected: 5 hours * 10 people * 52 weeks * $100 = $260,000
	expectedDirect := 5.0 * 10 * 52 * 100.0
	assert.InDelta(t, expectedDirect, annualBenefit, 1.0, "Should match direct benefit calculation")
}

func TestCalculateROIFromParams_ProductivityBenefitOnly(t *testing.T) {
	params := tasks.ROIParameters{
		TeamSize:           10,
		HourlyCost:         100.0,
		AIAdoptionPercent:  100.0,
		ProductivityGain:   20.0, // 20% productivity gain
		QualityImprovement: 0.0,
		HoursSavedPerWeek:  0.0,
	}

	annualBenefit, _, _ := tasks.CalculateROIFromParams(params, 0, 100)

	// Expected: 10 * 40 * 52 * 0.20 * $100 = $416,000
	expectedProductivity := 10.0 * 40 * 52 * 0.20 * 100.0
	assert.InDelta(t, expectedProductivity, annualBenefit, 1.0, "Should match productivity benefit calculation")
}

func TestCalculateROIFromParams_ThreeYearROICalculation(t *testing.T) {
	params := tasks.ROIParameters{
		TeamSize:           5,
		HourlyCost:         75.0,
		AIAdoptionPercent:  80.0,
		ProductivityGain:   10.0,
		QualityImprovement: 5.0,
		HoursSavedPerWeek:  2.0,
	}

	upfrontCost := 10000.0
	monthlyCost := 1000.0

	annualBenefit, _, threeYearROI := tasks.CalculateROIFromParams(params, upfrontCost, monthlyCost)

	// 3-year cost = $10k + $36k = $46k
	// 3-year benefit = annual * 3
	// ROI = (benefit - cost) / cost * 100

	threeYearCost := 10000.0 + (1000.0 * 36)
	threeYearBenefit := annualBenefit * 3
	expectedROI := ((threeYearBenefit - threeYearCost) / threeYearCost) * 100

	assert.InDelta(t, expectedROI, threeYearROI, 0.1, "3-year ROI should match manual calculation")
}
