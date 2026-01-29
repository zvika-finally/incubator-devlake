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

// TeamHealthScore stores DORA-based health scoring for a project
// Each DORA metric contributes 0-25 points for a total of 0-100
type TeamHealthScore struct {
	Id          string    `gorm:"primaryKey;type:varchar(255)"`
	ProjectName string    `gorm:"type:varchar(255);index"`
	PeriodStart time.Time `gorm:"index"`
	PeriodEnd   time.Time `gorm:"index"`

	// Raw DORA metrics
	DeployFrequency   float64 `gorm:"type:decimal(10,4)"` // Deploys per day
	LeadTimeHours     float64 `gorm:"type:decimal(10,2)"` // Hours from commit to deploy
	ChangeFailureRate float64 `gorm:"type:decimal(5,2)"`  // Percentage (0-100)
	MTTRHours         float64 `gorm:"type:decimal(10,2)"` // Mean time to recovery in hours

	// Individual scores (0-25 each, scaled against elite benchmarks)
	DeployFreqScore int `gorm:"type:int"`
	LeadTimeScore   int `gorm:"type:int"`
	CFRScore        int `gorm:"type:int"`
	MTTRScore       int `gorm:"type:int"`

	// Total score (0-100) and health level
	TotalScore  int    `gorm:"type:int"`
	HealthLevel string `gorm:"type:varchar(20)"` // elite, high, medium, low

	CalculatedAt time.Time `gorm:"index"`
}

func (TeamHealthScore) TableName() string {
	return "team_health_scores"
}
