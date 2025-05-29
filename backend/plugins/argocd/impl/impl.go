package impl

import (
	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/argocd/models"
	"github.com/apache/incubator-devlake/plugins/argocd/models/migrationscripts"
)

type Plugin struct{}

func (p Plugin) Description() string {
	return "ArgoCD plugin for Apache DevLake. Collects Applications, Projects, and Clusters from ArgoCD."
}

func (p Plugin) Name() string {
	return "argocd"
}

func (p Plugin) RootPkgPath() string {
	return "github.com/apache/incubator-devlake/backend/plugins/argocd"
}

func (p Plugin) Init(basicRes context.BasicRes) errors.Error {
	// Initialize any global variables or connections here
	return nil
}

func (p Plugin) MigrationScripts() []plugin.MigrationScript {
	return migrationscripts.All()
}

func (p Plugin) ModelPkgPath() string {
	return "github.com/apache/incubator-devlake/plugins/argocd/models"
}

func (p Plugin) Models() []any {
	return []any{
		&models.ArgoCDApplication{},
		&models.ArgoCDProject{},
		&models.ArgoCDCluster{},
	}
}

func (p Plugin) GetTablesInfo() []dal.Tabler {
	return []dal.Tabler{
		&models.ArgoCDApplication{},
		&models.ArgoCDProject{},
		&models.ArgoCDCluster{},
	}
}

var _ plugin.PluginMeta = (*Plugin)(nil)
var _ plugin.PluginInit = (*Plugin)(nil)
var _ plugin.PluginMigration = (*Plugin)(nil)
var _ plugin.PluginModel = (*Plugin)(nil)
