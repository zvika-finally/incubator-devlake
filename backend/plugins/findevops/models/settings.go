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

// FinDevOpsSettings stores project-scoped configuration for the FinDevOps plugin
type FinDevOpsSettings struct {
	Id          uint64 `json:"id" gorm:"primaryKey;autoIncrement"`
	ProjectName string `json:"projectName" gorm:"type:varchar(255);uniqueIndex"`

	// Default hourly rate when individual rates aren't available
	DefaultHourlyRate float64 `json:"defaultHourlyRate" gorm:"default:87.0"` // Default blended rate

	// Role-based hourly rates (JSON)
	// Format: {"engineer": 72.0, "seniorEngineer": 96.0, "staffEngineer": 120.0}
	RoleRates string `json:"roleRates" gorm:"type:text"`

	// Story point to hours conversion
	HoursPerStoryPoint float64 `json:"hoursPerStoryPoint" gorm:"default:4.0"` // Default hours per story point

	// Capitalization framework
	CapitalizationFramework string `json:"capitalizationFramework" gorm:"type:varchar(50);default:'asc_350_40_stages'"`

	// ASC 350-40 label mappings (JSON arrays)
	// Labels that indicate preliminary stage (expensed) - defaults via NewDefaultSettings()
	PreliminaryLabels string `json:"preliminaryLabels" gorm:"type:text"`

	// Labels that indicate post-implementation stage (expensed) - defaults via NewDefaultSettings()
	PostImplementationLabels string `json:"postImplementationLabels" gorm:"type:text"`

	// Issue type mappings (JSON arrays) - defaults via NewDefaultSettings()
	PreliminaryTypes        string `json:"preliminaryTypes" gorm:"type:text"`
	DevelopmentTypes        string `json:"developmentTypes" gorm:"type:text"`
	PostImplementationTypes string `json:"postImplementationTypes" gorm:"type:text"`
}

func (FinDevOpsSettings) TableName() string {
	return "_tool_findevops_settings"
}

// GetProjectName implements MetricSettings interface
func (s *FinDevOpsSettings) GetProjectName() string {
	return s.ProjectName
}

// SetProjectName implements MetricSettings interface
func (s *FinDevOpsSettings) SetProjectName(name string) {
	s.ProjectName = name
}

// NewDefaultSettings creates settings with sensible defaults
func NewDefaultSettings() *FinDevOpsSettings {
	return &FinDevOpsSettings{
		DefaultHourlyRate:       87.0,
		HoursPerStoryPoint:      4.0,
		CapitalizationFramework: "asc_350_40_stages",
		PreliminaryLabels:       `["research","spike","investigation","feasibility","discovery","poc","proof-of-concept","planning"]`,
		PostImplementationLabels: `["bug","hotfix","maintenance","ktlo","support","incident","fix","patch","tech-debt"]`,
		PreliminaryTypes:        `["Spike","Research","Discovery"]`,
		DevelopmentTypes:        `["Story","Feature","Enhancement","Epic"]`,
		PostImplementationTypes: `["Bug","Defect","Hotfix","Support"]`,
	}
}
