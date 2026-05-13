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

func TestIsHotfixTitle(t *testing.T) {
	cases := []struct {
		title string
		want  bool
	}{
		{"feat: add widget", false},
		{"hotfix: prod down", true},
		{"HOTFIX(api): null deref", true},
		{"Urgent: revert deploy", true},
		{"emergency-rollback", true},
		{"fixup! 1a2b3c4 hotfix part 2", true},
		{"chore: clean up", false},
	}
	for _, c := range cases {
		if got := IsHotfixTitle(c.title); got != c.want {
			t.Errorf("IsHotfixTitle(%q) = %v, want %v", c.title, got, c.want)
		}
	}
}

func TestFileOverlapRatio(t *testing.T) {
	cases := []struct {
		name string
		a, b []string
		want float64
	}{
		{"identical", []string{"x.go", "y.go"}, []string{"x.go", "y.go"}, 1.0},
		{"disjoint", []string{"a"}, []string{"b"}, 0.0},
		{"half overlap", []string{"x.go", "y.go"}, []string{"x.go"}, 0.5},
		{"empty a", nil, []string{"x.go"}, 0.0},
		{"empty b", []string{"x.go"}, nil, 0.0},
		{"both empty", nil, nil, 0.0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := FileOverlapRatio(c.a, c.b)
			if got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}
