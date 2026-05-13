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

type businessMetricsSettings20260130 struct {
	Id          uint64 `gorm:"primaryKey;autoIncrement"`
	ProjectName string `gorm:"type:varchar(255);uniqueIndex"`

	// DORA Elite Benchmarks
	EliteDeployFreq    float64 `gorm:"default:1.0"`
	EliteLeadTimeHours float64 `gorm:"default:24.0"`
	EliteCFR           float64 `gorm:"default:5.0"`
	EliteMTTRHours     float64 `gorm:"default:1.0"`

	// Health level thresholds
	EliteThreshold  int `gorm:"default:80"`
	HighThreshold   int `gorm:"default:60"`
	MediumThreshold int `gorm:"default:40"`

	// Label prefixes
	InvestmentLabelPrefix string `gorm:"type:varchar(100);default:'investment:'"`
	StageLabelPrefix      string `gorm:"type:varchar(100);default:'stage:'"`

	// Business value weights (JSON)
	BusinessValueWeights string `gorm:"type:text"`
}

func (businessMetricsSettings20260130) TableName() string {
	return "_tool_businessmetrics_settings"
}

func (u *addSettings) Up(baseRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		baseRes,
		&businessMetricsSettings20260130{},
	)
}

func (*addSettings) Version() uint64 {
	return 20260130000002
}

func (*addSettings) Name() string {
	return "businessmetrics: add settings table for project-scoped configuration"
}
