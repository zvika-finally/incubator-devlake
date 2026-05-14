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

// EngineerDxiProxy holds a behavioral sentiment proxy plus optional survey data
// for an engineer in a given ISO week. Survey fields are nullable — populated
// only if a DXI/eNPS survey ingest exists (out of scope for Phase B).
type EngineerDxiProxy struct {
	EngineerId         string     `gorm:"primaryKey;type:varchar(255)" json:"engineerId"`
	PeriodWeek         time.Time  `gorm:"primaryKey;type:date" json:"periodWeek"`
	SentimentScore     float64    `gorm:"type:decimal(5,2)" json:"sentimentScore"`     // 0–100, behavioral
	BadDeveloperDayFlag bool      `gorm:"type:bool" json:"badDeveloperDayFlag"`
	LastSurveyDate     *time.Time `gorm:"type:date" json:"lastSurveyDate,omitempty"`
	LastSurveyDxi      *float64   `gorm:"type:decimal(5,2)" json:"lastSurveyDxi,omitempty"`
	ComputedAt         time.Time  `gorm:"not null" json:"computedAt"`
}

func (EngineerDxiProxy) TableName() string {
	return "engineer_dxi_proxy"
}
