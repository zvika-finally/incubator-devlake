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

// EffortSource constants
const (
	EffortSourceJiraTime       = "jira_time"
	EffortSourceJiraEstimate   = "jira_estimate"
	EffortSourceStoryPoints    = "story_points"
	EffortSourceGitInferred    = "git_inferred"
	EffortSourceFteDistributed = "fte_distributed"
)

// ConfidenceLevel constants
const (
	ConfidenceHigh     = "high"
	ConfidenceMedium   = "medium"
	ConfidenceInferred = "inferred"
	ConfidenceLow      = "low"
)

// CostAllocation stores calculated costs per initiative per fiscal period
type CostAllocation struct {
	Id           string `gorm:"primaryKey;type:varchar(255)"`
	InitiativeId string `gorm:"type:varchar(255);index"`
	IssueId      string `gorm:"type:varchar(255);index"`
	FiscalMonth  string `gorm:"type:varchar(10);index"` // "2026-01"
	DeveloperId  string `gorm:"type:varchar(255);index"`

	// Time data
	HoursWorked float64 `gorm:"type:decimal(10,2)"`
	HourlyRate  float64 `gorm:"type:decimal(10,2)"`

	// Cost calculations
	DeveloperCost float64 `gorm:"type:decimal(12,2)"` // hours * rate
	AIToolCost    float64 `gorm:"type:decimal(12,2)"` // estimated AI tool cost
	TotalCost     float64 `gorm:"type:decimal(12,2)"`

	// ASC 350-40 categorization
	CapitalizationCategory string `gorm:"type:varchar(50)"` // capitalizable, expense
	ProjectPhase           string `gorm:"type:varchar(50)"` // preliminary, development, post_implementation
	CapitalizationPercent  int    `gorm:"type:int"`         // 0 or 100

	// Audit trail
	CategoryReason string `gorm:"type:varchar(255)"` // Why this categorization
	IssueType      string `gorm:"type:varchar(50)"`
	IssueLabels    string `gorm:"type:text"`

	// Budget variance tracking (from Jira estimates)
	EstimatedMinutes int64   `gorm:"type:bigint"`       // From OriginalEstimateMinutes
	ActualMinutes    int64   `gorm:"type:bigint"`       // From TimeSpentMinutes
	VarianceMinutes  int64   `gorm:"type:bigint"`       // Estimated - Actual (positive = under budget)
	VariancePercent  float64 `gorm:"type:decimal(8,2)"` // (Estimated - Actual) / Estimated * 100
	OverBudget       bool    `gorm:"type:bool"`         // ActualMinutes > EstimatedMinutes

	// Unallocated tracking
	IsUnallocated bool `gorm:"type:bool;index"` // No epic/initiative attribution

	// === EFFORT SOURCE TRACKING ===
	EffortSource    string `gorm:"type:varchar(50)"` // jira_time, jira_estimate, story_points, git_inferred, fte_distributed
	ConfidenceLevel string `gorm:"type:varchar(20)"` // high, medium, inferred, low

	// === GIT-INFERRED EFFORT BREAKDOWN ===
	GitCodingHours      float64 `gorm:"type:decimal(10,2)"` // Hours from coding activity
	GitReviewHours      float64 `gorm:"type:decimal(10,2)"` // Hours from review activity
	GitComplexityFactor float64 `gorm:"type:decimal(5,2)"`  // Complexity multiplier
	GitActiveDays       int     `gorm:"type:int"`           // Days with commits

	// === VALIDATION FLAGS ===
	EffortValidated       bool    `gorm:"type:bool;default:false"` // Jira vs Git cross-validated
	ValidationVariancePct float64 `gorm:"type:decimal(8,2)"`       // Variance between Jira and Git

	// === AUDIT TRAIL FOR R&D COMPLIANCE ===
	LinkedCommitShas      string `gorm:"type:text"` // JSON array of commit SHAs
	LinkedPrIds           string `gorm:"type:text"` // JSON array of PR IDs
	ClassificationSignals string `gorm:"type:text"` // JSON: what triggered ASC 350-40 category

	// === FTE CONTEXT ===
	DeveloperMonthlyFte float64 `gorm:"type:decimal(3,2)"` // Developer's FTE for this month
	FteAllocationPct    float64 `gorm:"type:decimal(5,2)"` // % of developer's month on this issue

	CalculatedAt time.Time
	CreatedAt    time.Time
}

func (CostAllocation) TableName() string {
	return "cost_allocations"
}

// MonthlyCostSummary aggregates costs by month for reporting
type MonthlyCostSummary struct {
	Id          string `gorm:"primaryKey;type:varchar(255)"`
	ProjectName string `gorm:"type:varchar(255);index"`
	FiscalMonth string `gorm:"type:varchar(10);index"`

	TotalCost          float64 `gorm:"type:decimal(14,2)"`
	CapitalizableCost  float64 `gorm:"type:decimal(14,2)"`
	ExpenseCost        float64 `gorm:"type:decimal(14,2)"`
	CapitalizationRate float64 `gorm:"type:decimal(5,2)"` // percentage

	// Breakdown by phase
	PreliminaryCost float64 `gorm:"type:decimal(14,2)"`
	DevelopmentCost float64 `gorm:"type:decimal(14,2)"`
	PostImplCost    float64 `gorm:"type:decimal(14,2)"`

	// Breakdown by investment category
	NewBusinessCost float64 `gorm:"type:decimal(14,2)"`
	KTLOCost        float64 `gorm:"type:decimal(14,2)"`
	PlatformCost    float64 `gorm:"type:decimal(14,2)"`
	TechDebtCost    float64 `gorm:"type:decimal(14,2)"`

	// Unallocated cost tracking
	UnallocatedCost    float64 `gorm:"type:decimal(14,2)"` // Cost not attributed to initiatives
	UnallocatedPercent float64 `gorm:"type:decimal(5,2)"`  // UnallocatedCost/TotalCost * 100 (target: <10%)
	OrphanIssueCount   int     `gorm:"type:int"`           // Count of issues without epic/initiative

	// Budget variance tracking
	TotalEstimatedCost   float64 `gorm:"type:decimal(14,2)"` // Sum of estimated costs
	TotalActualCost      float64 `gorm:"type:decimal(14,2)"` // Sum of actual costs
	BudgetVariance       float64 `gorm:"type:decimal(8,2)"`  // (Estimated - Actual) / Estimated * 100
	OverBudgetIssueCount int     `gorm:"type:int"`           // Count of issues exceeding estimate

	CalculatedAt time.Time
}

func (MonthlyCostSummary) TableName() string {
	return "monthly_cost_summaries"
}

// DeveloperHourlyRate stores hourly rates for cost calculations
type DeveloperHourlyRate struct {
	Id            string  `gorm:"primaryKey;type:varchar(255)"`
	DeveloperId   string  `gorm:"type:varchar(255);uniqueIndex"`
	DeveloperName string  `gorm:"type:varchar(255)"`
	HourlyRate    float64 `gorm:"type:decimal(10,2)"`
	Role          string  `gorm:"type:varchar(100)"` // engineer, senior, staff
	CostCenter    string  `gorm:"type:varchar(100)"`
	EffectiveDate time.Time
	CreatedAt     time.Time
}

func (DeveloperHourlyRate) TableName() string {
	return "developer_hourly_rates"
}
