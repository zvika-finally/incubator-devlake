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

	"github.com/apache/incubator-devlake/plugins/aidetector/tasks"
	"github.com/stretchr/testify/assert"
)

// Tests for AI Impact calculation logic

func TestCalculatePercentChange_PositiveIncrease(t *testing.T) {
	// Current is 20% higher than baseline
	result := tasks.CalculatePercentChange(100, 120)
	assert.InDelta(t, 20.0, result, 0.01)
}

func TestCalculatePercentChange_PositiveDecrease(t *testing.T) {
	// Current is 25% lower than baseline
	result := tasks.CalculatePercentChange(100, 75)
	assert.InDelta(t, -25.0, result, 0.01)
}

func TestCalculatePercentChange_NoChange(t *testing.T) {
	result := tasks.CalculatePercentChange(100, 100)
	assert.InDelta(t, 0.0, result, 0.01)
}

func TestCalculatePercentChange_ZeroBaseline(t *testing.T) {
	// Should return 0 to avoid division by zero
	result := tasks.CalculatePercentChange(0, 100)
	assert.Equal(t, 0.0, result)
}

func TestCalculatePercentChange_BothZero(t *testing.T) {
	result := tasks.CalculatePercentChange(0, 0)
	assert.Equal(t, 0.0, result)
}

func TestCalculatePercentChange_SmallValues(t *testing.T) {
	// Baseline 0.5, current 0.75 = 50% increase
	result := tasks.CalculatePercentChange(0.5, 0.75)
	assert.InDelta(t, 50.0, result, 0.01)
}

func TestCalculatePercentChange_LargeIncrease(t *testing.T) {
	// Current is 3x baseline = 200% increase
	result := tasks.CalculatePercentChange(10, 30)
	assert.InDelta(t, 200.0, result, 0.01)
}

// Test the impact interpretation
func TestAIImpactInterpretation_PRThroughputImprovement(t *testing.T) {
	// If baseline is 5 PRs/week and current is 7 PRs/week
	// That's a 40% improvement (positive is good for throughput)
	change := tasks.CalculatePercentChange(5, 7)
	assert.InDelta(t, 40.0, change, 0.01)
	assert.True(t, change > 0, "Higher throughput should show positive change")
}

func TestAIImpactInterpretation_ReviewTimeImprovement(t *testing.T) {
	// If baseline is 24 hours and current is 18 hours
	// That's -25% change, but we want to show it as improvement
	// So we invert the sign: faster review time = positive change
	rawChange := tasks.CalculatePercentChange(24, 18)
	assert.InDelta(t, -25.0, rawChange, 0.01)

	// After inversion (as done in the subtask):
	displayChange := -rawChange
	assert.InDelta(t, 25.0, displayChange, 0.01)
	assert.True(t, displayChange > 0, "Faster review time should show positive change after inversion")
}

func TestAIImpactInterpretation_LeadTimeRegression(t *testing.T) {
	// If baseline is 48 hours and current is 60 hours
	// That's +25% change (longer lead time = worse)
	// After inversion: -25% (negative = regression)
	rawChange := tasks.CalculatePercentChange(48, 60)
	assert.InDelta(t, 25.0, rawChange, 0.01)

	displayChange := -rawChange
	assert.InDelta(t, -25.0, displayChange, 0.01)
	assert.True(t, displayChange < 0, "Longer lead time should show negative change after inversion")
}
