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

type initSchema struct{}

type aiUsageSignal20260129 struct {
	Id                    string    `gorm:"primaryKey;type:varchar(255)"`
	PullRequestId         string    `gorm:"type:varchar(255);not null;index"`
	AIConfidenceScore     int       `gorm:"type:int"`
	DetectedTool          string    `gorm:"type:varchar(50)"`
	RapidCommitScore      int       `gorm:"type:int"`
	PRSizeScore           int       `gorm:"type:int"`
	LinesPerMinuteScore   int       `gorm:"type:int"`
	DuplicationScore      int       `gorm:"type:int"`
	GenericMessageScore   int       `gorm:"type:int"`
	AvgTimeBetweenCommits float64   `gorm:"type:decimal(10,2)"`
	LinesPerMinute        float64   `gorm:"type:decimal(10,2)"`
	PRAdditions           int
	PRDeletions           int
	CommitCount           int
	CycleTimeHours        float64   `gorm:"type:decimal(10,2)"`
	VelocityMultiplier    float64   `gorm:"type:decimal(5,2)"`
	PatternSignatures     string    `gorm:"type:text"`
	DetectedAt            time.Time
	CreatedAt             time.Time
}

func (aiUsageSignal20260129) TableName() string {
	return "ai_usage_signals"
}

type developerBaseline20260129 struct {
	Id                    string    `gorm:"primaryKey;type:varchar(255)"`
	DeveloperId           string    `gorm:"type:varchar(255);not null;uniqueIndex"`
	AvgPRAdditions        float64   `gorm:"type:decimal(10,2)"`
	AvgPRDeletions        float64   `gorm:"type:decimal(10,2)"`
	AvgCycleTimeHours     float64   `gorm:"type:decimal(10,2)"`
	AvgCommitsPerPR       float64   `gorm:"type:decimal(10,2)"`
	AvgTimeBetweenCommits float64   `gorm:"type:decimal(10,2)"`
	PRCount               int
	CalculatedAt          time.Time
}

func (developerBaseline20260129) TableName() string {
	return "developer_baselines"
}

func (u *initSchema) Up(baseRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		baseRes,
		&aiUsageSignal20260129{},
		&developerBaseline20260129{},
	)
}

func (*initSchema) Version() uint64 {
	return 20260129000002
}

func (*initSchema) Name() string {
	return "aidetector: init schema for AI usage signals and developer baselines"
}
