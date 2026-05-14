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

func TestSentimentScore(t *testing.T) {
	cases := []struct {
		name            string
		afterHoursRatio float64
		reviewToAuthor  float64
		messageDropPct  float64 // 0..1 fraction of WoW drop
		want            float64 // approx
	}{
		{"perfect week", 0.0, 0.8, 0.0, 100},
		{"all night work", 1.0, 0.8, 0.0, 60},  // -40
		{"heavy reviewer", 0.0, 3.0, 0.0, 70},  // (3-1.5)/1.5=1.0 → -30
		{"disengaged", 0.0, 0.8, 0.5, 90},      // -10
		{"compound burnout", 1.0, 3.0, 0.6, 20}, // -40 -30 -10
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := SentimentScore(c.afterHoursRatio, c.reviewToAuthor, c.messageDropPct)
			if got < c.want-0.5 || got > c.want+0.5 {
				t.Errorf("got %v, want approx %v", got, c.want)
			}
		})
	}
}

func TestBadDeveloperDayFlag(t *testing.T) {
	cases := []struct {
		name            string
		score           float64
		afterHoursRatio float64
		messageDropPct  float64
		want            bool
	}{
		{"healthy week", 90, 0.0, 0.0, false},
		{"score below 50", 40, 0.0, 0.0, true},
		{"after-hours + disengaged", 70, 0.20, 0.60, true},
		{"after-hours alone", 70, 0.20, 0.10, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := BadDeveloperDayFlag(c.score, c.afterHoursRatio, c.messageDropPct); got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}
