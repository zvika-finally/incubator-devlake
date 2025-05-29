package api

import (
	"context"
	"net/http"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/argocd/models"
)

func testConnection(ctx context.Context, connection models.ArgoCDConnection) (*plugin.ApiResourceOutput, errors.Error) {
	// Create API client from connection
	apiClient, err := api.NewApiClientFromConnection(ctx, basicRes, &connection)
	if err != nil {
		return nil, err
	}

	// Test connection by making a simple API call to ArgoCD
	// Try to get the version information or list applications
	res, err := apiClient.Get("api/version", nil, nil)
	if err != nil {
		return nil, err
	}

	switch res.StatusCode {
	case http.StatusOK:
		return &plugin.ApiResourceOutput{
			Body: map[string]interface{}{
				"success": true,
				"message": "Connection test successful",
			},
			Status: http.StatusOK,
		}, nil
	case http.StatusUnauthorized:
		return nil, errors.HttpStatus(http.StatusBadRequest).New("Authentication failed. Please check your token.")
	case http.StatusForbidden:
		return nil, errors.HttpStatus(http.StatusBadRequest).New("Access denied. Please check your permissions.")
	default:
		return nil, errors.HttpStatus(http.StatusBadRequest).New("Failed to connect to ArgoCD server")
	}
}

// TestConnection tests ArgoCD connection
// @Summary test ArgoCD connection
// @Description Test ArgoCD Connection
// @Tags plugins/argocd
// @Param body body models.ArgoCDConnection true "json body"
// @Success 200  {object} map[string]interface{} "Success"
// @Failure 400  {string} errcode.Error "Bad Request"
// @Failure 500  {string} errcode.Error "Internal Error"
// @Router /plugins/argocd/test [POST]
func TestConnection(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	var connection models.ArgoCDConnection
	if err := api.Decode(input.Body, &connection, vld); err != nil {
		return nil, err
	}

	result, err := testConnection(context.TODO(), connection)
	if err != nil {
		return nil, plugin.WrapTestConnectionErrResp(basicRes, err)
	}
	return result, nil
}

// TestExistingConnection tests existing ArgoCD connection
// @Summary test ArgoCD connection
// @Description Test ArgoCD Connection
// @Tags plugins/argocd
// @Success 200  {object} map[string]interface{} "Success"
// @Failure 400  {string} errcode.Error "Bad Request"
// @Failure 500  {string} errcode.Error "Internal Error"
// @Router /plugins/argocd/connections/{connectionId}/test [POST]
func TestExistingConnection(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	connection, err := dsHelper.ConnApi.GetMergedConnection(input)
	if err != nil {
		return nil, errors.BadInput.Wrap(err, "can't read connection from database")
	}

	result, err := testConnection(context.TODO(), *connection)
	if err != nil {
		return nil, plugin.WrapTestConnectionErrResp(basicRes, err)
	}
	return result, nil
}

// PostConnections creates ArgoCD connection
// @Summary create ArgoCD connection
// @Description Create ArgoCD connection
// @Tags plugins/argocd
// @Param body body models.ArgoCDConnection true "json body"
// @Success 200  {object} models.ArgoCDConnection "Success"
// @Failure 400  {string} errcode.Error "Bad Request"
// @Failure 500  {string} errcode.Error "Internal Error"
// @Router /plugins/argocd/connections [POST]
func PostConnections(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return dsHelper.ConnApi.Post(input)
}

func ListConnections(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return dsHelper.ConnApi.GetAll(input)
}

func GetConnection(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return dsHelper.ConnApi.GetDetail(input)
}

func PatchConnection(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return dsHelper.ConnApi.Patch(input)
}

func DeleteConnection(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return dsHelper.ConnApi.Delete(input)
}

// Scope API handlers
func GetScope(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return dsHelper.ScopeApi.GetScopeDetail(input)
}

func PatchScope(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return dsHelper.ScopeApi.Patch(input)
}

func DeleteScope(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return dsHelper.ScopeApi.Delete(input)
}

func GetScopeList(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return dsHelper.ScopeApi.GetPage(input)
}

func PutScopes(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return dsHelper.ScopeApi.PutMultiple(input)
}

// RemoteScopes fetches applications from ArgoCD server
// @Summary Get remote scopes
// @Description Fetch available applications from ArgoCD server
// @Tags plugins/argocd
// @Param connectionId path int true "connection ID"
// @Success 200  {object} []map[string]interface{} "Success"
// @Failure 400  {string} errcode.Error "Bad Request"
// @Failure 500  {string} errcode.Error "Internal Error"
// @Router /plugins/argocd/connections/{connectionId}/remote-scopes [GET]
func RemoteScopes(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	connection, err := dsHelper.ConnApi.GetMergedConnection(input)
	if err != nil {
		return nil, errors.BadInput.Wrap(err, "can't read connection from database")
	}

	// Create API client
	apiClient, err := api.NewApiClientFromConnection(context.TODO(), basicRes, connection)
	if err != nil {
		return nil, err
	}

	// Fetch applications from ArgoCD
	res, err := apiClient.Get("api/v1/applications", nil, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Items []map[string]interface{} `json:"items"`
	}
	err = api.UnmarshalResponse(res, &response)
	if err != nil {
		return nil, err
	}

	// Convert to scope format
	scopes := make([]map[string]interface{}, 0, len(response.Items))
	for _, item := range response.Items {
		if metadata, ok := item["metadata"].(map[string]interface{}); ok {
			scope := map[string]interface{}{
				"id":   metadata["uid"],
				"name": metadata["name"],
			}
			if spec, ok := item["spec"].(map[string]interface{}); ok {
				scope["project"] = spec["project"]
			}
			scopes = append(scopes, scope)
		}
	}

	return &plugin.ApiResourceOutput{Body: scopes}, nil
}

// Scope config API handlers
func CreateScopeConfig(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return dsHelper.ScopeConfigApi.Post(input)
}

func GetScopeConfigList(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return dsHelper.ScopeConfigApi.GetAll(input)
}

func GetScopeConfig(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return dsHelper.ScopeConfigApi.GetDetail(input)
}

func PatchScopeConfig(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return dsHelper.ScopeConfigApi.Patch(input)
}

func DeleteScopeConfig(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return dsHelper.ScopeConfigApi.Delete(input)
}
