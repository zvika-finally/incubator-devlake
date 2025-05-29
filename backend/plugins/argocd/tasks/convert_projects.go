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

var ConvertProjectsMeta = plugin.SubTaskMeta{
	Name:             "convertProjects",
	EntryPoint:       ConvertProjects,
	EnabledByDefault: true,
	Description:      "Convert ArgoCD projects to domain layer",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CICD},
}

func ConvertProjects(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*ArgoCDTaskData)
	connectionId := data.Options.ConnectionId

	cursor, err := db.Cursor(
		dal.From(models.ArgoCDProject{}),
		dal.Where("connection_id = ?", connectionId),
	)
	if err != nil {
		return err
	}
	defer cursor.Close()

	converter, err := helper.NewDataConverter(helper.DataConverterArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Table: models.ArgoCDProject{}.TableName(),
		},
		InputRowType: reflect.TypeOf(models.ArgoCDProject{}),
		Input:        cursor,
		Convert: func(inputRow interface{}) ([]interface{}, errors.Error) {
			project := inputRow.(*models.ArgoCDProject)
			
			// Convert to CICD Scope
			scope := &devops.CicdScope{
				DomainEntity: domainlayer.DomainEntity{
					Id: project.ArgoCDId,
				},
				Name:        project.Name,
				Description: project.Description,
				CreatedDate: project.CreatedDate,
			}

			return []interface{}{scope}, nil
		},
	})
	if err != nil {
		return err
	}

	return converter.Execute()
}