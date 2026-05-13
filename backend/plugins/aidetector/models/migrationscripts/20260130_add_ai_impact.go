/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package migrationscripts

import (
	"time"

	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
)

var _ plugin.MigrationScript = (*addAIImpact)(nil)

type addAIImpact struct{}

func (script *addAIImpact) Up(basicRes context.BasicRes) errors.Error {
	db := basicRes.GetDal()

	err := db.AutoMigrate(&aiImpactMetric20260130{})
	if err != nil {
		return errors.Default.Wrap(err, "failed to create ai_impact_metrics table")
	}

	return nil
}

func (script *addAIImpact) Version() uint64 {
	return 20260130000001
}

func (script *addAIImpact) Name() string {
	return "aidetector: add AI impact metrics table"
}

// Model for migration
type aiImpactMetric20260130 struct {
	Id             string     `gorm:"primaryKey;type:varchar(255)"`
	ProjectName    string     `gorm:"type:varchar(255);index"`
	AIAdoptionDate *time.Time `gorm:"index"`

	BaselinePRThroughput float64 `gorm:"type:decimal(10,2)"`
	BaselineReviewTime   float64 `gorm:"type:decimal(10,2)"`
	BaselineLeadTime     float64 `gorm:"type:decimal(10,2)"`

	CurrentPRThroughput float64 `gorm:"type:decimal(10,2)"`
	CurrentReviewTime   float64 `gorm:"type:decimal(10,2)"`
	CurrentLeadTime     float64 `gorm:"type:decimal(10,2)"`

	PRThroughputChange float64 `gorm:"type:decimal(10,2)"`
	ReviewTimeChange   float64 `gorm:"type:decimal(10,2)"`
	LeadTimeChange     float64 `gorm:"type:decimal(10,2)"`

	CalculatedAt time.Time `gorm:"index"`
}

func (aiImpactMetric20260130) TableName() string {
	return "ai_impact_metrics"
}
