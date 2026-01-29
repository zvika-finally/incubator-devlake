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

// CapacityPlannerSettings stores project-scoped configuration for the capacity planner plugin
type CapacityPlannerSettings struct {
	Id          uint64 `json:"id" gorm:"primaryKey;autoIncrement"`
	ProjectName string `json:"projectName" gorm:"type:varchar(255);uniqueIndex"`

	// Monte Carlo simulation parameters
	MonteCarloIterations int     `json:"monteCarloIterations" gorm:"default:1000"` // Number of simulation iterations
	VelocityVariance     float64 `json:"velocityVariance" gorm:"default:0.25"`     // Variance factor for velocity (0-1)
	DefaultVelocity      float64 `json:"defaultVelocity" gorm:"default:20.0"`      // Default story points per week

	// Sprint parameters
	SprintDurationWeeks int `json:"sprintDurationWeeks" gorm:"default:2"` // Sprint duration in weeks
	VelocitySprintCount int `json:"velocitySprintCount" gorm:"default:6"` // Sprints to use for velocity calculation

	// Brooks's Law model parameters
	RampUpWeeks         float64 `json:"rampUpWeeks" gorm:"default:4.0"`    // Weeks for new hire to ramp up
	NewHireProductivity float64 `json:"newHireProductivity" gorm:"default:0.5"` // New hire productivity during ramp-up (0-1)
	ChannelOverhead     float64 `json:"channelOverhead" gorm:"default:0.1"`     // Communication overhead per new channel (0-1)

	// ROI calculation parameters
	DefaultDeveloperCost  float64 `json:"defaultDeveloperCost" gorm:"default:150000.0"` // Annual developer cost
	RoiTimeHorizonMonths  int     `json:"roiTimeHorizonMonths" gorm:"default:12"`       // ROI calculation horizon

	// Forecast percentiles (JSON array)
	// Format: [50, 80, 90, 95] for P50, P80, P90, P95 percentiles
	ForecastPercentiles string `json:"forecastPercentiles" gorm:"type:varchar(255);default:'[50,80,90,95]'"`
}

func (CapacityPlannerSettings) TableName() string {
	return "_tool_capacityplanner_settings"
}

// GetProjectName implements MetricSettings interface
func (s *CapacityPlannerSettings) GetProjectName() string {
	return s.ProjectName
}

// SetProjectName implements MetricSettings interface
func (s *CapacityPlannerSettings) SetProjectName(name string) {
	s.ProjectName = name
}

// NewDefaultSettings creates settings with sensible defaults
func NewDefaultSettings() *CapacityPlannerSettings {
	return &CapacityPlannerSettings{
		MonteCarloIterations:  1000,
		VelocityVariance:      0.25,
		DefaultVelocity:       20.0,
		SprintDurationWeeks:   2,
		VelocitySprintCount:   6,
		RampUpWeeks:           4.0,
		NewHireProductivity:   0.5,
		ChannelOverhead:       0.1,
		DefaultDeveloperCost:  150000.0,
		RoiTimeHorizonMonths:  12,
		ForecastPercentiles:   "[50,80,90,95]",
	}
}
