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

var CalculateVelocityMeta = plugin.SubTaskMeta{
	Name:             "calculateVelocity",
	EntryPoint:       CalculateVelocity,
	EnabledByDefault: true,
	Description:      "Calculate team velocity from completed sprints",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

func CalculateVelocity(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*CapacityPlannerTaskData)
	logger := taskCtx.GetLogger()

	logger.Info("Starting calculateVelocity for project: %s", data.Options.ProjectName)

	// Get completed sprints for this project
	var sprints []ticket.Sprint
	clauses := []dal.Clause{
		dal.From(&ticket.Sprint{}),
		dal.Join("LEFT JOIN board_sprints bs ON bs.sprint_id = sprints.id"),
		dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'boards' AND pm.row_id = bs.board_id"),
		dal.Where("pm.project_name = ? AND sprints.status = ?", data.Options.ProjectName, "closed"),
		dal.Orderby("sprints.completed_date DESC"),
		dal.Limit(data.Options.VelocitySprintCount),
	}

	if err := db.All(&sprints, clauses...); err != nil {
		return errors.Default.Wrap(err, "failed to query sprints")
	}

	logger.Info("Found %d completed sprints", len(sprints))

	for _, sprint := range sprints {
		velocity, err := calculateSprintVelocity(db, data.Options.ProjectName, sprint, logger)
		if err != nil {
			logger.Error(err, "failed to calculate velocity for sprint %s", sprint.Name)
			continue
		}

		if err := db.CreateOrUpdate(velocity); err != nil {
			logger.Error(err, "failed to save velocity for sprint %s", sprint.Name)
		}
	}

	logger.Info("Completed calculateVelocity")
	return nil
}

func calculateSprintVelocity(db dal.Dal, projectName string, sprint ticket.Sprint, logger log.Logger) (*models.TeamVelocity, error) {
	velocity := &models.TeamVelocity{
		Id:              fmt.Sprintf("%s:%s", projectName, sprint.Id),
		ProjectName:     projectName,
		SprintId:        sprint.Id,
		SprintName:      sprint.Name,
		SprintStartDate: sprint.StartedDate,
		SprintEndDate:   sprint.CompletedDate,
		CalculatedAt:    time.Now(),
	}

	// Calculate fiscal week
	if sprint.StartedDate != nil {
		year, week := sprint.StartedDate.ISOWeek()
		velocity.FiscalWeek = fmt.Sprintf("%d-W%02d", year, week)
	}

	// Count completed story points in this sprint
	var storyPoints int64
	err := db.First(&storyPoints,
		dal.Select("COALESCE(SUM(story_point), 0)"),
		dal.From("sprint_issues si"),
		dal.Join("LEFT JOIN issues i ON i.id = si.issue_id"),
		dal.Where("si.sprint_id = ? AND i.status = ?", sprint.Id, "Done"),
	)
	if err == nil {
		velocity.StoryPointsCompleted = int(storyPoints)
	}

	// Count completed issues
	var issueCount int64
	err = db.First(&issueCount,
		dal.Select("COUNT(*)"),
		dal.From("sprint_issues si"),
		dal.Join("LEFT JOIN issues i ON i.id = si.issue_id"),
		dal.Where("si.sprint_id = ? AND i.status = ?", sprint.Id, "Done"),
	)
	if err == nil {
		velocity.IssuesCompleted = int(issueCount)
	}

	// Calculate average cycle time for issues in this sprint
	var avgCycleTime float64
	err = db.First(&avgCycleTime,
		dal.Select("AVG((UNIX_TIMESTAMP(i.resolution_date) - UNIX_TIMESTAMP(i.created_date)) / 3600)"),
		dal.From("sprint_issues si"),
		dal.Join("LEFT JOIN issues i ON i.id = si.issue_id"),
		dal.Where("si.sprint_id = ? AND i.resolution_date IS NOT NULL", sprint.Id),
	)
	if err == nil {
		velocity.AvgCycleTimeHours = avgCycleTime
	}

	logger.Info("Sprint %s: %d story points, %d issues, %.1f hrs avg cycle time",
		sprint.Name, velocity.StoryPointsCompleted, velocity.IssuesCompleted, velocity.AvgCycleTimeHours)

	return velocity, nil
}
