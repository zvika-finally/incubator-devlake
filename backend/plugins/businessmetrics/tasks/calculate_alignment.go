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
	"github.com/apache/incubator-devlake/plugins/businessmetrics/models"
)

var CalculateAlignmentMeta = plugin.SubTaskMeta{
	Name:             "calculateAlignment",
	EntryPoint:       CalculateAlignment,
	EnabledByDefault: true,
	Description:      "Link issues, PRs, and commits to business initiatives via Epic relationships",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET, plugin.DOMAIN_TYPE_CODE},
}

func CalculateAlignment(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*BusinessMetricsTaskData)
	logger := taskCtx.GetLogger()

	logger.Info("Starting calculateAlignment for project: %s", data.Options.ProjectName)

	// Get all initiatives for this project
	var initiatives []models.BusinessInitiative
	err := db.All(&initiatives, dal.From(&models.BusinessInitiative{}))
	if err != nil {
		return errors.Default.Wrap(err, "failed to query initiatives")
	}

	logger.Info("Processing %d initiatives", len(initiatives))

	// For each initiative, find all related issues (child issues of the Epic)
	for _, initiative := range initiatives {
		// Find issues that belong to this Epic
		var issues []ticket.Issue
		clauses := []dal.Clause{
			dal.From(&ticket.Issue{}),
			dal.Where("epic_key = ?", initiative.JiraEpicKey),
		}

		err := db.All(&issues, clauses...)
		if err != nil {
			logger.Error(err, "failed to query issues for epic %s", initiative.JiraEpicKey)
			continue
		}

		logger.Info("Found %d issues for epic %s", len(issues), initiative.JiraEpicKey)

		// Create work allocations for each issue
		for _, issue := range issues {
			var storyPoints int
			var estimatedHours, actualHours float64
			if issue.StoryPoint != nil {
				storyPoints = int(*issue.StoryPoint)
			}
			if issue.OriginalEstimateMinutes != nil {
				estimatedHours = float64(*issue.OriginalEstimateMinutes) / 60.0
			}
			if issue.TimeSpentMinutes != nil {
				actualHours = float64(*issue.TimeSpentMinutes) / 60.0
			}

			allocation := &models.WorkAllocation{
				Id:             fmt.Sprintf("%s:%s", initiative.Id, issue.Id),
				InitiativeId:   initiative.Id,
				EntityType:     "issue",
				EntityId:       issue.Id,
				DeveloperId:    issue.AssigneeId,
				StoryPoints:    storyPoints,
				EstimatedHours: estimatedHours,
				ActualHours:    actualHours,
				CreatedAt:      time.Now(),
			}

			err = db.CreateOrUpdate(allocation)
			if err != nil {
				logger.Error(err, "failed to save allocation for issue %s", issue.IssueKey)
				continue
			}
		}

		// Also link PRs that are connected to these issues via pull_request_issues table
		linkPRsToInitiative(db, logger, initiative, issues)
	}

	logger.Info("Completed calculateAlignment")
	return nil
}

func linkPRsToInitiative(db dal.Dal, logger log.Logger, initiative models.BusinessInitiative, issues []ticket.Issue) {
	if len(issues) == 0 {
		return
	}

	// Collect issue IDs
	issueIds := make([]string, len(issues))
	for i, issue := range issues {
		issueIds[i] = issue.Id
	}

	// Query PRs linked to these issues
	rows, err := db.Cursor(
		dal.Select("DISTINCT pull_request_id"),
		dal.From("pull_request_issues"),
		dal.Where("issue_id IN ?", issueIds),
	)
	if err != nil {
		logger.Error(err, "failed to query PRs for initiative %s", initiative.Id)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var prId string
		if err := rows.Scan(&prId); err != nil {
			continue
		}

		allocation := &models.WorkAllocation{
			Id:           fmt.Sprintf("%s:pr:%s", initiative.Id, prId),
			InitiativeId: initiative.Id,
			EntityType:   "pull_request",
			EntityId:     prId,
			CreatedAt:    time.Now(),
		}

		if err := db.CreateOrUpdate(allocation); err != nil {
			logger.Error(err, "failed to save PR allocation %s", prId)
		}
	}
}
