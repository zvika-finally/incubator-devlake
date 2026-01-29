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

// BusinessInitiative represents a strategic business goal extracted from Jira Epics
type BusinessInitiative struct {
	Id                 string     `gorm:"primaryKey;type:varchar(255)"`
	Name               string     `gorm:"type:varchar(500);not null"`
	JiraEpicKey        string     `gorm:"type:varchar(100);not null;index"`
	GoalType           string     `gorm:"type:varchar(50)"` // revenue, efficiency, compliance, innovation
	InvestmentCategory string     `gorm:"type:varchar(50)"` // business, ktlo, platform, techdebt, support, rd
	DevelopmentStage   string     `gorm:"type:varchar(50)"` // development, maintenance, research
	FiscalQuarter      string     `gorm:"type:varchar(20)"` // 2026-Q1
	TargetDate         *time.Time
	Status             string     `gorm:"type:varchar(50)"` // planned, active, completed, cancelled

	// Business capability classification
	BusinessCapability string `gorm:"type:varchar(50)"` // core_product, growth, monetization, platform, etc.
	RevenueImpact      string `gorm:"type:varchar(20)"` // direct, enabling, supporting, cost_center

	// Business value metrics
	BusinessValueScore int     `gorm:"type:int"`          // 0-100 calculated score
	RevenueToCostRatio float64 `gorm:"type:decimal(5,2)"` // If applicable

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (BusinessInitiative) TableName() string {
	return "business_initiatives"
}

// WorkAllocation links work items (issues, PRs, commits) to business initiatives
type WorkAllocation struct {
	Id             string  `gorm:"primaryKey;type:varchar(255)"`
	InitiativeId   string  `gorm:"type:varchar(255);not null;index"`
	EntityType     string  `gorm:"type:varchar(50);not null"` // issue, pull_request, commit
	EntityId       string  `gorm:"type:varchar(255);not null;index"`
	DeveloperId    string  `gorm:"type:varchar(255);index"`
	StoryPoints    int
	EstimatedHours float64
	ActualHours    float64
	CreatedAt      time.Time
}

func (WorkAllocation) TableName() string {
	return "work_allocations"
}
