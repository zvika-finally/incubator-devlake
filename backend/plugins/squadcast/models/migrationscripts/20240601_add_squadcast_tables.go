package migrationscripts

import (
	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/helpers/migrationhelper"
	"github.com/apache/incubator-devlake/plugins/squadcast/models"
)

type AddSquadcastIncidentTable struct{}

func (*AddSquadcastIncidentTable) Up(baseRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(baseRes, &models.SquadcastIncident{}, &models.SquadcastConnection{})
}

func (*AddSquadcastIncidentTable) Version() uint64 {
	return 20240601000000
}

func (*AddSquadcastIncidentTable) Name() string {
	return "add _tool_squadcast_incidents and _tool_squadcast_connections tables"
}
