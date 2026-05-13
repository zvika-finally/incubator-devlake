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

// PRDefectSignals records whether a merged PR was followed by a defect indicator
// (revert / hotfix / incident) within a 14-day window. One row per PR; recomputed
// nightly until WindowCloseDate passes (30 days after merge).
type PRDefectSignals struct {
	PRId                  string    `gorm:"primaryKey;type:varchar(255)" json:"prId"`
	HasRevert14d          bool      `gorm:"type:bool" json:"hasRevert14d"`
	HasHotfix14d          bool      `gorm:"type:bool" json:"hasHotfix14d"`
	HasIncident14d        bool      `gorm:"type:bool" json:"hasIncident14d"`
	IncidentDataAvailable bool      `gorm:"type:bool" json:"incidentDataAvailable"`
	TotalDefectCount      int       `gorm:"type:int" json:"totalDefectCount"`
	WindowCloseDate       time.Time `gorm:"not null;index" json:"windowCloseDate"`
	ComputedAt            time.Time `gorm:"not null" json:"computedAt"`
}

func (PRDefectSignals) TableName() string {
	return "pr_defect_signals"
}
