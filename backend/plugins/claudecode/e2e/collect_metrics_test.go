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

package e2e

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/apache/incubator-devlake/plugins/claudecode/models"
)

func TestClaudeCodeAPIResponseParsing(t *testing.T) {
	// Simulated Claude Code Analytics Admin API response
	// Docs: https://docs.anthropic.com/en/api/claude-code-analytics-api
	apiResponse := `{
		"data": [{
			"date": "2025-09-01T00:00:00Z",
			"actor": {
				"type": "user_actor",
				"email_address": "developer@company.com"
			},
			"organization_id": "dc9f6c26-b22c-4831-8d01-0446bada88f1",
			"customer_type": "api",
			"terminal_type": "vscode",
			"core_metrics": {
				"num_sessions": 5,
				"lines_of_code": {
					"added": 1543,
					"removed": 892
				},
				"commits_by_claude_code": 12,
				"pull_requests_by_claude_code": 2
			},
			"tool_actions": {
				"edit_tool": {"accepted": 45, "rejected": 5},
				"multi_edit_tool": {"accepted": 12, "rejected": 2},
				"write_tool": {"accepted": 8, "rejected": 1},
				"notebook_edit_tool": {"accepted": 3, "rejected": 0}
			},
			"model_breakdown": [{
				"model": "claude-sonnet-4-5-20250929",
				"tokens": {
					"input": 100000,
					"output": 35000,
					"cache_read": 10000,
					"cache_creation": 5000
				},
				"estimated_cost": {
					"currency": "USD",
					"amount": 1025
				}
			}]
		}],
		"has_more": false,
		"next_page": null
	}`

	var data map[string]interface{}
	err := json.Unmarshal([]byte(apiResponse), &data)
	assert.NoError(t, err)

	dataArr := data["data"].([]interface{})
	record := dataArr[0].(map[string]interface{})

	actor := record["actor"].(map[string]interface{})
	assert.Equal(t, "user_actor", actor["type"])
	assert.Equal(t, "developer@company.com", actor["email_address"])

	coreMetrics := record["core_metrics"].(map[string]interface{})
	assert.Equal(t, float64(5), coreMetrics["num_sessions"])

	linesOfCode := coreMetrics["lines_of_code"].(map[string]interface{})
	assert.Equal(t, float64(1543), linesOfCode["added"])
}

func TestClaudeCodeUsageMetricStorage(t *testing.T) {
	metric := models.ClaudeCodeUsageMetric{
		Id:                       "1:dev@company.com:2025-09-01T00:00:00Z",
		ConnectionId:             1,
		OrganizationId:           "org-123",
		Date:                     time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC),
		ActorType:                "user_actor",
		ActorEmail:               "dev@company.com",
		CustomerType:             "api",
		TerminalType:             "vscode",
		NumSessions:              5,
		LinesAdded:               1543,
		LinesRemoved:             892,
		CommitsByClaudeCode:      12,
		PullRequestsByClaudeCode: 2,
		EditToolAccepted:         45,
		EditToolRejected:         5,
		MultiEditToolAccepted:    12,
		MultiEditToolRejected:    2,
		WriteToolAccepted:        8,
		WriteToolRejected:        1,
		NotebookEditAccepted:     3,
		NotebookEditRejected:     0,
		EditToolAcceptanceRate:   90.0,
		WriteToolAcceptanceRate:  88.89,
		OverallAcceptanceRate:    89.47,
		InputTokens:              100000,
		OutputTokens:             35000,
		CacheReadTokens:          10000,
		CacheCreationTokens:      5000,
		EstimatedCostCents:       1025,
		CostCurrency:             "USD",
		CollectedAt:              time.Now(),
	}

	// Verify metric fields
	assert.NotEmpty(t, metric.Id)
	assert.Equal(t, uint64(1), metric.ConnectionId)
	assert.Equal(t, "org-123", metric.OrganizationId)
	assert.Equal(t, 5, metric.NumSessions)
	assert.Equal(t, 1543, metric.LinesAdded)
	assert.Equal(t, 12, metric.CommitsByClaudeCode)
	assert.InDelta(t, 90.0, metric.EditToolAcceptanceRate, 0.1)
}

func TestClaudeCodeConnectionModel(t *testing.T) {
	connection := models.ClaudeCodeConnection{
		Name:               "Test Connection",
		AdminApiKey:        "sk-ant-admin-xxx",
		RateLimitPerSecond: 5,
	}

	assert.Equal(t, "Test Connection", connection.Name)
	assert.NotEmpty(t, connection.AdminApiKey)
	assert.Equal(t, 5, connection.RateLimitPerSecond)
}

