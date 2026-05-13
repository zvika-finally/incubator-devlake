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

type addCodeChurn struct{}

func (script *addCodeChurn) Up(basicRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&aiChurnMetric20260131{},
		&projectChurnSummary20260131{},
	)
}

func (script *addCodeChurn) Version() uint64 {
	return 20260131000004
}

func (script *addCodeChurn) Name() string {
	return "aidetector: add code churn tracking for AI vs non-AI PRs"
}

// Migration models

type aiChurnMetric20260131 struct {
	Id                string     `gorm:"primaryKey;type:varchar(255)"`
	PullRequestId     string     `gorm:"type:varchar(255);index"`
	PullRequestKey    int        `gorm:"type:int"`
	ProjectName       string     `gorm:"type:varchar(255);index"`
	AuthorId          string     `gorm:"type:varchar(255);index"`
	AuthorName        string     `gorm:"type:varchar(255)"`
	AIConfidenceScore int        `gorm:"type:int"`
	IsAIAssisted      bool       `gorm:"type:bool;index"`
	InitialAdditions  int        `gorm:"type:int"`
	InitialDeletions  int        `gorm:"type:int"`
	MergedAt          *time.Time `gorm:"index"`
	ChurnWithin7Days  int        `gorm:"type:int"`
	ChurnWithin30Days int        `gorm:"type:int"`
	FollowUpCommits7  int        `gorm:"type:int"`
	FollowUpCommits30 int        `gorm:"type:int"`
	ChurnRatio7Days   float64    `gorm:"type:decimal(8,4)"`
	ChurnRatio30Days  float64    `gorm:"type:decimal(8,4)"`
	FilePaths         string     `gorm:"type:text"`
	CalculatedAt      time.Time
}

func (aiChurnMetric20260131) TableName() string {
	return "ai_churn_metrics"
}

type projectChurnSummary20260131 struct {
	Id                     string    `gorm:"primaryKey;type:varchar(255)"`
	ProjectName            string    `gorm:"type:varchar(255);index"`
	PeriodStart            time.Time `gorm:"index"`
	PeriodEnd              time.Time `gorm:"index"`
	TotalPRsAnalyzed       int       `gorm:"type:int"`
	AIPRCount              int       `gorm:"type:int"`
	NonAIPRCount           int       `gorm:"type:int"`
	AIAvgChurnRatio7       float64   `gorm:"type:decimal(8,4)"`
	AIAvgChurnRatio30      float64   `gorm:"type:decimal(8,4)"`
	AITotalChurn30         int       `gorm:"type:int"`
	AITotalAdditions       int       `gorm:"type:int"`
	NonAIAvgChurnRatio7    float64   `gorm:"type:decimal(8,4)"`
	NonAIAvgChurnRatio30   float64   `gorm:"type:decimal(8,4)"`
	NonAITotalChurn30      int       `gorm:"type:int"`
	NonAITotalAdditions    int       `gorm:"type:int"`
	ChurnDifferencePercent float64   `gorm:"type:decimal(8,2)"`
	CalculatedAt           time.Time
}

func (projectChurnSummary20260131) TableName() string {
	return "project_churn_summaries"
}
