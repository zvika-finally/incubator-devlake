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

// EngineerVerificationEffort records how much time an engineer spent reviewing
// vs authoring code per ISO week. One row per (engineer_id, period_week).
// Written by computeVerificationEffort, recomputed on every run.
type EngineerVerificationEffort struct {
	EngineerId               string    `gorm:"primaryKey;type:varchar(255)" json:"engineerId"`
	PeriodWeek               time.Time `gorm:"primaryKey;type:date" json:"periodWeek"` // ISO week start (Monday) at 00:00 UTC
	AuthorMinutes            int       `gorm:"type:int" json:"authorMinutes"`
	ReviewerMinutes          int       `gorm:"type:int" json:"reviewerMinutes"`
	ReviewToAuthorRatio      float64   `gorm:"type:decimal(8,4)" json:"reviewToAuthorRatio"`
	ReviewCommentsTotal      int       `gorm:"type:int" json:"reviewCommentsTotal"`
	ReviewCommentsPerLoc     float64   `gorm:"type:decimal(8,4)" json:"reviewCommentsPerLoc"`
	ReviewCommentsHighCohort int       `gorm:"type:int" json:"reviewCommentsHighCohort"`
	ReviewCommentsPerLocHigh float64   `gorm:"type:decimal(8,4)" json:"reviewCommentsPerLocHigh"`
	ComputedAt               time.Time `gorm:"not null" json:"computedAt"`
}

func (EngineerVerificationEffort) TableName() string {
	return "engineer_verification_effort"
}
