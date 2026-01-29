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

package migrationscripts

import (
	"time"

	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/helpers/migrationhelper"
)

type addFlowEfficiency struct{}

func (script *addFlowEfficiency) Up(basicRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&issueFlowMetric20260131{},
		&projectFlowSummary20260131{},
	)
}

func (script *addFlowEfficiency) Version() uint64 {
	return 20260131000003
}

func (script *addFlowEfficiency) Name() string {
	return "capacityplanner: add flow efficiency metrics"
}

// Migration models

type issueFlowMetric20260131 struct {
	Id              string     `gorm:"primaryKey;type:varchar(255)"`
	ProjectName     string     `gorm:"type:varchar(255);index"`
	IssueId         string     `gorm:"type:varchar(255);index"`
	IssueKey        string     `gorm:"type:varchar(100)"`
	IssueType       string     `gorm:"type:varchar(50);index"`
	TotalDays       float64    `gorm:"type:decimal(10,2)"`
	ActiveDays      float64    `gorm:"type:decimal(10,2)"`
	WaitingDays     float64    `gorm:"type:decimal(10,2)"`
	FlowEfficiency  float64    `gorm:"type:decimal(5,2)"`
	StartedAt       *time.Time `gorm:"index"`
	CompletedAt     *time.Time `gorm:"index"`
	StatusBreakdown string     `gorm:"type:text"`
	TransitionCount int        `gorm:"type:int"`
	CalculatedAt    time.Time
}

func (issueFlowMetric20260131) TableName() string {
	return "issue_flow_metrics"
}

type projectFlowSummary20260131 struct {
	Id                   string    `gorm:"primaryKey;type:varchar(255)"`
	ProjectName          string    `gorm:"type:varchar(255);index"`
	SprintId             string    `gorm:"type:varchar(255);index"`
	SprintName           string    `gorm:"type:varchar(255)"`
	PeriodStart          time.Time `gorm:"index"`
	PeriodEnd            time.Time `gorm:"index"`
	IssueCount           int       `gorm:"type:int"`
	AvgFlowEfficiency    float64   `gorm:"type:decimal(5,2)"`
	MedianFlowEfficiency float64   `gorm:"type:decimal(5,2)"`
	P90FlowEfficiency    float64   `gorm:"type:decimal(5,2)"`
	AvgTotalDays         float64   `gorm:"type:decimal(10,2)"`
	AvgActiveDays        float64   `gorm:"type:decimal(10,2)"`
	AvgWaitingDays       float64   `gorm:"type:decimal(10,2)"`
	ExcellentCount       int       `gorm:"type:int"`
	GoodCount            int       `gorm:"type:int"`
	AverageCount         int       `gorm:"type:int"`
	PoorCount            int       `gorm:"type:int"`
	CalculatedAt         time.Time
}

func (projectFlowSummary20260131) TableName() string {
	return "project_flow_summaries"
}
