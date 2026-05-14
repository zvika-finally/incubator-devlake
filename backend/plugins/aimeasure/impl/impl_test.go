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

package impl

import (
	"encoding/json"
	"testing"
)

func TestMakeMetricPluginPipelinePlanV200_EmptyOptions(t *testing.T) {
	var p AIMeasure
	plan, err := p.MakeMetricPluginPipelinePlanV200("demo", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(plan) != 1 || len(plan[0]) != 1 {
		t.Fatalf("expected single-stage single-task plan, got %v", plan)
	}
	task := plan[0][0]
	if task.Plugin != "aimeasure" {
		t.Errorf("expected plugin 'aimeasure', got %q", task.Plugin)
	}
	if task.Options["projectName"] != "demo" {
		t.Errorf("expected projectName 'demo', got %v", task.Options["projectName"])
	}
	if len(task.Subtasks) != 3 {
		t.Errorf("expected 3 subtasks, got %d: %v", len(task.Subtasks), task.Subtasks)
	}
}

func TestMakeMetricPluginPipelinePlanV200_WithOptions(t *testing.T) {
	var p AIMeasure
	opts, _ := json.Marshal(map[string]interface{}{
		"highCohortThreshold": 70,
		"lowCohortThreshold":  40,
		"defectWindowDays":    21,
	})
	plan, err := p.MakeMetricPluginPipelinePlanV200("demo", opts)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	task := plan[0][0]
	if task.Options["highCohortThreshold"] != 70 {
		t.Errorf("expected highCohortThreshold 70, got %v", task.Options["highCohortThreshold"])
	}
	if task.Options["defectWindowDays"] != 21 {
		t.Errorf("expected defectWindowDays 21, got %v", task.Options["defectWindowDays"])
	}
	if task.Options["projectName"] != "demo" {
		t.Errorf("expected projectName from arg to override options, got %v", task.Options["projectName"])
	}
}

func TestRunAfter_IncludesAIDetector(t *testing.T) {
	var p AIMeasure
	deps, err := p.RunAfter()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, d := range deps {
		if d == "aidetector" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected aidetector in RunAfter() deps, got %v", deps)
	}
}

func TestRequiredDataEntities_ListsUpstreamModels(t *testing.T) {
	var p AIMeasure
	entities, err := p.RequiredDataEntities()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := map[string]bool{
		"pull_requests":        false,
		"commits":              false,
		"commit_files":         false,
		"pull_request_commits": false,
	}
	for _, e := range entities {
		if m, ok := e["model"].(string); ok {
			if _, want := expected[m]; want {
				expected[m] = true
			}
		}
	}
	for m, seen := range expected {
		if !seen {
			t.Errorf("expected model %q in RequiredDataEntities, missing", m)
		}
	}
}

func TestIsProjectMetric(t *testing.T) {
	var p AIMeasure
	if !p.IsProjectMetric() {
		t.Error("expected IsProjectMetric() to be true")
	}
}
