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
	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/helpers/migrationhelper"
)

type addSettings struct{}

type finDevOpsSettings20260130 struct {
	Id          uint64 `gorm:"primaryKey;autoIncrement"`
	ProjectName string `gorm:"type:varchar(255);uniqueIndex"`

	// Cost parameters
	DefaultHourlyRate  float64 `gorm:"default:87.0"`
	RoleRates          string  `gorm:"type:text"`
	HoursPerStoryPoint float64 `gorm:"default:4.0"`

	// Capitalization framework
	CapitalizationFramework string `gorm:"type:varchar(50);default:'asc_350_40_stages'"`

	// ASC 350-40 label mappings (JSON arrays)
	PreliminaryLabels        string `gorm:"type:text;default:'[\"research\",\"spike\",\"investigation\",\"feasibility\",\"discovery\",\"poc\",\"proof-of-concept\",\"planning\"]'"`
	PostImplementationLabels string `gorm:"type:text;default:'[\"bug\",\"hotfix\",\"maintenance\",\"ktlo\",\"support\",\"incident\",\"fix\",\"patch\",\"tech-debt\"]'"`

	// Issue type mappings (JSON arrays)
	PreliminaryTypes        string `gorm:"type:text;default:'[\"Spike\",\"Research\",\"Discovery\"]'"`
	DevelopmentTypes        string `gorm:"type:text;default:'[\"Story\",\"Feature\",\"Enhancement\",\"Epic\"]'"`
	PostImplementationTypes string `gorm:"type:text;default:'[\"Bug\",\"Defect\",\"Hotfix\",\"Support\"]'"`
}

func (finDevOpsSettings20260130) TableName() string {
	return "_tool_findevops_settings"
}

func (u *addSettings) Up(baseRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		baseRes,
		&finDevOpsSettings20260130{},
	)
}

func (*addSettings) Version() uint64 {
	return 20260130000004
}

func (*addSettings) Name() string {
	return "findevops: add settings table for project-scoped configuration"
}
