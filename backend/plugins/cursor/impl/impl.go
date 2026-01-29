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
	"github.com/apache/incubator-devlake/plugins/cursor/api"
	"github.com/apache/incubator-devlake/plugins/cursor/models"
	"github.com/apache/incubator-devlake/plugins/cursor/models/migrationscripts"
	"github.com/apache/incubator-devlake/plugins/cursor/tasks"
)

// make sure interface is implemented
var _ interface {
	plugin.PluginMeta
	plugin.PluginInit
	plugin.PluginTask
	plugin.PluginModel
	plugin.PluginMigration
	plugin.PluginApi
} = (*Cursor)(nil)

type Cursor struct{}

func (p Cursor) Init(basicRes context.BasicRes) errors.Error {
	api.Init(basicRes)
	return nil
}

func (p Cursor) Description() string {
	return "Collect AI coding assistant metrics from Cursor Business Analytics API"
}

func (p Cursor) Dashboards() []plugin.GrafanaDashboard {
	return nil
}

func (p Cursor) SvgIcon() string {
	return `<svg viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
<path d="M8 1C4.1 1 1 4.1 1 8s3.1 7 7 7 7-3.1 7-7-3.1-7-7-7zm0 12.5c-3 0-5.5-2.5-5.5-5.5S5 2.5 8 2.5s5.5 2.5 5.5 5.5-2.5 5.5-5.5 5.5z" fill="#444"/>
<path d="M10.5 5.5l-5 5m0-5l5 5" stroke="#444" stroke-width="1.5"/>
</svg>`
}

func (p Cursor) GetTablesInfo() []dal.Tabler {
	return []dal.Tabler{
		&models.CursorConnection{},
		&models.CursorUsageMetric{},
		&models.CursorUserMetric{},
	}
}

func (p Cursor) Name() string {
	return "cursor"
}

func (p Cursor) Connection() dal.Tabler {
	return &models.CursorConnection{}
}

func (p Cursor) SubTaskMetas() []plugin.SubTaskMeta {
	return []plugin.SubTaskMeta{
		tasks.CollectMetricsMeta,
	}
}

func (p Cursor) PrepareTaskData(taskCtx plugin.TaskContext, options map[string]interface{}) (interface{}, errors.Error) {
	op, err := tasks.DecodeAndValidateTaskOptions(options)
	if err != nil {
		return nil, err
	}

	connection, connErr := api.GetConnectionForTask(op.ConnectionId)
	if connErr != nil {
		return nil, errors.Default.Wrap(connErr, "failed to load connection")
	}

	return &tasks.CursorTaskData{
		Options:    op,
		Connection: connection,
	}, nil
}

func (p Cursor) RootPkgPath() string {
	return "github.com/apache/incubator-devlake/plugins/cursor"
}

func (p Cursor) MigrationScripts() []plugin.MigrationScript {
	return migrationscripts.All()
}

func (p Cursor) ApiResources() map[string]map[string]plugin.ApiResourceHandler {
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
