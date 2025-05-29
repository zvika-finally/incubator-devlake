package migrationscripts

import (
	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/helpers/migrationhelper"
	"github.com/apache/incubator-devlake/plugins/argocd/models"
)

type addArgoCDCluster struct{}

func (*addArgoCDCluster) Version() uint64 {
	return 20240601000003
}

func (*addArgoCDCluster) Name() string {
	return "add argocd cluster table"
}

func (*addArgoCDCluster) Up(basicRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&models.ArgoCDCluster{},
	)
}
