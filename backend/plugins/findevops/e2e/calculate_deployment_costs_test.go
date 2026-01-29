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

	"github.com/apache/incubator-devlake/plugins/findevops/tasks"
	"github.com/stretchr/testify/assert"
)

// Tests for deployment cost calculation logic

func TestCalculateCostPerDeployment_NormalCase(t *testing.T) {
	// $10,000 cost / 100 deployments = $100/deployment
	result := tasks.CalculateCostPerDeployment(10000, 100)
	assert.Equal(t, 100.0, result)
}

func TestCalculateCostPerDeployment_ZeroDeployments(t *testing.T) {
	// Should return 0 to avoid division by zero
	result := tasks.CalculateCostPerDeployment(10000, 0)
	assert.Equal(t, 0.0, result)
}

func TestCalculateCostPerDeployment_NegativeDeployments(t *testing.T) {
	// Edge case: negative count should be treated as zero
	result := tasks.CalculateCostPerDeployment(10000, -5)
	assert.Equal(t, 0.0, result)
}

func TestCalculateCostPerDeployment_ZeroCost(t *testing.T) {
	// Zero cost is valid
	result := tasks.CalculateCostPerDeployment(0, 100)
	assert.Equal(t, 0.0, result)
}

func TestCalculateCostPerDeployment_FractionalResult(t *testing.T) {
	// $1000 cost / 3 deployments = $333.33...
	result := tasks.CalculateCostPerDeployment(1000, 3)
	assert.InDelta(t, 333.33, result, 0.01)
}

func TestCalculateCostPerDeployment_SingleDeployment(t *testing.T) {
	result := tasks.CalculateCostPerDeployment(5000, 1)
	assert.Equal(t, 5000.0, result)
}

func TestCostTimeWindows_ContainsExpectedValues(t *testing.T) {
	// Verify the time windows are as specified in the plan
	assert.Contains(t, tasks.CostTimeWindows, 7, "Should include 7-day window")
	assert.Contains(t, tasks.CostTimeWindows, 30, "Should include 30-day window")
	assert.Contains(t, tasks.CostTimeWindows, 90, "Should include 90-day window")
	assert.Equal(t, 3, len(tasks.CostTimeWindows), "Should have exactly 3 windows")
}

// Test cost trend scenarios
func TestCostPerDeployment_ImprovementScenario(t *testing.T) {
	// Before AI tools: $500/deployment
	// After AI tools: $300/deployment
	// Improvement: 40% reduction

	beforeCost := tasks.CalculateCostPerDeployment(50000, 100)
	afterCost := tasks.CalculateCostPerDeployment(30000, 100)

	improvement := ((beforeCost - afterCost) / beforeCost) * 100
	assert.InDelta(t, 40.0, improvement, 0.1, "Should show 40% cost reduction")
}

func TestCostPerDeployment_MoreFrequentDeploymentsScenario(t *testing.T) {
	// Same total cost, but more deployments = lower per-deployment cost
	// Month 1: $10k, 50 deploys = $200/deploy
	// Month 2: $10k, 100 deploys = $100/deploy

	month1 := tasks.CalculateCostPerDeployment(10000, 50)
	month2 := tasks.CalculateCostPerDeployment(10000, 100)

	assert.Equal(t, 200.0, month1)
	assert.Equal(t, 100.0, month2)
	assert.Less(t, month2, month1, "More frequent deployments should lower per-deployment cost")
}
