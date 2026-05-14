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

package impl

import (
	"encoding/json"

	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	coreModels "github.com/apache/incubator-devlake/core/models"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/aimeasure/models"
	"github.com/apache/incubator-devlake/plugins/aimeasure/models/migrationscripts"
	"github.com/apache/incubator-devlake/plugins/aimeasure/tasks"
)

// make sure interfaces are implemented
var _ interface {
	plugin.PluginMeta
	plugin.PluginInit
	plugin.PluginTask
	plugin.PluginModel
	plugin.PluginMetric
	plugin.PluginMigration
	plugin.MetricPluginBlueprintV200
} = (*AIMeasure)(nil)

type AIMeasure struct{}

func (p AIMeasure) Init(basicRes context.BasicRes) errors.Error {
	return nil
}

func (p AIMeasure) Description() string {
	return "Analytics layer: classifies PRs by AI cohort and computes quality/verification/cost signals from upstream plugin data"
}

func (p AIMeasure) Name() string {
	return "aimeasure"
}

func (p AIMeasure) RootPkgPath() string {
	return "github.com/apache/incubator-devlake/plugins/aimeasure"
}

func (p AIMeasure) MigrationScripts() []plugin.MigrationScript {
	return migrationscripts.All()
}

func (p AIMeasure) GetTablesInfo() []dal.Tabler {
	return []dal.Tabler{
		&models.PRAICohort{},
		&models.PRDefectSignals{},
		&models.PRChangeComposition{},
		&models.AccountOverride{},
		&models.EngineerRole{},
		// Phase B
		&models.EngineerVerificationEffort{},
		&models.EngineerSlackSignals{},
		&models.EngineerDxiProxy{},
		&models.SlackChannelCategory{},
	}
}

func (p AIMeasure) Dashboards() []plugin.GrafanaDashboard {
	return nil
}

func (p AIMeasure) SvgIcon() string {
	return `<svg viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
<path d="M8 1L2 4v4c0 3.5 2.5 6.5 6 8 3.5-1.5 6-4.5 6-8V4L8 1z" fill="#444444"/>
<path d="M5 7l2 2 4-4" stroke="#FFFFFF" stroke-width="1.5" fill="none" stroke-linecap="round" stroke-linejoin="round"/>
</svg>`
}

func (p AIMeasure) RequiredDataEntities() (data []map[string]interface{}, err errors.Error) {
	return []map[string]interface{}{
		{"model": "pull_requests"},
		{"model": "commits"},
		{"model": "commit_files"},
		{"model": "pull_request_commits"},
		{"model": "pull_request_comments"},
		{"model": "pull_request_reviewers"},
	}, nil
}

func (p AIMeasure) IsProjectMetric() bool {
	return true
}

func (p AIMeasure) RunAfter() ([]string, errors.Error) {
	return []string{"aidetector"}, nil // reads ai_usage_signals produced by aidetector
}

func (p AIMeasure) Settings() interface{} {
	return nil
}

func (p AIMeasure) SubTaskMetas() []plugin.SubTaskMeta {
	return []plugin.SubTaskMeta{
		// Phase A
		tasks.ClassifyPRCohortMeta,         // run first — produces pr_ai_cohort
		tasks.ComputeChangeCompositionMeta, // independent — produces pr_change_composition
		tasks.ComputeQualityCohortMeta,     // independent — produces pr_defect_signals
		// Phase B — depend on Phase A's outputs being present
		tasks.ComputeVerificationEffortMeta, // reads pr_ai_cohort, writes engineer_verification_effort
		tasks.ComputeSlackSignalsMeta,       // reads slack tool tables, writes engineer_slack_signals
		tasks.ComputeSentimentProxyMeta,     // reads both above, writes engineer_dxi_proxy (run last)
	}
}

func (p AIMeasure) PrepareTaskData(taskCtx plugin.TaskContext, options map[string]interface{}) (interface{}, errors.Error) {
	opts, err := tasks.DecodeAndValidateTaskOptions(options)
	if err != nil {
		return nil, err
	}
	return &tasks.AIMeasureTaskData{Options: opts}, nil
}

func (p AIMeasure) MakeMetricPluginPipelinePlanV200(projectName string, options json.RawMessage) (coreModels.PipelinePlan, errors.Error) {
	op := &tasks.AIMeasureOptions{}
	if options != nil && string(options) != "\"\"" {
		err := json.Unmarshal(options, op)
		if err != nil {
			return nil, errors.Default.WrapRaw(err)
		}
	}
	op.ProjectName = projectName

	plan := coreModels.PipelinePlan{
		{
			{
				Plugin: "aimeasure",
				Options: map[string]interface{}{
					"projectName":         projectName,
					"highCohortThreshold": op.HighCohortThreshold,
					"lowCohortThreshold":  op.LowCohortThreshold,
					"defectWindowDays":    op.DefectWindowDays,
				},
				Subtasks: []string{
					"classifyPRCohort",
					"computeChangeComposition",
					"computeQualityCohort",
					"computeVerificationEffort",
					"computeSlackSignals",
					"computeSentimentProxy",
				},
			},
		},
	}
	return plan, nil
}
