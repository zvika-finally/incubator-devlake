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

import "time"

// ChannelCategory enumerates the buckets aimeasure groups Slack channels into.
// Stored as varchar; mapping is kept in slack_channel_categories.
type ChannelCategory string

const (
	CategoryEngineering        ChannelCategory = "engineering"
	CategoryIncidentSupport    ChannelCategory = "incident_support"
	CategoryDesignArchitecture ChannelCategory = "design_architecture"
	CategoryGeneral            ChannelCategory = "general"
)

// EngineerSlackSignals records per-engineer per-week Slack participation
// in each channel category. One row per (engineer_id, period_week, channel_category).
type EngineerSlackSignals struct {
	EngineerId               string          `gorm:"primaryKey;type:varchar(255)" json:"engineerId"`
	PeriodWeek               time.Time       `gorm:"primaryKey;type:date" json:"periodWeek"`
	ChannelCategory          ChannelCategory `gorm:"primaryKey;type:varchar(32)" json:"channelCategory"`
	MessageCount             int             `gorm:"type:int" json:"messageCount"`
	ThreadParticipationCount int             `gorm:"type:int" json:"threadParticipationCount"`
	AfterHoursMessageCount   int             `gorm:"type:int" json:"afterHoursMessageCount"`
	AfterHoursRatio          float64         `gorm:"type:decimal(5,4)" json:"afterHoursRatio"`
	ComputedAt               time.Time       `gorm:"not null" json:"computedAt"`
}

func (EngineerSlackSignals) TableName() string {
	return "engineer_slack_signals"
}
