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

// BusinessMetricsSettings stores project-scoped configuration for the business metrics plugin
type BusinessMetricsSettings struct {
	Id          uint64 `json:"id" gorm:"primaryKey;autoIncrement"`
	ProjectName string `json:"projectName" gorm:"type:varchar(255);uniqueIndex"`

	// DORA Elite Benchmarks - used in CalculateHealthScore
	EliteDeployFreq    float64 `json:"eliteDeployFreq" gorm:"default:1.0"`    // Deploys per day for elite
	EliteLeadTimeHours float64 `json:"eliteLeadTimeHours" gorm:"default:24.0"` // Hours for elite lead time
	EliteCFR           float64 `json:"eliteCfr" gorm:"default:5.0"`            // Elite change failure rate %
	EliteMTTRHours     float64 `json:"eliteMttrHours" gorm:"default:1.0"`      // Elite MTTR in hours

	// Health level thresholds (total score out of 100)
	EliteThreshold  int `json:"eliteThreshold" gorm:"default:80"`  // Score >= this = Elite
	HighThreshold   int `json:"highThreshold" gorm:"default:60"`   // Score >= this = High
	MediumThreshold int `json:"mediumThreshold" gorm:"default:40"` // Score >= this = Medium, below = Low

	// Label prefixes for extracting business categorization from issues
	InvestmentLabelPrefix string `json:"investmentLabelPrefix" gorm:"type:varchar(100);default:'investment:'"` // e.g., "investment:ktlo"
	StageLabelPrefix      string `json:"stageLabelPrefix" gorm:"type:varchar(100);default:'stage:'"`           // e.g., "stage:development"

	// Business value weights for ROI calculation (JSON)
	// Format: {"feature": 1.0, "bugfix": 0.5, "tech_debt": 0.3}
	BusinessValueWeights string `json:"businessValueWeights" gorm:"type:text"`
}

func (BusinessMetricsSettings) TableName() string {
	return "_tool_businessmetrics_settings"
}

// GetProjectName implements MetricSettings interface
func (s *BusinessMetricsSettings) GetProjectName() string {
	return s.ProjectName
}

// SetProjectName implements MetricSettings interface
func (s *BusinessMetricsSettings) SetProjectName(name string) {
	s.ProjectName = name
}

// NewDefaultSettings creates settings with sensible defaults
func NewDefaultSettings() *BusinessMetricsSettings {
	return &BusinessMetricsSettings{
		EliteDeployFreq:       1.0,
		EliteLeadTimeHours:    24.0,
		EliteCFR:              5.0,
		EliteMTTRHours:        1.0,
		EliteThreshold:        80,
		HighThreshold:         60,
		MediumThreshold:       40,
		InvestmentLabelPrefix: "investment:",
		StageLabelPrefix:      "stage:",
	}
}
