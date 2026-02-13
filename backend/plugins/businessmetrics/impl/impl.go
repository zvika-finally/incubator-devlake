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
	"github.com/apache/incubator-devlake/plugins/businessmetrics/api"
	"github.com/apache/incubator-devlake/plugins/businessmetrics/models"
	"github.com/apache/incubator-devlake/plugins/businessmetrics/models/migrationscripts"
	"github.com/apache/incubator-devlake/plugins/businessmetrics/tasks"
)

// make sure interface is implemented
var _ interface {
	plugin.PluginMeta
	plugin.PluginInit
	plugin.PluginTask
	plugin.PluginModel
	plugin.PluginMetric
	plugin.PluginMigration
	plugin.PluginApi
	plugin.MetricPluginBlueprintV200
} = (*BusinessMetrics)(nil)

type BusinessMetrics struct{}

func (p BusinessMetrics) Init(basicRes context.BasicRes) errors.Error {
	api.Init(basicRes, p)
	return nil
}

func (p BusinessMetrics) Description() string {
	return "Extract business initiatives from Jira Epics and calculate work alignment"
}

func (p BusinessMetrics) Dashboards() []plugin.GrafanaDashboard {
	return nil
}

func (p BusinessMetrics) SvgIcon() string {
	return `<svg viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
<path fill-rule="evenodd" clip-rule="evenodd" d="M2 2h3v3H2V2zm0 4.5h3v3H2v-3zM2 11h3v3H2v-3zm4.5-9h3v3h-3V2zm0 4.5h3v3h-3v-3zm0 4.5h3v3h-3v-3zM11 2h3v3h-3V2zm0 4.5h3v3h-3v-3zm0 4.5h3v3h-3v-3z" fill="#444444"/>
</svg>`
}

func (p BusinessMetrics) RequiredDataEntities() (data []map[string]interface{}, err errors.Error) {
	return []map[string]interface{}{
		{
			"model": "issues",
			"requiredFields": map[string]string{
				"column":        "type",
				"expectedValue": "Epic",
			},
		},
	}, nil
}

func (p BusinessMetrics) GetTablesInfo() []dal.Tabler {
	return []dal.Tabler{
		&models.BusinessInitiative{},
		&models.WorkAllocation{},
		&models.TeamHealthScore{},
		&models.BusinessMetricsSettings{},
		&models.WorkingAgreement{},
		&models.AgreementViolation{},
		&models.AgreementComplianceSummary{},
	}
}

func (p BusinessMetrics) Name() string {
	return "businessmetrics"
}

func (p BusinessMetrics) IsProjectMetric() bool {
	return true
}

func (p BusinessMetrics) RunAfter() ([]string, errors.Error) {
	return []string{"dora"}, nil // Needs DORA metrics (project_pr_metrics table)
}

func (p BusinessMetrics) Settings() interface{} {
	return nil
}

func (p BusinessMetrics) SubTaskMetas() []plugin.SubTaskMeta {
	return []plugin.SubTaskMeta{
		tasks.ExtractBusinessGoalsMeta,
		tasks.CalculateAlignmentMeta,
		tasks.CalculateHealthScoreMeta,
		tasks.CalculateBusinessValueMeta,
		tasks.CheckAgreementsMeta,
	}
}

func (p BusinessMetrics) PrepareTaskData(taskCtx plugin.TaskContext, options map[string]interface{}) (interface{}, errors.Error) {
	op, err := tasks.DecodeAndValidateTaskOptions(options)
	if err != nil {
		return nil, err
	}

	// Load settings for the project
	settings, settingsErr := api.GetSettingsForProject(op.ProjectName)
	if settingsErr != nil {
		taskCtx.GetLogger().Warn(settingsErr, "Failed to load settings for project %s, using defaults", op.ProjectName)
		settings = models.NewDefaultSettings()
	}

	// Let project settings override hardcoded option defaults when caller didn't provide explicit values.
	if (op.InvestmentLabelPrefix == "" || op.InvestmentLabelPrefix == "investment:") && settings.InvestmentLabelPrefix != "" {
		op.InvestmentLabelPrefix = settings.InvestmentLabelPrefix
	}
	if (op.StageLabelPrefix == "" || op.StageLabelPrefix == "stage:") && settings.StageLabelPrefix != "" {
		op.StageLabelPrefix = settings.StageLabelPrefix
	}
	taskCtx.GetLogger().Info("Loaded business metrics settings for project: %s", op.ProjectName)

	return &tasks.BusinessMetricsTaskData{
		Options:  op,
		Settings: settings,
	}, nil
}

func (p BusinessMetrics) RootPkgPath() string {
	return "github.com/apache/incubator-devlake/plugins/businessmetrics"
}

func (p BusinessMetrics) MigrationScripts() []plugin.MigrationScript {
	return migrationscripts.All()
}

func (p BusinessMetrics) ApiResources() map[string]map[string]plugin.ApiResourceHandler {
	return map[string]map[string]plugin.ApiResourceHandler{
		"settings": {
			"GET": api.ListSettings,
		},
		"settings/:projectName": {
			"GET":    api.GetSettings,
			"PUT":    api.PutSettings,
			"DELETE": api.DeleteSettings,
		},
		"agreements/:projectName": {
			"GET":  api.ListAgreements,
			"POST": api.CreateAgreement,
		},
		"agreements/:projectName/:agreementType": {
			"GET":    api.GetAgreement,
			"PUT":    api.UpdateAgreement,
			"DELETE": api.DeleteAgreement,
		},
		"violations/:projectName": {
			"GET": api.ListViolations,
		},
		"compliance/:projectName": {
			"GET": api.GetComplianceSummary,
		},
	}
}

func (p BusinessMetrics) MakeMetricPluginPipelinePlanV200(projectName string, options json.RawMessage) (coreModels.PipelinePlan, errors.Error) {
	op := &tasks.BusinessMetricsOptions{}
	if options != nil && string(options) != "\"\"" {
		err := json.Unmarshal(options, op)
		if err != nil {
			return nil, errors.Default.WrapRaw(err)
		}
	}

	plan := coreModels.PipelinePlan{
		{
			{
				Plugin: "businessmetrics",
				Options: map[string]interface{}{
					"projectName": projectName,
				},
				Subtasks: []string{
					"extractBusinessGoals",
					"calculateAlignment",
					"calculateHealthScore",
					"calculateBusinessValue",
					"checkAgreements",
				},
			},
		},
	}
	return plan, nil
}
