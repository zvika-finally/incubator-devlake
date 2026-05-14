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

package tasks

import (
	"time"

	"github.com/apache/incubator-devlake/plugins/aimeasure/models"
)

// PeriodWeekStart returns the Monday-00:00 UTC of the ISO week containing t.
// All Phase B per-week aggregations bucket by this value.
func PeriodWeekStart(t time.Time) time.Time {
	u := t.UTC()
	// Go's Weekday: Sunday=0, Monday=1 ... Saturday=6. Convert to Mon=0..Sun=6.
	offset := int(u.Weekday()) - 1
	if offset < 0 {
		offset = 6 // Sunday → 6 days back to last Monday
	}
	monday := u.AddDate(0, 0, -offset)
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.UTC)
}

// IsAfterHours returns true when t falls outside Mon–Fri 09:00–18:00 UTC.
// Phase B is workspace-uniform; per-engineer timezone is Phase C work and
// will replace this with a lookup against an engineer_timezone table.
func IsAfterHours(t time.Time) bool {
	u := t.UTC()
	switch u.Weekday() {
	case time.Saturday, time.Sunday:
		return true
	}
	h := u.Hour()
	return h < 9 || h >= 18
}

// CategorizeChannel resolves a channel to its aimeasure category. It tries
// the channel name first (more stable for human-curated mappings), then the
// channel ID. Unmapped channels fall back to CategoryGeneral.
func CategorizeChannel(mapping map[string]models.ChannelCategory, channelName, channelId string) models.ChannelCategory {
	if c, ok := mapping[channelName]; ok && channelName != "" {
		return c
	}
	if c, ok := mapping[channelId]; ok && channelId != "" {
		return c
	}
	return models.CategoryGeneral
}
