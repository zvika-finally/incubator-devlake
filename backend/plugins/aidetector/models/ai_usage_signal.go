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

// AIUsageSignal stores AI detection results for pull requests
type AIUsageSignal struct {
	Id                string `gorm:"primaryKey;type:varchar(255)"`
	PullRequestId     string `gorm:"type:varchar(255);not null;index"`
	AIConfidenceScore int    `gorm:"type:int"`         // 0-100
	DetectedTool      string `gorm:"type:varchar(50)"` // unknown, copilot, cursor, claude

	// Explicit signals (HIGH confidence - from git trailers, PR body markers)
	ExplicitToolDetected bool   `gorm:"type:bool"`         // True if explicit AI marker found
	ExplicitTools        string `gorm:"type:varchar(255)"` // Comma-separated list of detected tools
	ExplicitPatterns     string `gorm:"type:text"`         // Matched patterns for audit trail
	ExplicitSignalScore  int    `gorm:"type:int"`          // Score from explicit markers (max 70)

	// Behavioral signals (each contributes to confidence score)
	RapidCommitScore    int `gorm:"type:int"` // Score from rapid commit velocity (max 30)
	PRSizeScore         int `gorm:"type:int"` // Score from PR size anomaly (max 20)
	LinesPerMinuteScore int `gorm:"type:int"` // Score from code production rate (max 25)
	DuplicationScore    int `gorm:"type:int"` // Score from code duplication patterns (max 15)
	GenericMessageScore int `gorm:"type:int"` // Score from generic commit messages (max 10)

	// Metrics
	AvgTimeBetweenCommits float64 `gorm:"type:decimal(10,2)"` // Minutes
	LinesPerMinute        float64 `gorm:"type:decimal(10,2)"`
	PRAdditions           int
	PRDeletions           int
	CommitCount           int
	CycleTimeHours        float64 `gorm:"type:decimal(10,2)"`

	// Velocity impact (compared to developer baseline)
	VelocityMultiplier float64 `gorm:"type:decimal(5,2)"` // e.g., 1.28 = 28% faster

	// Raw pattern data (JSON)
	PatternSignatures string `gorm:"type:text"` // JSON with detailed pattern analysis

	DetectedAt time.Time
	CreatedAt  time.Time
}

func (AIUsageSignal) TableName() string {
	return "ai_usage_signals"
}

// DeveloperBaseline stores baseline metrics per developer for comparison
type DeveloperBaseline struct {
	Id                    string  `gorm:"primaryKey;type:varchar(255)"`
	DeveloperId           string  `gorm:"type:varchar(255);not null;uniqueIndex"`
	AvgPRAdditions        float64 `gorm:"type:decimal(10,2)"`
	AvgPRDeletions        float64 `gorm:"type:decimal(10,2)"`
	AvgCycleTimeHours     float64 `gorm:"type:decimal(10,2)"`
	AvgCommitsPerPR       float64 `gorm:"type:decimal(10,2)"`
	AvgTimeBetweenCommits float64 `gorm:"type:decimal(10,2)"` // Minutes
	PRCount               int     // Number of PRs used to calculate baseline
	CalculatedAt          time.Time
}

func (DeveloperBaseline) TableName() string {
	return "developer_baselines"
}
