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
	"github.com/apache/incubator-devlake/plugins/capacityplanner/models"
)

var CalculateThroughputMeta = plugin.SubTaskMeta{
	Name:             "calculateThroughput",
	EntryPoint:       CalculateThroughput,
	EnabledByDefault: true,
	Description:      "Calculate team throughput using Kanban metrics (weekly/monthly completed issues)",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

func CalculateThroughput(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*CapacityPlannerTaskData)
	logger := taskCtx.GetLogger()

	logger.Info("Starting calculateThroughput for project: %s (Kanban metrics)", data.Options.ProjectName)

	// Get time period duration (default: 1 week)
	periodDays := 7
	if data.Options.SprintDurationWeeks > 0 {
		periodDays = data.Options.SprintDurationWeeks * 7
	}

	// Calculate how many periods to analyze (default: 6 periods)
	periodCount := data.Options.VelocitySprintCount
	if periodCount == 0 {
		periodCount = 6
	}

	logger.Info("Analyzing %d periods of %d days each", periodCount, periodDays)

	// Calculate throughput for each time period
	now := time.Now()
	for i := 0; i < periodCount; i++ {
		periodEnd := now.AddDate(0, 0, -i*periodDays)
		periodStart := periodEnd.AddDate(0, 0, -periodDays)

		throughput, err := calculatePeriodThroughput(db, data.Options.ProjectName, periodStart, periodEnd, i+1, logger)
		if err != nil {
			logger.Error(err, "failed to calculate throughput for period %d", i+1)
			continue
		}

		if err := db.CreateOrUpdate(throughput); err != nil {
			logger.Error(err, "failed to save throughput for period %d", i+1)
		}
	}

	logger.Info("Completed calculateThroughput")
	return nil
}

func calculatePeriodThroughput(db dal.Dal, projectName string, startDate, endDate time.Time, periodNum int, logger log.Logger) (*models.TeamVelocity, error) {
	// Create a synthetic "sprint" ID for this time period
	year, week := endDate.ISOWeek()
	periodId := fmt.Sprintf("%s-W%d-%02d-%d", projectName, year, week, periodNum)
	periodName := fmt.Sprintf("Week %d-%02d", year, week)

	velocity := &models.TeamVelocity{
		Id:              periodId,
		ProjectName:     projectName,
		SprintId:        periodId,
		SprintName:      periodName,
		FiscalWeek:      fmt.Sprintf("%d-W%02d", year, week),
		SprintStartDate: &startDate,
		SprintEndDate:   &endDate,
		CalculatedAt:    time.Now(),
	}

	// Count completed issues in this time period
	// Using resolution_date to identify when issues were completed
	// Issues are mapped to projects through boards, not directly
	var issueCount int64
	err := db.First(&issueCount,
		dal.Select("COUNT(DISTINCT issues.id)"),
		dal.From(&ticket.Issue{}),
		dal.Join("LEFT JOIN board_issues bi ON bi.issue_id = issues.id"),
		dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'boards' AND pm.row_id = bi.board_id"),
		dal.Where("pm.project_name = ? AND issues.resolution_date >= ? AND issues.resolution_date < ? AND issues.status = 'DONE'",
			projectName, startDate, endDate),
	)
	if err == nil {
		velocity.IssuesCompleted = int(issueCount)
	} else {
		logger.Warn(err, "Could not query issues by resolution_date, trying alternative approach")

		// Alternative: use updated_date if resolution_date is not available
		err = db.First(&issueCount,
			dal.Select("COUNT(DISTINCT issues.id)"),
			dal.From(&ticket.Issue{}),
			dal.Join("LEFT JOIN board_issues bi ON bi.issue_id = issues.id"),
			dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'boards' AND pm.row_id = bi.board_id"),
			dal.Where("pm.project_name = ? AND issues.updated_date >= ? AND issues.updated_date < ? AND issues.status = 'DONE'",
				projectName, startDate, endDate),
		)
		if err == nil {
			velocity.IssuesCompleted = int(issueCount)
		}
	}

	// Count story points completed (if available)
	var storyPoints int64
	err = db.First(&storyPoints,
		dal.Select("COALESCE(SUM(issues.story_point), 0)"),
		dal.From(&ticket.Issue{}),
		dal.Join("LEFT JOIN board_issues bi ON bi.issue_id = issues.id"),
		dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'boards' AND pm.row_id = bi.board_id"),
		dal.Where("pm.project_name = ? AND issues.resolution_date >= ? AND issues.resolution_date < ? AND issues.status = 'DONE'",
			projectName, startDate, endDate),
	)
	if err == nil {
		velocity.StoryPointsCompleted = int(storyPoints)
	}

	// Calculate average cycle time for issues completed in this period
	var avgCycleTimeHours float64
	err = db.First(&avgCycleTimeHours,
		dal.Select("AVG(TIMESTAMPDIFF(HOUR, issues.created_date, issues.resolution_date))"),
		dal.From(&ticket.Issue{}),
		dal.Join("LEFT JOIN board_issues bi ON bi.issue_id = issues.id"),
		dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'boards' AND pm.row_id = bi.board_id"),
		dal.Where("pm.project_name = ? AND issues.resolution_date >= ? AND issues.resolution_date < ? AND issues.resolution_date IS NOT NULL",
			projectName, startDate, endDate),
	)
	if err == nil {
		velocity.AvgCycleTimeHours = avgCycleTimeHours
	}

	// Calculate average lead time
	var avgLeadTimeHours float64
	err = db.First(&avgLeadTimeHours,
		dal.Select("AVG(TIMESTAMPDIFF(HOUR, issues.created_date, issues.resolution_date))"),
		dal.From(&ticket.Issue{}),
		dal.Join("LEFT JOIN board_issues bi ON bi.issue_id = issues.id"),
		dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'boards' AND pm.row_id = bi.board_id"),
		dal.Where("pm.project_name = ? AND issues.resolution_date >= ? AND issues.resolution_date < ? AND issues.resolution_date IS NOT NULL",
			projectName, startDate, endDate),
	)
	if err == nil {
		velocity.AvgLeadTimeHours = avgLeadTimeHours
	}

	// Count PRs merged in this period
	var prCount int64
	err = db.First(&prCount,
		dal.Select("COUNT(*)"),
		dal.From("pull_requests pr"),
		dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'repos' AND pm.row_id = pr.base_repo_id"),
		dal.Where("pm.project_name = ? AND pr.merged_date >= ? AND pr.merged_date < ? AND pr.status = 'MERGED'",
			projectName, startDate, endDate),
	)
	if err == nil {
		velocity.PRsMerged = int(prCount)
	}

	// Count commits in this period
	var commitCount int64
	err = db.First(&commitCount,
		dal.Select("COUNT(*)"),
		dal.From("commits c"),
		dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'repos' AND pm.row_id = c.repo_id"),
		dal.Where("pm.project_name = ? AND c.authored_date >= ? AND c.authored_date < ?",
			projectName, startDate, endDate),
	)
	if err == nil {
		velocity.CommitCount = int(commitCount)
	}

	logger.Info("Period %s (%s to %s): %d issues, %d story points, %.1f hrs avg cycle time, %.1f hrs avg lead time",
		periodName,
		startDate.Format("2006-01-02"),
		endDate.Format("2006-01-02"),
		velocity.IssuesCompleted,
		velocity.StoryPointsCompleted,
		velocity.AvgCycleTimeHours,
		velocity.AvgLeadTimeHours,
	)

	return velocity, nil
}
