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
	"github.com/apache/incubator-devlake/core/models/domainlayer/crossdomain"
	"github.com/apache/incubator-devlake/core/models/domainlayer/devops"
	"github.com/apache/incubator-devlake/helpers/e2ehelper"
	"github.com/apache/incubator-devlake/plugins/aidetector/impl"
	"github.com/apache/incubator-devlake/plugins/aidetector/models"
	"github.com/apache/incubator-devlake/plugins/aidetector/tasks"
)

func TestAnalyzeCodeChurnDataFlow(t *testing.T) {
	var plugin impl.AIDetector
	dataflowTester := e2ehelper.NewDataFlowTester(t, "aidetector", plugin)

	taskData := &tasks.AIDetectorTaskData{
		Options: &tasks.AIDetectorOptions{
			ProjectName:         "project1",
			ConfidenceThreshold: 65,
		},
	}

	// Import input data - following DORA pattern
	dataflowTester.ImportCsvIntoTabler("./code_churn/project_mapping.csv", &crossdomain.ProjectMapping{})
	dataflowTester.ImportCsvIntoTabler("./code_churn/cicd_scopes.csv", &devops.CicdScope{})
	dataflowTester.ImportCsvIntoTabler("./code_churn/repos.csv", &code.Repo{})
	dataflowTester.ImportCsvIntoTabler("./code_churn/pull_requests.csv", &code.PullRequest{})
	dataflowTester.ImportCsvIntoTabler("./code_churn/ai_usage_signals.csv", &models.AIUsageSignal{})
	dataflowTester.ImportCsvIntoTabler("./code_churn/pull_request_commits.csv", &code.PullRequestCommit{})
	dataflowTester.ImportCsvIntoTabler("./code_churn/commit_files.csv", &code.CommitFile{})
	dataflowTester.ImportCsvIntoTabler("./code_churn/commits.csv", &code.Commit{})

	// Flush output tables
	dataflowTester.FlushTabler(&models.AIChurnMetric{})
	dataflowTester.FlushTabler(&models.ProjectChurnSummary{})

	// Run the subtask
	dataflowTester.Subtask(tasks.AnalyzeCodeChurnMeta, taskData)

	// Verify output
	dataflowTester.VerifyTableWithOptions(&models.AIChurnMetric{}, e2ehelper.TableOptions{
		CSVRelPath:  "./code_churn/ai_churn_metrics.csv",
		IgnoreTypes: []interface{}{common.NoPKModel{}},
		IgnoreFields: []string{
			"merged_at",
			"file_paths",
			"calculated_at",
		},
		NumericEpsilon: map[string]float64{
			"churn_ratio7_days":  0.0001,
			"churn_ratio30_days": 0.0001,
		},
	})
}
