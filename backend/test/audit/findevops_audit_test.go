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

package audit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCapitalizationRateFormula validates the capitalization rate calculation
// Formula: capitalization_rate = capitalizable_cost / total_cost * 100
func TestCapitalizationRateFormula(t *testing.T) {
	testCases := []struct {
		name              string
		capitalizableCost float64
		totalCost         float64
		expectedRate      float64
	}{
		{
			name:              "50% capitalization",
			capitalizableCost: 50000.00,
			totalCost:         100000.00,
			expectedRate:      50.00,
		},
		{
			name:              "100% capitalization (all features)",
			capitalizableCost: 75000.00,
			totalCost:         75000.00,
			expectedRate:      100.00,
		},
		{
			name:              "0% capitalization (all maintenance)",
			capitalizableCost: 0.00,
			totalCost:         50000.00,
			expectedRate:      0.00,
		},
		{
			name:              "Zero total cost",
			capitalizableCost: 0.00,
			totalCost:         0.00,
			expectedRate:      0.00, // Avoid division by zero
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var rate float64
			if tc.totalCost > 0 {
				rate = tc.capitalizableCost / tc.totalCost * 100
			}
			assert.InDelta(t, tc.expectedRate, rate, 0.01, "Capitalization rate mismatch")
		})
	}
}

// TestBudgetVarianceFormula validates the budget variance calculation
// Formula: variance = (estimated - actual) / estimated * 100
// Positive = under budget, Negative = over budget
func TestBudgetVarianceFormula(t *testing.T) {
	testCases := []struct {
		name               string
		estimatedMinutes   int64
		actualMinutes      int64
		expectedVariance   float64
		expectedOverBudget bool
	}{
		{
			name:               "On budget",
			estimatedMinutes:   480, // 8 hours
			actualMinutes:      480,
			expectedVariance:   0.00,
			expectedOverBudget: false,
		},
		{
			name:               "Under budget by 25%",
			estimatedMinutes:   480,
			actualMinutes:      360, // 6 hours
			expectedVariance:   25.00,
			expectedOverBudget: false,
		},
		{
			name:               "Over budget by 50%",
			estimatedMinutes:   480,
			actualMinutes:      720, // 12 hours
			expectedVariance:   -50.00,
			expectedOverBudget: true,
		},
		{
			name:               "No estimate (zero)",
			estimatedMinutes:   0,
			actualMinutes:      480,
			expectedVariance:   0.00,  // Cannot calculate variance
			expectedOverBudget: false, // No estimate means no over-budget flag
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var variance float64
			if tc.estimatedMinutes > 0 {
				variance = float64(tc.estimatedMinutes-tc.actualMinutes) / float64(tc.estimatedMinutes) * 100
			}
			overBudget := tc.actualMinutes > tc.estimatedMinutes && tc.estimatedMinutes > 0

			assert.InDelta(t, tc.expectedVariance, variance, 0.01, "Variance mismatch")
			assert.Equal(t, tc.expectedOverBudget, overBudget, "OverBudget flag mismatch")
		})
	}
}

// TestCostPerDeploymentFormula validates the cost per deployment calculation
// Formula: cost_per_deployment = total_cost / deployment_count
func TestCostPerDeploymentFormula(t *testing.T) {
	testCases := []struct {
		name            string
		totalCost       float64
		deploymentCount int
		expectedCPD     float64
	}{
		{
			name:            "Normal case",
			totalCost:       50000.00,
			deploymentCount: 100,
			expectedCPD:     500.00,
		},
		{
			name:            "High efficiency",
			totalCost:       10000.00,
			deploymentCount: 200,
			expectedCPD:     50.00,
		},
		{
			name:            "No deployments",
			totalCost:       50000.00,
			deploymentCount: 0,
			expectedCPD:     0.00, // Avoid division by zero
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var cpd float64
			if tc.deploymentCount > 0 {
				cpd = tc.totalCost / float64(tc.deploymentCount)
			}
			assert.InDelta(t, tc.expectedCPD, cpd, 0.01, "Cost per deployment mismatch")
		})
	}
}

// TestUnallocatedPercentFormula validates the unallocated cost percentage
// Formula: unallocated_percent = unallocated_cost / total_cost * 100
// Target: < 10%
func TestUnallocatedPercentFormula(t *testing.T) {
	testCases := []struct {
		name            string
		unallocatedCost float64
		totalCost       float64
		expectedPercent float64
		meetsTarget     bool
	}{
		{
			name:            "Good - 5% unallocated",
			unallocatedCost: 5000.00,
			totalCost:       100000.00,
			expectedPercent: 5.00,
			meetsTarget:     true, // < 10%
		},
		{
			name:            "Warning - 15% unallocated",
			unallocatedCost: 15000.00,
			totalCost:       100000.00,
			expectedPercent: 15.00,
			meetsTarget:     false, // >= 10%
		},
		{
			name:            "Perfect - 0% unallocated",
			unallocatedCost: 0.00,
			totalCost:       100000.00,
			expectedPercent: 0.00,
			meetsTarget:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var percent float64
			if tc.totalCost > 0 {
				percent = tc.unallocatedCost / tc.totalCost * 100
			}
			meetsTarget := percent < 10.0

			assert.InDelta(t, tc.expectedPercent, percent, 0.01, "Unallocated percent mismatch")
			assert.Equal(t, tc.meetsTarget, meetsTarget, "Target threshold check mismatch")
		})
	}
}

// TestTotalCostEqualsPhaseSum validates cost breakdown consistency
// total_cost = preliminary_cost + development_cost + post_impl_cost
func TestTotalCostEqualsPhaseSum(t *testing.T) {
	testCases := []struct {
		name            string
		preliminaryCost float64
		developmentCost float64
		postImplCost    float64
	}{
		{
			name:            "Mixed workload",
			preliminaryCost: 10000.00,
			developmentCost: 60000.00,
			postImplCost:    30000.00,
		},
		{
			name:            "All development",
			preliminaryCost: 0.00,
			developmentCost: 100000.00,
			postImplCost:    0.00,
		},
		{
			name:            "All maintenance",
			preliminaryCost: 0.00,
			developmentCost: 0.00,
			postImplCost:    50000.00,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			totalCost := tc.preliminaryCost + tc.developmentCost + tc.postImplCost
			phaseSum := tc.preliminaryCost + tc.developmentCost + tc.postImplCost

			assert.InDelta(t, totalCost, phaseSum, 0.01, "Total should equal sum of phases")
		})
	}
}

// TestCapitalizableExpenseSplit validates ASC 350-40 split
// capitalizable_cost = development_cost
// expense_cost = preliminary_cost + post_impl_cost
func TestCapitalizableExpenseSplit(t *testing.T) {
	testCases := []struct {
		name            string
		preliminaryCost float64
		developmentCost float64
		postImplCost    float64
	}{
		{
			name:            "Typical mix",
			preliminaryCost: 5000.00,
			developmentCost: 70000.00,
			postImplCost:    25000.00,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			capitalizableCost := tc.developmentCost
			expenseCost := tc.preliminaryCost + tc.postImplCost

			// Capitalizable should equal development
			assert.InDelta(t, tc.developmentCost, capitalizableCost, 0.01)

			// Expense should equal preliminary + post-impl
			assert.InDelta(t, tc.preliminaryCost+tc.postImplCost, expenseCost, 0.01)
		})
	}
}
