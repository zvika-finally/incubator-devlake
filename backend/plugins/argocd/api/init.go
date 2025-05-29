package api

import (
	"github.com/apache/incubator-devlake/core/context"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/argocd/models"
	"github.com/go-playground/validator/v10"
)

var dsHelper *helper.DsHelper[models.ArgoCDConnection, models.ArgoCDApplication, models.ArgoCDScopeConfig]
var vld *validator.Validate
var basicRes context.BasicRes

func Init(br context.BasicRes, p any) {
	basicRes = br
	vld = validator.New()
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
