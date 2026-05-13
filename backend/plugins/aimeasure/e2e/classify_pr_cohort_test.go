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

package e2e

import (
	"testing"

	"github.com/apache/incubator-devlake/core/models/common"
	"github.com/apache/incubator-devlake/core/models/domainlayer/code"
	"github.com/apache/incubator-devlake/helpers/e2ehelper"
	aidetectorModels "github.com/apache/incubator-devlake/plugins/aidetector/models"
	"github.com/apache/incubator-devlake/plugins/aimeasure/impl"
	"github.com/apache/incubator-devlake/plugins/aimeasure/models"
	"github.com/apache/incubator-devlake/plugins/aimeasure/tasks"
)

func TestClassifyPRCohortDataFlow(t *testing.T) {
	var plugin impl.AIMeasure
	dataflowTester := e2ehelper.NewDataFlowTester(t, "aimeasure", plugin)

	taskData := &tasks.AIMeasureTaskData{
		Options: &tasks.AIMeasureOptions{
			ProjectName:         "demo",
			HighCohortThreshold: 65,
			LowCohortThreshold:  30,
			DefectWindowDays:    14,
		},
	}

	dataflowTester.ImportCsvIntoTabler("./fixtures/pull_requests.csv", &code.PullRequest{})
	dataflowTester.ImportCsvIntoTabler("./fixtures/commits.csv", &code.Commit{})
	dataflowTester.ImportCsvIntoTabler("./fixtures/pull_request_commits.csv", &code.PullRequestCommit{})
	dataflowTester.ImportCsvIntoTabler("./fixtures/ai_usage_signals.csv", &aidetectorModels.AIUsageSignal{})

	dataflowTester.FlushTabler(&models.PRAICohort{})
	dataflowTester.Subtask(tasks.ClassifyPRCohortMeta, taskData)
	dataflowTester.VerifyTableWithOptions(&models.PRAICohort{}, e2ehelper.TableOptions{
		CSVRelPath:   "./fixtures/expected_pr_ai_cohort.csv",
		IgnoreTypes:  []interface{}{common.NoPKModel{}},
		IgnoreFields: []string{"classified_at"},
	})
}
