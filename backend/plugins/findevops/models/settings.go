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

	// === FTE NORMALIZATION (Swarmia Model) ===
	FteMaxPerMonth             float64 `json:"fteMaxPerMonth" gorm:"default:1.0"`
	FteBaselineMultiplier      float64 `json:"fteBaselineMultiplier" gorm:"default:1.2"`
	FteInactivityThresholdDays int     `json:"fteInactivityThresholdDays" gorm:"default:5"`
	FteWorkingHoursPerMonth    float64 `json:"fteWorkingHoursPerMonth" gorm:"default:160.0"`

	// === ACTIVITY WEIGHTS (Swarmia Model) ===
	ActivityWeightPrAuthored     float64 `json:"activityWeightPrAuthored" gorm:"default:1.0"`
	ActivityWeightPrReviewed     float64 `json:"activityWeightPrReviewed" gorm:"default:0.3"`
	ActivityWeightCommitAuthored float64 `json:"activityWeightCommitAuthored" gorm:"default:0.2"`
	ActivityWeightIssueUpdated   float64 `json:"activityWeightIssueUpdated" gorm:"default:0.1"`
	ActivityWeightCommentAdded   float64 `json:"activityWeightCommentAdded" gorm:"default:0.05"`

	// === GIT INFERENCE (git2effort methodology) ===
	GitProductiveHoursPerActiveDay float64 `json:"gitProductiveHoursPerActiveDay" gorm:"default:6.0"`
	GitReviewHoursPerCycle         float64 `json:"gitReviewHoursPerCycle" gorm:"default:1.5"`
	GitCommentsPerReviewCycle      int     `json:"gitCommentsPerReviewCycle" gorm:"default:3"`
	GitMinHoursPerIssue            float64 `json:"gitMinHoursPerIssue" gorm:"default:1.0"`
	GitMaxHoursPerIssue            float64 `json:"gitMaxHoursPerIssue" gorm:"default:80.0"`

	// === VALIDATION ===
	ValidationJiraGitVarianceThresholdPct float64 `json:"validationJiraGitVarianceThresholdPct" gorm:"default:50.0"`

	// === ASC 350-40 COMMIT KEYWORDS ===
	PreliminaryCommitKeywords        string `json:"preliminaryCommitKeywords" gorm:"type:text"`
	DevelopmentCommitKeywords        string `json:"developmentCommitKeywords" gorm:"type:text"`
	PostImplementationCommitKeywords string `json:"postImplementationCommitKeywords" gorm:"type:text"`

	// === FEATURE FLAGS ===
	EnableGitEffortInference bool `json:"enableGitEffortInference" gorm:"default:true"`
	EnableFteNormalization   bool `json:"enableFteNormalization" gorm:"default:true"`
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
		DefaultHourlyRate:        87.0,
		HoursPerStoryPoint:       4.0,
		CapitalizationFramework:  "asc_350_40_stages",
		PreliminaryLabels:        `["research","spike","investigation","feasibility","discovery","poc","proof-of-concept","planning"]`,
		PostImplementationLabels: `["bug","hotfix","maintenance","ktlo","support","incident","fix","patch","tech-debt"]`,
		PreliminaryTypes:         `["Spike","Research","Discovery"]`,
		DevelopmentTypes:         `["Story","Feature","Enhancement","Epic"]`,
		PostImplementationTypes:  `["Bug","Defect","Hotfix","Support"]`,

		// FTE Normalization (Swarmia Model)
		FteMaxPerMonth:             1.0,
		FteBaselineMultiplier:      1.2,
		FteInactivityThresholdDays: 5,
		FteWorkingHoursPerMonth:    160.0,

		// Activity Weights (Swarmia Model)
		ActivityWeightPrAuthored:     1.0,
		ActivityWeightPrReviewed:     0.3,
		ActivityWeightCommitAuthored: 0.2,
		ActivityWeightIssueUpdated:   0.1,
		ActivityWeightCommentAdded:   0.05,

		// Git Inference (git2effort methodology)
		GitProductiveHoursPerActiveDay: 6.0,
		GitReviewHoursPerCycle:         1.5,
		GitCommentsPerReviewCycle:      3,
		GitMinHoursPerIssue:            1.0,
		GitMaxHoursPerIssue:            80.0,

		// Validation
		ValidationJiraGitVarianceThresholdPct: 50.0,

		// ASC 350-40 Commit Keywords
		PreliminaryCommitKeywords:        `["spike","research","investigate","explore","feasibility","discovery","planning"]`,
		DevelopmentCommitKeywords:        `["feat","feature","implement","add","build","create","develop"]`,
		PostImplementationCommitKeywords: `["fix","bug","hotfix","patch","maintenance","refactor","chore"]`,

		// Feature Flags
		EnableGitEffortInference: true,
		EnableFteNormalization:   true,
	}
}
