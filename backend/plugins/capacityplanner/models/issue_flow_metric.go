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

// IssueFlowMetric tracks flow efficiency metrics per issue
// Flow efficiency = (Active time) / (Total time) * 100
// Active time = time spent in "In Progress" or similar active statuses
// Total time = calendar days from start to done
type IssueFlowMetric struct {
	Id          string `gorm:"primaryKey;type:varchar(255)"`
	ProjectName string `gorm:"type:varchar(255);index"`
	IssueId     string `gorm:"type:varchar(255);index"`
	IssueKey    string `gorm:"type:varchar(100)"`
	IssueType   string `gorm:"type:varchar(50);index"`

	// Time metrics (in days)
	TotalDays      float64 `gorm:"type:decimal(10,2)"` // Calendar days from first transition to done
	ActiveDays     float64 `gorm:"type:decimal(10,2)"` // Days in "In Progress" or active statuses
	WaitingDays    float64 `gorm:"type:decimal(10,2)"` // Days in waiting/blocked statuses
	FlowEfficiency float64 `gorm:"type:decimal(5,2)"`  // (ActiveDays / TotalDays) * 100

	// Date range
	StartedAt   *time.Time `gorm:"index"`
	CompletedAt *time.Time `gorm:"index"`

	// Status breakdown stored as JSON: {"In Progress": 5.5, "Blocked": 2.1, "Review": 1.3}
	StatusBreakdown string `gorm:"type:text"`

	// Transition count
	TransitionCount int `gorm:"type:int"`

	CalculatedAt time.Time
}

func (IssueFlowMetric) TableName() string {
	return "issue_flow_metrics"
}

// FlowEfficiencyCategory returns a category based on flow efficiency percentage
func (m *IssueFlowMetric) FlowEfficiencyCategory() string {
	switch {
	case m.FlowEfficiency >= 40:
		return "excellent" // World-class (rare)
	case m.FlowEfficiency >= 25:
		return "good" // Above average
	case m.FlowEfficiency >= 15:
		return "average" // Typical for most teams
	default:
		return "poor" // Significant improvement opportunity
	}
}

// ProjectFlowSummary aggregates flow metrics at project/sprint level
type ProjectFlowSummary struct {
	Id          string    `gorm:"primaryKey;type:varchar(255)"`
	ProjectName string    `gorm:"type:varchar(255);index"`
	SprintId    string    `gorm:"type:varchar(255);index"`
	SprintName  string    `gorm:"type:varchar(255)"`
	PeriodStart time.Time `gorm:"index"`
	PeriodEnd   time.Time `gorm:"index"`

	// Aggregate metrics
	IssueCount           int     `gorm:"type:int"`
	AvgFlowEfficiency    float64 `gorm:"type:decimal(5,2)"`
	MedianFlowEfficiency float64 `gorm:"type:decimal(5,2)"`
	P90FlowEfficiency    float64 `gorm:"type:decimal(5,2)"`

	AvgTotalDays   float64 `gorm:"type:decimal(10,2)"`
	AvgActiveDays  float64 `gorm:"type:decimal(10,2)"`
	AvgWaitingDays float64 `gorm:"type:decimal(10,2)"`

	// Count by category
	ExcellentCount int `gorm:"type:int"` // >= 40%
	GoodCount      int `gorm:"type:int"` // 25-39%
	AverageCount   int `gorm:"type:int"` // 15-24%
	PoorCount      int `gorm:"type:int"` // < 15%

	CalculatedAt time.Time
}

func (ProjectFlowSummary) TableName() string {
	return "project_flow_summaries"
}
