package tasks

import (
	"encoding/json"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/common"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/argocd/models"
)

var ExtractApplicationsMeta = plugin.SubTaskMeta{
	Name:             "extractApplications",
	EntryPoint:       ExtractApplications,
	EnabledByDefault: true,
	Description:      "Extract ArgoCD applications",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CICD},
}

type ApiApplication struct {
	Metadata struct {
		Name              string            `json:"name"`
		Namespace         string            `json:"namespace"`
		CreationTimestamp *common.Iso8601Time `json:"creationTimestamp"`
		UID               string            `json:"uid"`
	} `json:"metadata"`
	Spec struct {
		Project string `json:"project"`
		Source  struct {
			RepoURL        string `json:"repoURL"`
			Path           string `json:"path"`
			TargetRevision string `json:"targetRevision"`
		} `json:"source"`
		Destination struct {
			Server    string `json:"server"`
			Namespace string `json:"namespace"`
		} `json:"destination"`
	} `json:"spec"`
	Status struct {
		Health struct {
			Status string `json:"status"`
		} `json:"health"`
		Sync struct {
			Status string `json:"status"`
		} `json:"sync"`
	} `json:"status"`
}

func ExtractApplications(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*ArgoCDTaskData)
	extractor, err := helper.NewApiExtractor(helper.ApiExtractorArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: ArgoCDApiParams{
				ConnectionId: data.Options.ConnectionId,
			},
			Table: models.RawArgoCDApplication{}.TableName(),
		},
		Extract: func(row *helper.RawData) ([]interface{}, errors.Error) {
			var apiApp ApiApplication
			err := errors.Convert(json.Unmarshal(row.Data, &apiApp))
			if err != nil {
				return nil, err
			}

			app := &models.ArgoCDApplication{
				ArgoCDId:     apiApp.Metadata.UID,
				Name:         apiApp.Metadata.Name,
				Project:      apiApp.Spec.Project,
				Namespace:    apiApp.Metadata.Namespace,
				RepoURL:      apiApp.Spec.Source.RepoURL,
				Path:         apiApp.Spec.Source.Path,
				TargetRev:    apiApp.Spec.Source.TargetRevision,
				Health:       apiApp.Status.Health.Status,
				SyncStatus:   apiApp.Status.Sync.Status,
				Cluster:      apiApp.Spec.Destination.Server,
			}
			
			app.ConnectionId = data.Options.ConnectionId
			
			if apiApp.Metadata.CreationTimestamp != nil {
				createdTime := apiApp.Metadata.CreationTimestamp.ToTime()
				app.CreatedDate = &createdTime
			}

			return []interface{}{app}, nil
		},
	})
	if err != nil {
		return err
	}

	return extractor.Execute()
}