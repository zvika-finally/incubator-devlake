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
	"time"

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

func TestResolveClassifiedAt_NewRowReturnsNow(t *testing.T) {
	now := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	fresh := models.PRAICohort{AICohort: models.CohortHigh, ConfidenceScore: 90, ClassifierVersion: "v1"}
	got := ResolveClassifiedAt(nil, fresh, now)
	if !got.Equal(now) {
		t.Errorf("new row should get current time; got %v want %v", got, now)
	}
}

func TestResolveClassifiedAt_NoChangePreservesTimestamp(t *testing.T) {
	earlier := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	now := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	existing := models.PRAICohort{
		AICohort: models.CohortHigh, ConfidenceScore: 90,
		HasExplicitMarker: true, HasCommitTrailer: false,
		ClassifierVersion: "v1", ClassifiedAt: earlier,
	}
	fresh := existing
	fresh.ClassifiedAt = time.Time{} // caller hasn't set it yet
	got := ResolveClassifiedAt(&existing, fresh, now)
	if !got.Equal(earlier) {
		t.Errorf("identical classification should preserve timestamp; got %v want %v", got, earlier)
	}
}

func TestResolveClassifiedAt_DimensionChangeBumpsTimestamp(t *testing.T) {
	earlier := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	now := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	base := models.PRAICohort{
		AICohort: models.CohortHigh, ConfidenceScore: 90,
		HasExplicitMarker: true, HasCommitTrailer: false,
		ClassifierVersion: "v1", ClassifiedAt: earlier,
	}
	cases := []struct {
		name   string
		mutate func(*models.PRAICohort)
	}{
		{"cohort changed", func(p *models.PRAICohort) { p.AICohort = models.CohortMedium }},
		{"confidence changed", func(p *models.PRAICohort) { p.ConfidenceScore = 50 }},
		{"explicit marker changed", func(p *models.PRAICohort) { p.HasExplicitMarker = false }},
		{"commit trailer changed", func(p *models.PRAICohort) { p.HasCommitTrailer = true }},
		{"classifier version changed", func(p *models.PRAICohort) { p.ClassifierVersion = "v2" }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fresh := base
			c.mutate(&fresh)
			got := ResolveClassifiedAt(&base, fresh, now)
			if !got.Equal(now) {
				t.Errorf("%s should bump timestamp; got %v want %v", c.name, got, now)
			}
		})
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