func TestToolAcceptanceRateCalculation(t *testing.T) {
	// Edit tool: 45 accepted, 5 rejected
	editAccepted := 45
	editRejected := 5
	editTotal := editAccepted + editRejected

	editRate := float64(editAccepted) / float64(editTotal) * 100
	assert.InDelta(t, 90.0, editRate, 0.1)

	// Write tool: 8 accepted, 1 rejected
	writeAccepted := 8
	writeRejected := 1
	writeTotal := writeAccepted + writeRejected

	writeRate := float64(writeAccepted) / float64(writeTotal) * 100
	assert.InDelta(t, 88.89, writeRate, 0.1)

	// Overall: all tools combined
	totalAccepted := 45 + 12 + 8 + 3 // edit + multi_edit + write + notebook
	totalRejected := 5 + 2 + 1 + 0
	totalAll := totalAccepted + totalRejected

	overallRate := float64(totalAccepted) / float64(totalAll) * 100
	assert.InDelta(t, 89.47, overallRate, 0.1)
}

func TestAPIKeyValidation(t *testing.T) {
	tests := []struct {
		name    string
		apiKey  string
		isValid bool
	}{
		{
			name:    "valid admin key",
			apiKey:  "sk-ant-admin-abc123xyz",
			isValid: true,
		},
		{
			name:    "invalid prefix - regular API key",
			apiKey:  "sk-ant-api01-abc123xyz",
			isValid: false,
		},
		{
			name:    "empty key",
			apiKey:  "",
			isValid: false,
		},
		{
			name:    "too short",
			apiKey:  "sk-ant-admin-",
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := len(tt.apiKey) > 13 && tt.apiKey[:13] == "sk-ant-admin-"
			assert.Equal(t, tt.isValid, isValid)
		})
	}
}

func TestClaudeCodeUserMetricModel(t *testing.T) {
	metric := models.ClaudeCodeUserMetric{
		Id:             "user-metric-123",
		ConnectionId:   1,
		OrganizationId: "org-123",
		UserId:         "user-456",
		UserEmail:      "alice@example.com",
		Date:           time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		EditToolUses:   100,
		WriteToolUses:  50,
		TotalToolUses:  300,
		LinesWritten:   500,
		SessionCount:   15,
		InputTokens:    200000,
		OutputTokens:   50000,
		CollectedAt:    time.Now(),
	}

	assert.NotEmpty(t, metric.Id)
	assert.Equal(t, "user-456", metric.UserId)
	assert.Equal(t, "alice@example.com", metric.UserEmail)
	assert.Equal(t, 300, metric.TotalToolUses)
}

func TestCostCalculation(t *testing.T) {
	// Cost is returned in cents USD
	costCents := int64(1025)

	// Convert to dollars
	costDollars := float64(costCents) / 100.0
	assert.InDelta(t, 10.25, costDollars, 0.01)
}

func TestPaginationHandling(t *testing.T) {
	// Test pagination with has_more flag
	responses := []struct {
		hasMore  bool
		nextPage *string
	}{
		{hasMore: true, nextPage: strPtr("page_abc123")},
		{hasMore: true, nextPage: strPtr("page_def456")},
		{hasMore: false, nextPage: nil},
	}

	pageCount := 0
	for _, resp := range responses {
		pageCount++
		if !resp.hasMore || resp.nextPage == nil {
			break
		}
	}

	assert.Equal(t, 3, pageCount)
}

func strPtr(s string) *string {
	return &s
}

func TestDateRFC3339Parsing(t *testing.T) {
	// Claude Code API returns dates in RFC 3339 format
	dateStr := "2025-09-01T00:00:00Z"
	date, err := time.Parse(time.RFC3339, dateStr)

	assert.NoError(t, err)
	assert.Equal(t, 2025, date.Year())
	assert.Equal(t, time.September, date.Month())
	assert.Equal(t, 1, date.Day())
}

func TestActorTypeParsing(t *testing.T) {
	// Actor can be user_actor or api_actor
	tests := []struct {
		actorType    string
		emailAddress string
		apiKeyName   string
		expectedId   string
	}{
		{
			actorType:    "user_actor",
			emailAddress: "dev@company.com",
			apiKeyName:   "",
			expectedId:   "dev@company.com",
		},
		{
			actorType:    "api_actor",
			emailAddress: "",
			apiKeyName:   "production-key",
			expectedId:   "production-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.actorType, func(t *testing.T) {
			var actorId string
			if tt.emailAddress != "" {
				actorId = tt.emailAddress
			} else {
				actorId = tt.apiKeyName
			}
			assert.Equal(t, tt.expectedId, actorId)
		})
	}
}
