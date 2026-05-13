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

import (
	"time"

	"github.com/apache/incubator-devlake/core/models/common"
)

// ClaudeCodeConnection stores connection configuration for Claude Code Admin API
// Admin API keys can be created at: https://console.anthropic.com/settings/admin-keys
// Only organization admins can provision Admin API keys
type ClaudeCodeConnection struct {
	common.Model
	Name               string `json:"name" gorm:"type:varchar(255);not null"`
	AdminApiKey        string `json:"adminApiKey" gorm:"type:text;serializer:encdec"` // sk-ant-admin-...
	RateLimitPerSecond int    `json:"rateLimitPerSecond" gorm:"default:5"`
}

func (ClaudeCodeConnection) TableName() string {
	return "_tool_claudecode_connections"
}

// ClaudeCodeUsageMetric stores daily user-level usage metrics from Claude Code Admin API
// API endpoint: GET /v1/organizations/usage_report/claude_code
type ClaudeCodeUsageMetric struct {
	Id             string    `gorm:"primaryKey;type:varchar(255)"`
	ConnectionId   uint64    `gorm:"type:bigint;index"`
	OrganizationId string    `gorm:"type:varchar(255);index"`
	Date           time.Time `gorm:"index"`

	// Actor identification (user_actor or api_actor)
	ActorType    string `gorm:"type:varchar(50)"`  // "user_actor" or "api_actor"
	ActorEmail   string `gorm:"type:varchar(255)"` // For user_actor
	ActorApiKey  string `gorm:"type:varchar(255)"` // For api_actor
	CustomerType string `gorm:"type:varchar(50)"`  // "api" or "subscription"
	TerminalType string `gorm:"type:varchar(100)"` // e.g., "vscode", "iTerm.app", "tmux"

	// Core metrics from API
	NumSessions              int `gorm:"type:int"` // Number of distinct Claude Code sessions
	LinesAdded               int `gorm:"type:int"` // Lines of code added
	LinesRemoved             int `gorm:"type:int"` // Lines of code removed
	CommitsByClaudeCode      int `gorm:"type:int"` // Git commits created through Claude Code
	PullRequestsByClaudeCode int `gorm:"type:int"` // PRs created through Claude Code

	// Tool action metrics (acceptance/rejection)
	EditToolAccepted      int `gorm:"type:int"`
	EditToolRejected      int `gorm:"type:int"`
	MultiEditToolAccepted int `gorm:"type:int"`
	MultiEditToolRejected int `gorm:"type:int"`
	WriteToolAccepted     int `gorm:"type:int"`
	WriteToolRejected     int `gorm:"type:int"`
	NotebookEditAccepted  int `gorm:"type:int"`
	NotebookEditRejected  int `gorm:"type:int"`

	// Calculated acceptance rates
	EditToolAcceptanceRate  float64 `gorm:"type:decimal(5,2)"`
	WriteToolAcceptanceRate float64 `gorm:"type:decimal(5,2)"`
	OverallAcceptanceRate   float64 `gorm:"type:decimal(5,2)"`

	// Token usage (aggregated across models)
	InputTokens         int64 `gorm:"type:bigint"`
	OutputTokens        int64 `gorm:"type:bigint"`
	CacheReadTokens     int64 `gorm:"type:bigint"`
	CacheCreationTokens int64 `gorm:"type:bigint"`

	// Cost (in cents USD)
	EstimatedCostCents int64  `gorm:"type:bigint"`
	CostCurrency       string `gorm:"type:varchar(10);default:'USD'"`

	CollectedAt time.Time
}

func (ClaudeCodeUsageMetric) TableName() string {
	return "claude_code_usage_metrics"
}

// ClaudeCodeUserMetric stores per-user usage metrics
type ClaudeCodeUserMetric struct {
	Id             string    `gorm:"primaryKey;type:varchar(255)"`
	ConnectionId   uint64    `gorm:"type:bigint;index"`
	OrganizationId string    `gorm:"type:varchar(255);index"`
	UserId         string    `gorm:"type:varchar(255);index"`
	UserEmail      string    `gorm:"type:varchar(255)"`
	Date           time.Time `gorm:"index"`

	// Tool usage
	EditToolUses  int `gorm:"type:int"`
	WriteToolUses int `gorm:"type:int"`
	TotalToolUses int `gorm:"type:int"`

	// Code metrics
	LinesWritten int `gorm:"type:int"`
	SessionCount int `gorm:"type:int"`

	// Token usage
	InputTokens  int64 `gorm:"type:bigint"`
	OutputTokens int64 `gorm:"type:bigint"`

	CollectedAt time.Time
}

func (ClaudeCodeUserMetric) TableName() string {
	return "claude_code_user_metrics"
}
