package tasks

import (
	"github.com/apache/incubator-devlake/core/plugin"
)

// All subtasks for the ArgoCD plugin
var SubTaskMetas = []plugin.SubTaskMeta{
	CollectApplicationsMeta,
	ExtractApplicationsMeta,
	ConvertApplicationsMeta,
	CollectProjectsMeta,
	ExtractProjectsMeta,
	ConvertProjectsMeta,
	CollectClustersMeta,
	ExtractClustersMeta,
	ConvertClustersMeta,
}
