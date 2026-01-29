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
	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/claudecode/api"
	"github.com/apache/incubator-devlake/plugins/claudecode/models"
	"github.com/apache/incubator-devlake/plugins/claudecode/models/migrationscripts"
	"github.com/apache/incubator-devlake/plugins/claudecode/tasks"
)

// make sure interface is implemented
var _ interface {
	plugin.PluginMeta
	plugin.PluginInit
	plugin.PluginTask
	plugin.PluginModel
	plugin.PluginMigration
	plugin.PluginApi
} = (*ClaudeCode)(nil)

type ClaudeCode struct{}

func (p ClaudeCode) Init(basicRes context.BasicRes) errors.Error {
	api.Init(basicRes)
	return nil
}

func (p ClaudeCode) Description() string {
	return "Collect AI coding assistant metrics from Claude Code Team Admin API"
}

func (p ClaudeCode) Dashboards() []plugin.GrafanaDashboard {
	return nil
}

func (p ClaudeCode) SvgIcon() string {
	return `<svg viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
<path d="M8 1C4.1 1 1 4.1 1 8s3.1 7 7 7 7-3.1 7-7-3.1-7-7-7zm0 12.5c-3 0-5.5-2.5-5.5-5.5S5 2.5 8 2.5s5.5 2.5 5.5 5.5-2.5 5.5-5.5 5.5z" fill="#D97706"/>
<path d="M6 6.5h4M6 8h4M6 9.5h2.5" stroke="#D97706" stroke-width="1"/>
</svg>`
}

func (p ClaudeCode) GetTablesInfo() []dal.Tabler {
	return []dal.Tabler{
		&models.ClaudeCodeConnection{},
		&models.ClaudeCodeUsageMetric{},
		&models.ClaudeCodeUserMetric{},
	}
}

func (p ClaudeCode) Name() string {
	return "claudecode"
}

func (p ClaudeCode) Connection() dal.Tabler {
	return &models.ClaudeCodeConnection{}
}

func (p ClaudeCode) SubTaskMetas() []plugin.SubTaskMeta {
	return []plugin.SubTaskMeta{
		tasks.CollectMetricsMeta,
	}
}

func (p ClaudeCode) PrepareTaskData(taskCtx plugin.TaskContext, options map[string]interface{}) (interface{}, errors.Error) {
	op, err := tasks.DecodeAndValidateTaskOptions(options)
	if err != nil {
		return nil, err
	}

	connection, connErr := api.GetConnectionForTask(op.ConnectionId)
	if connErr != nil {
		return nil, errors.Default.Wrap(connErr, "failed to load connection")
	}

	return &tasks.ClaudeCodeTaskData{
		Options:    op,
		Connection: connection,
	}, nil
}

func (p ClaudeCode) RootPkgPath() string {
	return "github.com/apache/incubator-devlake/plugins/claudecode"
}

func (p ClaudeCode) MigrationScripts() []plugin.MigrationScript {
	return migrationscripts.All()
}

func (p ClaudeCode) ApiResources() map[string]map[string]plugin.ApiResourceHandler {
	return map[string]map[string]plugin.ApiResourceHandler{
		"connections": {
			"GET":  api.GetConnections,
			"POST": api.PostConnections,
		},
		"connections/:connectionId": {
			"GET":    api.GetConnection,
			"PATCH":  api.PatchConnection,
			"DELETE": api.DeleteConnection,
		},
	}
}
