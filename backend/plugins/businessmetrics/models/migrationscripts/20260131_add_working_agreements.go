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

type addWorkingAgreements struct{}

func (script *addWorkingAgreements) Up(basicRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&workingAgreement20260131{},
		&agreementViolation20260131{},
		&agreementComplianceSummary20260131{},
	)
}

func (script *addWorkingAgreements) Version() uint64 {
	return 20260131000002
}

func (script *addWorkingAgreements) Name() string {
	return "businessmetrics: add working agreements, violations, and compliance summaries"
}

// Migration models

type workingAgreement20260131 struct {
	Id             string  `gorm:"primaryKey;type:varchar(255)"`
	ProjectName    string  `gorm:"type:varchar(255);index"`
	AgreementType  string  `gorm:"type:varchar(50);index"`
	ThresholdValue float64 `gorm:"type:decimal(10,2)"`
	ThresholdUnit  string  `gorm:"type:varchar(20)"`
	AlertEnabled   bool    `gorm:"type:bool;default:true"`
	Description    string  `gorm:"type:varchar(255)"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (workingAgreement20260131) TableName() string {
	return "working_agreements"
}

type agreementViolation20260131 struct {
	Id             string    `gorm:"primaryKey;type:varchar(255)"`
	AgreementId    string    `gorm:"type:varchar(255);index"`
	AgreementType  string    `gorm:"type:varchar(50);index"`
	ProjectName    string    `gorm:"type:varchar(255);index"`
	EntityType     string    `gorm:"type:varchar(50)"`
	EntityId       string    `gorm:"type:varchar(255)"`
	EntityKey      string    `gorm:"type:varchar(255)"`
	CurrentValue   float64   `gorm:"type:decimal(10,2)"`
	ThresholdValue float64   `gorm:"type:decimal(10,2)"`
	ExcessValue    float64   `gorm:"type:decimal(10,2)"`
	ViolatedAt     time.Time `gorm:"index"`
	ResolvedAt     *time.Time
	IsResolved     bool `gorm:"type:bool;default:false;index"`
	CreatedAt      time.Time
}

func (agreementViolation20260131) TableName() string {
	return "agreement_violations"
}

type agreementComplianceSummary20260131 struct {
	Id              string    `gorm:"primaryKey;type:varchar(255)"`
	ProjectName     string    `gorm:"type:varchar(255);index"`
	AgreementType   string    `gorm:"type:varchar(50);index"`
	PeriodStart     time.Time `gorm:"index"`
	PeriodEnd       time.Time `gorm:"index"`
	TotalChecked    int       `gorm:"type:int"`
	TotalCompliant  int       `gorm:"type:int"`
	TotalViolations int       `gorm:"type:int"`
	ComplianceRate  float64   `gorm:"type:decimal(5,2)"`
	AverageValue    float64   `gorm:"type:decimal(10,2)"`
	P50Value        float64   `gorm:"type:decimal(10,2)"`
	P90Value        float64   `gorm:"type:decimal(10,2)"`
	CalculatedAt    time.Time
}

func (agreementComplianceSummary20260131) TableName() string {
	return "agreement_compliance_summaries"
}
