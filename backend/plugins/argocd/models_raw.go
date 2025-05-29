package models

import (
	"github.com/apache/incubator-devlake/core/models/common"
)

// RawArgoCDApplication stores the raw JSON for an ArgoCD application
// Table name: _raw_argocd_applications
type RawArgoCDApplication struct {
	common.Model
	ConnectionId uint64 `gorm:"index;not null"`
	RawData      []byte `gorm:"type:json"`
}

func (RawArgoCDApplication) TableName() string { return "_raw_argocd_applications" }

// RawArgoCDProject stores the raw JSON for an ArgoCD project
// Table name: _raw_argocd_projects
type RawArgoCDProject struct {
	common.Model
	ConnectionId uint64 `gorm:"index;not null"`
	RawData      []byte `gorm:"type:json"`
}

func (RawArgoCDProject) TableName() string { return "_raw_argocd_projects" }

// RawArgoCDCluster stores the raw JSON for an ArgoCD cluster
// Table name: _raw_argocd_clusters
type RawArgoCDCluster struct {
	common.Model
	ConnectionId uint64 `gorm:"index;not null"`
	RawData      []byte `gorm:"type:json"`
}

func (RawArgoCDCluster) TableName() string { return "_raw_argocd_clusters" }
