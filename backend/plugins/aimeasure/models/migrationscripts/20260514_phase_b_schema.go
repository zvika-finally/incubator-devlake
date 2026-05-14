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

type phaseBSchema struct{}

// Phase B tables. Snapshot types are frozen to this migration date so future
// model changes do not retroactively alter what this migration creates.

type engineerVerificationEffort20260514 struct {
	EngineerId               string    `gorm:"primaryKey;type:varchar(255)"`
	PeriodWeek               time.Time `gorm:"primaryKey;type:date"`
	AuthorMinutes            int       `gorm:"type:int"`
	ReviewerMinutes          int       `gorm:"type:int"`
	ReviewToAuthorRatio      float64   `gorm:"type:decimal(8,4)"`
	ReviewCommentsTotal      int       `gorm:"type:int"`
	ReviewCommentsPerLoc     float64   `gorm:"type:decimal(8,4)"`
	ReviewCommentsHighCohort int       `gorm:"type:int"`
	ReviewCommentsPerLocHigh float64   `gorm:"type:decimal(8,4)"`
	ComputedAt               time.Time `gorm:"not null"`
}

func (engineerVerificationEffort20260514) TableName() string { return "engineer_verification_effort" }

type engineerSlackSignals20260514 struct {
	EngineerId               string    `gorm:"primaryKey;type:varchar(255)"`
	PeriodWeek               time.Time `gorm:"primaryKey;type:date"`
	ChannelCategory          string    `gorm:"primaryKey;type:varchar(32)"`
	MessageCount             int       `gorm:"type:int"`
	ThreadParticipationCount int       `gorm:"type:int"`
	AfterHoursMessageCount   int       `gorm:"type:int"`
	AfterHoursRatio          float64   `gorm:"type:decimal(5,4)"`
	ComputedAt               time.Time `gorm:"not null"`
}

func (engineerSlackSignals20260514) TableName() string { return "engineer_slack_signals" }

type engineerDxiProxy20260514 struct {
	EngineerId          string     `gorm:"primaryKey;type:varchar(255)"`
	PeriodWeek          time.Time  `gorm:"primaryKey;type:date"`
	SentimentScore      float64    `gorm:"type:decimal(5,2)"`
	BadDeveloperDayFlag bool       `gorm:"type:bool"`
	LastSurveyDate      *time.Time `gorm:"type:date"`
	LastSurveyDxi       *float64   `gorm:"type:decimal(5,2)"`
	ComputedAt          time.Time  `gorm:"not null"`
}

func (engineerDxiProxy20260514) TableName() string { return "engineer_dxi_proxy" }

type slackChannelCategory20260514 struct {
	ChannelKey string `gorm:"primaryKey;type:varchar(255)"`
	Category   string `gorm:"type:varchar(32);not null"`
	Note       string `gorm:"type:varchar(500)"`
}

func (slackChannelCategory20260514) TableName() string { return "aimeasure_slack_channel_categories" }

func (*phaseBSchema) Up(basicRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&engineerVerificationEffort20260514{},
		&engineerSlackSignals20260514{},
		&engineerDxiProxy20260514{},
		&slackChannelCategory20260514{},
	)
}

func (*phaseBSchema) Version() uint64 {
	return 20260514000001
}

func (*phaseBSchema) Name() string {
	return "aimeasure: Phase B schema (verification, slack, sentiment + channel mapping)"
}
