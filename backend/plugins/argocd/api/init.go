package api

import (
	"github.com/apache/incubator-devlake/core/context"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/argocd/models"
)

var dsHelper *helper.DsHelper[models.ArgoCDConnection, models.ArgoCDApplication, models.ArgoCDScopeConfig]

func Init(br context.BasicRes, p any) {
	dsHelper = helper.NewDataSourceHelper[
		models.ArgoCDConnection, models.ArgoCDApplication, models.ArgoCDScopeConfig,
	](
		br,
		"argocd",
		[]string{"name"},
		nil,
		nil,
		nil,
	)
}
