package impl

import (
	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/squadcast/models"
	"github.com/apache/incubator-devlake/plugins/squadcast/models/migrationscripts"
	"github.com/apache/incubator-devlake/plugins/squadcast/tasks"
)

type SquadcastPlugin struct{}

func (p SquadcastPlugin) Name() string {
	return "squadcast"
}

func (p SquadcastPlugin) MigrationScripts() []plugin.MigrationScript {
	return []plugin.MigrationScript{
		&migrationscripts.AddSquadcastIncidentTable{},
	}
}

func (p SquadcastPlugin) SubTaskMetas() []plugin.SubTaskMeta {
	return []plugin.SubTaskMeta{
		tasks.CollectIncidentsMeta,
		tasks.ExtractIncidentsMeta,
	}
}

func (p SquadcastPlugin) PrepareTaskData(taskCtx plugin.TaskContext, options map[string]interface{}) (interface{}, errors.Error) {
	var op tasks.SquadcastOptions
	err := helper.Decode(options, &op, nil)
	if err != nil {
		return nil, errors.Default.Wrap(err, "could not decode Squadcast options")
	}
	if op.ConnectionId == 0 {
		return nil, errors.Default.New("squadcast connectionId is invalid")
	}
	connection := &models.SquadcastConnection{}
	err = taskCtx.GetDal().First(connection, dal.Where("id = ?", op.ConnectionId))
	if err != nil {
		return nil, errors.Default.Wrap(err, "unable to get Squadcast connection")
	}
	return &tasks.SquadcastTaskData{
		Options:    &op,
		Connection: connection,
	}, nil
}

var Plugin SquadcastPlugin
