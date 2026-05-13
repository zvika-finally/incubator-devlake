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

	"github.com/apache/incubator-devlake/core/models/domainlayer/devops"
	"github.com/apache/incubator-devlake/helpers/e2ehelper"
	"github.com/apache/incubator-devlake/plugins/github/impl"
	"github.com/apache/incubator-devlake/plugins/github/models"
	"github.com/apache/incubator-devlake/plugins/github/tasks"
	githubGraphQLTasks "github.com/apache/incubator-devlake/plugins/github_graphql/tasks"
)

// TestGithubDeploymentConvertorDedup verifies that the converter collapses multiple
// _tool_github_deployments rows that share the same (commit_oid, environment, ref_name)
// into a single cicd_deployment_commits row, keeping the row with the latest updated_date.
//
// This guards against the row inflation we hit in production where GitHub's GraphQL
// Deployments API emits one Deployment entity per deploy job (3 INACTIVE + 1 ACTIVE
// per workflow), which previously produced 4x rows per real deployment.
func TestGithubDeploymentConvertorDedup(t *testing.T) {
	var github impl.Github
	dataflowTester := e2ehelper.NewDataFlowTester(t, "github", github)
	taskData := &tasks.GithubTaskData{
		Options: &tasks.GithubOptions{
			ConnectionId: 1,
			Name:         "example/repo",
			GithubId:     99999999,
		},
	}

	// Seed _tool_github_deployments directly with 4 duplicate rows + 1 unique row.
	// Bypassing Extract here keeps this test focused on the convertor's dedup behavior.
	dataflowTester.ImportCsvIntoTabler("./raw_tables/_tool_github_deployments_dedup.csv", &models.GithubDeployment{})

	dataflowTester.FlushTabler(&devops.CicdDeploymentCommit{})
	dataflowTester.FlushTabler(&devops.CICDDeployment{})
	dataflowTester.Subtask(githubGraphQLTasks.ConvertDeploymentsMeta, taskData)

	dataflowTester.VerifyTable(&devops.CicdDeploymentCommit{},
		"./snapshot_tables/cicd_deployment_commits_dedup.csv",
		[]string{
			"cicd_scope_id",
			"cicd_deployment_id",
			"name",
			"result",
			"status",
			"original_status",
			"environment",
			"original_environment",
			"created_date",
			"queued_date",
			"started_date",
			"finished_date",
			"commit_sha",
			"commit_msg",
			"ref_name",
			"repo_id",
			"repo_url",
			"prev_success_deployment_commit_id",
			"display_title",
			"url",
			"duration_sec",
		},
	)
}
