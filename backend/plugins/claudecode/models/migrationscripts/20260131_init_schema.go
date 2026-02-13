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

package migrationscripts

import (
	"time"

	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/helpers/migrationhelper"
)

type initSchema struct{}

func (script *initSchema) Up(basicRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&claudeCodeConnection20260131{},
		&claudeCodeUsageMetric20260131{},
		&claudeCodeUserMetric20260131{},
	)
}

func (script *initSchema) Version() uint64 {
	return 20260131000006
}

func (script *initSchema) Name() string {
	return "claudecode: init schema for connections and metrics"
}

// Migration models

type claudeCodeConnection20260131 struct {
	ID                 uint64 `gorm:"primaryKey;autoIncrement"`
	Name               string `gorm:"type:varchar(255);not null"`
	AdminApiKey        string `gorm:"type:text;serializer:encdec"`
	RateLimitPerSecond int    `gorm:"default:5"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (claudeCodeConnection20260131) TableName() string {
	return "_tool_claudecode_connections"
}

type claudeCodeUsageMetric20260131 struct {
	Id             string    `gorm:"primaryKey;type:varchar(255)"`
	ConnectionId   uint64    `gorm:"type:bigint;index"`
	OrganizationId string    `gorm:"type:varchar(255);index"`
	Date           time.Time `gorm:"index"`

	ActorType    string `gorm:"type:varchar(50)"`
	ActorEmail   string `gorm:"type:varchar(255)"`
	ActorApiKey  string `gorm:"type:varchar(255)"`
	CustomerType string `gorm:"type:varchar(50)"`
	TerminalType string `gorm:"type:varchar(100)"`

	NumSessions              int `gorm:"type:int"`
	LinesAdded               int `gorm:"type:int"`
	LinesRemoved             int `gorm:"type:int"`
	CommitsByClaudeCode      int `gorm:"type:int"`
	PullRequestsByClaudeCode int `gorm:"type:int"`

	EditToolAccepted      int `gorm:"type:int"`
	EditToolRejected      int `gorm:"type:int"`
	MultiEditToolAccepted int `gorm:"type:int"`
	MultiEditToolRejected int `gorm:"type:int"`
	WriteToolAccepted     int `gorm:"type:int"`
	WriteToolRejected     int `gorm:"type:int"`
	NotebookEditAccepted  int `gorm:"type:int"`
	NotebookEditRejected  int `gorm:"type:int"`

	EditToolAcceptanceRate  float64 `gorm:"type:decimal(5,2)"`
	WriteToolAcceptanceRate float64 `gorm:"type:decimal(5,2)"`
	OverallAcceptanceRate   float64 `gorm:"type:decimal(5,2)"`

	InputTokens         int64 `gorm:"type:bigint"`
	OutputTokens        int64 `gorm:"type:bigint"`
	CacheReadTokens     int64 `gorm:"type:bigint"`
	CacheCreationTokens int64 `gorm:"type:bigint"`

	EstimatedCostCents int64  `gorm:"type:bigint"`
	CostCurrency       string `gorm:"type:varchar(10);default:'USD'"`

	CollectedAt time.Time
}

func (claudeCodeUsageMetric20260131) TableName() string {
	return "claude_code_usage_metrics"
}

type claudeCodeUserMetric20260131 struct {
	Id             string    `gorm:"primaryKey;type:varchar(255)"`
	ConnectionId   uint64    `gorm:"type:bigint;index"`
	OrganizationId string    `gorm:"type:varchar(255);index"`
	UserId         string    `gorm:"type:varchar(255);index"`
	UserEmail      string    `gorm:"type:varchar(255)"`
	Date           time.Time `gorm:"index"`
	EditToolUses   int       `gorm:"type:int"`
	WriteToolUses  int       `gorm:"type:int"`
	TotalToolUses  int       `gorm:"type:int"`
	LinesWritten   int       `gorm:"type:int"`
	SessionCount   int       `gorm:"type:int"`
	InputTokens    int64     `gorm:"type:bigint"`
	OutputTokens   int64     `gorm:"type:bigint"`
	CollectedAt    time.Time
}

func (claudeCodeUserMetric20260131) TableName() string {
	return "claude_code_user_metrics"
}
