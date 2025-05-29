package models

import (
	"time"

	"github.com/apache/incubator-devlake/core/models/common"
)

type ArgoCDProject struct {
	common.NoPKModel
	ConnectionId uint64     `json:"connectionId" gorm:"primaryKey"`
	ArgoCDId     string     `gorm:"primaryKey;type:varchar(255)" json:"argoCdId"`
	Name         string     `gorm:"type:varchar(255)" json:"name"`
	Description  string     `gorm:"type:varchar(1024)" json:"description"`
	CreatedDate  *time.Time `json:"createdAt"`
}

func (ArgoCDProject) TableName() string {
	return "_tool_argocd_projects"
}
