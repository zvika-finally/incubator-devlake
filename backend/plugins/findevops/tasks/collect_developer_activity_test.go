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

func TestCalculateActivityScore(t *testing.T) {
	weights := &ActivityWeights{
		PrAuthored:     1.0,
		PrReviewed:     0.3,
		CommitAuthored: 0.2,
		IssueUpdated:   0.1,
		CommentAdded:   0.05,
	}

	testCases := []struct {
		name          string
		prsAuthored   int
		prsReviewed   int
		commits       int
		issuesUpdated int
		comments      int
		expectedScore float64
	}{
		{
			name:          "Typical developer month",
			prsAuthored:   10,
			prsReviewed:   15,
			commits:       50,
			issuesUpdated: 20,
			comments:      30,
			expectedScore: 28.0,
		},
		{
			name:          "Zero activity",
			prsAuthored:   0,
			prsReviewed:   0,
			commits:       0,
			issuesUpdated: 0,
			comments:      0,
			expectedScore: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := calculateActivityScore(
				tc.prsAuthored,
				tc.prsReviewed,
				tc.commits,
				tc.issuesUpdated,
				tc.comments,
				weights,
			)
			assert.InDelta(t, tc.expectedScore, score, 0.01)
		})
	}
}

func TestCalculateFte(t *testing.T) {
	testCases := []struct {
		name          string
		rawScore      float64
		baselineScore float64
		maxFte        float64
		expectedFte   float64
	}{
		{
			name:          "Full-time developer",
			rawScore:      30.0,
			baselineScore: 25.0,
			maxFte:        1.0,
			expectedFte:   1.0,
		},
		{
			name:          "Part-time developer",
			rawScore:      12.5,
			baselineScore: 25.0,
			maxFte:        1.0,
			expectedFte:   0.5,
		},
		{
			name:          "Inactive developer",
			rawScore:      0,
			baselineScore: 25.0,
			maxFte:        1.0,
			expectedFte:   0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fte := calculateFte(tc.rawScore, tc.baselineScore, tc.maxFte)
			assert.InDelta(t, tc.expectedFte, fte, 0.01)
		})
	}
}
