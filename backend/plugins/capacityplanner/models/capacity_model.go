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

// CapacityModel stores Brooks's Law capacity impact analysis for team size changes
// Uses communication overhead formula: channels = n*(n-1)/2
type CapacityModel struct {
	Id           string `gorm:"primaryKey;type:varchar(255)"`
	ProjectName  string `gorm:"type:varchar(255);index"`
	ScenarioName string `gorm:"type:varchar(255)"`

	// Team parameters
	CurrentTeamSize int `gorm:"type:int"`
	TeamSizeDelta   int `gorm:"type:int"`           // +2 or -1
	RampUpWeeks     int `gorm:"type:int;default:8"` // New hire ramp-up period

	// Communication overhead (Brooks's Law)
	CurrentChannels int     `gorm:"type:int"`          // n*(n-1)/2
	NewChannels     int     `gorm:"type:int"`          // After team size change
	OverheadFactor  float64 `gorm:"type:decimal(5,2)"` // Communication overhead multiplier

	// Projected impact
	ProductivityFactor   float64 `gorm:"type:decimal(5,2)"` // Effective capacity multiplier
	ProjectedDeployDelta float64 `gorm:"type:decimal(5,2)"` // Percentage change in deploy frequency
	ProjectedLeadDelta   float64 `gorm:"type:decimal(5,2)"` // Percentage change in lead time

	CalculatedAt time.Time `gorm:"index"`
}

func (CapacityModel) TableName() string {
	return "capacity_models"
}
