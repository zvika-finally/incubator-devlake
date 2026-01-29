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

// InvestmentROI stores ROI calculations for development investments
// Supports AI tools, hiring, and tech debt investment types
type InvestmentROI struct {
	Id             string `gorm:"primaryKey;type:varchar(255)"`
	InvestmentName string `gorm:"type:varchar(255);index"`
	InvestmentType string `gorm:"type:varchar(50)"` // ai_tools, hiring, tech_debt

	// Costs
	UpfrontCost float64 `gorm:"type:decimal(12,2)"`
	MonthlyCost float64 `gorm:"type:decimal(12,2)"`
	AnnualCost  float64 `gorm:"type:decimal(12,2)"`

	// Benefits (annual USD)
	DirectBenefit       float64 `gorm:"type:decimal(12,2)"` // hours_saved * 52 * hourly_cost
	ProductivityBenefit float64 `gorm:"type:decimal(12,2)"` // team_hours * gain% * hourly_cost
	QualityBenefit      float64 `gorm:"type:decimal(12,2)"` // bug_hours * improvement% * hourly_cost
	TotalAnnualBenefit  float64 `gorm:"type:decimal(12,2)"`

	// ROI metrics
	PaybackMonths float64 `gorm:"type:decimal(10,2)"` // Months to break even
	ThreeYearROI  float64 `gorm:"type:decimal(10,2)"` // 3-year ROI percentage

	// Input parameters (JSON)
	Parameters string `gorm:"type:text"` // JSON with detailed input parameters

	CalculatedAt time.Time `gorm:"index"`
}

func (InvestmentROI) TableName() string {
	return "investment_rois"
}
