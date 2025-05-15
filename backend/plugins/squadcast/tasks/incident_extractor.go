package tasks

import (
	"encoding/json"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/squadcast/models"
)

var ExtractIncidentsMeta = plugin.SubTaskMeta{
	Name:             "Extract Squadcast Incidents",
	EntryPoint:       ExtractIncidents,
	EnabledByDefault: true,
	Description:      "Extract raw incident data into tool layer table squadcast_incidents",
	DomainTypes:      []string{"INCIDENT"},
}

func ExtractIncidents(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*SquadcastTaskData)
	extractor, err := api.NewApiExtractor(api.ApiExtractorArgs{
		RawDataSubTaskArgs: api.RawDataSubTaskArgs{
			Ctx:   taskCtx,
			Table: "_raw_squadcast_incidents",
			Params: map[string]interface{}{
				"connectionId": data.Connection.ID,
			},
		},
		Extract: func(row *api.RawData) ([]interface{}, errors.Error) {
			var apiIncident SquadcastIncidentAPIItem
			if err := json.Unmarshal(row.Data, &apiIncident); err != nil {
				return nil, errors.Default.Wrap(err, "failed to unmarshal raw incident")
			}

			incident := &models.SquadcastIncident{
				ConnectionId: data.Connection.ID,
				IncidentId:   apiIncident.ID,
				Title:        apiIncident.Title,
				Status:       apiIncident.Status,
				Severity:     apiIncident.Severity,
				Assignee:     apiIncident.Assignee,
				CreatedAt:    apiIncident.CreatedAt,
				UpdatedAt:    apiIncident.UpdatedAt,
				ResolvedAt:   apiIncident.ResolvedAt,
				URL:          apiIncident.URL,
				RawData:      row.Data,
			}
			return []interface{}{incident}, nil
		},
	})
	if err != nil {
		return err
	}
	return extractor.Execute()
}
