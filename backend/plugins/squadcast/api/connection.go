package api

import (
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/squadcast/models"
)

func RegisterSquadcastConnectionApi(router plugin.Router) {
	api.RegisterConnectionApi[models.SquadcastConnection](router, "_tool_squadcast_connections")
}
