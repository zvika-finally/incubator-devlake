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

// AgreementType defines the type of working agreement
type AgreementType string

const (
	AgreementPRMergeTime        AgreementType = "pr_merge_time"
	AgreementReviewTurnaround   AgreementType = "review_turnaround"
	AgreementWIPLimit           AgreementType = "wip_limit"
	AgreementIssuesInProgress   AgreementType = "issues_in_progress"
)

// ThresholdUnit defines the unit for threshold values
type ThresholdUnit string

const (
	UnitDays   ThresholdUnit = "days"
	UnitHours  ThresholdUnit = "hours"
	UnitCount  ThresholdUnit = "count"
)

// WorkingAgreement defines a team's working agreement threshold
// (Swarmia-style agreements for PR merge time, review turnaround, WIP limits)
type WorkingAgreement struct {
	Id             string        `gorm:"primaryKey;type:varchar(255)"`
	ProjectName    string        `gorm:"type:varchar(255);index"`
	AgreementType  AgreementType `gorm:"type:varchar(50);index"`
	ThresholdValue float64       `gorm:"type:decimal(10,2)"`
	ThresholdUnit  ThresholdUnit `gorm:"type:varchar(20)"`
	AlertEnabled   bool          `gorm:"type:bool;default:true"`
	Description    string        `gorm:"type:varchar(255)"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (WorkingAgreement) TableName() string {
	return "working_agreements"
}

// AgreementViolation records when a working agreement threshold is exceeded
type AgreementViolation struct {
	Id             string        `gorm:"primaryKey;type:varchar(255)"`
	AgreementId    string        `gorm:"type:varchar(255);index"`
	AgreementType  AgreementType `gorm:"type:varchar(50);index"`
	ProjectName    string        `gorm:"type:varchar(255);index"`
	EntityType     string        `gorm:"type:varchar(50)"`  // "pull_request", "issue", "developer"
	EntityId       string        `gorm:"type:varchar(255)"`
	EntityKey      string        `gorm:"type:varchar(255)"` // PR number, issue key, developer name
	CurrentValue   float64       `gorm:"type:decimal(10,2)"`
	ThresholdValue float64       `gorm:"type:decimal(10,2)"`
	ExcessValue    float64       `gorm:"type:decimal(10,2)"` // How much over threshold
	ViolatedAt     time.Time     `gorm:"index"`
	ResolvedAt     *time.Time    // When violation was resolved (PR merged, issue done)
	IsResolved     bool          `gorm:"type:bool;default:false;index"`
	CreatedAt      time.Time
}

func (AgreementViolation) TableName() string {
	return "agreement_violations"
}

// AgreementComplianceSummary aggregates compliance metrics by project and period
type AgreementComplianceSummary struct {
	Id                   string        `gorm:"primaryKey;type:varchar(255)"`
	ProjectName          string        `gorm:"type:varchar(255);index"`
	AgreementType        AgreementType `gorm:"type:varchar(50);index"`
	PeriodStart          time.Time     `gorm:"index"`
	PeriodEnd            time.Time     `gorm:"index"`
	TotalChecked         int           `gorm:"type:int"`
	TotalCompliant       int           `gorm:"type:int"`
	TotalViolations      int           `gorm:"type:int"`
	ComplianceRate       float64       `gorm:"type:decimal(5,2)"` // (Compliant/Total)*100
	AverageValue         float64       `gorm:"type:decimal(10,2)"`
	P50Value             float64       `gorm:"type:decimal(10,2)"` // Median
	P90Value             float64       `gorm:"type:decimal(10,2)"` // 90th percentile
	CalculatedAt         time.Time
}

func (AgreementComplianceSummary) TableName() string {
	return "agreement_compliance_summaries"
}

// DefaultWorkingAgreements returns standard Swarmia-style agreements
func DefaultWorkingAgreements(projectName string) []WorkingAgreement {
	now := time.Now()
	return []WorkingAgreement{
		{
			Id:             projectName + ":pr_merge_time",
			ProjectName:    projectName,
			AgreementType:  AgreementPRMergeTime,
			ThresholdValue: 7,
			ThresholdUnit:  UnitDays,
			AlertEnabled:   true,
			Description:    "PR merge time should be less than 7 days",
			CreatedAt:      now,
			UpdatedAt:      now,
		},
		{
			Id:             projectName + ":review_turnaround",
			ProjectName:    projectName,
			AgreementType:  AgreementReviewTurnaround,
			ThresholdValue: 24,
			ThresholdUnit:  UnitHours,
			AlertEnabled:   true,
			Description:    "First review should happen within 24 hours",
			CreatedAt:      now,
			UpdatedAt:      now,
		},
		{
			Id:             projectName + ":wip_limit",
			ProjectName:    projectName,
			AgreementType:  AgreementWIPLimit,
			ThresholdValue: 3,
			ThresholdUnit:  UnitCount,
			AlertEnabled:   true,
			Description:    "No more than 3 open PRs per developer",
			CreatedAt:      now,
			UpdatedAt:      now,
		},
		{
			Id:             projectName + ":issues_in_progress",
			ProjectName:    projectName,
			AgreementType:  AgreementIssuesInProgress,
			ThresholdValue: 2,
			ThresholdUnit:  UnitCount,
			AlertEnabled:   true,
			Description:    "No more than 2 issues in progress per developer",
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}
}
