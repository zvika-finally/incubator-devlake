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
	"github.com/apache/incubator-devlake/plugins/aidetector/api"
	"github.com/apache/incubator-devlake/plugins/aidetector/models"
	"github.com/apache/incubator-devlake/plugins/aidetector/models/migrationscripts"
	"github.com/apache/incubator-devlake/plugins/aidetector/tasks"
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
} = (*AIDetector)(nil)

type AIDetector struct{}

func (p AIDetector) Init(basicRes context.BasicRes) errors.Error {
	api.Init(basicRes, p)
	return nil
}

func (p AIDetector) Description() string {
	return "Detect AI-assisted code by analyzing commit and PR patterns (Jellyfish-style approach)"
}

func (p AIDetector) Dashboards() []plugin.GrafanaDashboard {
	return nil
}

func (p AIDetector) SvgIcon() string {
	return `<svg viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
<path d="M8 1L2 4v4c0 3.5 2.5 6.5 6 8 3.5-1.5 6-4.5 6-8V4L8 1zm0 2l4 2v3c0 2.5-1.8 4.8-4 6-2.2-1.2-4-3.5-4-6V5l4-2z" fill="#444444"/>
<path d="M6 7h4v1H6V7zm0 2h4v1H6V9z" fill="#444444"/>
</svg>`
}

func (p AIDetector) RequiredDataEntities() (data []map[string]interface{}, err errors.Error) {
	return []map[string]interface{}{
		{
			"model": "pull_requests",
		},
		{
			"model": "commits",
		},
	}, nil
}

func (p AIDetector) GetTablesInfo() []dal.Tabler {
	return []dal.Tabler{
		&models.AIUsageSignal{},
		&models.DeveloperBaseline{},
		&models.AIImpactMetric{},
		&models.AIDetectorSettings{},
	}
}

func (p AIDetector) Name() string {
	return "aidetector"
}

func (p AIDetector) IsProjectMetric() bool {
	return true
}

func (p AIDetector) RunAfter() ([]string, errors.Error) {
	return []string{}, nil
}

func (p AIDetector) Settings() interface{} {
	return nil
}

func (p AIDetector) SubTaskMetas() []plugin.SubTaskMeta {
	return []plugin.SubTaskMeta{
		tasks.DetectExplicitSignalsMeta,   // Run first: explicit markers (HIGH confidence)
		tasks.AnalyzeCommitPatternsMeta,   // Then: behavioral patterns
		tasks.AnalyzePRCharacteristicsMeta,
		tasks.ScoreAIConfidenceMeta,       // Combine all scores
		tasks.CalculateAIImpactMeta,       // Finally: calculate productivity impact
	}
}

func (p AIDetector) PrepareTaskData(taskCtx plugin.TaskContext, options map[string]interface{}) (interface{}, errors.Error) {
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
	taskCtx.GetLogger().Info("Loaded AI detector settings for project: %s", op.ProjectName)

	return &tasks.AIDetectorTaskData{
		Options:  op,
		Settings: settings,
	}, nil
}

func (p AIDetector) RootPkgPath() string {
	return "github.com/apache/incubator-devlake/plugins/aidetector"
}

func (p AIDetector) MigrationScripts() []plugin.MigrationScript {
	return migrationscripts.All()
}

func (p AIDetector) ApiResources() map[string]map[string]plugin.ApiResourceHandler {
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

func (p AIDetector) MakeMetricPluginPipelinePlanV200(projectName string, options json.RawMessage) (coreModels.PipelinePlan, errors.Error) {
	op := &tasks.AIDetectorOptions{}
	if options != nil && string(options) != "\"\"" {
		err := json.Unmarshal(options, op)
		if err != nil {
			return nil, errors.Default.WrapRaw(err)
		}
	}

	plan := coreModels.PipelinePlan{
		{
			{
				Plugin: "aidetector",
				Options: map[string]interface{}{
					"projectName": projectName,
				},
				Subtasks: []string{
					"detectExplicitSignals",
					"analyzeCommitPatterns",
					"analyzePRCharacteristics",
					"scoreAIConfidence",
					"calculateAIImpact",
				},
			},
		},
	}
	return plan, nil
}
