package tasks

import (
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/argocd/models"
)

type ArgoCDOptions struct {
	ConnectionId   uint64                    `json:"connectionId" mapstructure:"connectionId,omitempty"`
	ScopeConfigId  uint64                    `json:"scopeConfigId" mapstructure:"scopeConfigId,omitempty"`
	ScopeConfig    *models.ArgoCDScopeConfig `json:"scopeConfig" mapstructure:"scopeConfig,omitempty"`
	AppId          string                    `json:"appId" mapstructure:"appId,omitempty"`
}

type ArgoCDTaskData struct {
	Options   *ArgoCDOptions
	ApiClient *helper.ApiAsyncClient
}

type ArgoCDApiParams struct {
	ConnectionId uint64 `json:"connectionId"`
	AppId        string `json:"appId,omitempty"`
}

func DecodeAndValidateTaskOptions(options map[string]interface{}) (*ArgoCDOptions, errors.Error) {
	var op ArgoCDOptions
	if err := helper.Decode(options, &op, nil); err != nil {
		return nil, err
	}
	if op.ConnectionId == 0 {
		return nil, errors.BadInput.New("connectionId is required")
	}
	return &op, nil
}

func NewArgoCDApiClient(taskCtx plugin.TaskContext, connection *models.ArgoCDConnection) (*helper.ApiAsyncClient, errors.Error) {
	apiClient, err := helper.NewApiClientFromConnection(taskCtx.GetContext(), taskCtx, connection)
	if err != nil {
		return nil, err
	}
	asyncClient, err := helper.CreateAsyncApiClient(taskCtx, apiClient, nil)
	if err != nil {
		return nil, err
	}
	return asyncClient, nil
}