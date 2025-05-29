package migrationscripts

import (
	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/helpers/migrationhelper"
	"github.com/apache/incubator-devlake/plugins/argocd/models"
)

type addArgoCDProject struct{}

func (*addArgoCDProject) Version() uint64 {
	return 20240601000002
}

func (*addArgoCDProject) Name() string {
	return "add argocd project table"
}

func (*addArgoCDProject) Up(basicRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&models.ArgoCDProject{},
	)
}
