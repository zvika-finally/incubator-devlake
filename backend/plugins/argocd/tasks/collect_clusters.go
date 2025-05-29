package tasks

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/argocd/models"
)

var CollectClustersMeta = plugin.SubTaskMeta{
	Name:             "collectClusters",
	EntryPoint:       CollectClusters,
	EnabledByDefault: true,
	Description:      "Collect ArgoCD clusters",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CICD},
}

func CollectClusters(taskCtx plugin.SubTaskContext) errors.Error {
	logger := taskCtx.GetLogger()
	data := taskCtx.GetData().(*ArgoCDTaskData)

	collector, err := helper.NewApiCollector(helper.ApiCollectorArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Table: models.RawArgoCDCluster{}.TableName(),
		},
		ApiClient:   data.ApiClient,
		PageSize:    50,
		UrlTemplate: "api/v1/clusters",
		Query: func(reqData *helper.RequestData) (url.Values, errors.Error) {
			query := url.Values{}
			return query, nil
		},
		GetTotalPages: func(res *http.Response, args *helper.ApiCollectorArgs) (int, errors.Error) {
			return 1, nil
		},
		ResponseParser: func(res *http.Response) ([]json.RawMessage, errors.Error) {
			var data struct {
				Items []json.RawMessage `json:"items"`
			}
			err := helper.UnmarshalResponse(res, &data)
			return data.Items, err
		},
	})

	if err != nil {
		return err
	}

	err = collector.Execute()
	if err != nil {
		logger.Error(err, "failed to collect ArgoCD clusters")
		return err
	}

	return nil
}