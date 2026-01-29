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

// MonteCarloForecast stores probabilistic completion forecasts using Monte Carlo simulation
// Runs 1000 iterations with variable velocity to produce percentile-based completion estimates
type MonteCarloForecast struct {
	Id               string  `gorm:"primaryKey;type:varchar(255)"`
	InitiativeId     string  `gorm:"type:varchar(255);index"`
	SimulationCount  int     `gorm:"type:int"`          // Number of iterations (default: 1000)
	VelocityVariance float64 `gorm:"type:decimal(5,2)"` // Velocity variance factor (default: 0.25)

	// Percentile outcomes in sprints
	P50Sprints int `gorm:"type:int"` // 50th percentile (median)
	P75Sprints int `gorm:"type:int"` // 75th percentile
	P90Sprints int `gorm:"type:int"` // 90th percentile
	P95Sprints int `gorm:"type:int"` // 95th percentile (conservative)

	// Percentile dates (calculated from current date + sprints)
	P50Date *time.Time
	P75Date *time.Time
	P90Date *time.Time
	P95Date *time.Time

	// Range statistics
	EarliestDays int `gorm:"type:int"` // Best case scenario
	LatestDays   int `gorm:"type:int"` // Worst case scenario

	CalculatedAt time.Time `gorm:"index"`
}

func (MonteCarloForecast) TableName() string {
	return "monte_carlo_forecasts"
}
