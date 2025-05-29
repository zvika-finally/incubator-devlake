package impl

import (
	"github.com/apache/incubator-devlake/core/plugin"
)

// Plugin entrypoint for ArgoCD
// Implements the plugin.Plugin interface

type ArgoCDPlugin struct{}

func (p ArgoCDPlugin) Description() string {
	return "Collects and integrates data from ArgoCD."
}

func (p ArgoCDPlugin) RootPkgPath() string {
	return "github.com/apache/incubator-devlake/plugins/argocd"
}

func (p ArgoCDPlugin) Name() string {
	return "argocd"
}

func (p ArgoCDPlugin) Connection() interface{} {
	return &ArgoCDConnection{}
}

func (p ArgoCDPlugin) MigrationScripts() []plugin.MigrationScript {
	return nil // Add migration scripts if needed
}

func (p ArgoCDPlugin) SubTaskMetas() []plugin.SubTaskMeta {
	return SubTaskMetas
}

var PluginInstance ArgoCDPlugin
