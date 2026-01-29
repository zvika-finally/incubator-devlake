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

package tasks

import (
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/businessmetrics/models"
)

var CalculateBusinessValueMeta = plugin.SubTaskMeta{
	Name:             "calculateBusinessValue",
	EntryPoint:       CalculateBusinessValue,
	EnabledByDefault: true,
	Description:      "Calculate business value score based on capability and revenue impact",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

// Revenue impact weights from eng-product-metrics
const (
	BaseScore                 = 50
	RevenueImpactDirect       = 30
	RevenueImpactEnabling     = 20
	RevenueImpactSupporting   = 10
	RevenueImpactCostCenter   = 0

	// Revenue-to-cost efficiency bonuses
	EfficiencyRatioExcellent  = 20 // ratio >= 5
	EfficiencyRatioGood       = 15 // ratio >= 2
	EfficiencyRatioFair       = 10 // ratio >= 1
	EfficiencyRatioPositive   = 5  // ratio > 0
)

func CalculateBusinessValue(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*BusinessMetricsTaskData)
	logger := taskCtx.GetLogger()

	logger.Info("Starting calculateBusinessValue for project: %s", data.Options.ProjectName)

	// Get all initiatives for this project
	var initiatives []models.BusinessInitiative
	clauses := []dal.Clause{
		dal.From(&models.BusinessInitiative{}),
		dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'issues' AND pm.row_id = business_initiatives.jira_epic_key"),
		dal.Where("pm.project_name = ?", data.Options.ProjectName),
	}

	err := db.All(&initiatives, clauses...)
	if err != nil {
		return errors.Default.Wrap(err, "failed to query business initiatives")
	}

	logger.Info("Calculating business value for %d initiatives", len(initiatives))

	for _, initiative := range initiatives {
		// Calculate business value score
		score := CalculateBusinessValueScore(
			initiative.RevenueImpact,
			initiative.RevenueToCostRatio,
		)

		// Update the initiative with the score
		initiative.BusinessValueScore = score
		initiative.UpdatedAt = time.Now()

		if err := db.Update(&initiative); err != nil {
			logger.Error(err, "failed to update business value score for initiative %s", initiative.Id)
			continue
		}

		logger.Info("Business value score for %s: %d (impact=%s, ratio=%.2f)",
			initiative.Name, score, initiative.RevenueImpact, initiative.RevenueToCostRatio)
	}

	logger.Info("Completed business value calculation for %d initiatives", len(initiatives))
	return nil
}

// CalculateBusinessValueScore calculates the business value score (0-100)
// Based on revenue impact weight and revenue-to-cost efficiency
// Exported for testing
func CalculateBusinessValueScore(revenueImpact string, revenueToCostRatio float64) int {
	score := BaseScore

	// Add revenue impact weight
	score += GetRevenueImpactWeight(revenueImpact)

	// Add efficiency bonus
	score += GetEfficiencyBonus(revenueToCostRatio)

	// Cap at 100
	if score > 100 {
		return 100
	}
	if score < 0 {
		return 0
	}

	return score
}

// GetRevenueImpactWeight returns the weight for a revenue impact type
// Exported for testing
func GetRevenueImpactWeight(revenueImpact string) int {
	switch revenueImpact {
	case models.RevenueImpactDirect:
		return RevenueImpactDirect
	case models.RevenueImpactEnabling:
		return RevenueImpactEnabling
	case models.RevenueImpactSupporting:
		return RevenueImpactSupporting
	case models.RevenueImpactCostCenter:
		return RevenueImpactCostCenter
	default:
		return 0
	}
}

// GetEfficiencyBonus returns the bonus based on revenue-to-cost ratio
// Exported for testing
func GetEfficiencyBonus(ratio float64) int {
	if ratio >= 5 {
		return EfficiencyRatioExcellent
	}
	if ratio >= 2 {
		return EfficiencyRatioGood
	}
	if ratio >= 1 {
		return EfficiencyRatioFair
	}
	if ratio > 0 {
		return EfficiencyRatioPositive
	}
	return 0
}
