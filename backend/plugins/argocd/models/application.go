package models

import (
	"time"

	"github.com/apache/incubator-devlake/core/models/common"
)

type ArgoCDApplication struct {
	common.NoPKModel
	Id          string `gorm:"primaryKey;type:varchar(255)"`
	Name        string `gorm:"type:varchar(255)"`
	Project     string `gorm:"type:varchar(255)"`
	Cluster     string `gorm:"type:varchar(255)"`
	CreatedDate *time.Time
	// Add other fields as needed
}

func (ArgoCDApplication) TableName() string {
	return "_tool_argocd_applications"
}
