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

package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTopologicalSort_Simple(t *testing.T) {
	// Test: A depends on B
	deps := map[string][]string{
		"A": {"B"},
		"B": {},
	}

	sorted, err := topologicalSort(deps)
	assert.NoError(t, err)
	assert.Equal(t, []string{"B", "A"}, sorted)
}

func TestTopologicalSort_NoDependencies(t *testing.T) {
	// Test: Independent plugins
	deps := map[string][]string{
		"A": {},
		"B": {},
		"C": {},
	}

	sorted, err := topologicalSort(deps)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(sorted))
	// All should be present (order doesn't matter for independent plugins)
	assert.Contains(t, sorted, "A")
	assert.Contains(t, sorted, "B")
	assert.Contains(t, sorted, "C")
}

func TestTopologicalSort_Diamond(t *testing.T) {
	// Test: D depends on B,C; B,C depend on A
	deps := map[string][]string{
		"A": {},
		"B": {"A"},
		"C": {"A"},
		"D": {"B", "C"},
	}

	sorted, err := topologicalSort(deps)
	assert.NoError(t, err)
	assert.Equal(t, 4, len(sorted))
	assert.Equal(t, "A", sorted[0]) // A must be first
	assert.Equal(t, "D", sorted[3]) // D must be last
}

func TestTopologicalSort_Cycle(t *testing.T) {
	// Test: A -> B -> C -> A (cycle)
	deps := map[string][]string{
		"A": {"C"},
		"B": {"A"},
		"C": {"B"},
	}

	_, err := topologicalSort(deps)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circular dependency")
}

func TestTopologicalSort_SelfCycle(t *testing.T) {
	// Test: A depends on itself
	deps := map[string][]string{
		"A": {"A"},
	}

	_, err := topologicalSort(deps)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circular dependency")
}

func TestTopologicalSort_RealMetricPlugins(t *testing.T) {
	// Test: Real DevLake metric plugin dependencies
	deps := map[string][]string{
		"dora":            {},
		"aidetector":      {"dora"},
		"businessmetrics": {"dora"},
		"capacityplanner": {}, // No dependencies - reads only from domain tables
		"findevops":       {"businessmetrics"},
	}

	sorted, err := topologicalSort(deps)
	assert.NoError(t, err)
	assert.Equal(t, 5, len(sorted))

	// Verify order constraints
	doraIdx := indexOf(sorted, "dora")
	aiIdx := indexOf(sorted, "aidetector")
	bizIdx := indexOf(sorted, "businessmetrics")
	capIdx := indexOf(sorted, "capacityplanner")
	finIdx := indexOf(sorted, "findevops")

	assert.True(t, doraIdx < aiIdx, "dora must run before aidetector")
	assert.True(t, doraIdx < bizIdx, "dora must run before businessmetrics")
	assert.True(t, bizIdx < finIdx, "businessmetrics must run before findevops")
	// capacityplanner has no dependencies, so it can run anytime
	_ = capIdx // Suppress unused variable warning
}

func TestTopologicalSort_LinearChain(t *testing.T) {
	// Test: A -> B -> C -> D (linear chain)
	deps := map[string][]string{
		"A": {},
		"B": {"A"},
		"C": {"B"},
		"D": {"C"},
	}

	sorted, err := topologicalSort(deps)
	assert.NoError(t, err)
	assert.Equal(t, []string{"A", "B", "C", "D"}, sorted)
}

func TestTopologicalSort_ComplexDependencies(t *testing.T) {
	// Test: Complex dependency graph
	// E depends on C,D; C depends on A; D depends on A,B
	deps := map[string][]string{
		"A": {},
		"B": {},
		"C": {"A"},
		"D": {"A", "B"},
		"E": {"C", "D"},
	}

	sorted, err := topologicalSort(deps)
	assert.NoError(t, err)
	assert.Equal(t, 5, len(sorted))

	// Verify order constraints
	aIdx := indexOf(sorted, "A")
	bIdx := indexOf(sorted, "B")
	cIdx := indexOf(sorted, "C")
	dIdx := indexOf(sorted, "D")
	eIdx := indexOf(sorted, "E")

	assert.True(t, aIdx < cIdx, "A must run before C")
	assert.True(t, aIdx < dIdx, "A must run before D")
	assert.True(t, bIdx < dIdx, "B must run before D")
	assert.True(t, cIdx < eIdx, "C must run before E")
	assert.True(t, dIdx < eIdx, "D must run before E")
}

// Helper function to find index of element in slice
func indexOf(slice []string, value string) int {
	for i, v := range slice {
		if v == value {
			return i
		}
	}
	return -1
}
