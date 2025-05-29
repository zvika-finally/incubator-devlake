package models

import (
	"github.com/apache/incubator-devlake/core/models/common"
)

type ArgoCDScopeConfig struct {
	common.ScopeConfig `mapstructure:",squash"`
	Name               string `mapstructure:"name" json:"name" gorm:"type:varchar(255);index:idx_name_argocd,unique" validate:"required"`
	
	// Add ArgoCD-specific scope configuration fields here
	// For example:
	EnvironmentPattern  string `mapstructure:"environmentPattern" json:"environmentPattern" gorm:"type:text"`
	DeploymentPattern   string `mapstructure:"deploymentPattern" json:"deploymentPattern" gorm:"type:text"`
	ProductionPattern   string `mapstructure:"productionPattern" json:"productionPattern" gorm:"type:text"`
}

func (r *ArgoCDScopeConfig) SetConnectionId(connectionId uint64) {
	r.ConnectionId = connectionId
}

func (ArgoCDScopeConfig) TableName() string {
	return "_tool_argocd_scope_configs"
}

func (r ArgoCDScopeConfig) ScopeConfigTableName() string {
	return "_tool_argocd_scope_configs"
}

// Interface compliance check - remove if not needed