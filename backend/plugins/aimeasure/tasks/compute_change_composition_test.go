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

	"github.com/apache/incubator-devlake/plugins/aimeasure/models"
)

func TestBatchBucket(t *testing.T) {
	cases := []struct {
		loc  int
		want models.BatchBucket
	}{
		{0, models.BucketXS},
		{49, models.BucketXS},
		{50, models.BucketS},
		{199, models.BucketS},
		{200, models.BucketM},
		{499, models.BucketM},
		{500, models.BucketL},
		{999, models.BucketL},
		{1000, models.BucketXL},
		{50_000, models.BucketXL},
	}
	for _, c := range cases {
		if got := BucketFor(c.loc); got != c.want {
			t.Errorf("BucketFor(%d) = %s, want %s", c.loc, got, c.want)
		}
	}
}

func TestRefactorRatio(t *testing.T) {
	cases := []struct {
		name       string
		additive   int
		refactor   int
		want       float64
		wantApprox bool
	}{
		{"all additive", 100, 0, 0.0, false},
		{"all refactor", 0, 100, 1.0, false},
		{"half and half", 50, 50, 0.5, false},
		{"empty PR (avoid div by 0)", 0, 0, 0.0, false},
		{"33% refactor", 200, 100, 0.3333, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ComputeRefactorRatio(c.additive, c.refactor)
			if c.wantApprox {
				if got < c.want-0.001 || got > c.want+0.001 {
					t.Errorf("got %v, want approx %v", got, c.want)
				}
			} else if got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}
