package tasks

import (
	"context"

	"encoding/json"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	tasks "github.com/apache/incubator-devlake/plugins/argocd/tasks"
)

// Subtask metadata for collecting ArgoCD applications
type CollectApplicationsOptions struct {
	ConnectionId uint64 `mapstructure:"connectionId" validate:"required"`
}

var CollectApplicationsMeta = plugin.SubTaskMeta{
	Name:             "collectApplications",
	EntryPoint:       CollectApplications,
	EnabledByDefault: true,
	Description:      "Collect ArgoCD applications via API",
}

// CollectApplications is the main subtask for fetching ArgoCD applications
func CollectApplications(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData()
	conn, ok := data.(*ArgoCDConnection)
	if !ok {
		return errors.Default.New("invalid connection data")
	}
	apps, err := FetchApplications(context.Background(), conn)
	if err != nil {
		return errors.Default.Wrap(err, "fetching applications")
	}
	dal := taskCtx.GetDal()
	connectionId := conn.ConnectionId
	if connectionId == 0 {
		// fallback: try to get from options if not set
		if opts, ok := data.(*CollectApplicationsOptions); ok {
			connectionId = opts.ConnectionId
		}
	}
	for _, app := range apps {
		raw, err := json.Marshal(app)
		if err != nil {
			return errors.Default.Wrap(err, "marshalling application")
		}
		record := &RawArgoCDApplication{
			ConnectionId: connectionId,
			RawData:      raw,
		}
		if err := dal.Create(record); err != nil {
			return errors.Default.Wrap(err, "saving application")
		}
	}
	return nil
}

// Subtask metadata for collecting ArgoCD projects
type CollectProjectsOptions struct {
	ConnectionId uint64 `mapstructure:"connectionId" validate:"required"`
}

var CollectProjectsMeta = plugin.SubTaskMeta{
	Name:             "collectProjects",
	EntryPoint:       CollectProjects,
	EnabledByDefault: true,
	Description:      "Collect ArgoCD projects via API",
}

func CollectProjects(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData()
	conn, ok := data.(*ArgoCDConnection)
	if !ok {
		return errors.Default.New("invalid connection data")
	}
	projects, err := FetchProjects(context.Background(), conn)
	if err != nil {
		return errors.Default.Wrap(err, "fetching projects")
	}
	dal := taskCtx.GetDal()
	connectionId := conn.ConnectionId
	if connectionId == 0 {
		if opts, ok := data.(*CollectProjectsOptions); ok {
			connectionId = opts.ConnectionId
		}
	}
	for _, project := range projects {
		raw, err := json.Marshal(project)
		if err != nil {
			return errors.Default.Wrap(err, "marshalling project")
		}
		record := &RawArgoCDProject{
			ConnectionId: connectionId,
			RawData:      raw,
		}
		if err := dal.Create(record); err != nil {
			return errors.Default.Wrap(err, "saving project")
		}
	}
	return nil
}

// Subtask metadata for collecting ArgoCD clusters
type CollectClustersOptions struct {
	ConnectionId uint64 `mapstructure:"connectionId" validate:"required"`
}

var CollectClustersMeta = plugin.SubTaskMeta{
	Name:             "collectClusters",
	EntryPoint:       CollectClusters,
	EnabledByDefault: true,
	Description:      "Collect ArgoCD clusters via API",
}

func CollectClusters(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData()
	conn, ok := data.(*ArgoCDConnection)
	if !ok {
		return errors.Default.New("invalid connection data")
	}
	clusters, err := FetchClusters(context.Background(), conn)
	if err != nil {
		return errors.Default.Wrap(err, "fetching clusters")
	}
	dal := taskCtx.GetDal()
	connectionId := conn.ConnectionId
	if connectionId == 0 {
		if opts, ok := data.(*CollectClustersOptions); ok {
			connectionId = opts.ConnectionId
		}
	}
	for _, cluster := range clusters {
		raw, err := json.Marshal(cluster)
		if err != nil {
			return errors.Default.Wrap(err, "marshalling cluster")
		}
		record := &RawArgoCDCluster{
			ConnectionId: connectionId,
			RawData:      raw,
		}
		if err := dal.Create(record); err != nil {
			return errors.Default.Wrap(err, "saving cluster")
		}
	}
	return nil
}

// Register all subtasks for plugin registration
var SubTaskMetas = []plugin.SubTaskMeta{
	CollectApplicationsMeta,
	CollectProjectsMeta,
	CollectClustersMeta,
	tasks.ExtractApplicationsMeta,
	tasks.ConvertApplicationsMeta,
}
