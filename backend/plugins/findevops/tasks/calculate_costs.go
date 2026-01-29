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
	"github.com/apache/incubator-devlake/core/log"
	"github.com/apache/incubator-devlake/core/models/domainlayer/ticket"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/findevops/models"
)

var CalculateCostsMeta = plugin.SubTaskMeta{
	Name:             "calculateCosts",
	EntryPoint:       CalculateCosts,
	EnabledByDefault: true,
	Description:      "Calculate development costs from worklogs and story points",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

func CalculateCosts(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*FinDevOpsTaskData)
	logger := taskCtx.GetLogger()

	logger.Info("Starting calculateCosts for project: %s", data.Options.ProjectName)

	// Load developer hourly rates
	rateMap := loadHourlyRates(db, data.Options.DefaultHourlyRate, logger)

	// Get all issues for this project
	var issues []ticket.Issue
	clauses := []dal.Clause{
		dal.From(&ticket.Issue{}),
		dal.Join("LEFT JOIN board_issues bi ON bi.issue_id = issues.id"),
		dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'boards' AND pm.row_id = bi.board_id"),
		dal.Where("pm.project_name = ?", data.Options.ProjectName),
	}

	if err := db.All(&issues, clauses...); err != nil {
		return errors.Default.Wrap(err, "failed to query issues")
	}

	logger.Info("Calculating costs for %d issues", len(issues))

	for _, issue := range issues {
		// Get hours worked (from worklogs or estimate from story points)
		hoursWorked := getHoursWorked(issue)
		if hoursWorked == 0 {
			continue // Skip issues with no time data
		}

		// Get hourly rate for assignee
		hourlyRate := getHourlyRate(issue.AssigneeId, rateMap, data.Options.DefaultHourlyRate)

		// Calculate fiscal month from issue resolution or update date
		fiscalMonth := getFiscalMonth(issue)

		// Get labels from tool layer
		var labels string
		_ = db.First(&labels,
			dal.Select("labels"),
			dal.From("_tool_jira_issues"),
			dal.Where("issue_key = ?", issue.IssueKey),
		)

		// Create cost allocation record
		allocation := &models.CostAllocation{
			Id:            fmt.Sprintf("%s:%s", issue.Id, fiscalMonth),
			IssueId:       issue.Id,
			FiscalMonth:   fiscalMonth,
			DeveloperId:   issue.AssigneeId,
			HoursWorked:   hoursWorked,
			HourlyRate:    hourlyRate,
			DeveloperCost: hoursWorked * hourlyRate,
			TotalCost:     hoursWorked * hourlyRate,
			IssueType:     issue.Type,
			IssueLabels:   labels,
			CalculatedAt:  time.Now(),
			CreatedAt:     time.Now(),
		}

		if err := db.CreateOrUpdate(allocation); err != nil {
			logger.Error(err, "failed to save cost allocation for issue %s", issue.IssueKey)
		}
	}

	logger.Info("Completed calculateCosts")
	return nil
}

func loadHourlyRates(db dal.Dal, defaultRate float64, logger log.Logger) map[string]float64 {
	rateMap := make(map[string]float64)

	var rates []models.DeveloperHourlyRate
	if err := db.All(&rates, dal.From(&models.DeveloperHourlyRate{})); err != nil {
		logger.Warn(err, "Could not load hourly rates, using default")
		return rateMap
	}

	for _, rate := range rates {
		rateMap[rate.DeveloperId] = rate.HourlyRate
	}

	return rateMap
}

func getHourlyRate(developerId string, rateMap map[string]float64, defaultRate float64) float64 {
	if rate, exists := rateMap[developerId]; exists {
		return rate
	}
	return defaultRate
}

func getHoursWorked(issue ticket.Issue) float64 {
	// First try actual time spent (from worklogs)
	if issue.TimeSpentMinutes != nil && *issue.TimeSpentMinutes > 0 {
		return float64(*issue.TimeSpentMinutes) / 60.0
	}

	// Fall back to original estimate
	if issue.OriginalEstimateMinutes != nil && *issue.OriginalEstimateMinutes > 0 {
		return float64(*issue.OriginalEstimateMinutes) / 60.0
	}

	// Fall back to story points (estimate 4 hours per story point)
	if issue.StoryPoint != nil && *issue.StoryPoint > 0 {
		return *issue.StoryPoint * 4.0
	}

	return 0
}

func getFiscalMonth(issue ticket.Issue) string {
	// Use resolution date if available, otherwise use updated date
	var refDate time.Time
	if issue.ResolutionDate != nil {
		refDate = *issue.ResolutionDate
	} else if issue.UpdatedDate != nil {
		refDate = *issue.UpdatedDate
	} else {
		refDate = time.Now()
	}

	return refDate.Format("2006-01")
}
