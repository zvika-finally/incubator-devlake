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

package models

import "testing"

func TestTableNames(t *testing.T) {
	cases := map[string]string{
		"pr_ai_cohort":                PRAICohort{}.TableName(),
		"pr_defect_signals":           PRDefectSignals{}.TableName(),
		"pr_change_composition":       PRChangeComposition{}.TableName(),
		"aimeasure_account_overrides": AccountOverride{}.TableName(),
		"aimeasure_engineer_roles":    EngineerRole{}.TableName(),
	}
	for expected, actual := range cases {
		if expected != actual {
			t.Errorf("expected table name %q, got %q", expected, actual)
		}
	}
}

func TestPhaseBTableNames(t *testing.T) {
	cases := map[string]string{
		"engineer_verification_effort":       EngineerVerificationEffort{}.TableName(),
		"engineer_slack_signals":             EngineerSlackSignals{}.TableName(),
		"engineer_dxi_proxy":                 EngineerDxiProxy{}.TableName(),
		"aimeasure_slack_channel_categories": SlackChannelCategory{}.TableName(),
	}
	for expected, actual := range cases {
		if expected != actual {
			t.Errorf("expected table name %q, got %q", expected, actual)
		}
	}
}
