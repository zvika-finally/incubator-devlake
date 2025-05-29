package tasks

import (
	"encoding/json"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/argocd/models"
)

var ExtractClustersMeta = plugin.SubTaskMeta{
	Name:             "extractClusters",
	EntryPoint:       ExtractClusters,
	EnabledByDefault: true,
	Description:      "Extract ArgoCD clusters",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CICD},
}

type ApiCluster struct {
	Name   string `json:"name"`
	Server string `json:"server"`
	Config struct {
		BearerToken     string `json:"bearerToken"`
		TlsClientConfig struct {
			Insecure bool `json:"insecure"`
		} `json:"tlsClientConfig"`
	} `json:"config"`
	ConnectionState struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	} `json:"connectionState"`
	ServerVersion string `json:"serverVersion"`
	Info          struct {
		ConnectionState struct {
			Status  string `json:"status"`
			Message string `json:"message"`
		} `json:"connectionState"`
		ServerVersion   string            `json:"serverVersion"`
		ApplicationsCount int             `json:"applicationsCount"`
	} `json:"info"`
}

func ExtractClusters(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*ArgoCDTaskData)
	extractor, err := helper.NewApiExtractor(helper.ApiExtractorArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: ArgoCDApiParams{
				ConnectionId: data.Options.ConnectionId,
			},
			Table: models.RawArgoCDCluster{}.TableName(),
		},
		Extract: func(row *helper.RawData) ([]interface{}, errors.Error) {
			var apiCluster ApiCluster
			err := errors.Convert(json.Unmarshal(row.Data, &apiCluster))
			if err != nil {
				return nil, err
			}

			cluster := &models.ArgoCDCluster{
				ArgoCDId:      apiCluster.Name, // Use name as ID for clusters
				Name:          apiCluster.Name,
				Server:        apiCluster.Server,
				ServerVersion: apiCluster.ServerVersion,
				Status:        apiCluster.ConnectionState.Status,
			}
			
			cluster.ConnectionId = data.Options.ConnectionId

			return []interface{}{cluster}, nil
		},
	})
	if err != nil {
		return err
	}

	return extractor.Execute()
}