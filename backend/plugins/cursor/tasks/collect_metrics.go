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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/cursor/models"
)

var CollectMetricsMeta = plugin.SubTaskMeta{
	Name:             "collectMetrics",
	EntryPoint:       CollectMetrics,
	EnabledByDefault: true,
	Description:      "Collect usage metrics from Cursor Analytics API",
	DomainTypes:      []string{}, // AI tool metrics don't map to standard domain types
}

// Cursor Analytics API response structures
// Docs: https://cursor.com/docs/account/teams/analytics-api
// Authentication: Basic Auth with API key from team settings (Enterprise only)

type cursorAgentEditsResponse struct {
	Data []struct {
		Date           string `json:"date"`
		AcceptedEdits  int    `json:"acceptedEdits"`
		RejectedEdits  int    `json:"rejectedEdits"`
		TotalEdits     int    `json:"totalEdits"`
		AcceptanceRate float64 `json:"acceptanceRate"`
	} `json:"data"`
	Params cursorParams `json:"params"`
}

type cursorTabsResponse struct {
	Data []struct {
		Date              string `json:"date"`
		Suggestions       int    `json:"suggestions"`
		Acceptances       int    `json:"acceptances"`
		AcceptanceRate    float64 `json:"acceptanceRate"`
		CharactersAdded   int    `json:"charactersAdded"`
		CharactersDeleted int    `json:"charactersDeleted"`
	} `json:"data"`
	Params cursorParams `json:"params"`
}

type cursorDAUResponse struct {
	Data []struct {
		Date       string `json:"date"`
		TotalUsers int    `json:"totalUsers"`
		CliUsers   int    `json:"cliUsers"`
		AgentUsers int    `json:"agentUsers"`
		BugBotUsers int   `json:"bugBotUsers"`
	} `json:"data"`
	Params cursorParams `json:"params"`
}

type cursorLeaderboardResponse struct {
	Data []struct {
		Email             string  `json:"email"`
		UserId            string  `json:"userId"`
		AcceptedEdits     int     `json:"acceptedEdits"`
		TabAcceptances    int     `json:"tabAcceptances"`
		TotalSuggestions  int     `json:"totalSuggestions"`
		AcceptanceRate    float64 `json:"acceptanceRate"`
		CharactersAdded   int     `json:"charactersAdded"`
	} `json:"data"`
	Params cursorParams `json:"params"`
	Pagination struct {
		Page     int `json:"page"`
		PageSize int `json:"pageSize"`
		Total    int `json:"total"`
	} `json:"pagination"`
}

