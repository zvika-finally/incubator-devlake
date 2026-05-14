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

import "testing"

func TestResolveSlackEngineer(t *testing.T) {
	overrides := map[string]string{
		"slack:U001": "account-alice",
	}
	if got := ResolveSlackEngineer(overrides, "U001"); got != "account-alice" {
		t.Errorf("expected mapped account, got %q", got)
	}
	if got := ResolveSlackEngineer(overrides, "U999"); got != "slack:U999" {
		t.Errorf("expected synthetic id, got %q", got)
	}
	if got := ResolveSlackEngineer(overrides, ""); got != "" {
		t.Errorf("expected empty for empty user, got %q", got)
	}
}

func TestIsThreadParticipation(t *testing.T) {
	cases := []struct {
		name              string
		ts, threadTs      string
		wantParticipation bool
	}{
		{"top-level message", "1700000000.000100", "", false},
		{"thread root", "1700000000.000100", "1700000000.000100", false},
		{"thread reply", "1700000001.000200", "1700000000.000100", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsThreadParticipation(c.ts, c.threadTs); got != c.wantParticipation {
				t.Errorf("IsThreadParticipation(%q,%q) = %v, want %v", c.ts, c.threadTs, got, c.wantParticipation)
			}
		})
	}
}

func TestSlackTsToTime(t *testing.T) {
	// Slack ts format: "<unix seconds>.<microseconds>". Parsing must round-trip.
	got, err := SlackTsToTime("1700000000.000123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Unix() != 1700000000 {
		t.Errorf("expected unix=1700000000, got %d", got.Unix())
	}
}
