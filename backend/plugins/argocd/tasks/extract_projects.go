package tasks

import (
	"encoding/json"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/common"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/argocd/models"
)

var ExtractProjectsMeta = plugin.SubTaskMeta{
	Name:             "extractProjects",
	EntryPoint:       ExtractProjects,
	EnabledByDefault: true,
	Description:      "Extract ArgoCD projects",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CICD},
}

type ApiProject struct {
	Metadata struct {
		Name              string            `json:"name"`
		Namespace         string            `json:"namespace"`
		CreationTimestamp *common.Iso8601Time `json:"creationTimestamp"`
		UID               string            `json:"uid"`
	} `json:"metadata"`
	Spec struct {
		Description  string   `json:"description"`
		Destinations []struct {
			Server    string `json:"server"`
			Namespace string `json:"namespace"`
		} `json:"destinations"`
		SourceRepos []string `json:"sourceRepos"`
	} `json:"spec"`
}

func ExtractProjects(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*ArgoCDTaskData)
	extractor, err := helper.NewApiExtractor(helper.ApiExtractorArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: ArgoCDApiParams{
				ConnectionId: data.Options.ConnectionId,
			},
			Table: models.RawArgoCDProject{}.TableName(),
		},
		Extract: func(row *helper.RawData) ([]interface{}, errors.Error) {
			var apiProject ApiProject
			err := errors.Convert(json.Unmarshal(row.Data, &apiProject))
			if err != nil {
				return nil, err
			}

			project := &models.ArgoCDProject{
				ArgoCDId:    apiProject.Metadata.UID,
				Name:        apiProject.Metadata.Name,
				Description: apiProject.Spec.Description,
			}
			
			project.ConnectionId = data.Options.ConnectionId
			
			if apiProject.Metadata.CreationTimestamp != nil {
				createdTime := apiProject.Metadata.CreationTimestamp.ToTime()
				project.CreatedDate = &createdTime
			}

			return []interface{}{project}, nil
		},
	})
	if err != nil {
		return err
	}

	return extractor.Execute()
}