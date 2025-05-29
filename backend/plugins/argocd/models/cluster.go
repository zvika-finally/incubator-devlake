package models

import (
	"time"

	"github.com/apache/incubator-devlake/core/models/common"
)

type ArgoCDCluster struct {
	common.NoPKModel
	ConnectionId  uint64     `json:"connectionId" gorm:"primaryKey"`
	ArgoCDId      string     `gorm:"primaryKey;type:varchar(255)" json:"argoCdId"`
	Name          string     `gorm:"type:varchar(255)" json:"name"`
	Server        string     `gorm:"type:varchar(255)" json:"server"`
	ServerVersion string     `gorm:"type:varchar(255)" json:"serverVersion"`
	Status        string     `gorm:"type:varchar(100)" json:"status"`
	CreatedDate   *time.Time `json:"createdAt"`
}

func (ArgoCDCluster) TableName() string {
	return "_tool_argocd_clusters"
}
