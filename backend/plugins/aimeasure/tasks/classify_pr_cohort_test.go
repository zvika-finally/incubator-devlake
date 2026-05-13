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

	"github.com/apache/incubator-devlake/plugins/aimeasure/models"
)

func TestClassify_ExplicitMarkerOverridesScore(t *testing.T) {
	input := ClassifyInput{
		ConfidenceScore:   10,
		HasExplicitMarker: true,
		HasCommitTrailer:  false,
		HighThreshold:     65,
		LowThreshold:      30,
	}
	if got := Classify(input); got != models.CohortHigh {
		t.Errorf("expected HIGH due to explicit marker, got %s", got)
	}
}

func TestClassify_CommitTrailerOverridesScore(t *testing.T) {
	input := ClassifyInput{
		ConfidenceScore:   10,
		HasExplicitMarker: false,
		HasCommitTrailer:  true,
		HighThreshold:     65,
		LowThreshold:      30,
	}
	if got := Classify(input); got != models.CohortHigh {
		t.Errorf("expected HIGH due to commit trailer, got %s", got)
	}
}

func TestClassify_MediumByScore(t *testing.T) {
	input := ClassifyInput{
		ConfidenceScore: 75,
		HighThreshold:   65,
		LowThreshold:    30,
	}
	if got := Classify(input); got != models.CohortMedium {
		t.Errorf("expected MEDIUM, got %s", got)
	}
}

func TestClassify_LowByScore(t *testing.T) {
	input := ClassifyInput{
		ConfidenceScore: 45,
		HighThreshold:   65,
		LowThreshold:    30,
	}
	if got := Classify(input); got != models.CohortLow {
		t.Errorf("expected LOW, got %s", got)
	}
}

func TestClassify_NoneByScore(t *testing.T) {
	input := ClassifyInput{
		ConfidenceScore: 15,
		HighThreshold:   65,
		LowThreshold:    30,
	}
	if got := Classify(input); got != models.CohortNone {
		t.Errorf("expected NONE, got %s", got)
	}
}

func TestClassify_ThresholdEdgeCases(t *testing.T) {
	hi := ClassifyInput{ConfidenceScore: 65, HighThreshold: 65, LowThreshold: 30}
	if got := Classify(hi); got != models.CohortMedium {
		t.Errorf("score == high threshold should be MEDIUM, got %s", got)
	}
	lo := ClassifyInput{ConfidenceScore: 30, HighThreshold: 65, LowThreshold: 30}
	if got := Classify(lo); got != models.CohortLow {
		t.Errorf("score == low threshold should be LOW, got %s", got)
	}
	just := ClassifyInput{ConfidenceScore: 29, HighThreshold: 65, LowThreshold: 30}
	if got := Classify(just); got != models.CohortNone {
		t.Errorf("score < low threshold should be NONE, got %s", got)
	}
}

func TestHasAITrailer(t *testing.T) {
	cases := []struct {
		name  string
		input []string
		want  bool
	}{
		{"empty", nil, false},
		{"plain commit", []string{"fix: nil deref in widget"}, false},
		{"claude trailer", []string{"feat: thing\n\nCo-authored-by: Claude <claude@anthropic.com>"}, true},
		{"copilot trailer", []string{"feat: thing\n\nCo-authored-by: GitHub Copilot <copilot@github.com>"}, true},
		{"cursor trailer", []string{"feat: thing\n\nCo-authored-by: Cursor"}, true},
		{"trailer in second commit", []string{"a", "b\n\nCo-authored-by: Claude"}, true},
		{"trailer in body only is fine too", []string{"Refactor stuff\nCo-authored-by: Claude"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := HasAITrailer(c.input); got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}
