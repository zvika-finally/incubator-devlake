package models

import (
	"time"

	"github.com/apache/incubator-devlake/core/models/common"
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
)

// Incident model
type SquadcastIncident struct {
	common.NoPKModel `json:"-" mapstructure:"-"`
	ConnectionId     uint64     `json:"connection_id" gorm:"primaryKey"`
	IncidentId       string     `json:"incident_id" gorm:"primaryKey;type:varchar(255)"`
	Title            string     `json:"title"`
	Status           string     `json:"status"`
	Severity         string     `json:"severity"`
	Assignee         string     `json:"assignee"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	ResolvedAt       *time.Time `json:"resolved_at"`
	URL              string     `json:"url"`
	RawData          []byte     `json:"raw_data"`
}

func (SquadcastIncident) TableName() string {
	return "_tool_squadcast_incidents"
}

// Connection model
type SquadcastConnection struct {
	api.BaseConnection
	ApiKey string `json:"apiKey" gorm:"column:api_key"`
}

func (SquadcastConnection) TableName() string {
	return "_tool_squadcast_connections"
}
