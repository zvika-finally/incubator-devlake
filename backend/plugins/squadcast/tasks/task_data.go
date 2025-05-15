package tasks

import "github.com/apache/incubator-devlake/plugins/squadcast/models"

// SquadcastOptions holds options for a Squadcast pipeline task
type SquadcastOptions struct {
	ConnectionId uint64 `json:"connectionId" mapstructure:"connectionId"`
}

// SquadcastTaskData holds the options and connection for use in subtasks
type SquadcastTaskData struct {
	Options    *SquadcastOptions
	Connection *models.SquadcastConnection
}
