package models

import (
	"github.com/apache/incubator-devlake/core/models/common"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
)

type ArgoCDConnection struct {
	helper.RestConnection `mapstructure:",squash"`
	common.Model          `mapstructure:",squash"`
}

func (ArgoCDConnection) TableName() string {
	return "_tool_argocd_connections"
}

func (c ArgoCDConnection) ConnectionId() uint64 {
	return c.Model.ID
}