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
)

// AIChurnMetric tracks code churn for AI-detected PRs vs non-AI PRs
// Based on GitClear's research showing AI code has 41% higher churn rate
// Churn = lines modified in follow-up commits / original lines added
type AIChurnMetric struct {
	Id                string    `gorm:"primaryKey;type:varchar(255)"`
	PullRequestId     string    `gorm:"type:varchar(255);index"`
	PullRequestKey    int       `gorm:"type:int"`
	ProjectName       string    `gorm:"type:varchar(255);index"`
	AuthorId          string    `gorm:"type:varchar(255);index"`
	AuthorName        string    `gorm:"type:varchar(255)"`

	// AI detection linkage
	AIConfidenceScore int       `gorm:"type:int"`      // From AIUsageSignal
	IsAIAssisted      bool      `gorm:"type:bool;index"` // Confidence >= threshold (default 70)

	// Original PR metrics
	InitialAdditions  int       `gorm:"type:int"`      // Lines added in original PR
	InitialDeletions  int       `gorm:"type:int"`      // Lines deleted in original PR
	MergedAt          *time.Time `gorm:"index"`

	// Churn metrics (modifications to PR's files after merge)
	ChurnWithin7Days  int       `gorm:"type:int"`      // Lines modified within 7 days
	ChurnWithin30Days int       `gorm:"type:int"`      // Lines modified within 30 days
	FollowUpCommits7  int       `gorm:"type:int"`      // Count of commits touching these files within 7 days
	FollowUpCommits30 int       `gorm:"type:int"`      // Count of commits touching these files within 30 days

	// Calculated churn ratios
	ChurnRatio7Days   float64   `gorm:"type:decimal(8,4)"` // Churn7 / InitialAdditions
	ChurnRatio30Days  float64   `gorm:"type:decimal(8,4)"` // Churn30 / InitialAdditions

	// Files from the original PR
	FilePaths         string    `gorm:"type:text"` // JSON array of file paths

	CalculatedAt      time.Time
}

func (AIChurnMetric) TableName() string {
	return "ai_churn_metrics"
}

// ChurnCategory returns a category based on churn ratio
func (m *AIChurnMetric) ChurnCategory() string {
	ratio := m.ChurnRatio30Days
	switch {
	case ratio < 0.1:
		return "stable" // Less than 10% churn
	case ratio < 0.25:
		return "normal" // 10-25% churn
	case ratio < 0.5:
		return "elevated" // 25-50% churn
	default:
		return "high" // > 50% churn
	}
}

// ProjectChurnSummary aggregates churn metrics at project level
type ProjectChurnSummary struct {
	Id                    string    `gorm:"primaryKey;type:varchar(255)"`
	ProjectName           string    `gorm:"type:varchar(255);index"`
	PeriodStart           time.Time `gorm:"index"`
	PeriodEnd             time.Time `gorm:"index"`

	// Counts
	TotalPRsAnalyzed      int       `gorm:"type:int"`
	AIPRCount             int       `gorm:"type:int"` // PRs with AI confidence >= threshold
	NonAIPRCount          int       `gorm:"type:int"` // PRs with AI confidence < threshold

	// AI-assisted PR churn
	AIAvgChurnRatio7      float64   `gorm:"type:decimal(8,4)"`
	AIAvgChurnRatio30     float64   `gorm:"type:decimal(8,4)"`
	AITotalChurn30        int       `gorm:"type:int"`
	AITotalAdditions      int       `gorm:"type:int"`

	// Non-AI PR churn
	NonAIAvgChurnRatio7   float64   `gorm:"type:decimal(8,4)"`
	NonAIAvgChurnRatio30  float64   `gorm:"type:decimal(8,4)"`
	NonAITotalChurn30     int       `gorm:"type:int"`
	NonAITotalAdditions   int       `gorm:"type:int"`

	// Comparison metrics
	ChurnDifferencePercent float64  `gorm:"type:decimal(8,2)"` // (AI - NonAI) / NonAI * 100
	// Positive = AI code has more churn (expected based on research)
	// GitClear benchmark: +41%

	CalculatedAt          time.Time
}

func (ProjectChurnSummary) TableName() string {
	return "project_churn_summaries"
}
