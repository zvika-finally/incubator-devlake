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

type capacityPlannerSettings20260130 struct {
	Id          uint64 `gorm:"primaryKey;autoIncrement"`
	ProjectName string `gorm:"type:varchar(255);uniqueIndex"`

	// Monte Carlo parameters
	MonteCarloIterations int     `gorm:"default:1000"`
	VelocityVariance     float64 `gorm:"default:0.25"`
	DefaultVelocity      float64 `gorm:"default:20.0"`

	// Sprint parameters
	SprintDurationWeeks int `gorm:"default:2"`
	VelocitySprintCount int `gorm:"default:6"`

	// Brooks's Law parameters
	RampUpWeeks         float64 `gorm:"default:4.0"`
	NewHireProductivity float64 `gorm:"default:0.5"`
	ChannelOverhead     float64 `gorm:"default:0.1"`

	// ROI parameters
	DefaultDeveloperCost float64 `gorm:"default:150000.0"`
	RoiTimeHorizonMonths int     `gorm:"default:12"`

	// Forecast percentiles (JSON)
	ForecastPercentiles string `gorm:"type:varchar(255);default:'[50,80,90,95]'"`
}

func (capacityPlannerSettings20260130) TableName() string {
	return "_tool_capacityplanner_settings"
}

func (u *addSettings) Up(baseRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		baseRes,
		&capacityPlannerSettings20260130{},
	)
}

func (*addSettings) Version() uint64 {
	return 20260130000003
}

func (*addSettings) Name() string {
	return "capacityplanner: add settings table for project-scoped configuration"
}
