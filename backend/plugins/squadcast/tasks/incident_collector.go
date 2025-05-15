package tasks

import (
	"encoding/json"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	api "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
)

var CollectIncidentsMeta = plugin.SubTaskMeta{
	Name:             "Collect Squadcast Incidents",
	EntryPoint:       CollectIncidents,
	EnabledByDefault: true,
	Description:      "Collect incidents from Squadcast API",
	DomainTypes:      []string{"INCIDENT"},
}

func CollectIncidents(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*SquadcastTaskData)
	apiKey := data.Connection.ApiKey
	client := NewSquadcastClient(apiKey)

	incidents, err := client.FetchIncidents()
	if err != nil {
		return errors.Default.Wrap(err, "failed to fetch incidents from Squadcast")
	}

	db := taskCtx.GetDal()
	params := map[string]interface{}{
		"connectionId": data.Connection.ID,
	}
	paramsString := plugin.MarshalScopeParams(params)

	for _, incident := range incidents {
		raw, err := json.Marshal(incident)
		if err != nil {
			return errors.Default.Wrap(err, "failed to marshal incident")
		}
		rawData := &api.RawData{
			Params: paramsString,
			Data:   raw,
		}
		err = db.Create(rawData, dal.From("_raw_squadcast_incidents"))
		if err != nil {
			return errors.Default.Wrap(err, "failed to save raw incident data")
		}
	}

	return nil
}
