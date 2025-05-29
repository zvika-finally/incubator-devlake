package tasks

import (
	"context"
	"net/http"
	"net/url"

	corecontext "github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/log"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
)

// testBasicRes is a minimal mock for corecontext.BasicRes for testing purposes.
type testBasicRes struct{}

func (t *testBasicRes) GetConfigReader() interface {
	GetString(string) string
	GetBool(string) bool
} {
	return &mockConfigReader{}
}

type mockConfigReader struct{}

func (m *mockConfigReader) GetString(key string) string { return "" }
func (m *mockConfigReader) GetBool(key string) bool     { return false }

type noopLogger struct{}

func (n *noopLogger) IsLevelEnabled(level log.LogLevel) bool                  { return false }
func (n *noopLogger) Printf(format string, a ...interface{})                  {}
func (n *noopLogger) Log(level log.LogLevel, format string, a ...interface{}) {}
func (n *noopLogger) Debug(format string, a ...interface{})                   {}
func (n *noopLogger) Info(format string, a ...interface{})                    {}
func (n *noopLogger) Warn(err error, format string, a ...interface{})         {}
func (n *noopLogger) Error(err error, format string, a ...interface{})        {}
func (n *noopLogger) Nested(name string) log.Logger                           { return n }
func (n *noopLogger) GetConfig() *log.LoggerConfig                            { return &log.LoggerConfig{} }
func (n *noopLogger) SetStream(config *log.LoggerStreamConfig)                {}

func (t *testBasicRes) GetLogger() log.Logger { return &noopLogger{} }

// FetchApplications fetches applications from the ArgoCD API endpoint using DevLake's ApiClient.
func FetchApplications(ctx context.Context, connection *ArgoCDConnection) ([]Application, errors.Error) {
	var br corecontext.BasicRes = nil
	if v := ctx.Value("testBasicRes"); v != nil {
		br, _ = v.(corecontext.BasicRes)
	}
	apiClient, err := helper.NewApiClient(ctx, connection.GetEndpoint(), nil, 0, connection.GetProxy(), br)
	if err != nil {
		return nil, err
	}
	// Set Bearer token for authentication
	apiClient.SetHeaders(map[string]string{
		"Authorization": "Bearer " + connection.Token,
	})
	res, err := apiClient.Get("api/v1/applications", url.Values{}, http.Header{})
	if err != nil {
		return nil, err
	}
	var result struct {
		Items []Application `json:"items"`
	}
	err = helper.UnmarshalResponse(res, &result)
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

// FetchProjects fetches projects from the ArgoCD API endpoint using DevLake's ApiClient.
func FetchProjects(ctx context.Context, connection *ArgoCDConnection) ([]Project, errors.Error) {
	var br corecontext.BasicRes = nil
	if v := ctx.Value("testBasicRes"); v != nil {
		br, _ = v.(corecontext.BasicRes)
	}
	apiClient, err := helper.NewApiClient(ctx, connection.GetEndpoint(), nil, 0, connection.GetProxy(), br)
	if err != nil {
		return nil, err
	}
	apiClient.SetHeaders(map[string]string{
		"Authorization": "Bearer " + connection.Token,
	})
	res, err := apiClient.Get("api/v1/projects", url.Values{}, http.Header{})
	if err != nil {
		return nil, err
	}
	var result struct {
		Items []Project `json:"items"`
	}
	err = helper.UnmarshalResponse(res, &result)
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

// FetchClusters fetches clusters from the ArgoCD API endpoint using DevLake's ApiClient.
func FetchClusters(ctx context.Context, connection *ArgoCDConnection) ([]Cluster, errors.Error) {
	var br corecontext.BasicRes = nil
	if v := ctx.Value("testBasicRes"); v != nil {
		br, _ = v.(corecontext.BasicRes)
	}
	apiClient, err := helper.NewApiClient(ctx, connection.GetEndpoint(), nil, 0, connection.GetProxy(), br)
	if err != nil {
		return nil, err
	}
	apiClient.SetHeaders(map[string]string{
		"Authorization": "Bearer " + connection.Token,
	})
	res, err := apiClient.Get("api/v1/clusters", url.Values{}, http.Header{})
	if err != nil {
		return nil, err
	}
	var result struct {
		Items []Cluster `json:"items"`
	}
	err = helper.UnmarshalResponse(res, &result)
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}
