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

var _ plugin.MigrationScript = (*addAdvancedPlanning)(nil)

type addAdvancedPlanning struct{}

func (script *addAdvancedPlanning) Up(basicRes context.BasicRes) errors.Error {
	db := basicRes.GetDal()

	// Create Monte Carlo forecasts table
	err := db.AutoMigrate(&monteCarloForecast20260130{})
	if err != nil {
		return errors.Default.Wrap(err, "failed to create monte_carlo_forecasts table")
	}

	// Create capacity models table (Brooks's Law)
	err = db.AutoMigrate(&capacityModel20260130{})
	if err != nil {
		return errors.Default.Wrap(err, "failed to create capacity_models table")
	}

	// Create investment ROI table
	err = db.AutoMigrate(&investmentROI20260130{})
	if err != nil {
		return errors.Default.Wrap(err, "failed to create investment_rois table")
	}

	return nil
}

func (script *addAdvancedPlanning) Version() uint64 {
	return 20260130000002
}

func (script *addAdvancedPlanning) Name() string {
	return "capacityplanner: add Monte Carlo, Brooks's Law, and ROI tables"
}

// Migration models

type monteCarloForecast20260130 struct {
	Id               string  `gorm:"primaryKey;type:varchar(255)"`
	InitiativeId     string  `gorm:"type:varchar(255);index"`
	SimulationCount  int     `gorm:"type:int"`
	VelocityVariance float64 `gorm:"type:decimal(5,2)"`

	P50Sprints int `gorm:"type:int"`
	P75Sprints int `gorm:"type:int"`
	P90Sprints int `gorm:"type:int"`
	P95Sprints int `gorm:"type:int"`

	P50Date *time.Time
	P75Date *time.Time
	P90Date *time.Time
	P95Date *time.Time

	EarliestDays int `gorm:"type:int"`
	LatestDays   int `gorm:"type:int"`

	CalculatedAt time.Time `gorm:"index"`
}

func (monteCarloForecast20260130) TableName() string {
	return "monte_carlo_forecasts"
}

type capacityModel20260130 struct {
	Id           string `gorm:"primaryKey;type:varchar(255)"`
	ProjectName  string `gorm:"type:varchar(255);index"`
	ScenarioName string `gorm:"type:varchar(255)"`

	CurrentTeamSize int `gorm:"type:int"`
	TeamSizeDelta   int `gorm:"type:int"`
	RampUpWeeks     int `gorm:"type:int;default:8"`

	CurrentChannels int     `gorm:"type:int"`
	NewChannels     int     `gorm:"type:int"`
	OverheadFactor  float64 `gorm:"type:decimal(5,2)"`

	ProductivityFactor   float64 `gorm:"type:decimal(5,2)"`
	ProjectedDeployDelta float64 `gorm:"type:decimal(5,2)"`
	ProjectedLeadDelta   float64 `gorm:"type:decimal(5,2)"`

	CalculatedAt time.Time `gorm:"index"`
}

func (capacityModel20260130) TableName() string {
	return "capacity_models"
}

type investmentROI20260130 struct {
	Id             string `gorm:"primaryKey;type:varchar(255)"`
	InvestmentName string `gorm:"type:varchar(255);index"`
	InvestmentType string `gorm:"type:varchar(50)"`

	UpfrontCost float64 `gorm:"type:decimal(12,2)"`
	MonthlyCost float64 `gorm:"type:decimal(12,2)"`
	AnnualCost  float64 `gorm:"type:decimal(12,2)"`

	DirectBenefit       float64 `gorm:"type:decimal(12,2)"`
	ProductivityBenefit float64 `gorm:"type:decimal(12,2)"`
	QualityBenefit      float64 `gorm:"type:decimal(12,2)"`
	TotalAnnualBenefit  float64 `gorm:"type:decimal(12,2)"`

	PaybackMonths float64 `gorm:"type:decimal(10,2)"`
	ThreeYearROI  float64 `gorm:"type:decimal(10,2)"`

	Parameters string `gorm:"type:text"`

	CalculatedAt time.Time `gorm:"index"`
}

func (investmentROI20260130) TableName() string {
	return "investment_rois"
}
