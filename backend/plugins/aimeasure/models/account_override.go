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

// AccountOverride is a manually-maintained mapping from a source identity
// (Slack user ID, GitHub login, Jira accountId) to a DevLake account ID.
// Populated by engineering leadership when automatic resolution fails.
type AccountOverride struct {
	SourceSystem string `gorm:"primaryKey;type:varchar(50)" json:"sourceSystem"` // "slack" / "github" / "jira"
	SourceId     string `gorm:"primaryKey;type:varchar(255)" json:"sourceId"`
	AccountId    string `gorm:"type:varchar(255);not null" json:"accountId"`
	Note         string `gorm:"type:varchar(500)" json:"note"`
}

func (AccountOverride) TableName() string {
	return "aimeasure_account_overrides"
}
