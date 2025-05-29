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

var ConvertClustersMeta = plugin.SubTaskMeta{
	Name:             "convertClusters",
	EntryPoint:       ConvertClusters,
	EnabledByDefault: true,
	Description:      "Convert ArgoCD clusters to domain layer",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CICD},
}

func ConvertClusters(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*ArgoCDTaskData)
	connectionId := data.Options.ConnectionId

	cursor, err := db.Cursor(
		dal.From(models.ArgoCDCluster{}),
		dal.Where("connection_id = ?", connectionId),
	)
	if err != nil {
		return err
	}
	defer cursor.Close()

	converter, err := helper.NewDataConverter(helper.DataConverterArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Table: models.ArgoCDCluster{}.TableName(),
		},
		InputRowType: reflect.TypeOf(models.ArgoCDCluster{}),
		Input:        cursor,
		Convert: func(inputRow interface{}) ([]interface{}, errors.Error) {
			cluster := inputRow.(*models.ArgoCDCluster)
			
			// Convert to CICD Scope for cluster management
			scope := &devops.CicdScope{
				DomainEntity: domainlayer.DomainEntity{
					Id: cluster.ArgoCDId,
				},
				Name:        cluster.Name,
				Description: "ArgoCD Cluster: " + cluster.Server,
				CreatedDate: cluster.CreatedDate,
			}

			return []interface{}{scope}, nil
		},
	})
	if err != nil {
		return err
	}

	return converter.Execute()
}