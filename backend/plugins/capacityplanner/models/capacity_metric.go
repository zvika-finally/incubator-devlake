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

// TeamVelocity stores calculated velocity metrics per sprint
type TeamVelocity struct {
	Id              string    `gorm:"primaryKey;type:varchar(255)"`
	ProjectName     string    `gorm:"type:varchar(255);index"`
	SprintId        string    `gorm:"type:varchar(255);index"`
	SprintName      string    `gorm:"type:varchar(255)"`
	FiscalWeek      string    `gorm:"type:varchar(10)"` // "2026-W05"

	// Velocity metrics
	StoryPointsCompleted int       `gorm:"type:int"`
	IssuesCompleted      int       `gorm:"type:int"`
	PRsMerged            int       `gorm:"type:int"`
	CommitCount          int       `gorm:"type:int"`

	// Time metrics
	AvgCycleTimeHours    float64   `gorm:"type:decimal(10,2)"`
	AvgLeadTimeHours     float64   `gorm:"type:decimal(10,2)"`

	// Capacity
	TeamSize             int       `gorm:"type:int"`
	AvailableHours       float64   `gorm:"type:decimal(10,2)"`

	SprintStartDate      *time.Time
	SprintEndDate        *time.Time
	CalculatedAt         time.Time
}

func (TeamVelocity) TableName() string {
	return "team_velocities"
}

// InitiativeForecast stores forecasted completion data per initiative
type InitiativeForecast struct {
	Id                      string    `gorm:"primaryKey;type:varchar(255)"`
	InitiativeId            string    `gorm:"type:varchar(255);index"`
	InitiativeName          string    `gorm:"type:varchar(500)"`

	// Current state
	TotalStoryPoints        int       `gorm:"type:int"`
	CompletedStoryPoints    int       `gorm:"type:int"`
	RemainingStoryPoints    int       `gorm:"type:int"`
	PercentComplete         float64   `gorm:"type:decimal(5,2)"`

	// Velocity-based forecast
	AvgVelocity             float64   `gorm:"type:decimal(10,2)"` // Story points per sprint
	EstimatedSprints        int       `gorm:"type:int"`
	EstimatedCompletionDate *time.Time

	// Confidence
	ConfidenceLevel         string    `gorm:"type:varchar(20)"` // high, medium, low
	VelocityStdDev          float64   `gorm:"type:decimal(10,2)"`

	// Scenario data (JSON)
	ScenarioData            string    `gorm:"type:text"` // JSON with best/worst case scenarios

	CalculatedAt            time.Time
}

func (InitiativeForecast) TableName() string {
	return "initiative_forecasts"
}

// CapacityScenario stores what-if scenario calculations
type CapacityScenario struct {
	Id                     string    `gorm:"primaryKey;type:varchar(255)"`
	InitiativeId           string    `gorm:"type:varchar(255);index"`
	ScenarioName           string    `gorm:"type:varchar(255)"`
	ScenarioType           string    `gorm:"type:varchar(50)"` // team_change, scope_change, velocity_change

	// Scenario parameters
	TeamSizeDelta          int       `gorm:"type:int"` // +2 or -1
	ScopeDelta             int       `gorm:"type:int"` // Story points added/removed
	VelocityMultiplier     float64   `gorm:"type:decimal(5,2)"` // 1.0 = normal, 1.2 = 20% faster

	// Forecasted impact
	OriginalSprints        int       `gorm:"type:int"`
	ScenarioSprints        int       `gorm:"type:int"`
	SprintsDelta           int       `gorm:"type:int"`
	OriginalCompletionDate *time.Time
	ScenarioCompletionDate *time.Time

	CalculatedAt           time.Time
}

func (CapacityScenario) TableName() string {
	return "capacity_scenarios"
}
