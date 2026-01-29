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
	"github.com/apache/incubator-devlake/core/plugin"
)

var _ plugin.MigrationScript = (*addHealthAndCapability)(nil)

type addHealthAndCapability struct{}

func (script *addHealthAndCapability) Up(basicRes context.BasicRes) errors.Error {
	db := basicRes.GetDal()

	// Create team health scores table
	err := db.AutoMigrate(&teamHealthScore20260130{})
	if err != nil {
		return errors.Default.Wrap(err, "failed to create team_health_scores table")
	}

	// Add new columns to business_initiatives table
	err = db.AutoMigrate(&businessInitiativeCapability20260130{})
	if err != nil {
		return errors.Default.Wrap(err, "failed to add capability columns to business_initiatives")
	}

	return nil
}

func (script *addHealthAndCapability) Version() uint64 {
	return 20260130000004
}

func (script *addHealthAndCapability) Name() string {
	return "businessmetrics: add health scores and capability columns"
}

// Migration models

type teamHealthScore20260130 struct {
	Id          string    `gorm:"primaryKey;type:varchar(255)"`
	ProjectName string    `gorm:"type:varchar(255);index"`
	PeriodStart time.Time `gorm:"index"`
	PeriodEnd   time.Time `gorm:"index"`

	DeployFrequency   float64 `gorm:"type:decimal(10,4)"`
	LeadTimeHours     float64 `gorm:"type:decimal(10,2)"`
	ChangeFailureRate float64 `gorm:"type:decimal(5,2)"`
	MTTRHours         float64 `gorm:"type:decimal(10,2)"`

	DeployFreqScore int `gorm:"type:int"`
	LeadTimeScore   int `gorm:"type:int"`
	CFRScore        int `gorm:"type:int"`
	MTTRScore       int `gorm:"type:int"`

	TotalScore  int    `gorm:"type:int"`
	HealthLevel string `gorm:"type:varchar(20)"`

	CalculatedAt time.Time `gorm:"index"`
}

func (teamHealthScore20260130) TableName() string {
	return "team_health_scores"
}

type businessInitiativeCapability20260130 struct {
	BusinessCapability string  `gorm:"type:varchar(50)"`
	RevenueImpact      string  `gorm:"type:varchar(20)"`
	BusinessValueScore int     `gorm:"type:int"`
	RevenueToCostRatio float64 `gorm:"type:decimal(5,2)"`
}

func (businessInitiativeCapability20260130) TableName() string {
	return "business_initiatives"
}
