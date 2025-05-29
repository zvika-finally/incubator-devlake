package migrationscripts

import (
	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/helpers/migrationhelper"
	"github.com/apache/incubator-devlake/plugins/argocd/models"
)

type addArgoCDApplication struct{}

func (*addArgoCDApplication) Version() uint64 {
	return 20250529000001
}

func (*addArgoCDApplication) Name() string {
	return "add argocd application and connection tables"
}

func (*addArgoCDApplication) Up(basicRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&models.ArgoCDApplication{},
		&models.ArgoCDConnection{},
		&models.RawArgoCDApplication{},
	)
}
