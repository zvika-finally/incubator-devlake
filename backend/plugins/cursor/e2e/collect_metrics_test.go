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

	"github.com/apache/incubator-devlake/plugins/cursor/models"
)

func TestCursorAgentEditsResponseParsing(t *testing.T) {
	// Simulated Cursor Analytics API response for agent-edits endpoint
	// Docs: https://cursor.com/docs/account/teams/analytics-api
	apiResponse := `{
		"data": [{
			"date": "2024-01-15",
			"acceptedEdits": 3500,
			"rejectedEdits": 1500,
			"totalEdits": 5000,
			"acceptanceRate": 70.0
		}],
		"params": {
			"metric": "agent-edits",
			"teamId": 12345,
			"startDate": "2024-01-15",
			"endDate": "2024-01-15"
		}
	}`

	var data map[string]interface{}
	err := json.Unmarshal([]byte(apiResponse), &data)
	assert.NoError(t, err)

	dataArr := data["data"].([]interface{})
	record := dataArr[0].(map[string]interface{})
	assert.Equal(t, "2024-01-15", record["date"])
	assert.Equal(t, float64(5000), record["totalEdits"])
	assert.Equal(t, float64(3500), record["acceptedEdits"])
	assert.Equal(t, float64(70.0), record["acceptanceRate"])
}

func TestCursorUsageMetricStorage(t *testing.T) {
	metric := models.CursorUsageMetric{
		Id:                  "1:2024-01-15",
		ConnectionId:        1,
		Date:                time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		TotalSuggestions:    5000,
		TotalAcceptances:    3500,
		AcceptanceRate:      70.0,
		GreenLinesAccepted:  2000,
		GreenLinesSuggested: 3000,
		TabSuggestions:      4000,
		TabAcceptances:      2800,
		TabAcceptanceRate:   70.0,
		ComposerSuggestions: 1000,
		ComposerAcceptances: 700,
		DailyActiveUsers:    25,
		CollectedAt:         time.Now(),
	}

	// Verify metric fields
	assert.NotEmpty(t, metric.Id)
	assert.Equal(t, uint64(1), metric.ConnectionId)
	assert.Equal(t, 5000, metric.TotalSuggestions)
	assert.Equal(t, 3500, metric.TotalAcceptances)
	assert.InDelta(t, 70.0, metric.AcceptanceRate, 0.1)
}

func TestCursorConnectionModel(t *testing.T) {
	connection := models.CursorConnection{
		Name:               "Test Connection",
		ApiKey:             "cursor-api-key-xxx",
		RateLimitPerSecond: 5,
	}

	assert.Equal(t, "Test Connection", connection.Name)
	assert.NotEmpty(t, connection.ApiKey)
	assert.Equal(t, 5, connection.RateLimitPerSecond)
}

func TestMetricAggregation(t *testing.T) {
	metrics := []models.CursorUsageMetric{
		{TotalSuggestions: 1500, TotalAcceptances: 1000},
		{TotalSuggestions: 1200, TotalAcceptances: 800},
		{TotalSuggestions: 800, TotalAcceptances: 600},
	}

	// Calculate team totals
	var totalSuggestions, totalAcceptances int
	for _, m := range metrics {
		totalSuggestions += m.TotalSuggestions
		totalAcceptances += m.TotalAcceptances
	}

	assert.Equal(t, 3500, totalSuggestions)
	assert.Equal(t, 2400, totalAcceptances)

	// Calculate acceptance rate
	acceptanceRate := float64(totalAcceptances) / float64(totalSuggestions) * 100
	assert.InDelta(t, 68.57, acceptanceRate, 0.1)
}

func TestRateLimitHandling(t *testing.T) {
	// Cursor API limits: 100 requests/min for team endpoints, 50/min for by-user
	teamRateLimit := 100 // requests per minute
	requestInterval := time.Minute / time.Duration(teamRateLimit)

	assert.Equal(t, 600*time.Millisecond, requestInterval)
}

func TestCursorUserMetricModel(t *testing.T) {
	metric := models.CursorUserMetric{
		Id:                  "1:user-456:2024-01-15",
		ConnectionId:        1,
		UserId:              "user-456",
		UserEmail:           "alice@example.com",
		Date:                time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		TabSuggestions:      500,
		TabAcceptances:      350,
		ComposerSuggestions: 100,
		ComposerAcceptances: 70,
		AcceptanceRate:      70.0,
		LinesAccepted:       800,
		LinesSuggested:      1200,
		CollectedAt:         time.Now(),
	}

	assert.NotEmpty(t, metric.Id)
	assert.Equal(t, "user-456", metric.UserId)
	assert.Equal(t, "alice@example.com", metric.UserEmail)
	assert.Equal(t, 500, metric.TabSuggestions)
}

func TestDateParsing(t *testing.T) {
	dateStr := "2024-01-15"
	date, err := time.Parse("2006-01-02", dateStr)

	assert.NoError(t, err)
	assert.Equal(t, 2024, date.Year())
	assert.Equal(t, time.January, date.Month())
	assert.Equal(t, 15, date.Day())
}

func TestCursorDAUResponseParsing(t *testing.T) {
	// DAU endpoint response
	apiResponse := `{
		"data": [{
			"date": "2024-01-15",
			"totalUsers": 50,
			"cliUsers": 10,
			"agentUsers": 35,
			"bugBotUsers": 5
		}],
		"params": {
			"metric": "dau",
			"startDate": "2024-01-15",
			"endDate": "2024-01-15"
		}
	}`

	var data map[string]interface{}
	err := json.Unmarshal([]byte(apiResponse), &data)
	assert.NoError(t, err)

	dataArr := data["data"].([]interface{})
	record := dataArr[0].(map[string]interface{})
	assert.Equal(t, float64(50), record["totalUsers"])
}

func TestCursorLeaderboardResponseParsing(t *testing.T) {
	// Leaderboard endpoint response
	apiResponse := `{
		"data": [{
			"email": "alice@example.com",
			"userId": "user-123",
			"acceptedEdits": 500,
			"tabAcceptances": 1200,
			"totalSuggestions": 2000,
			"acceptanceRate": 85.0,
			"charactersAdded": 15000
		}],
		"params": {
			"metric": "leaderboard",
			"startDate": "2024-01-01",
			"endDate": "2024-01-31"
		},
		"pagination": {
			"page": 1,
			"pageSize": 100,
			"total": 25
		}
	}`

	var data map[string]interface{}
	err := json.Unmarshal([]byte(apiResponse), &data)
	assert.NoError(t, err)

	dataArr := data["data"].([]interface{})
	user := dataArr[0].(map[string]interface{})
	assert.Equal(t, "alice@example.com", user["email"])
	assert.Equal(t, float64(85.0), user["acceptanceRate"])
}

func TestDateRangeValidation(t *testing.T) {
	// Cursor API limits date range to 30 days
	maxDays := 30

	startDate := time.Now().AddDate(0, 0, -45)
	endDate := time.Now()

	daysDiff := int(endDate.Sub(startDate).Hours() / 24)

	if daysDiff > maxDays {
		// Should truncate to 30 days
		startDate = endDate.AddDate(0, 0, -maxDays)
		daysDiff = maxDays
	}

	assert.Equal(t, 30, daysDiff)
}
