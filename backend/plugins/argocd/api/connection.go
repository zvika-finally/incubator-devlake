package api

import (
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
)

// ArgoCDConnection holds the connection info for ArgoCD API
// Follows DevLake plugin conventions
type ArgoCDConnection struct {
	helper.RestConnection `mapstructure:",squash"`
	Token                 string `mapstructure:"token" json:"token" validate:"required"`
	ConnectionID          uint64 `mapstructure:"connectionId" json:"connectionId" gorm:"primaryKey;autoIncrement:false"`
}

// TableName returns the table name for GORM
func (ArgoCDConnection) TableName() string {
	return "_tool_argocd_connections"
}

// ConnectionId returns the connection ID (required by ToolLayerConnection)
func (c ArgoCDConnection) ConnectionId() uint64 {
	return c.ConnectionID
}

// CreateConnection handles POST /connections
func CreateConnection(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return dsHelper.ConnApi.Post(input)
}

// ListConnections handles GET /connections
func ListConnections(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return dsHelper.ConnApi.GetAll(input)
}

// GetConnection handles GET /connections/:connectionId
func GetConnection(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return dsHelper.ConnApi.GetDetail(input)
}

// UpdateConnection handles PUT /connections/:connectionId
func UpdateConnection(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return dsHelper.ConnApi.Patch(input)
}

// DeleteConnection handles DELETE /connections/:connectionId
func DeleteConnection(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return dsHelper.ConnApi.Delete(input)
}

// ApiResources returns the API resource map for the plugin
var ApiResources = map[string]map[string]plugin.ApiResourceHandler{
	"connections": {
		"POST": CreateConnection,
		"GET":  ListConnections,
	},
	"connections/:connectionId": {
		"GET":    GetConnection,
		"PUT":    UpdateConnection,
		"DELETE": DeleteConnection,
	},
}
