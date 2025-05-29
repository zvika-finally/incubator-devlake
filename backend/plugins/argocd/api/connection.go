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

// Connection API handlers
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

// Scope API handlers (simplified)
func GetScope(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return &plugin.ApiResourceOutput{}, nil
}

func PatchScope(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return &plugin.ApiResourceOutput{}, nil
}

func DeleteScope(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return &plugin.ApiResourceOutput{}, nil
}

func GetScopeList(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return &plugin.ApiResourceOutput{}, nil
}

func PutScopes(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return &plugin.ApiResourceOutput{}, nil
}

// Remote scope API handlers (simplified)
func RemoteScopes(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return &plugin.ApiResourceOutput{}, nil
}

// Scope config API handlers (simplified)
func CreateScopeConfig(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return &plugin.ApiResourceOutput{}, nil
}

func GetScopeConfigList(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return &plugin.ApiResourceOutput{}, nil
}

func GetScopeConfig(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return &plugin.ApiResourceOutput{}, nil
}

func PatchScopeConfig(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return &plugin.ApiResourceOutput{}, nil
}

func DeleteScopeConfig(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return &plugin.ApiResourceOutput{}, nil
}
