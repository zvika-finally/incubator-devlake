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

// Phase A tables. Mirrored shape of models/* — defined here so that this
// migration is self-contained and idempotent even if model definitions change later.

type prAICohort20260513 struct {
	PRId              string    `gorm:"primaryKey;type:varchar(255)"`
	AICohort          string    `gorm:"type:varchar(20);not null;index"`
	ConfidenceScore   int       `gorm:"type:int"`
	HasExplicitMarker bool      `gorm:"type:bool"`
	HasCommitTrailer  bool      `gorm:"type:bool"`
	ClassifierVersion string    `gorm:"type:varchar(32);not null"`
	ClassifiedAt      time.Time `gorm:"not null"`
}

func (prAICohort20260513) TableName() string { return "pr_ai_cohort" }

type prDefectSignals20260513 struct {
	PRId                  string    `gorm:"primaryKey;type:varchar(255)"`
	HasRevert14d          bool      `gorm:"type:bool"`
	HasHotfix14d          bool      `gorm:"type:bool"`
	HasIncident14d        bool      `gorm:"type:bool"`
	IncidentDataAvailable bool      `gorm:"type:bool"`
	TotalDefectCount      int       `gorm:"type:int"`
	WindowCloseDate       time.Time `gorm:"not null;index"`
	ComputedAt            time.Time `gorm:"not null"`
}

func (prDefectSignals20260513) TableName() string { return "pr_defect_signals" }

type prChangeComposition20260513 struct {
	PRId          string    `gorm:"primaryKey;type:varchar(255)"`
	Additions     int       `gorm:"type:int"`
	Deletions     int       `gorm:"type:int"`
	FileCount     int       `gorm:"type:int"`
	AdditiveLines int       `gorm:"type:int"`
	RefactorLines int       `gorm:"type:int"`
	RefactorRatio float64   `gorm:"type:decimal(5,4)"`
	BatchBucket   string    `gorm:"type:varchar(4);not null;index"`
	ComputedAt    time.Time `gorm:"not null"`
}

func (prChangeComposition20260513) TableName() string { return "pr_change_composition" }

type accountOverride20260513 struct {
	SourceSystem string `gorm:"primaryKey;type:varchar(50)"`
	SourceId     string `gorm:"primaryKey;type:varchar(255)"`
	AccountId    string `gorm:"type:varchar(255);not null"`
	Note         string `gorm:"type:varchar(500)"`
}

func (accountOverride20260513) TableName() string { return "aimeasure_account_overrides" }

type engineerRole20260513 struct {
	AccountId string `gorm:"primaryKey;type:varchar(255)"`
	Role      string `gorm:"type:varchar(50);not null"`
	UpdatedAt string `gorm:"type:varchar(32)"`
}

func (engineerRole20260513) TableName() string { return "aimeasure_engineer_roles" }

func (*initSchema) Up(basicRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&prAICohort20260513{},
		&prDefectSignals20260513{},
		&prChangeComposition20260513{},
		&accountOverride20260513{},
		&engineerRole20260513{},
	)
}

func (*initSchema) Version() uint64 {
	return 20260513000001
}

func (*initSchema) Name() string {
	return "aimeasure: initialize Phase A schema"
}
