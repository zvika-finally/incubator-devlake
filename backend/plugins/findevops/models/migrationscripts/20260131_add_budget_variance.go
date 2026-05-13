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

var _ = (*addBudgetVariance)(nil)

type addBudgetVariance struct{}

func (script *addBudgetVariance) Up(basicRes context.BasicRes) errors.Error {
	// AutoMigrateTables will add missing columns to existing tables
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&costAllocation20260131{},
		&monthlyCostSummary20260131{},
	)
}

func (script *addBudgetVariance) Version() uint64 {
	return 20260131000001
}

func (script *addBudgetVariance) Name() string {
	return "findevops: add budget variance and unallocated cost tracking"
}

// Migration model for cost_allocations - includes new variance fields
type costAllocation20260131 struct {
	Id                     string  `gorm:"primaryKey;type:varchar(255)"`
	InitiativeId           string  `gorm:"type:varchar(255);index"`
	IssueId                string  `gorm:"type:varchar(255);index"`
	FiscalMonth            string  `gorm:"type:varchar(10);index"`
	DeveloperId            string  `gorm:"type:varchar(255);index"`
	HoursWorked            float64 `gorm:"type:decimal(10,2)"`
	HourlyRate             float64 `gorm:"type:decimal(10,2)"`
	DeveloperCost          float64 `gorm:"type:decimal(12,2)"`
	AIToolCost             float64 `gorm:"type:decimal(12,2)"`
	TotalCost              float64 `gorm:"type:decimal(12,2)"`
	CapitalizationCategory string  `gorm:"type:varchar(50)"`
	ProjectPhase           string  `gorm:"type:varchar(50)"`
	CapitalizationPercent  int     `gorm:"type:int"`
	CategoryReason         string  `gorm:"type:varchar(255)"`
	IssueType              string  `gorm:"type:varchar(50)"`
	IssueLabels            string  `gorm:"type:text"`
	// New budget variance fields
	EstimatedMinutes int64   `gorm:"type:bigint"`
	ActualMinutes    int64   `gorm:"type:bigint"`
	VarianceMinutes  int64   `gorm:"type:bigint"`
	VariancePercent  float64 `gorm:"type:decimal(8,2)"`
	OverBudget       bool    `gorm:"type:bool"`
	IsUnallocated    bool    `gorm:"type:bool;index"`
	CalculatedAt     time.Time
	CreatedAt        time.Time
}

func (costAllocation20260131) TableName() string {
	return "cost_allocations"
}

// Migration model for monthly_cost_summaries - includes new unallocated/variance fields
type monthlyCostSummary20260131 struct {
	Id                 string  `gorm:"primaryKey;type:varchar(255)"`
	ProjectName        string  `gorm:"type:varchar(255);index"`
	FiscalMonth        string  `gorm:"type:varchar(10);index"`
	TotalCost          float64 `gorm:"type:decimal(14,2)"`
	CapitalizableCost  float64 `gorm:"type:decimal(14,2)"`
	ExpenseCost        float64 `gorm:"type:decimal(14,2)"`
	CapitalizationRate float64 `gorm:"type:decimal(5,2)"`
	PreliminaryCost    float64 `gorm:"type:decimal(14,2)"`
	DevelopmentCost    float64 `gorm:"type:decimal(14,2)"`
	PostImplCost       float64 `gorm:"type:decimal(14,2)"`
	NewBusinessCost    float64 `gorm:"type:decimal(14,2)"`
	KTLOCost           float64 `gorm:"type:decimal(14,2)"`
	PlatformCost       float64 `gorm:"type:decimal(14,2)"`
	TechDebtCost       float64 `gorm:"type:decimal(14,2)"`
	// New unallocated cost fields
	UnallocatedCost    float64 `gorm:"type:decimal(14,2)"`
	UnallocatedPercent float64 `gorm:"type:decimal(5,2)"`
	OrphanIssueCount   int     `gorm:"type:int"`
	// New budget variance fields
	TotalEstimatedCost   float64 `gorm:"type:decimal(14,2)"`
	TotalActualCost      float64 `gorm:"type:decimal(14,2)"`
	BudgetVariance       float64 `gorm:"type:decimal(8,2)"`
	OverBudgetIssueCount int     `gorm:"type:int"`
	CalculatedAt         time.Time
}

func (monthlyCostSummary20260131) TableName() string {
	return "monthly_cost_summaries"
}
