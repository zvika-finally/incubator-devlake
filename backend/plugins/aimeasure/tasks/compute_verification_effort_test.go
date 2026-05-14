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

import "testing"

func TestEstimateAuthorMinutes(t *testing.T) {
	cases := []struct {
		loc  int
		want int
	}{
		{0, 10},      // floor
		{50, 10},     // 10 min == floor
		{100, 20},
		{500, 100},
		{2000, 240},  // ceiling
		{10000, 240}, // ceiling holds
	}
	for _, c := range cases {
		if got := EstimateAuthorMinutes(c.loc); got != c.want {
			t.Errorf("EstimateAuthorMinutes(%d) = %d, want %d", c.loc, got, c.want)
		}
	}
}

func TestEstimateReviewerMinutes(t *testing.T) {
	cases := []struct {
		numComments int
		want        int
	}{
		{0, 15},   // baseline
		{5, 25},   // 15 + 2*5
		{30, 75},
		{60, 120}, // ceiling
		{100, 120},
	}
	for _, c := range cases {
		if got := EstimateReviewerMinutes(c.numComments); got != c.want {
			t.Errorf("EstimateReviewerMinutes(%d) = %d, want %d", c.numComments, got, c.want)
		}
	}
}

func TestSafeRatio(t *testing.T) {
	cases := []struct {
		num, denom int
		want       float64
	}{
		{0, 0, 0.0},
		{100, 0, 0.0},   // div by zero → 0
		{50, 100, 0.5},
		{300, 100, 3.0},
	}
	for _, c := range cases {
		got := SafeRatio(c.num, c.denom)
		if got != c.want {
			t.Errorf("SafeRatio(%d,%d) = %v, want %v", c.num, c.denom, got, c.want)
		}
	}
}
