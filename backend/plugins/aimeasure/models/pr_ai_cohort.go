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

// AICohort enumerates the AI-assistance level of a PR.
// Values are stored as varchar strings, never numeric IDs.
type AICohort string

const (
	CohortNone   AICohort = "NONE"
	CohortLow    AICohort = "LOW"
	CohortMedium AICohort = "MEDIUM"
	CohortHigh   AICohort = "HIGH"
)

// PRAICohort is the cohort classification for a merged pull request.
// One row per PR. Rewritten by the classifyPRCohort subtask.
type PRAICohort struct {
	PRId              string    `gorm:"primaryKey;type:varchar(255)" json:"prId"`
	AICohort          AICohort  `gorm:"type:varchar(20);not null;index" json:"aiCohort"`
	ConfidenceScore   int       `gorm:"type:int" json:"confidenceScore"`
	HasExplicitMarker bool      `gorm:"type:bool" json:"hasExplicitMarker"`
	HasCommitTrailer  bool      `gorm:"type:bool" json:"hasCommitTrailer"`
	ClassifierVersion string    `gorm:"type:varchar(32);not null" json:"classifierVersion"`
	ClassifiedAt      time.Time `gorm:"not null" json:"classifiedAt"`
}

func (PRAICohort) TableName() string {
	return "pr_ai_cohort"
}
