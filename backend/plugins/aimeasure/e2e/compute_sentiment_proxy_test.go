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
	"github.com/apache/incubator-devlake/helpers/e2ehelper"
	"github.com/apache/incubator-devlake/plugins/aimeasure/impl"
	"github.com/apache/incubator-devlake/plugins/aimeasure/models"
	"github.com/apache/incubator-devlake/plugins/aimeasure/tasks"
)

func TestComputeSentimentProxyDataFlow(t *testing.T) {
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

	// Seed the two upstream Phase B tables directly so this test stays
	// independent of TestComputeVerificationEffortDataFlow / TestComputeSlackSignalsDataFlow.
	dataflowTester.ImportCsvIntoTabler("./fixtures/expected_engineer_verification_effort_for_sentiment.csv", &models.EngineerVerificationEffort{})
	dataflowTester.ImportCsvIntoTabler("./fixtures/expected_engineer_slack_signals_for_sentiment.csv", &models.EngineerSlackSignals{})

	dataflowTester.FlushTabler(&models.EngineerDxiProxy{})
	dataflowTester.Subtask(tasks.ComputeSentimentProxyMeta, taskData)
	dataflowTester.VerifyTableWithOptions(&models.EngineerDxiProxy{}, e2ehelper.TableOptions{
		CSVRelPath:   "./fixtures/expected_engineer_dxi_proxy.csv",
		IgnoreTypes:  []interface{}{common.NoPKModel{}},
		IgnoreFields: []string{"computed_at", "last_survey_date", "last_survey_dxi"},
		// NumericEpsilon on sentiment_score at 0.01 (column is decimal(5,2)).
		NumericEpsilon: map[string]float64{
			"sentiment_score": 0.01,
		},
	})
}
