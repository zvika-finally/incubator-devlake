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

// AIDetectorSettings stores project-scoped configuration for the AI detector plugin
type AIDetectorSettings struct {
	Id          uint64 `json:"id" gorm:"primaryKey;autoIncrement"`
	ProjectName string `json:"projectName" gorm:"type:varchar(255);uniqueIndex"`

	// Confidence thresholds (0-100) - used in DetectExplicitSignals
	ConfidenceTrailer int `json:"confidenceTrailer" gorm:"default:95"` // Git trailers - highest confidence
	ConfidenceBody    int `json:"confidenceBody" gorm:"default:85"`    // PR body/commit markers
	ConfidenceGeneric int `json:"confidenceGeneric" gorm:"default:60"` // Generic AI indicators
	ConfidenceEmail   int `json:"confidenceEmail" gorm:"default:90"`   // Known AI bot emails

	// Detection threshold - minimum confidence to flag as AI-assisted
	DetectionThreshold int `json:"detectionThreshold" gorm:"default:70"`

	// Scoring weights for composite AI confidence score (should sum to 100)
	ExplicitSignalWeight   int `json:"explicitSignalWeight" gorm:"default:50"`   // Weight for explicit tool markers
	BehavioralSignalWeight int `json:"behavioralSignalWeight" gorm:"default:30"` // Weight for behavioral patterns
	PRPatternWeight        int `json:"prPatternWeight" gorm:"default:20"`        // Weight for PR characteristics

	// Custom tool patterns (JSON array of patterns)
	// Format: [{"tool": "my_tool", "patterns": ["pattern1", "pattern2"]}]
	CustomToolPatterns string `json:"customToolPatterns" gorm:"type:text"`

	// File exclusion patterns for churn analysis (JSON array of glob patterns)
	// Infrastructure files like .github/, package.json often have high churn but aren't application code
	// Format: [".github/", "package.json", "*.lock", "Dockerfile"]
	ExcludeFilePatterns string `json:"excludeFilePatterns" gorm:"type:text;default:'[\".github/\",\"package.json\",\"package-lock.json\",\"yarn.lock\",\"pnpm-lock.yaml\",\"Dockerfile\",\"docker-compose.yml\",\".gitignore\",\".eslintrc\",\"tsconfig.json\"]'"`

	// Whether to analyze historical PRs (before plugin was installed)
	AnalyzeHistorical bool `json:"analyzeHistorical" gorm:"default:true"`
}

func (AIDetectorSettings) TableName() string {
	return "_tool_aidetector_settings"
}

// GetProjectName implements MetricSettings interface
func (s *AIDetectorSettings) GetProjectName() string {
	return s.ProjectName
}

// SetProjectName implements MetricSettings interface
func (s *AIDetectorSettings) SetProjectName(name string) {
	s.ProjectName = name
}

// NewDefaultSettings creates settings with sensible defaults
func NewDefaultSettings() *AIDetectorSettings {
	return &AIDetectorSettings{
		ConfidenceTrailer:      95,
		ConfidenceBody:         85,
		ConfidenceGeneric:      60,
		ConfidenceEmail:        90,
		DetectionThreshold:     70,
		ExplicitSignalWeight:   50,
		BehavioralSignalWeight: 30,
		PRPatternWeight:        20,
		AnalyzeHistorical:      true,
		ExcludeFilePatterns:    `[".github/","package.json","package-lock.json","yarn.lock","pnpm-lock.yaml","Dockerfile","docker-compose.yml",".gitignore",".eslintrc","tsconfig.json"]`,
	}
}
