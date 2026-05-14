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

// SlackChannelCategory maps a Slack channel (by name or ID) to one of the
// aimeasure ChannelCategory values. Manually maintained — engineering leadership
// curates the mapping. Lookup falls back to "general" for unmapped channels.
type SlackChannelCategory struct {
	ChannelKey string `gorm:"primaryKey;type:varchar(255)" json:"channelKey"` // channel name OR channel ID (whichever is more stable)
	Category   string `gorm:"type:varchar(32);not null" json:"category"`
	Note       string `gorm:"type:varchar(500)" json:"note,omitempty"`
}

func (SlackChannelCategory) TableName() string {
	return "aimeasure_slack_channel_categories"
}
