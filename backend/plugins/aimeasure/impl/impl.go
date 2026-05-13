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
	plugin.PluginMigration
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
	}
}

func (p AIMeasure) SubTaskMetas() []plugin.SubTaskMeta {
	return []plugin.SubTaskMeta{
		tasks.ClassifyPRCohortMeta,         // run first — produces pr_ai_cohort
		tasks.ComputeChangeCompositionMeta, // independent — produces pr_change_composition
		tasks.ComputeQualityCohortMeta,     // independent — produces pr_defect_signals
	}
}

func (p AIMeasure) PrepareTaskData(taskCtx plugin.TaskContext, options map[string]interface{}) (interface{}, errors.Error) {
	opts, err := tasks.DecodeAndValidateTaskOptions(options)
	if err != nil {
		return nil, err
	}
	return &tasks.AIMeasureTaskData{Options: opts}, nil
}
