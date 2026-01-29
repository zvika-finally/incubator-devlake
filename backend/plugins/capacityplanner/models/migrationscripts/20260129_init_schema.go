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

type initSchema struct{}

type teamVelocity20260129 struct {
	Id                   string     `gorm:"primaryKey;type:varchar(255)"`
	ProjectName          string     `gorm:"type:varchar(255);index"`
	SprintId             string     `gorm:"type:varchar(255);index"`
	SprintName           string     `gorm:"type:varchar(255)"`
	FiscalWeek           string     `gorm:"type:varchar(10)"`
	StoryPointsCompleted int        `gorm:"type:int"`
	IssuesCompleted      int        `gorm:"type:int"`
	PRsMerged            int        `gorm:"type:int"`
	CommitCount          int        `gorm:"type:int"`
	AvgCycleTimeHours    float64    `gorm:"type:decimal(10,2)"`
	AvgLeadTimeHours     float64    `gorm:"type:decimal(10,2)"`
	TeamSize             int        `gorm:"type:int"`
	AvailableHours       float64    `gorm:"type:decimal(10,2)"`
	SprintStartDate      *time.Time
	SprintEndDate        *time.Time
	CalculatedAt         time.Time
}

func (teamVelocity20260129) TableName() string {
	return "team_velocities"
}

type initiativeForecast20260129 struct {
	Id                      string     `gorm:"primaryKey;type:varchar(255)"`
	InitiativeId            string     `gorm:"type:varchar(255);index"`
	InitiativeName          string     `gorm:"type:varchar(500)"`
	TotalStoryPoints        int        `gorm:"type:int"`
	CompletedStoryPoints    int        `gorm:"type:int"`
	RemainingStoryPoints    int        `gorm:"type:int"`
	PercentComplete         float64    `gorm:"type:decimal(5,2)"`
	AvgVelocity             float64    `gorm:"type:decimal(10,2)"`
	EstimatedSprints        int        `gorm:"type:int"`
	EstimatedCompletionDate *time.Time
	ConfidenceLevel         string     `gorm:"type:varchar(20)"`
	VelocityStdDev          float64    `gorm:"type:decimal(10,2)"`
	ScenarioData            string     `gorm:"type:text"`
	CalculatedAt            time.Time
}

func (initiativeForecast20260129) TableName() string {
	return "initiative_forecasts"
}

type capacityScenario20260129 struct {
	Id                     string     `gorm:"primaryKey;type:varchar(255)"`
	InitiativeId           string     `gorm:"type:varchar(255);index"`
	ScenarioName           string     `gorm:"type:varchar(255)"`
	ScenarioType           string     `gorm:"type:varchar(50)"`
	TeamSizeDelta          int        `gorm:"type:int"`
	ScopeDelta             int        `gorm:"type:int"`
	VelocityMultiplier     float64    `gorm:"type:decimal(5,2)"`
	OriginalSprints        int        `gorm:"type:int"`
	ScenarioSprints        int        `gorm:"type:int"`
	SprintsDelta           int        `gorm:"type:int"`
	OriginalCompletionDate *time.Time
	ScenarioCompletionDate *time.Time
	CalculatedAt           time.Time
}

func (capacityScenario20260129) TableName() string {
	return "capacity_scenarios"
}

func (u *initSchema) Up(baseRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		baseRes,
		&teamVelocity20260129{},
		&initiativeForecast20260129{},
		&capacityScenario20260129{},
	)
}

func (*initSchema) Version() uint64 {
	return 20260129000004
}

func (*initSchema) Name() string {
	return "capacityplanner: init schema for velocity, forecasts, and scenarios"
}
