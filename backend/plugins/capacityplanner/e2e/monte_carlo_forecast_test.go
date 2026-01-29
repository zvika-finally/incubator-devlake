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
	"math/rand"
	"testing"

	"github.com/apache/incubator-devlake/plugins/capacityplanner/tasks"
	"github.com/stretchr/testify/assert"
)

// Tests for Monte Carlo forecasting logic

func TestNearestRankPercentile_EmptyData(t *testing.T) {
	result := tasks.NearestRankPercentile([]int{}, 50)
	assert.Equal(t, 0, result)
}

func TestNearestRankPercentile_SingleValue(t *testing.T) {
	result := tasks.NearestRankPercentile([]int{42}, 50)
	assert.Equal(t, 42, result)
}

func TestNearestRankPercentile_P50Median(t *testing.T) {
	// Sorted: [10, 20, 30, 40, 50]
	// P50 should be ~30 (median)
	data := []int{10, 20, 30, 40, 50}
	result := tasks.NearestRankPercentile(data, 50)
	// Index = 5 * 50 / 100 = 2, so data[2] = 30
	assert.Equal(t, 30, result)
}

func TestNearestRankPercentile_P90(t *testing.T) {
	// 10 values: [1,2,3,4,5,6,7,8,9,10]
	// P90 index = 10 * 90 / 100 = 9, data[9] = 10
	data := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	result := tasks.NearestRankPercentile(data, 90)
	assert.Equal(t, 10, result)
}

func TestNearestRankPercentile_P75(t *testing.T) {
	// 8 values: [1,2,3,4,5,6,7,8]
	// P75 index = 8 * 75 / 100 = 6, data[6] = 7
	data := []int{1, 2, 3, 4, 5, 6, 7, 8}
	result := tasks.NearestRankPercentile(data, 75)
	assert.Equal(t, 7, result)
}

func TestNearestRankPercentile_P0(t *testing.T) {
	data := []int{10, 20, 30, 40, 50}
	result := tasks.NearestRankPercentile(data, 0)
	assert.Equal(t, 10, result)
}

func TestNearestRankPercentile_P100(t *testing.T) {
	data := []int{10, 20, 30, 40, 50}
	result := tasks.NearestRankPercentile(data, 100)
	// Should return last element
	assert.Equal(t, 50, result)
}

func TestGaussianRandom_MeanAndStdDev(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	// Generate 10000 samples
	sum := 0.0
	count := 10000
	mean := 20.0
	stddev := 5.0

	for i := 0; i < count; i++ {
		sum += tasks.GaussianRandom(rng, mean, stddev)
	}

	// Average should be close to mean
	avg := sum / float64(count)
	assert.InDelta(t, mean, avg, 0.5, "Average should be close to mean")
}

func TestGaussianRandom_Distribution(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	mean := 100.0
	stddev := 10.0

	// Count how many fall within 1, 2, 3 standard deviations
	within1Sigma := 0
	within2Sigma := 0
	within3Sigma := 0
	count := 10000

	for i := 0; i < count; i++ {
		value := tasks.GaussianRandom(rng, mean, stddev)
		diff := abs(value - mean)

		if diff <= stddev {
			within1Sigma++
		}
		if diff <= 2*stddev {
			within2Sigma++
		}
		if diff <= 3*stddev {
			within3Sigma++
		}
	}

	// ~68% should be within 1 sigma
	pct1 := float64(within1Sigma) / float64(count) * 100
	assert.InDelta(t, 68.0, pct1, 3.0, "~68%% should be within 1 sigma")

	// ~95% should be within 2 sigma
	pct2 := float64(within2Sigma) / float64(count) * 100
	assert.InDelta(t, 95.0, pct2, 2.0, "~95%% should be within 2 sigma")

	// ~99.7% should be within 3 sigma
	pct3 := float64(within3Sigma) / float64(count) * 100
	assert.InDelta(t, 99.7, pct3, 1.0, "~99.7%% should be within 3 sigma")
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
