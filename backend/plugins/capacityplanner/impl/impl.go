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
	"github.com/apache/incubator-devlake/plugins/capacityplanner/api"
	"github.com/apache/incubator-devlake/plugins/capacityplanner/models"
	"github.com/apache/incubator-devlake/plugins/capacityplanner/models/migrationscripts"
	"github.com/apache/incubator-devlake/plugins/capacityplanner/tasks"
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
} = (*CapacityPlanner)(nil)

type CapacityPlanner struct{}

func (p CapacityPlanner) Init(basicRes context.BasicRes) errors.Error {
	api.Init(basicRes, p)
	return nil
}

func (p CapacityPlanner) Description() string {
	return "Calculate team velocity (Scrum) or throughput (Kanban) and forecast initiative completion dates"
}

func (p CapacityPlanner) Dashboards() []plugin.GrafanaDashboard {
	return nil
}

func (p CapacityPlanner) SvgIcon() string {
	return `<svg viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
<path d="M14 2H2v12h12V2zM3 3h10v2H3V3zm0 3h4v7H3V6zm5 0h5v3H8V6zm0 4h5v3H8v-3z" fill="#444444"/>
</svg>`
}

func (p CapacityPlanner) RequiredDataEntities() (data []map[string]interface{}, err errors.Error) {
	return []map[string]interface{}{
		{
			"model": "issues", // Required for both Kanban and Scrum
		},
		{
			"model": "sprints", // Optional - only needed for Scrum velocity metrics
		},
	}, nil
}

func (p CapacityPlanner) GetTablesInfo() []dal.Tabler {
	return []dal.Tabler{
		&models.TeamVelocity{},
		&models.InitiativeForecast{},
		&models.CapacityScenario{},
		&models.MonteCarloForecast{},
		&models.CapacityModel{},
		&models.InvestmentROI{},
		&models.CapacityPlannerSettings{},
		&models.IssueFlowMetric{},
		&models.ProjectFlowSummary{},
	}
}

func (p CapacityPlanner) Name() string {
	return "capacityplanner"
}

func (p CapacityPlanner) IsProjectMetric() bool {
	return true
}

func (p CapacityPlanner) RunAfter() ([]string, errors.Error) {
	return []string{}, nil // No dependencies - reads from domain tables (issues, sprints, boards, pull_requests)
}

func (p CapacityPlanner) Settings() interface{} {
	return nil
}

func (p CapacityPlanner) SubTaskMetas() []plugin.SubTaskMeta {
	return []plugin.SubTaskMeta{
		tasks.CalculateVelocityMeta,        // Sprint-based velocity (Scrum)
		tasks.CalculateThroughputMeta,      // Time-based throughput (Kanban)
		tasks.ForecastCompletionKanbanMeta, // Kanban: issue-count forecasting
		tasks.MonteCarloForecastKanbanMeta, // Kanban: Monte Carlo with throughput
		tasks.ForecastCompletionMeta,       // Scrum: story-point forecasting
		tasks.MonteCarloForecastMeta,       // Scrum: Monte Carlo with velocity
		tasks.BrooksLawModelMeta,
		tasks.CalculateROIMeta,
		tasks.CalculateFlowEfficiencyMeta,
	}
}

func (p CapacityPlanner) PrepareTaskData(taskCtx plugin.TaskContext, options map[string]interface{}) (interface{}, errors.Error) {
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
	taskCtx.GetLogger().Info("Loaded capacity planner settings for project: %s", op.ProjectName)

	return &tasks.CapacityPlannerTaskData{
		Options:  op,
		Settings: settings,
	}, nil
}

func (p CapacityPlanner) RootPkgPath() string {
	return "github.com/apache/incubator-devlake/plugins/capacityplanner"
}

func (p CapacityPlanner) MigrationScripts() []plugin.MigrationScript {
	return migrationscripts.All()
}

func (p CapacityPlanner) ApiResources() map[string]map[string]plugin.ApiResourceHandler {
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

func (p CapacityPlanner) MakeMetricPluginPipelinePlanV200(projectName string, options json.RawMessage) (coreModels.PipelinePlan, errors.Error) {
	op := &tasks.CapacityPlannerOptions{}
	if options != nil && string(options) != "\"\"" {
		err := json.Unmarshal(options, op)
		if err != nil {
			return nil, errors.Default.WrapRaw(err)
		}
	}

	plan := coreModels.PipelinePlan{
		{
			{
				Plugin: "capacityplanner",
				Options: map[string]interface{}{
					"projectName": projectName,
				},
				Subtasks: []string{
					"calculateThroughput",      // Kanban: throughput metrics
					"forecastCompletionKanban", // Kanban: issue-based forecasting
					"monteCarloForecastKanban", // Kanban: probabilistic forecasts
					"calculateVelocity",        // Scrum: sprint-based (optional)
					"forecastCompletion",       // Scrum: story-point forecasting
					"monteCarloForecast",       // Scrum: Monte Carlo with velocity
					"brooksLawModel",
					"calculateROI",
					"calculateFlowEfficiency",
				},
			},
		},
	}
	return plan, nil
}
