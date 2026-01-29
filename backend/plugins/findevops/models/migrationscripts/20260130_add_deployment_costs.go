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

var _ plugin.MigrationScript = (*addDeploymentCosts)(nil)

type addDeploymentCosts struct{}

func (script *addDeploymentCosts) Up(basicRes context.BasicRes) errors.Error {
	db := basicRes.GetDal()

	err := db.AutoMigrate(&deploymentCost20260130{})
	if err != nil {
		return errors.Default.Wrap(err, "failed to create deployment_costs table")
	}

	return nil
}

func (script *addDeploymentCosts) Version() uint64 {
	return 20260130000003
}

func (script *addDeploymentCosts) Name() string {
	return "findevops: add deployment costs table"
}

// Migration model
type deploymentCost20260130 struct {
	Id                string    `gorm:"primaryKey;type:varchar(255)"`
	ProjectName       string    `gorm:"type:varchar(255);index"`
	WindowDays        int       `gorm:"type:int;index"`
	PeriodStart       time.Time `gorm:"index"`
	PeriodEnd         time.Time `gorm:"index"`

	TotalCost           float64 `gorm:"type:decimal(14,2)"`
	DeploymentCount     int     `gorm:"type:int"`
	CostPerDeployment   float64 `gorm:"type:decimal(12,2)"`

	CalculatedAt time.Time `gorm:"index"`
}

func (deploymentCost20260130) TableName() string {
	return "deployment_costs"
}
