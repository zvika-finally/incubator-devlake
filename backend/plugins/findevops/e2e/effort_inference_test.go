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

	"github.com/apache/incubator-devlake/core/models/domainlayer/code"
	"github.com/apache/incubator-devlake/core/models/domainlayer/crossdomain"
	"github.com/apache/incubator-devlake/core/models/domainlayer/ticket"
	"github.com/apache/incubator-devlake/helpers/e2ehelper"
	"github.com/apache/incubator-devlake/plugins/findevops/impl"
	"github.com/apache/incubator-devlake/plugins/findevops/models"
	"github.com/apache/incubator-devlake/plugins/findevops/tasks"
)

// TestEffortInferenceDataFlow tests the complete effort inference pipeline
// This test is currently a placeholder - CSV test data files need to be created
// in the ./effort_inference/ directory to fully enable this test.
func TestEffortInferenceDataFlow(t *testing.T) {
	t.Skip("Skipping: requires CSV test data in ./effort_inference/ directory")

	var plugin impl.FinDevOps
	dataflowTester := e2ehelper.NewDataFlowTester(t, "findevops", plugin)

	// Import test data
	dataflowTester.ImportCsvIntoTabler("./effort_inference/issues.csv", &ticket.Issue{})
	dataflowTester.ImportCsvIntoTabler("./effort_inference/commits.csv", &code.Commit{})
	dataflowTester.ImportCsvIntoTabler("./effort_inference/pull_requests.csv", &code.PullRequest{})
	dataflowTester.ImportCsvIntoTabler("./effort_inference/issue_commits.csv", &crossdomain.IssueCommit{})
	dataflowTester.ImportCsvIntoTabler("./effort_inference/pull_request_issues.csv", &crossdomain.PullRequestIssue{})
	dataflowTester.ImportCsvIntoTabler("./effort_inference/board_issues.csv", &ticket.BoardIssue{})
	dataflowTester.ImportCsvIntoTabler("./effort_inference/project_mapping.csv", &crossdomain.ProjectMapping{})

	taskData := &tasks.FinDevOpsTaskData{
		Options: &tasks.FinDevOpsOptions{
			ProjectName:       "test-project",
			DefaultHourlyRate: 87.0,
		},
		Settings: models.NewDefaultSettings(),
	}

	// Run subtasks in order
	dataflowTester.Subtask(tasks.CollectDeveloperActivityMeta, taskData)
	dataflowTester.Subtask(tasks.InferGitEffortMeta, taskData)
	dataflowTester.Subtask(tasks.CalculateCostsMeta, taskData)

	// Verify FTE records were created
	dataflowTester.VerifyTableWithOptions(
		&models.DeveloperMonthlyFte{},
		e2ehelper.TableOptions{
			CSVRelPath: "./effort_inference/developer_monthly_fte.csv",
		},
	)

	// Verify cost allocations have effort source tracking
	dataflowTester.VerifyTableWithOptions(
		&models.CostAllocation{},
		e2ehelper.TableOptions{
			CSVRelPath: "./effort_inference/cost_allocations.csv",
		},
	)
}
