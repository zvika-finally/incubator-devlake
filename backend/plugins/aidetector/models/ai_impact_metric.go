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

// AIImpactMetric stores before/after comparison of team productivity metrics
// relative to AI tool adoption date. Uses 90-day baseline before adoption
// and 30-day current period after adoption for statistical significance.
type AIImpactMetric struct {
	Id              string     `gorm:"primaryKey;type:varchar(255)"`
	ProjectName     string     `gorm:"type:varchar(255);index"`
	AIAdoptionDate  *time.Time `gorm:"index"` // Date of first explicit AI signal detection

	// Baseline period metrics (90 days pre-adoption)
	BaselinePRThroughput float64 `gorm:"type:decimal(10,2)"` // PRs merged per week
	BaselineReviewTime   float64 `gorm:"type:decimal(10,2)"` // Hours from PR open to merge
	BaselineLeadTime     float64 `gorm:"type:decimal(10,2)"` // Hours from first commit to deploy

	// Current period metrics (30 days post-adoption)
	CurrentPRThroughput float64 `gorm:"type:decimal(10,2)"`
	CurrentReviewTime   float64 `gorm:"type:decimal(10,2)"`
	CurrentLeadTime     float64 `gorm:"type:decimal(10,2)"`

	// Calculated percentage changes (positive = improvement)
	// For throughput: higher is better, so positive change = improvement
	// For time metrics: lower is better, so the sign is inverted (faster = positive)
	PRThroughputChange float64 `gorm:"type:decimal(10,2)"`
	ReviewTimeChange   float64 `gorm:"type:decimal(10,2)"`
	LeadTimeChange     float64 `gorm:"type:decimal(10,2)"`

	CalculatedAt time.Time `gorm:"index"`
}

func (AIImpactMetric) TableName() string {
	return "ai_impact_metrics"
}
