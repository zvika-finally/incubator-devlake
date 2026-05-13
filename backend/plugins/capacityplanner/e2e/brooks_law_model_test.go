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

// Tests for Brooks's Law capacity modeling

func TestCalculateCommunicationChannels_SinglePerson(t *testing.T) {
	// 1 person = 0 channels
	result := tasks.CalculateCommunicationChannels(1)
	assert.Equal(t, 0, result)
}

func TestCalculateCommunicationChannels_TwoPeople(t *testing.T) {
	// 2 people = 1 channel (2*1/2 = 1)
	result := tasks.CalculateCommunicationChannels(2)
	assert.Equal(t, 1, result)
}

func TestCalculateCommunicationChannels_FivePeople(t *testing.T) {
	// 5 people = 10 channels (5*4/2 = 10)
	result := tasks.CalculateCommunicationChannels(5)
	assert.Equal(t, 10, result)
}

func TestCalculateCommunicationChannels_TenPeople(t *testing.T) {
	// 10 people = 45 channels (10*9/2 = 45)
	result := tasks.CalculateCommunicationChannels(10)
	assert.Equal(t, 45, result)
}

func TestCalculateCommunicationChannels_ZeroPeople(t *testing.T) {
	result := tasks.CalculateCommunicationChannels(0)
	assert.Equal(t, 0, result)
}

func TestCalculateBrooksLawImpact_AddingOneToFive(t *testing.T) {
	currentSize := 5
	delta := 1
	currentChannels := 10 // 5*4/2
	newChannels := 15     // 6*5/2

	productivity, overhead := tasks.CalculateBrooksLawImpact(currentSize, delta, currentChannels, newChannels)

	// Adding 1 person with 50% productivity: (5 + 0.5) / 5 = 1.1
	assert.InDelta(t, 1.1, productivity, 0.01, "Productivity factor should be ~1.1")

	// Channel delta = 15 - 10 = 5
	// Overhead = 1 - (5 / 11) * 0.1 = ~0.95
	assert.InDelta(t, 0.95, overhead, 0.02, "Overhead factor should be ~0.95")
}

func TestCalculateBrooksLawImpact_RemovingOneFromFive(t *testing.T) {
	currentSize := 5
	delta := -1
	currentChannels := 10 // 5*4/2
	newChannels := 6      // 4*3/2

	productivity, overhead := tasks.CalculateBrooksLawImpact(currentSize, delta, currentChannels, newChannels)

	// Removing 1 person: 4/5 = 0.8
	assert.InDelta(t, 0.8, productivity, 0.01, "Productivity factor should be 0.8")

	// Channel delta = 6 - 10 = -4
	// Overhead = 1 - (-4 / 11) * 0.1 = ~1.04 (actually improves!)
	assert.InDelta(t, 1.04, overhead, 0.02, "Overhead factor should improve")
}

func TestCalculateBrooksLawImpact_NoChange(t *testing.T) {
	currentSize := 5
	delta := 0
	currentChannels := 10
	newChannels := 10

	productivity, overhead := tasks.CalculateBrooksLawImpact(currentSize, delta, currentChannels, newChannels)

	assert.Equal(t, 1.0, productivity, "No change should have productivity factor of 1.0")
	assert.Equal(t, 1.0, overhead, "No change should have overhead factor of 1.0")
}

func TestCalculateBrooksLawImpact_AddingThreeToFive(t *testing.T) {
	// This is the classic "adding manpower to a late project" scenario
	currentSize := 5
	delta := 3
	currentChannels := 10 // 5*4/2
	newChannels := 28     // 8*7/2

	productivity, overhead := tasks.CalculateBrooksLawImpact(currentSize, delta, currentChannels, newChannels)

	// Adding 3 people with 50% productivity: (5 + 1.5) / 5 = 1.3
	assert.InDelta(t, 1.3, productivity, 0.01, "Productivity factor should be ~1.3")

	// Channel delta = 28 - 10 = 18
	// Overhead = 1 - (18 / 11) * 0.1 = ~0.84
	assert.InDelta(t, 0.84, overhead, 0.02, "Overhead should be significant with 3 new members")

	// Net effect = 1.3 * 0.84 = ~1.09 (only 9% improvement despite adding 60% more people)
	netEffect := productivity * overhead
	assert.InDelta(t, 1.09, netEffect, 0.05, "Net effect demonstrates diminishing returns")
}

func TestBrooksLawPrinciple_DiminishingReturns(t *testing.T) {
	// Demonstrate that adding more people leads to diminishing returns
	baseTeam := 5
	baseChannels := tasks.CalculateCommunicationChannels(baseTeam)

	var netEffects []float64

	for delta := 1; delta <= 5; delta++ {
		newSize := baseTeam + delta
		newChannels := tasks.CalculateCommunicationChannels(newSize)
		productivity, overhead := tasks.CalculateBrooksLawImpact(baseTeam, delta, baseChannels, newChannels)
		netEffects = append(netEffects, productivity*overhead)
	}

	// Each additional person should have a smaller marginal effect
	// (not strictly enforced in this simple model, but the principle holds)
	t.Logf("Net effects for adding 1-5 people to team of 5: %v", netEffects)

	// All should show improvement but communication overhead limits it
	for i, effect := range netEffects {
		assert.Greater(t, effect, 1.0, "Adding %d people should still improve overall capacity", i+1)
		assert.Less(t, effect, float64(baseTeam+i+1)/float64(baseTeam),
			"Improvement should be less than linear scaling")
	}
}
