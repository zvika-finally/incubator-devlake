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
	"github.com/apache/incubator-devlake/plugins/findevops/api"
	"github.com/apache/incubator-devlake/plugins/findevops/models"
	"github.com/apache/incubator-devlake/plugins/findevops/models/migrationscripts"
	"github.com/apache/incubator-devlake/plugins/findevops/tasks"
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
} = (*FinDevOps)(nil)

type FinDevOps struct{}

func (p FinDevOps) Init(basicRes context.BasicRes) errors.Error {
	api.Init(basicRes, p)
	return nil
}

func (p FinDevOps) Description() string {
	return "Calculate development costs and categorize for US GAAP ASC 350-40 capitalization compliance"
}

func (p FinDevOps) Dashboards() []plugin.GrafanaDashboard {
	return nil
}

func (p FinDevOps) SvgIcon() string {
	return `<svg viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
<path d="M8 1C4.13 1 1 4.13 1 8s3.13 7 7 7 7-3.13 7-7-3.13-7-7-7zm.5 10.5h-1v-1h1v1zm0-2h-1v-4h1v4z" fill="#444444"/>
<text x="5.5" y="11" font-size="6" fill="#444444">$</text>
</svg>`
}

func (p FinDevOps) RequiredDataEntities() (data []map[string]interface{}, err errors.Error) {
	return []map[string]interface{}{
		{
			"model": "issues",
		},
	}, nil
}

func (p FinDevOps) GetTablesInfo() []dal.Tabler {
	return []dal.Tabler{
		&models.CostAllocation{},
		&models.MonthlyCostSummary{},
		&models.DeveloperHourlyRate{},
		&models.DeploymentCost{},
		&models.FinDevOpsSettings{},
	}
}

func (p FinDevOps) Name() string {
	return "findevops"
}

func (p FinDevOps) IsProjectMetric() bool {
	return true
}

func (p FinDevOps) RunAfter() ([]string, errors.Error) {
	return []string{"businessmetrics"}, nil // Needs business initiatives first
}

func (p FinDevOps) Settings() interface{} {
	return nil
}

func (p FinDevOps) SubTaskMetas() []plugin.SubTaskMeta {
	return []plugin.SubTaskMeta{
		tasks.CalculateCostsMeta,
		tasks.CategorizeCapitalizationMeta,
		tasks.CalculateDeploymentCostsMeta,
	}
}

func (p FinDevOps) PrepareTaskData(taskCtx plugin.TaskContext, options map[string]interface{}) (interface{}, errors.Error) {
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
	taskCtx.GetLogger().Info("Loaded FinDevOps settings for project: %s", op.ProjectName)

	return &tasks.FinDevOpsTaskData{
		Options:  op,
		Settings: settings,
	}, nil
}

func (p FinDevOps) RootPkgPath() string {
	return "github.com/apache/incubator-devlake/plugins/findevops"
}

func (p FinDevOps) MigrationScripts() []plugin.MigrationScript {
	return migrationscripts.All()
}

func (p FinDevOps) ApiResources() map[string]map[string]plugin.ApiResourceHandler {
	return map[string]map[string]plugin.ApiResourceHandler{
		"settings": {
			"GET": api.ListSettings,
		},
		"settings/:projectName": {
			"GET":    api.GetSettings,
			"PUT":    api.PutSettings,
			"DELETE": api.DeleteSettings,
		},
	}
}

func (p FinDevOps) MakeMetricPluginPipelinePlanV200(projectName string, options json.RawMessage) (coreModels.PipelinePlan, errors.Error) {
	op := &tasks.FinDevOpsOptions{}
	if options != nil && string(options) != "\"\"" {
		err := json.Unmarshal(options, op)
		if err != nil {
			return nil, errors.Default.WrapRaw(err)
		}
	}

	plan := coreModels.PipelinePlan{
		{
			{
				Plugin: "findevops",
				Options: map[string]interface{}{
					"projectName": projectName,
				},
				Subtasks: []string{
					"calculateCosts",
					"categorizeCapitalization",
					"calculateDeploymentCosts",
				},
			},
		},
	}
	return plan, nil
}
