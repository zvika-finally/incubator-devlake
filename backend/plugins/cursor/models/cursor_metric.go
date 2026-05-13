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

// CursorConnection stores connection configuration for Cursor Analytics API
// API credentials can be generated from: https://cursor.com/settings (Enterprise teams only)
// Authentication: Basic Auth with API key
type CursorConnection struct {
	common.Model
	Name               string `json:"name" gorm:"type:varchar(255);not null"`
	ApiKey             string `json:"apiKey" gorm:"type:text;serializer:encdec"` // Basic Auth key
	RateLimitPerSecond int    `json:"rateLimitPerSecond" gorm:"default:5"`
}

func (CursorConnection) TableName() string {
	return "_tool_cursor_connections"
}

// CursorUsageMetric stores daily team-level usage metrics from Cursor Analytics API
// API endpoints: /analytics/team/agent-edits, /analytics/team/tabs, /analytics/team/dau
type CursorUsageMetric struct {
	Id           string    `gorm:"primaryKey;type:varchar(255)"`
	ConnectionId uint64    `gorm:"type:bigint;index"`
	Date         time.Time `gorm:"index"`

	// Suggestion metrics
	TotalSuggestions int     `gorm:"type:int"`
	TotalAcceptances int     `gorm:"type:int"`
	AcceptanceRate   float64 `gorm:"type:decimal(5,2)"` // (Acceptances/Suggestions)*100

	// Lines metrics (green = additions, red = deletions)
	GreenLinesAccepted  int     `gorm:"type:int"`
	GreenLinesSuggested int     `gorm:"type:int"`
	RedLinesAccepted    int     `gorm:"type:int"`
	RedLinesSuggested   int     `gorm:"type:int"`
	LineAcceptanceRatio float64 `gorm:"type:decimal(5,2)"`

	// Tab completion metrics
	TabSuggestions    int     `gorm:"type:int"`
	TabAcceptances    int     `gorm:"type:int"`
	TabAcceptanceRate float64 `gorm:"type:decimal(5,2)"`

	// Composer (AI chat) metrics
	ComposerSuggestions    int     `gorm:"type:int"`
	ComposerAcceptances    int     `gorm:"type:int"`
	ComposerAcceptanceRate float64 `gorm:"type:decimal(5,2)"`

	// Activity
	DailyActiveUsers int `gorm:"type:int"`

	CollectedAt time.Time
}

func (CursorUsageMetric) TableName() string {
	return "cursor_usage_metrics"
}

// CursorUserMetric stores per-user usage metrics from Cursor Analytics API
// API endpoint: /analytics/team/leaderboard
type CursorUserMetric struct {
	Id           string    `gorm:"primaryKey;type:varchar(255)"`
	ConnectionId uint64    `gorm:"type:bigint;index"`
	UserId       string    `gorm:"type:varchar(255);index"`
	UserEmail    string    `gorm:"type:varchar(255)"`
	Date         time.Time `gorm:"index"`

	// Suggestion metrics
	TabSuggestions      int     `gorm:"type:int"`
	TabAcceptances      int     `gorm:"type:int"`
	ComposerSuggestions int     `gorm:"type:int"`
	ComposerAcceptances int     `gorm:"type:int"`
	AcceptanceRate      float64 `gorm:"type:decimal(5,2)"`

	// Lines
	LinesAccepted  int `gorm:"type:int"`
	LinesSuggested int `gorm:"type:int"`

	CollectedAt time.Time
}

func (CursorUserMetric) TableName() string {
	return "cursor_user_metrics"
}
