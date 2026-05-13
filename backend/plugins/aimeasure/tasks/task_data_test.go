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

func TestDecodeAndValidateTaskOptions_AppliesDefaults(t *testing.T) {
	opts, err := DecodeAndValidateTaskOptions(map[string]interface{}{
		"projectName": "demo",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if opts.ProjectName != "demo" {
		t.Errorf("expected projectName 'demo', got %q", opts.ProjectName)
	}
	if opts.HighCohortThreshold != 65 {
		t.Errorf("expected default threshold 65, got %d", opts.HighCohortThreshold)
	}
	if opts.LowCohortThreshold != 30 {
		t.Errorf("expected default low threshold 30, got %d", opts.LowCohortThreshold)
	}
	if opts.DefectWindowDays != 14 {
		t.Errorf("expected default window 14, got %d", opts.DefectWindowDays)
	}
}

func TestDecodeAndValidateTaskOptions_AcceptsOverrides(t *testing.T) {
	opts, err := DecodeAndValidateTaskOptions(map[string]interface{}{
		"projectName":         "demo",
		"highCohortThreshold": 70,
		"lowCohortThreshold":  40,
		"defectWindowDays":    21,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if opts.HighCohortThreshold != 70 {
		t.Errorf("expected 70, got %d", opts.HighCohortThreshold)
	}
	if opts.LowCohortThreshold != 40 {
		t.Errorf("expected 40, got %d", opts.LowCohortThreshold)
	}
	if opts.DefectWindowDays != 21 {
		t.Errorf("expected 21, got %d", opts.DefectWindowDays)
	}
}
