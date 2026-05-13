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

func (script *initSchema) Up(basicRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&cursorConnection20260131{},
		&cursorUsageMetric20260131{},
		&cursorUserMetric20260131{},
	)
}

func (script *initSchema) Version() uint64 {
	return 20260131000005
}

func (script *initSchema) Name() string {
	return "cursor: init schema for connections and metrics"
}

// Migration models

type cursorConnection20260131 struct {
	ID                 uint64 `gorm:"primaryKey;autoIncrement"`
	Name               string `gorm:"type:varchar(255);not null"`
	TeamId             string `gorm:"type:varchar(255)"`
	ApiKey             string `gorm:"type:text;serializer:encdec"`
	RateLimitPerSecond int    `gorm:"default:5"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (cursorConnection20260131) TableName() string {
	return "_tool_cursor_connections"
}

type cursorUsageMetric20260131 struct {
	Id                     string    `gorm:"primaryKey;type:varchar(255)"`
	ConnectionId           uint64    `gorm:"type:bigint;index"`
	TeamId                 string    `gorm:"type:varchar(255);index"`
	Date                   time.Time `gorm:"index"`
	TotalSuggestions       int       `gorm:"type:int"`
	TotalAcceptances       int       `gorm:"type:int"`
	AcceptanceRate         float64   `gorm:"type:decimal(5,2)"`
	GreenLinesAccepted     int       `gorm:"type:int"`
	GreenLinesSuggested    int       `gorm:"type:int"`
	RedLinesAccepted       int       `gorm:"type:int"`
	RedLinesSuggested      int       `gorm:"type:int"`
	LineAcceptanceRatio    float64   `gorm:"type:decimal(5,2)"`
	TabSuggestions         int       `gorm:"type:int"`
	TabAcceptances         int       `gorm:"type:int"`
	TabAcceptanceRate      float64   `gorm:"type:decimal(5,2)"`
	ComposerSuggestions    int       `gorm:"type:int"`
	ComposerAcceptances    int       `gorm:"type:int"`
	ComposerAcceptanceRate float64   `gorm:"type:decimal(5,2)"`
	DailyActiveUsers       int       `gorm:"type:int"`
	CollectedAt            time.Time
}

func (cursorUsageMetric20260131) TableName() string {
	return "cursor_usage_metrics"
}

type cursorUserMetric20260131 struct {
	Id                  string    `gorm:"primaryKey;type:varchar(255)"`
	ConnectionId        uint64    `gorm:"type:bigint;index"`
	TeamId              string    `gorm:"type:varchar(255);index"`
	UserId              string    `gorm:"type:varchar(255);index"`
	UserEmail           string    `gorm:"type:varchar(255)"`
	Date                time.Time `gorm:"index"`
	TabSuggestions      int       `gorm:"type:int"`
	TabAcceptances      int       `gorm:"type:int"`
	ComposerSuggestions int       `gorm:"type:int"`
	ComposerAcceptances int       `gorm:"type:int"`
	AcceptanceRate      float64   `gorm:"type:decimal(5,2)"`
	LinesAccepted       int       `gorm:"type:int"`
	LinesSuggested      int       `gorm:"type:int"`
	CollectedAt         time.Time
}

func (cursorUserMetric20260131) TableName() string {
	return "cursor_user_metrics"
}
