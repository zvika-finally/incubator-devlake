package models

import (
	"time"

	"github.com/apache/incubator-devlake/core/models/common"
	"github.com/apache/incubator-devlake/core/plugin"
)

type ArgoCDApplication struct {
	common.Scope `mapstructure:",squash"`
	ArgoCDId     string `gorm:"index;type:varchar(255)" json:"argoCdId" mapstructure:"argoCdId"`
	Name         string `gorm:"type:varchar(255)" json:"name" mapstructure:"name"`
	Project      string `gorm:"type:varchar(255)" json:"project" mapstructure:"project"`
	Namespace    string `gorm:"type:varchar(255)" json:"namespace" mapstructure:"namespace"`
	Cluster      string `gorm:"type:varchar(255)" json:"cluster" mapstructure:"cluster"`
	RepoURL      string `gorm:"type:varchar(512)" json:"repoURL" mapstructure:"repoURL"`
	Path         string `gorm:"type:varchar(255)" json:"path" mapstructure:"path"`
	TargetRev    string `gorm:"type:varchar(255)" json:"targetRevision" mapstructure:"targetRevision"`
	Health       string `gorm:"type:varchar(100)" json:"health" mapstructure:"health"`
	SyncStatus   string `gorm:"type:varchar(100)" json:"syncStatus" mapstructure:"syncStatus"`
	CreatedDate  *time.Time `json:"createdAt" mapstructure:"createdAt"`
	UpdatedDate  *time.Time `json:"updatedAt" mapstructure:"updatedAt"`
}

func (app ArgoCDApplication) ScopeId() string {
	return app.ArgoCDId
}

func (app ArgoCDApplication) ScopeName() string {
	return app.Name
}

func (app ArgoCDApplication) ScopeFullName() string {
	return app.Name
}

func (app ArgoCDApplication) ScopeParams() interface{} {
	return &ArgoCDScopeParams{
		ConnectionId: app.ConnectionId,
		ArgoCDId:     app.ArgoCDId,
	}
}

func (ArgoCDApplication) TableName() string {
	return "_tool_argocd_applications"
}

type ArgoCDScopeParams struct {
	ConnectionId uint64 `json:"connectionId"`
	ArgoCDId     string `json:"argoCdId"`
}

var _ plugin.ToolLayerScope = (*ArgoCDApplication)(nil)
