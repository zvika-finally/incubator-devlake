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
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/claudecode/models"
)

var CollectMetricsMeta = plugin.SubTaskMeta{
	Name:             "collectMetrics",
	EntryPoint:       CollectMetrics,
	EnabledByDefault: true,
	Description:      "Collect usage metrics from Claude Code Admin API",
	DomainTypes:      []string{}, // AI tool metrics don't map to standard domain types
}

// Claude Code Admin API response structure
// API: GET https://api.anthropic.com/v1/organizations/usage_report/claude_code?starting_at=YYYY-MM-DD
// Docs: https://docs.anthropic.com/en/api/claude-code-analytics-api
type claudeCodeUsageResponse struct {
	Data    []claudeCodeUserDayRecord `json:"data"`
	HasMore bool                      `json:"has_more"`
	NextPage *string                  `json:"next_page"`
}

type claudeCodeUserDayRecord struct {
	Date           string            `json:"date"` // RFC 3339 format
	Actor          claudeCodeActor   `json:"actor"`
	OrganizationId string            `json:"organization_id"`
	CustomerType   string            `json:"customer_type"` // "api" or "subscription"
	TerminalType   string            `json:"terminal_type"` // e.g., "vscode", "iTerm.app"
	CoreMetrics    claudeCodeCore    `json:"core_metrics"`
	ToolActions    claudeCodeTools   `json:"tool_actions"`
	ModelBreakdown []claudeCodeModel `json:"model_breakdown"`
}

type claudeCodeActor struct {
	Type         string `json:"type"` // "user_actor" or "api_actor"
	EmailAddress string `json:"email_address,omitempty"`
	ApiKeyName   string `json:"api_key_name,omitempty"`
}

type claudeCodeCore struct {
	NumSessions                int                  `json:"num_sessions"`
	LinesOfCode                claudeCodeLines      `json:"lines_of_code"`
	CommitsByClaudeCode        int                  `json:"commits_by_claude_code"`
	PullRequestsByClaudeCode   int                  `json:"pull_requests_by_claude_code"`
}

type claudeCodeLines struct {
	Added   int `json:"added"`
	Removed int `json:"removed"`
}

type claudeCodeTools struct {
	EditTool       claudeCodeToolAction `json:"edit_tool"`
	MultiEditTool  claudeCodeToolAction `json:"multi_edit_tool"`
	WriteTool      claudeCodeToolAction `json:"write_tool"`
	NotebookEdit   claudeCodeToolAction `json:"notebook_edit_tool"`
}

type claudeCodeToolAction struct {
	Accepted int `json:"accepted"`
	Rejected int `json:"rejected"`
}

type claudeCodeModel struct {
	Model         string              `json:"model"`
	Tokens        claudeCodeTokens    `json:"tokens"`
	EstimatedCost claudeCodeCost      `json:"estimated_cost"`
}

type claudeCodeTokens struct {
	Input         int64 `json:"input"`
	Output        int64 `json:"output"`
	CacheRead     int64 `json:"cache_read"`
	CacheCreation int64 `json:"cache_creation"`
}

type claudeCodeCost struct {
	Currency string `json:"currency"`
	Amount   int64  `json:"amount"` // in cents
}

func CollectMetrics(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*ClaudeCodeTaskData)
	logger := taskCtx.GetLogger()
	conn := data.Connection

	logger.Info("Starting collectMetrics for Claude Code")

	// Calculate start date
	startDate := time.Now().AddDate(0, 0, -data.Options.DaysBack)

	// Collect metrics for each day
	for date := startDate; !date.After(time.Now()); date = date.AddDate(0, 0, 1) {
		logger.Info("Fetching Claude Code metrics for %s", date.Format("2006-01-02"))

		var nextPage *string
		for {
			usageData, err := fetchUsageReport(conn, date, nextPage)
			if err != nil {
				logger.Error(err, "Failed to fetch Claude Code usage report for %s", date.Format("2006-01-02"))
				break
			}

			// Process user-level daily metrics
			for _, record := range usageData.Data {
				metric := convertToMetric(conn.ID, record)
				if err := db.CreateOrUpdate(metric); err != nil {
					logger.Error(err, "Failed to save metric for actor %s on %s", getActorId(record.Actor), record.Date)
				}
			}

			if !usageData.HasMore || usageData.NextPage == nil {
				break
			}
			nextPage = usageData.NextPage
		}
	}

	logger.Info("Completed collectMetrics for Claude Code")
	return nil
}

