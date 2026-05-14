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
	"testing"
	"time"

	"github.com/apache/incubator-devlake/plugins/aimeasure/models"
)

func TestPeriodWeekStart(t *testing.T) {
	cases := []struct {
		name string
		in   string // RFC3339
		want string // YYYY-MM-DD of Monday at 00:00 UTC
	}{
		{"monday stays put", "2026-05-11T00:00:00Z", "2026-05-11"},
		{"tuesday rolls back", "2026-05-12T15:30:00Z", "2026-05-11"},
		{"sunday rolls back to prior monday", "2026-05-17T23:59:59Z", "2026-05-11"},
		{"timezone is normalized to UTC", "2026-05-11T02:00:00-05:00", "2026-05-11"}, // = 07:00 UTC
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in, _ := time.Parse(time.RFC3339, c.in)
			got := PeriodWeekStart(in)
			gotStr := got.Format("2006-01-02")
			if gotStr != c.want {
				t.Errorf("PeriodWeekStart(%s) = %s, want %s", c.in, gotStr, c.want)
			}
			if got.Hour() != 0 || got.Minute() != 0 || got.Second() != 0 {
				t.Errorf("week start should be 00:00:00, got %v", got)
			}
		})
	}
}

func TestIsAfterHours(t *testing.T) {
	// After-hours = before 09:00 or after 18:00 local-business-day OR weekend.
	// For Phase B we use UTC and a fixed 09:00-18:00 window; per-engineer
	// timezone is Phase C work.
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"monday 10am UTC", "2026-05-11T10:00:00Z", false},
		{"monday 8am UTC", "2026-05-11T08:00:00Z", true},
		{"monday 8pm UTC", "2026-05-11T20:00:00Z", true},
		{"saturday noon UTC", "2026-05-16T12:00:00Z", true},
		{"sunday 2pm UTC", "2026-05-17T14:00:00Z", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in, _ := time.Parse(time.RFC3339, c.in)
			if got := IsAfterHours(in); got != c.want {
				t.Errorf("IsAfterHours(%s) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

func TestCategorizeChannel(t *testing.T) {
	mapping := map[string]models.ChannelCategory{
		"eng-platform":  models.CategoryEngineering,
		"C0123ABCDEF":   models.CategoryIncidentSupport, // mapped by ID
		"design-review": models.CategoryDesignArchitecture,
	}
	cases := []struct {
		name        string
		channelName string
		channelId   string
		want        models.ChannelCategory
	}{
		{"name match wins over id", "eng-platform", "C0099", models.CategoryEngineering},
		{"id match used if name missing", "", "C0123ABCDEF", models.CategoryIncidentSupport},
		{"unmapped falls back to general", "random-banter", "C9999", models.CategoryGeneral},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := CategorizeChannel(mapping, c.channelName, c.channelId); got != c.want {
				t.Errorf("CategorizeChannel(%q,%q) = %s, want %s", c.channelName, c.channelId, got, c.want)
			}
		})
	}
}
