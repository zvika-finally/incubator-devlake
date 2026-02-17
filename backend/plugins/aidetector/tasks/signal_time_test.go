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

	"github.com/apache/incubator-devlake/core/models/domainlayer/code"
)

func TestResolveSignalDetectedAt(t *testing.T) {
	created := time.Date(2025, 10, 1, 12, 0, 0, 0, time.UTC)
	merged := time.Date(2025, 10, 2, 8, 30, 0, 0, time.UTC)

	t.Run("prefer merged date", func(t *testing.T) {
		pr := &code.PullRequest{
			CreatedDate: created,
			MergedDate:  &merged,
		}
		got := resolveSignalDetectedAt(pr)
		if !got.Equal(merged) {
			t.Fatalf("expected merged date %v, got %v", merged, got)
		}
	})

	t.Run("fallback to created date", func(t *testing.T) {
		pr := &code.PullRequest{
			CreatedDate: created,
		}
		got := resolveSignalDetectedAt(pr)
		if !got.Equal(created) {
			t.Fatalf("expected created date %v, got %v", created, got)
		}
	})

	t.Run("fallback to now when missing pr dates", func(t *testing.T) {
		before := time.Now()
		got := resolveSignalDetectedAt(&code.PullRequest{})
		after := time.Now()
		if got.Before(before) || got.After(after) {
			t.Fatalf("expected now fallback between %v and %v, got %v", before, after, got)
		}
	})

	t.Run("fallback to now when pr is nil", func(t *testing.T) {
		before := time.Now()
		got := resolveSignalDetectedAt(nil)
		after := time.Now()
		if got.Before(before) || got.After(after) {
			t.Fatalf("expected now fallback between %v and %v, got %v", before, after, got)
		}
	})
}