func convertToMetric(connectionId uint64, record claudeCodeUserDayRecord) *models.ClaudeCodeUsageMetric {
	date, _ := time.Parse(time.RFC3339, record.Date)
	actorId := getActorId(record.Actor)

	// Aggregate tokens and costs across all models
	var inputTokens, outputTokens, cacheRead, cacheCreation, totalCostCents int64
	for _, model := range record.ModelBreakdown {
		inputTokens += model.Tokens.Input
		outputTokens += model.Tokens.Output
		cacheRead += model.Tokens.CacheRead
		cacheCreation += model.Tokens.CacheCreation
		totalCostCents += model.EstimatedCost.Amount
	}

	// Calculate acceptance rates
	editTotal := record.ToolActions.EditTool.Accepted + record.ToolActions.EditTool.Rejected
	writeTotal := record.ToolActions.WriteTool.Accepted + record.ToolActions.WriteTool.Rejected

	var editRate, writeRate, overallRate float64
	if editTotal > 0 {
		editRate = float64(record.ToolActions.EditTool.Accepted) / float64(editTotal) * 100
	}
	if writeTotal > 0 {
		writeRate = float64(record.ToolActions.WriteTool.Accepted) / float64(writeTotal) * 100
	}

	totalAccepted := record.ToolActions.EditTool.Accepted + record.ToolActions.MultiEditTool.Accepted +
		record.ToolActions.WriteTool.Accepted + record.ToolActions.NotebookEdit.Accepted
	totalRejected := record.ToolActions.EditTool.Rejected + record.ToolActions.MultiEditTool.Rejected +
		record.ToolActions.WriteTool.Rejected + record.ToolActions.NotebookEdit.Rejected
	totalAll := totalAccepted + totalRejected
	if totalAll > 0 {
		overallRate = float64(totalAccepted) / float64(totalAll) * 100
	}

	return &models.ClaudeCodeUsageMetric{
		Id:             fmt.Sprintf("%d:%s:%s", connectionId, actorId, record.Date),
		ConnectionId:   connectionId,
		OrganizationId: record.OrganizationId,
		Date:           date,

		ActorType:    record.Actor.Type,
		ActorEmail:   record.Actor.EmailAddress,
		ActorApiKey:  record.Actor.ApiKeyName,
		CustomerType: record.CustomerType,
		TerminalType: record.TerminalType,

		NumSessions:              record.CoreMetrics.NumSessions,
		LinesAdded:               record.CoreMetrics.LinesOfCode.Added,
		LinesRemoved:             record.CoreMetrics.LinesOfCode.Removed,
		CommitsByClaudeCode:      record.CoreMetrics.CommitsByClaudeCode,
		PullRequestsByClaudeCode: record.CoreMetrics.PullRequestsByClaudeCode,

		EditToolAccepted:      record.ToolActions.EditTool.Accepted,
		EditToolRejected:      record.ToolActions.EditTool.Rejected,
		MultiEditToolAccepted: record.ToolActions.MultiEditTool.Accepted,
		MultiEditToolRejected: record.ToolActions.MultiEditTool.Rejected,
		WriteToolAccepted:     record.ToolActions.WriteTool.Accepted,
		WriteToolRejected:     record.ToolActions.WriteTool.Rejected,
		NotebookEditAccepted:  record.ToolActions.NotebookEdit.Accepted,
		NotebookEditRejected:  record.ToolActions.NotebookEdit.Rejected,

		EditToolAcceptanceRate:  editRate,
		WriteToolAcceptanceRate: writeRate,
		OverallAcceptanceRate:   overallRate,

		InputTokens:         inputTokens,
		OutputTokens:        outputTokens,
		CacheReadTokens:     cacheRead,
		CacheCreationTokens: cacheCreation,

		EstimatedCostCents: totalCostCents,
		CostCurrency:       "USD",

		CollectedAt: time.Now(),
	}
}

func getActorId(actor claudeCodeActor) string {
	if actor.EmailAddress != "" {
		return actor.EmailAddress
	}
	return actor.ApiKeyName
}

func fetchUsageReport(conn *models.ClaudeCodeConnection, date time.Time, nextPage *string) (*claudeCodeUsageResponse, errors.Error) {
	url := fmt.Sprintf("https://api.anthropic.com/v1/organizations/usage_report/claude_code?starting_at=%s&limit=100",
		date.Format("2006-01-02"))

	if nextPage != nil {
		url = fmt.Sprintf("https://api.anthropic.com/v1/organizations/usage_report/claude_code?starting_at=%s&page=%s",
			date.Format("2006-01-02"), *nextPage)
	}

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("x-api-key", conn.AdminApiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "DevLake/1.0.0 (https://devlake.apache.org)")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Default.Wrap(err, "failed to call Claude Code Admin API")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.Default.New(fmt.Sprintf("Claude Code API returned %d: %s", resp.StatusCode, string(body)))
	}

	var usageData claudeCodeUsageResponse
	if err := json.NewDecoder(resp.Body).Decode(&usageData); err != nil {
		return nil, errors.Default.Wrap(err, "failed to decode Claude Code API response")
	}

	return &usageData, nil
}
