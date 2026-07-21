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
	"encoding/json"
	"testing"
	"time"

	"github.com/apache/incubator-devlake/plugins/claude_code/models"
	"github.com/stretchr/testify/assert"
)

func TestParseAnalyticsDateFromDayInput(t *testing.T) {
	input, err := json.Marshal(claudeCodeDayInput{Day: "2026-03-04"})
	assert.NoError(t, err)

	date, parseErr := parseAnalyticsDate(input)

	assert.Nil(t, parseErr)
	assert.Equal(t, time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC), date)
}

func TestParseAnalyticsDateFromRangeInput(t *testing.T) {
	input, err := json.Marshal(claudeCodeDateRangeInput{StartDate: "2026-03-04", EndDate: "2026-03-05"})
	assert.NoError(t, err)

	date, parseErr := parseAnalyticsDate(input)

	assert.Nil(t, parseErr)
	assert.Equal(t, time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC), date)
}

func TestParseAnalyticsDateReturnsErrorForInvalidInput(t *testing.T) {
	date, parseErr := parseAnalyticsDate(json.RawMessage(`{"unknown":"2026-03-04"}`))

	assert.NotNil(t, parseErr)
	assert.True(t, date.IsZero())
}

func TestExtractConsoleUserActivityAggregatesTokensAndCost(t *testing.T) {
	raw := []byte(`{
		"date": "2026-07-20",
		"actor": {"type": "user_actor", "email_address": "dev@example.com"},
		"core_metrics": {"num_sessions": 3, "lines_of_code": {"added": 10, "removed": 2}},
		"tool_actions": {"edit_tool": {"accepted": 5, "rejected": 1}},
		"model_breakdown": [
			{"model": "claude-opus", "tokens": {"input": 100, "output": 200, "cache_read": 300, "cache_creation": 400}, "estimated_cost": {"currency": "USD", "amount": 150}},
			{"model": "claude-sonnet", "tokens": {"input": 1, "output": 2, "cache_read": 3, "cache_creation": 4}, "estimated_cost": {"currency": "USD", "amount": 50}}
		]
	}`)
	data := &ClaudeCodeTaskData{Options: &ClaudeCodeOptions{ConnectionId: 1, ScopeId: "org1"}}

	results, err := extractConsoleUserActivity(data, time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC), raw)

	assert.Nil(t, err)
	assert.Len(t, results, 1)
	activity := results[0].(*models.ClaudeCodeUserActivity)
	assert.Equal(t, int64(101), activity.InputTokens)
	assert.Equal(t, int64(202), activity.OutputTokens)
	assert.Equal(t, int64(303), activity.CacheReadTokens)
	assert.Equal(t, int64(404), activity.CacheCreationTokens)
	assert.Equal(t, int64(200), activity.EstimatedCostCents)
	assert.Equal(t, "USD", activity.CostCurrency)
	// sanity: core metrics still populated
	assert.Equal(t, 3, activity.CCSessionCount)
	assert.Equal(t, "dev@example.com", activity.UserEmail)
}

func TestExtractConsoleUserActivityNoModelBreakdown(t *testing.T) {
	raw := []byte(`{
		"date": "2026-07-20",
		"actor": {"type": "user_actor", "email_address": "dev@example.com"},
		"core_metrics": {"num_sessions": 1, "lines_of_code": {"added": 1, "removed": 0}},
		"tool_actions": {}
	}`)
	data := &ClaudeCodeTaskData{Options: &ClaudeCodeOptions{ConnectionId: 1, ScopeId: "org1"}}

	results, err := extractConsoleUserActivity(data, time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC), raw)

	assert.Nil(t, err)
	assert.Len(t, results, 1)
	activity := results[0].(*models.ClaudeCodeUserActivity)
	assert.Equal(t, int64(0), activity.InputTokens)
	assert.Equal(t, int64(0), activity.EstimatedCostCents)
	assert.Equal(t, "", activity.CostCurrency)
}
