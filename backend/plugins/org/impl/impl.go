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
	coreModels "github.com/apache/incubator-devlake/core/models"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/org/api"
	"github.com/apache/incubator-devlake/plugins/org/tasks"
)

var _ interface {
	plugin.PluginMeta
	plugin.PluginInit
	plugin.PluginTask
	plugin.PluginModel
	plugin.ProjectMapper
} = (*Org)(nil)

type Org struct {
	handlers *api.Handlers
}

func (p *Org) Init(basicRes context.BasicRes) errors.Error {
	p.handlers = api.NewHandlers(basicRes)
	return nil
}

func (p Org) GetTablesInfo() []dal.Tabler {
	return []dal.Tabler{}
}

func (p Org) Description() string {
	return "collect data related to team and organization"
}

func (p Org) Name() string {
	return "org"
}

func (p Org) SubTaskMetas() []plugin.SubTaskMeta {
	return []plugin.SubTaskMeta{
		tasks.ConnectUserAccountsExactMeta,
		tasks.SetProjectMappingMeta,
		tasks.SleepMeta,
	}
}

func (p Org) MapProject(projectName string, scopes []plugin.Scope) (coreModels.PipelinePlan, errors.Error) {
	var plan coreModels.PipelinePlan
	var stage coreModels.PipelineStage

	// construct task options for Org
	options := make(map[string]interface{})
	options["projectMappings"] = []tasks.ProjectMapping{tasks.NewProjectMapping(projectName, scopes)}

	subtasks, err := helper.MakePipelinePlanSubtasks([]plugin.SubTaskMeta{tasks.SetProjectMappingMeta}, []string{plugin.DOMAIN_TYPE_CROSS})
	if err != nil {
		return nil, err
	}
	stage = append(stage, &coreModels.PipelineTask{
		Plugin:   "org",
		Subtasks: subtasks,
		Options:  options,
	})
	plan = append(plan, stage)
	return plan, nil
}

func (p Org) PrepareTaskData(taskCtx plugin.TaskContext, options map[string]interface{}) (interface{}, errors.Error) {
	var op tasks.Options
	err := helper.Decode(options, &op, nil)
	if err != nil {
		return nil, errors.BadInput.Wrap(err, "could not decode options")
	}
	taskData := &tasks.TaskData{
		Options: &op,
	}
	return taskData, nil
}

func (p Org) RootPkgPath() string {
	return "github.com/apache/incubator-devlake/plugins/org"
}

// ApiResources registers the plugin's HTTP endpoints.
//
// Handlers are resolved through closures at request time rather than bound as
// method values here. ApiResources() can be invoked during router setup before
// Init() has populated p.handlers: when a DB migration is pending and the server
// is awaiting confirmation, InitPlugins() (which calls Init) is deferred until the
// migration is executed. Binding p.handlers.GetX directly at that point would
// capture a nil *Handlers, and every org endpoint would panic with a nil-pointer
// dereference on the first request even after Init() later populates p.handlers.
// The closures defer the lookup until the request fires, by which time Init() has
// run. The receiver is a pointer so the closures observe the populated field.
func (p *Org) ApiResources() map[string]map[string]plugin.ApiResourceHandler {
	return map[string]map[string]plugin.ApiResourceHandler{
		"teams.csv": {
			"GET": func(i *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
				return p.handlers.GetTeam(i)
			},
			"PUT": func(i *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
				return p.handlers.CreateTeam(i)
			},
		},
		"users.csv": {
			"GET": func(i *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
				return p.handlers.GetUser(i)
			},
			"PUT": func(i *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
				return p.handlers.CreateUser(i)
			},
		},
		"user_account_mapping.csv": {
			"GET": func(i *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
				return p.handlers.GetUserAccountMapping(i)
			},
			"PUT": func(i *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
				return p.handlers.CreateUserAccountMapping(i)
			},
		},
		"project_mapping.csv": {
			"GET": func(i *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
				return p.handlers.GetProjectMapping(i)
			},
			"PUT": func(i *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
				return p.handlers.CreateProjectMapping(i)
			},
		},
	}
}
