package impl

import (
	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	coreModels "github.com/apache/incubator-devlake/core/models"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/argocd/api"
	"github.com/apache/incubator-devlake/plugins/argocd/models"
	"github.com/apache/incubator-devlake/plugins/argocd/models/migrationscripts"
	"github.com/apache/incubator-devlake/plugins/argocd/tasks"
)

// Ensure ArgoCDPlugin implements all required interfaces
var _ interface {
	plugin.PluginMeta
	plugin.PluginInit
	plugin.PluginTask
	plugin.PluginModel
	plugin.PluginMigration
	plugin.PluginSource
	plugin.DataSourcePluginBlueprintV200
	plugin.CloseablePluginTask
} = (*ArgoCDPlugin)(nil)

type ArgoCDPlugin struct{}

func (p ArgoCDPlugin) Init(basicRes context.BasicRes) errors.Error {
	api.Init(basicRes, p)
	return nil
}

func (p ArgoCDPlugin) Connection() dal.Tabler {
	return &models.ArgoCDConnection{}
}

func (p ArgoCDPlugin) Scope() plugin.ToolLayerScope {
	return &models.ArgoCDApplication{}
}

func (p ArgoCDPlugin) ScopeConfig() dal.Tabler {
	return &models.ArgoCDScopeConfig{}
}

func (p ArgoCDPlugin) MakeDataSourcePipelinePlanV200(
	connectionId uint64,
	scopes []*coreModels.BlueprintScope,
) (coreModels.PipelinePlan, []plugin.Scope, errors.Error) {
	return api.MakePipelinePlanV200(p.SubTaskMetas(), connectionId, scopes)
}

func (p ArgoCDPlugin) GetTablesInfo() []dal.Tabler {
	return []dal.Tabler{
		&models.ArgoCDConnection{},
		&models.ArgoCDApplication{},
		&models.ArgoCDProject{},
		&models.ArgoCDCluster{},
		&models.ArgoCDScopeConfig{},
		&models.RawArgoCDApplication{},
		&models.RawArgoCDProject{},
		&models.RawArgoCDCluster{},
	}
}

func (p ArgoCDPlugin) Description() string {
	return "Collects and integrates data from ArgoCD"
}

func (p ArgoCDPlugin) Name() string {
	return "argocd"
}

func (p ArgoCDPlugin) SubTaskMetas() []plugin.SubTaskMeta {
	return tasks.SubTaskMetas
}

func (p ArgoCDPlugin) PrepareTaskData(taskCtx plugin.TaskContext, options map[string]interface{}) (interface{}, errors.Error) {
	logger := taskCtx.GetLogger()
	logger.Debug("%v", options)
	op, err := tasks.DecodeAndValidateTaskOptions(options)
	if err != nil {
		return nil, err
	}
	if op.ConnectionId == 0 {
		return nil, errors.BadInput.New("connectionId is invalid")
	}
	connection := &models.ArgoCDConnection{}
	connectionHelper := helper.NewConnectionHelper(
		taskCtx,
		nil,
		p.Name(),
	)
	err = connectionHelper.FirstById(connection, op.ConnectionId)
	if err != nil {
		return nil, errors.BadInput.Wrap(err, "connection not found")
	}

	apiClient, err := tasks.NewArgoCDApiClient(taskCtx, connection)
	if err != nil {
		return nil, err
	}

	taskData := tasks.ArgoCDTaskData{
		Options:   op,
		ApiClient: apiClient,
	}

	return &taskData, nil
}

func (p ArgoCDPlugin) RootPkgPath() string {
	return "github.com/apache/incubator-devlake/plugins/argocd"
}

func (p ArgoCDPlugin) MigrationScripts() []plugin.MigrationScript {
	return migrationscripts.All()
}

func (p ArgoCDPlugin) ApiResources() map[string]map[string]plugin.ApiResourceHandler {
	return map[string]map[string]plugin.ApiResourceHandler{
		"test": {
			"POST": api.TestConnection,
		},
		"connections": {
			"POST": api.PostConnections,
			"GET":  api.ListConnections,
		},
		"connections/:connectionId": {
			"PATCH":  api.PatchConnection,
			"DELETE": api.DeleteConnection,
			"GET":    api.GetConnection,
		},
		"connections/:connectionId/test": {
			"POST": api.TestExistingConnection,
		},
		"connections/:connectionId/scopes/:scopeId": {
			"GET":    api.GetScope,
			"PATCH":  api.PatchScope,
			"DELETE": api.DeleteScope,
		},
		"connections/:connectionId/remote-scopes": {
			"GET": api.RemoteScopes,
		},
		"connections/:connectionId/scopes": {
			"GET": api.GetScopeList,
			"PUT": api.PutScopes,
		},
		"connections/:connectionId/scope-configs": {
			"POST": api.CreateScopeConfig,
			"GET":  api.GetScopeConfigList,
		},
		"connections/:connectionId/scope-configs/:scopeConfigId": {
			"PATCH":  api.PatchScopeConfig,
			"GET":    api.GetScopeConfig,
			"DELETE": api.DeleteScopeConfig,
		},
	}
}

func (p ArgoCDPlugin) Close(taskCtx plugin.TaskContext) errors.Error {
	data, ok := taskCtx.GetData().(*tasks.ArgoCDTaskData)
	if !ok {
		return errors.Default.New("GetData failed when trying to close ArgoCD plugin")
	}
	data.ApiClient.Release()
	return nil
}

// Export the plugin instance
var PluginInstance ArgoCDPlugin
