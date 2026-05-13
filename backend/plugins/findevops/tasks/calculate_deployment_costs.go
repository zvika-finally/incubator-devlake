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
	"fmt"
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/findevops/models"
)

var CalculateDeploymentCostsMeta = plugin.SubTaskMeta{
	Name:             "calculateDeploymentCosts",
	EntryPoint:       CalculateDeploymentCosts,
	EnabledByDefault: true,
	Description:      "Calculate cost per deployment for different time windows (7, 30, 90 days)",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CICD},
}

// CostTimeWindows lists the day-window sizes (7, 30, 90) used when
// computing rolling deployment costs.
var CostTimeWindows = []int{7, 30, 90}

func CalculateDeploymentCosts(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*FinDevOpsTaskData)
	logger := taskCtx.GetLogger()

	logger.Info("Starting calculateDeploymentCosts for project: %s", data.Options.ProjectName)

	now := time.Now()

	for _, windowDays := range CostTimeWindows {
		periodEnd := now
		periodStart := now.AddDate(0, 0, -windowDays)

		// Calculate total cost in window from cost_allocations table
		totalCost, err := calculateTotalCostInWindow(db, data.Options.ProjectName, periodStart, periodEnd)
		if err != nil {
			logger.Error(err, "failed to calculate total cost for %d-day window", windowDays)
			continue
		}

		// Count deployments in window
		deploymentCount, err := countDeploymentsInWindow(db, data.Options.ProjectName, periodStart, periodEnd)
		if err != nil {
			logger.Error(err, "failed to count deployments for %d-day window", windowDays)
			continue
		}

		// Calculate cost per deployment
		costPerDeployment := 0.0
		if deploymentCount > 0 {
			costPerDeployment = totalCost / float64(deploymentCount)
		}

		// Create/update deployment cost record
		deploymentCost := models.DeploymentCost{
			Id:          fmt.Sprintf("%s:%d:%s", data.Options.ProjectName, windowDays, periodEnd.Format("20060102")),
			ProjectName: data.Options.ProjectName,
			WindowDays:  windowDays,
			PeriodStart: periodStart,
			PeriodEnd:   periodEnd,

			TotalCost:         totalCost,
			DeploymentCount:   deploymentCount,
			CostPerDeployment: costPerDeployment,

			CalculatedAt: now,
		}

		if err := db.CreateOrUpdate(&deploymentCost); err != nil {
			logger.Error(err, "failed to save deployment cost for %d-day window", windowDays)
			continue
		}

		logger.Info("Deployment cost for %d-day window: $%.2f total, %d deploys, $%.2f per deploy",
			windowDays, totalCost, deploymentCount, costPerDeployment)
	}

	logger.Info("Completed deployment cost calculation for project %s", data.Options.ProjectName)
	return nil
}

// calculateTotalCostInWindow sums up all costs in the time window
func calculateTotalCostInWindow(db dal.Dal, projectName string, start, end time.Time) (float64, errors.Error) {
	var totalCost float64

	clauses := []dal.Clause{
		dal.Select("COALESCE(SUM(cost_allocations.total_cost), 0) as total"),
		dal.From(&models.CostAllocation{}),
		dal.Join("LEFT JOIN issues ON issues.id = cost_allocations.issue_id"),
		dal.Join("LEFT JOIN board_issues bi ON bi.issue_id = issues.id"),
		dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'boards' AND pm.row_id = bi.board_id"),
		dal.Where("pm.project_name = ? AND cost_allocations.calculated_at >= ? AND cost_allocations.calculated_at < ?",
			projectName, start, end),
	}

	err := db.First(&totalCost, clauses...)
	if err != nil && !db.IsErrorNotFound(err) {
		return 0, errors.Default.Wrap(err, "failed to sum costs")
	}

	return totalCost, nil
}

// countDeploymentsInWindow counts successful deployments in the time window
func countDeploymentsInWindow(db dal.Dal, projectName string, start, end time.Time) (int, errors.Error) {
	clauses := []dal.Clause{
		dal.From("cicd_deployment_commits"),
		dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'cicd_scopes' AND pm.row_id = cicd_deployment_commits.cicd_scope_id"),
		dal.Where("pm.project_name = ? AND cicd_deployment_commits.finished_date >= ? AND cicd_deployment_commits.finished_date < ? AND cicd_deployment_commits.result = ?",
			projectName, start, end, "SUCCESS"),
	}

	count, err := db.Count(clauses...)
	if err != nil {
		return 0, errors.Default.Wrap(err, "failed to count deployments")
	}

	return int(count), nil
}

// CalculateCostPerDeployment is a helper function for testing
// Exported for testing
func CalculateCostPerDeployment(totalCost float64, deploymentCount int) float64 {
	if deploymentCount <= 0 {
		return 0
	}
	return totalCost / float64(deploymentCount)
}
