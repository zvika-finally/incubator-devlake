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

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateGitInferredHours(t *testing.T) {
	config := &GitInferenceConfig{
		ProductiveHoursPerActiveDay: 6.0,
		ReviewHoursPerCycle:         1.5,
		CommentsPerReviewCycle:      3,
		MinHoursPerIssue:            1.0,
		MaxHoursPerIssue:            80.0,
	}

	testCases := []struct {
		name           string
		activeDays     int
		reviewComments int
		linesChanged   int
		filesChanged   int
		expectedMin    float64
		expectedMax    float64
	}{
		{
			name:           "Small task - 1 day, no reviews",
			activeDays:     1,
			reviewComments: 0,
			linesChanged:   50,
			filesChanged:   2,
			expectedMin:    1.0,
			expectedMax:    15.0,
		},
		{
			name:           "Large task - capped at max",
			activeDays:     20,
			reviewComments: 30,
			linesChanged:   5000,
			filesChanged:   50,
			expectedMin:    80.0,
			expectedMax:    80.0,
		},
		{
			name:           "Zero activity - returns zero",
			activeDays:     0,
			reviewComments: 0,
			linesChanged:   0,
			filesChanged:   0,
			expectedMin:    0.0,
			expectedMax:    0.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hours := calculateGitInferredHours(
				tc.activeDays,
				tc.reviewComments,
				tc.linesChanged,
				tc.filesChanged,
				config,
			)
			assert.GreaterOrEqual(t, hours, tc.expectedMin)
			assert.LessOrEqual(t, hours, tc.expectedMax)
		})
	}
}

func TestCalculateComplexityFactor(t *testing.T) {
	testCases := []struct {
		name         string
		linesChanged int
		filesChanged int
		expectedMin  float64
		expectedMax  float64
	}{
		{
			name:         "Simple change",
			linesChanged: 10,
			filesChanged: 1,
			expectedMin:  0.5,
			expectedMax:  2.0,
		},
		{
			name:         "Complex change",
			linesChanged: 1000,
			filesChanged: 20,
			expectedMin:  2.0,
			expectedMax:  5.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			factor := calculateComplexityFactor(tc.linesChanged, tc.filesChanged)
			assert.GreaterOrEqual(t, factor, tc.expectedMin)
			assert.LessOrEqual(t, factor, tc.expectedMax)
		})
	}
}
