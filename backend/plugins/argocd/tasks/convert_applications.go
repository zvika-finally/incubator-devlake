package tasks

import (
	"reflect"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/domainlayer"
	"github.com/apache/incubator-devlake/core/models/domainlayer/devops"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/argocd/models"
)

var ConvertApplicationsMeta = plugin.SubTaskMeta{
	Name:             "convertApplications",
	EntryPoint:       ConvertApplications,
	EnabledByDefault: true,
	Description:      "Convert ArgoCD applications to domain layer",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CICD},
}

func ConvertApplications(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*ArgoCDTaskData)
	connectionId := data.Options.ConnectionId

	cursor, err := db.Cursor(
		dal.From(models.ArgoCDApplication{}),
		dal.Where("connection_id = ?", connectionId),
	)
	if err != nil {
		return err
	}
	defer cursor.Close()

	converter, err := helper.NewDataConverter(helper.DataConverterArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Table: models.ArgoCDApplication{}.TableName(),
		},
		InputRowType: reflect.TypeOf(models.ArgoCDApplication{}),
		Input:        cursor,
		Convert: func(inputRow interface{}) ([]interface{}, errors.Error) {
			app := inputRow.(*models.ArgoCDApplication)
			
			// Convert to CICD Deployment
			deployment := &devops.CICDDeployment{
				DomainEntity: domainlayer.DomainEntity{
					Id: app.ArgoCDId,
				},
				Name:         app.Name,
				Result:       convertHealthToResult(app.Health),
				Status:       convertSyncStatusToStatus(app.SyncStatus),
				Environment:  app.Namespace,
			}
			
			// Set creation date if available
			if app.CreatedDate != nil {
				deployment.CreatedDate = *app.CreatedDate
			}

			return []interface{}{deployment}, nil
		},
	})
	if err != nil {
		return err
	}

	return converter.Execute()
}

func convertHealthToResult(health string) string {
	switch health {
	case "Healthy":
		return "SUCCESS"
	case "Degraded":
		return "FAILURE"
	case "Progressing":
		return "IN_PROGRESS"
	default:
		return "FAILURE"
	}
}

func convertSyncStatusToStatus(syncStatus string) string {
	switch syncStatus {
	case "Synced":
		return "DONE"
	case "OutOfSync":
		return "PENDING"
	case "Unknown":
		return "PENDING"
	default:
		return "PENDING"
	}
}