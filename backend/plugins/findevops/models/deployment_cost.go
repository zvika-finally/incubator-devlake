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

package models

import (
	"time"
)

// DeploymentCost stores cost per deployment metrics for different time windows
type DeploymentCost struct {
	Id                string    `gorm:"primaryKey;type:varchar(255)"`
	ProjectName       string    `gorm:"type:varchar(255);index"`
	WindowDays        int       `gorm:"type:int;index"` // 7, 30, or 90 days
	PeriodStart       time.Time `gorm:"index"`
	PeriodEnd         time.Time `gorm:"index"`

	TotalCost           float64 `gorm:"type:decimal(14,2)"`
	DeploymentCount     int     `gorm:"type:int"`
	CostPerDeployment   float64 `gorm:"type:decimal(12,2)"`

	CalculatedAt time.Time `gorm:"index"`
}

func (DeploymentCost) TableName() string {
	return "deployment_costs"
}
