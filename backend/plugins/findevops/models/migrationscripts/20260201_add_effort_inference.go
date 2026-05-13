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

type addEffortInference struct{}

// developerMonthlyFte20260201 - new table for FTE tracking
type developerMonthlyFte20260201 struct {
	Id                   string  `gorm:"primaryKey;type:varchar(255)"`
	DeveloperId          string  `gorm:"type:varchar(255);index"`
	FiscalMonth          string  `gorm:"type:varchar(10);index"`
	ProjectName          string  `gorm:"type:varchar(255);index"`
	PrsAuthored          int     `gorm:"type:int;default:0"`
	PrsReviewed          int     `gorm:"type:int;default:0"`
	CommitsAuthored      int     `gorm:"type:int;default:0"`
	IssuesUpdated        int     `gorm:"type:int;default:0"`
	CommentsAdded        int     `gorm:"type:int;default:0"`
	RawActivityScore     float64 `gorm:"type:decimal(10,2)"`
	BaselineScore        float64 `gorm:"type:decimal(10,2)"`
	RawFte               float64 `gorm:"type:decimal(3,2)"`
	InactiveDays         int     `gorm:"type:int;default:0"`
	AdjustedFte          float64 `gorm:"type:decimal(3,2)"`
	HoursFromJira        float64 `gorm:"type:decimal(10,2);default:0"`
	HoursFromGitInferred float64 `gorm:"type:decimal(10,2);default:0"`
	HoursDistributed     float64 `gorm:"type:decimal(10,2);default:0"`
	TotalAllocatedHours  float64 `gorm:"type:decimal(10,2);default:0"`
	CalculatedAt         time.Time
}

func (developerMonthlyFte20260201) TableName() string {
	return "developer_monthly_fte"
}

// costAllocation20260201 - add new columns to existing table
type costAllocation20260201 struct {
	Id                    string  `gorm:"primaryKey;type:varchar(255)"`
	EffortSource          string  `gorm:"type:varchar(50)"`
	ConfidenceLevel       string  `gorm:"type:varchar(20)"`
	GitCodingHours        float64 `gorm:"type:decimal(10,2)"`
	GitReviewHours        float64 `gorm:"type:decimal(10,2)"`
	GitComplexityFactor   float64 `gorm:"type:decimal(5,2)"`
	GitActiveDays         int     `gorm:"type:int"`
	EffortValidated       bool    `gorm:"type:bool;default:false"`
	ValidationVariancePct float64 `gorm:"type:decimal(8,2)"`
	LinkedCommitShas      string  `gorm:"type:text"`
	LinkedPrIds           string  `gorm:"type:text"`
	ClassificationSignals string  `gorm:"type:text"`
	DeveloperMonthlyFte   float64 `gorm:"type:decimal(3,2)"`
	FteAllocationPct      float64 `gorm:"type:decimal(5,2)"`
}

func (costAllocation20260201) TableName() string {
	return "cost_allocations"
}

// finDevOpsSettings20260201 - add new configuration fields
type finDevOpsSettings20260201 struct {
	Id                                    uint64  `gorm:"primaryKey;autoIncrement"`
	FteMaxPerMonth                        float64 `gorm:"default:1.0"`
	FteBaselineMultiplier                 float64 `gorm:"default:1.2"`
	FteInactivityThresholdDays            int     `gorm:"default:5"`
	FteWorkingHoursPerMonth               float64 `gorm:"default:160.0"`
	ActivityWeightPrAuthored              float64 `gorm:"default:1.0"`
	ActivityWeightPrReviewed              float64 `gorm:"default:0.3"`
	ActivityWeightCommitAuthored          float64 `gorm:"default:0.2"`
	ActivityWeightIssueUpdated            float64 `gorm:"default:0.1"`
	ActivityWeightCommentAdded            float64 `gorm:"default:0.05"`
	GitProductiveHoursPerActiveDay        float64 `gorm:"default:6.0"`
	GitReviewHoursPerCycle                float64 `gorm:"default:1.5"`
	GitCommentsPerReviewCycle             int     `gorm:"default:3"`
	GitMinHoursPerIssue                   float64 `gorm:"default:1.0"`
	GitMaxHoursPerIssue                   float64 `gorm:"default:80.0"`
	ValidationJiraGitVarianceThresholdPct float64 `gorm:"default:50.0"`
	PreliminaryCommitKeywords             string  `gorm:"type:text"`
	DevelopmentCommitKeywords             string  `gorm:"type:text"`
	PostImplementationCommitKeywords      string  `gorm:"type:text"`
	EnableGitEffortInference              bool    `gorm:"default:true"`
	EnableFteNormalization                bool    `gorm:"default:true"`
}

func (finDevOpsSettings20260201) TableName() string {
	return "_tool_findevops_settings"
}

func (u *addEffortInference) Up(baseRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		baseRes,
		&developerMonthlyFte20260201{},
		&costAllocation20260201{},
		&finDevOpsSettings20260201{},
	)
}

func (*addEffortInference) Version() uint64 {
	return 20260201000001
}

func (*addEffortInference) Name() string {
	return "findevops: add effort inference support (FTE normalization, git inference, audit trail)"
}
