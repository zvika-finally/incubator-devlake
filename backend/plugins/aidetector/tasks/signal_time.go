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

	"github.com/apache/incubator-devlake/core/models/domainlayer/code"
)

// resolveSignalDetectedAt aligns signal timestamp to PR event time so dashboard trends are stable across reruns.
func resolveSignalDetectedAt(pr *code.PullRequest) time.Time {
	if pr != nil {
		if pr.MergedDate != nil && !pr.MergedDate.IsZero() {
			return *pr.MergedDate
		}
		if !pr.CreatedDate.IsZero() {
			return pr.CreatedDate
		}
	}
	return time.Now()
}
