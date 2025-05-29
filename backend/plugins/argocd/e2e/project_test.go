/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with

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

	"github.com/apache/incubator-devlake/helpers/e2ehelper"
	"github.com/apache/incubator-devlake/plugins/argocd/impl"
	"github.com/apache/incubator-devlake/plugins/argocd/models"
	"github.com/apache/incubator-devlake/plugins/argocd/tasks"
)

func TestProjectDataFlow(t *testing.T) {
	var plugin impl.ArgoCDPlugin
	dataflowTester := e2ehelper.NewDataFlowTester(t, "argocd", plugin)

	taskData := &tasks.ArgoCDTaskData{
		Options: &tasks.ArgoCDOptions{
			ConnectionId: 1,
		},
	}

	// import raw data table
	dataflowTester.ImportCsvIntoRawTable("./raw_tables/_raw_argocd_api_projects.csv", "_raw_argocd_api_projects")

	// verify extraction
	dataflowTester.FlushTabler(&models.ArgoCDProject{})
	dataflowTester.Subtask(tasks.ExtractProjectsMeta, taskData)
	
	// Check that extraction worked
	var count int64
	dataflowTester.Db.Model(&models.ArgoCDProject{}).Count(&count)
	if count != 1 {
		t.Errorf("Expected 1 project, got %d", count)
	}
}