type cursorParams struct {
	Metric    string `json:"metric"`
	TeamId    int    `json:"teamId"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
}

func CollectMetrics(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*CursorTaskData)
	logger := taskCtx.GetLogger()
	conn := data.Connection

	logger.Info("Starting collectMetrics for Cursor")

	// Calculate date range (max 30 days per Cursor API)
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -data.Options.DaysBack)
	if data.Options.DaysBack > 30 {
		startDate = endDate.AddDate(0, 0, -30)
		logger.Warn(nil, "Cursor API limits date range to 30 days, truncating")
	}

	startDateStr := startDate.Format("2006-01-02")
	endDateStr := endDate.Format("2006-01-02")

	// Collect agent edits (AI-suggested code edits)
	agentEdits, err := fetchAgentEdits(conn, startDateStr, endDateStr)
	if err != nil {
		logger.Warn(err, "Failed to fetch agent edits")
	}

	// Collect tab completions
	tabs, err := fetchTabs(conn, startDateStr, endDateStr)
	if err != nil {
		logger.Warn(err, "Failed to fetch tabs metrics")
	}

	// Collect daily active users
	dau, err := fetchDAU(conn, startDateStr, endDateStr)
	if err != nil {
		logger.Warn(err, "Failed to fetch DAU metrics")
	}

	// Merge metrics by date and save
	metricsMap := make(map[string]*models.CursorUsageMetric)

	// Process agent edits
	if agentEdits != nil {
		for _, ae := range agentEdits.Data {
			metric := getOrCreateMetric(metricsMap, conn.ID, ae.Date)
			metric.TotalSuggestions = ae.TotalEdits
			metric.TotalAcceptances = ae.AcceptedEdits
			metric.AcceptanceRate = ae.AcceptanceRate
		}
	}

	// Process tab completions
	if tabs != nil {
		for _, tab := range tabs.Data {
			metric := getOrCreateMetric(metricsMap, conn.ID, tab.Date)
			metric.TabSuggestions = tab.Suggestions
			metric.TabAcceptances = tab.Acceptances
			metric.TabAcceptanceRate = tab.AcceptanceRate
			metric.GreenLinesAccepted = tab.CharactersAdded
			metric.RedLinesAccepted = tab.CharactersDeleted
		}
	}

	// Process DAU
	if dau != nil {
		for _, d := range dau.Data {
			metric := getOrCreateMetric(metricsMap, conn.ID, d.Date)
			metric.DailyActiveUsers = d.TotalUsers
		}
	}

	// Save all metrics
	for _, metric := range metricsMap {
		metric.CollectedAt = time.Now()
		if err := db.CreateOrUpdate(metric); err != nil {
			logger.Error(err, "Failed to save team metric for %s", metric.Date)
		}
	}

	// Collect per-user metrics from leaderboard
	leaderboard, err := fetchLeaderboard(conn, startDateStr, endDateStr)
	if err != nil {
		logger.Warn(err, "Failed to fetch leaderboard")
	} else {
		for _, user := range leaderboard.Data {
			userMetric := &models.CursorUserMetric{
				Id:              fmt.Sprintf("%d:%s:%s", conn.ID, user.UserId, startDateStr),
				ConnectionId:    conn.ID,
				UserId:          user.UserId,
				UserEmail:       user.Email,
				Date:            startDate,
				TabAcceptances:  user.TabAcceptances,
				LinesAccepted:   user.CharactersAdded,
				AcceptanceRate:  user.AcceptanceRate,
				CollectedAt:     time.Now(),
			}
			if err := db.CreateOrUpdate(userMetric); err != nil {
				logger.Error(err, "Failed to save user metric for %s", user.Email)
			}
		}
	}

	logger.Info("Completed collectMetrics for Cursor")
	return nil
}

func getOrCreateMetric(metricsMap map[string]*models.CursorUsageMetric, connectionId uint64, dateStr string) *models.CursorUsageMetric {
	if metric, exists := metricsMap[dateStr]; exists {
		return metric
	}

	date, _ := time.Parse("2006-01-02", dateStr)
	metric := &models.CursorUsageMetric{
		Id:           fmt.Sprintf("%d:%s", connectionId, dateStr),
		ConnectionId: connectionId,
		Date:         date,
	}
	metricsMap[dateStr] = metric
	return metric
}

func cursorRequest(conn *models.CursorConnection, endpoint string) (*http.Response, errors.Error) {
	url := fmt.Sprintf("https://cursor.com%s", endpoint)

	req, _ := http.NewRequest("GET", url, nil)
	// Basic Auth: username is API key, password is empty
	auth := base64.StdEncoding.EncodeToString([]byte(conn.ApiKey + ":"))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "DevLake/1.0.0 (https://devlake.apache.org)")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Default.Wrap(err, "failed to call Cursor API")
	}

	return resp, nil
}

func fetchAgentEdits(conn *models.CursorConnection, startDate, endDate string) (*cursorAgentEditsResponse, errors.Error) {
	endpoint := fmt.Sprintf("/analytics/team/agent-edits?startDate=%s&endDate=%s", startDate, endDate)
	resp, err := cursorRequest(conn, endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.Default.New(fmt.Sprintf("Cursor API returned %d: %s", resp.StatusCode, string(body)))
	}

	var data cursorAgentEditsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, errors.Default.Wrap(err, "failed to decode Cursor API response")
	}
	return &data, nil
}

func fetchTabs(conn *models.CursorConnection, startDate, endDate string) (*cursorTabsResponse, errors.Error) {
	endpoint := fmt.Sprintf("/analytics/team/tabs?startDate=%s&endDate=%s", startDate, endDate)
	resp, err := cursorRequest(conn, endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.Default.New(fmt.Sprintf("Cursor API returned %d: %s", resp.StatusCode, string(body)))
	}

	var data cursorTabsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, errors.Default.Wrap(err, "failed to decode Cursor API response")
	}
	return &data, nil
}

func fetchDAU(conn *models.CursorConnection, startDate, endDate string) (*cursorDAUResponse, errors.Error) {
	endpoint := fmt.Sprintf("/analytics/team/dau?startDate=%s&endDate=%s", startDate, endDate)
	resp, err := cursorRequest(conn, endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.Default.New(fmt.Sprintf("Cursor API returned %d: %s", resp.StatusCode, string(body)))
	}

	var data cursorDAUResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, errors.Default.Wrap(err, "failed to decode Cursor API response")
	}
	return &data, nil
}

func fetchLeaderboard(conn *models.CursorConnection, startDate, endDate string) (*cursorLeaderboardResponse, errors.Error) {
	endpoint := fmt.Sprintf("/analytics/team/leaderboard?startDate=%s&endDate=%s&pageSize=100", startDate, endDate)
	resp, err := cursorRequest(conn, endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.Default.New(fmt.Sprintf("Cursor API returned %d: %s", resp.StatusCode, string(body)))
	}

	var data cursorLeaderboardResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, errors.Default.Wrap(err, "failed to decode Cursor API response")
	}
	return &data, nil
}
