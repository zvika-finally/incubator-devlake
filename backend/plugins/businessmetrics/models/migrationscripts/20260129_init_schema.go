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

// BusinessInitiative represents a strategic business goal extracted from Jira Epics
type businessInitiative20260129 struct {
	Id                 string `gorm:"primaryKey;type:varchar(255)"`
	Name               string `gorm:"type:varchar(500);not null"`
	JiraEpicKey        string `gorm:"type:varchar(100);not null;index"`
	GoalType           string `gorm:"type:varchar(50)"`
	InvestmentCategory string `gorm:"type:varchar(50)"`
	DevelopmentStage   string `gorm:"type:varchar(50)"`
	FiscalQuarter      string `gorm:"type:varchar(20)"`
	TargetDate         *time.Time
	Status             string `gorm:"type:varchar(50)"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (businessInitiative20260129) TableName() string {
	return "business_initiatives"
}

// WorkAllocation links work items to business initiatives
type workAllocation20260129 struct {
	Id             string `gorm:"primaryKey;type:varchar(255)"`
	InitiativeId   string `gorm:"type:varchar(255);not null;index"`
	EntityType     string `gorm:"type:varchar(50);not null"`
	EntityId       string `gorm:"type:varchar(255);not null;index"`
	DeveloperId    string `gorm:"type:varchar(255);index"`
	StoryPoints    int
	EstimatedHours float64
	ActualHours    float64
	CreatedAt      time.Time
}

func (workAllocation20260129) TableName() string {
	return "work_allocations"
}

func (u *initSchema) Up(baseRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		baseRes,
		&businessInitiative20260129{},
		&workAllocation20260129{},
	)
}

func (*initSchema) Version() uint64 {
	return 20260129000001
}

func (*initSchema) Name() string {
	return "businessmetrics: init schema for business initiatives and work allocations"
}
