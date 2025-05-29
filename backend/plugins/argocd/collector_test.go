package tasks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/stretchr/testify/assert"
)

func TestFetchApplications(t *testing.T) {
	mockApps := struct {
		Items []Application `json:"items"`
	}{
		Items: []Application{{Name: "app1"}, {Name: "app2"}},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/applications", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		json.NewEncoder(w).Encode(mockApps)
	}))
	defer server.Close()

	conn := &ArgoCDConnection{
		RestConnection: helper.RestConnection{Endpoint: server.URL + "/"},
		Token:          "test-token",
	}
	ctx := context.WithValue(context.Background(), "testBasicRes", &testBasicRes{})
	apps, err := FetchApplications(ctx, conn)
	assert.NoError(t, err)
	assert.Len(t, apps, 2)
	assert.Equal(t, "app1", apps[0].Name)
}

func TestFetchProjects(t *testing.T) {
	mockProjects := struct {
		Items []Project `json:"items"`
	}{
		Items: []Project{{Name: "proj1"}},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		json.NewEncoder(w).Encode(mockProjects)
	}))
	defer server.Close()

	conn := &ArgoCDConnection{
		RestConnection: helper.RestConnection{Endpoint: server.URL + "/"},
		Token:          "test-token",
	}
	ctx := context.WithValue(context.Background(), "testBasicRes", &testBasicRes{})
	projects, err := FetchProjects(ctx, conn)
	assert.NoError(t, err)
	assert.Len(t, projects, 1)
	assert.Equal(t, "proj1", projects[0].Name)
}

func TestFetchClusters(t *testing.T) {
	mockClusters := struct {
		Items []Cluster `json:"items"`
	}{
		Items: []Cluster{{Name: "cluster1"}},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/clusters", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		json.NewEncoder(w).Encode(mockClusters)
	}))
	defer server.Close()

	conn := &ArgoCDConnection{
		RestConnection: helper.RestConnection{Endpoint: server.URL + "/"},
		Token:          "test-token",
	}
	ctx := context.WithValue(context.Background(), "testBasicRes", &testBasicRes{})
	clusters, err := FetchClusters(ctx, conn)
	assert.NoError(t, err)
	assert.Len(t, clusters, 1)
	assert.Equal(t, "cluster1", clusters[0].Name)
}

func TestFetchApplications_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	conn := &ArgoCDConnection{
		RestConnection: helper.RestConnection{Endpoint: server.URL + "/"},
		Token:          "test-token",
	}
	ctx := context.WithValue(context.Background(), "testBasicRes", &testBasicRes{})
	apps, err := FetchApplications(ctx, conn)
	assert.Error(t, err)
	assert.Nil(t, apps)
}
