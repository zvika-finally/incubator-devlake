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

import "time"

// DeveloperMonthlyFte tracks FTE normalization per developer per month
// Based on Swarmia's methodology: normalize activity to max 1 FTE, weight different activities
type DeveloperMonthlyFte struct {
	Id          string `gorm:"primaryKey;type:varchar(255)"` // {developer_id}:{fiscal_month}
	DeveloperId string `gorm:"type:varchar(255);index"`
	FiscalMonth string `gorm:"type:varchar(10);index"`
	ProjectName string `gorm:"type:varchar(255);index"`

	// Activity counts (raw signals from Git/Jira)
	PrsAuthored     int `gorm:"type:int;default:0"`
	PrsReviewed     int `gorm:"type:int;default:0"`
	CommitsAuthored int `gorm:"type:int;default:0"`
	IssuesUpdated   int `gorm:"type:int;default:0"`
	CommentsAdded   int `gorm:"type:int;default:0"`

	// FTE calculation (Swarmia methodology)
	RawActivityScore float64 `gorm:"type:decimal(10,2)"` // Weighted sum of activities
	BaselineScore    float64 `gorm:"type:decimal(10,2)"` // Team median × multiplier
	RawFte           float64 `gorm:"type:decimal(3,2)"`  // Before inactivity adjustment
	InactiveDays     int     `gorm:"type:int;default:0"` // Consecutive days with no activity
	AdjustedFte      float64 `gorm:"type:decimal(3,2)"`  // Final FTE after deductions

	// Hours allocation tracking
	HoursFromJira        float64 `gorm:"type:decimal(10,2);default:0"` // Hours from Jira time tracking
	HoursFromGitInferred float64 `gorm:"type:decimal(10,2);default:0"` // Hours inferred from Git
	HoursDistributed     float64 `gorm:"type:decimal(10,2);default:0"` // Hours distributed via FTE
	TotalAllocatedHours  float64 `gorm:"type:decimal(10,2);default:0"` // Sum of all allocated hours

	CalculatedAt time.Time
}

func (DeveloperMonthlyFte) TableName() string {
	return "developer_monthly_fte"
}
