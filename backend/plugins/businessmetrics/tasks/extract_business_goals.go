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
	"strings"
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/domainlayer/ticket"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/businessmetrics/models"
)

var ExtractBusinessGoalsMeta = plugin.SubTaskMeta{
	Name:             "extractBusinessGoals",
	EntryPoint:       ExtractBusinessGoals,
	EnabledByDefault: true,
	Description:      "Extract business initiatives from Jira Epics based on labels",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

func ExtractBusinessGoals(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*BusinessMetricsTaskData)
	logger := taskCtx.GetLogger()

	logger.Info("Starting extractBusinessGoals for project: %s", data.Options.ProjectName)

	// Query all Epics for this project
	var epics []ticket.Issue
	clauses := []dal.Clause{
		dal.From(&ticket.Issue{}),
		dal.Join("LEFT JOIN board_issues bi ON bi.issue_id = issues.id"),
		dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'boards' AND pm.row_id = bi.board_id"),
		dal.Where("pm.project_name = ? AND issues.type = ?", data.Options.ProjectName, "Epic"),
	}

	err := db.All(&epics, clauses...)
	if err != nil {
		return errors.Default.Wrap(err, "failed to query epics")
	}

	logger.Info("Found %d Epics to process", len(epics))

	// Process each Epic
	for _, epic := range epics {
		initiative := &models.BusinessInitiative{
			Id:            epic.Id,
			Name:          epic.Title,
			JiraEpicKey:   epic.IssueKey,
			Status:        mapStatus(epic.Status),
			CreatedAt:     time.Now(),
		}

		// Get labels from the tool layer (Jira issue)
		var labels string
		_ = db.First(&labels,
			dal.Select("labels"),
			dal.From("_tool_jira_issues"),
			dal.Where("issue_key = ?", epic.IssueKey),
		)

		// Parse labels for investment category and other metadata
		if labels != "" {
			labelList := strings.Split(labels, ",")
			for _, label := range labelList {
				label = strings.TrimSpace(label)

				// Check for investment category label
				if strings.HasPrefix(label, data.Options.InvestmentLabelPrefix) {
					initiative.InvestmentCategory = strings.TrimPrefix(label, data.Options.InvestmentLabelPrefix)
				}

				// Check for stage label
				if strings.HasPrefix(label, data.Options.StageLabelPrefix) {
					initiative.DevelopmentStage = strings.TrimPrefix(label, data.Options.StageLabelPrefix)
				}

				// Check for goal type patterns
				if strings.HasPrefix(label, "goal:") {
					initiative.GoalType = strings.TrimPrefix(label, "goal:")
				}

				// Check for fiscal quarter patterns (e.g., "q1-2026", "2026-q1")
				if strings.Contains(strings.ToLower(label), "q1") ||
				   strings.Contains(strings.ToLower(label), "q2") ||
				   strings.Contains(strings.ToLower(label), "q3") ||
				   strings.Contains(strings.ToLower(label), "q4") {
					initiative.FiscalQuarter = label
				}
			}
		}

		// Save or update the initiative
		err = db.CreateOrUpdate(initiative)
		if err != nil {
			logger.Error(err, "failed to save initiative for epic %s", epic.IssueKey)
			continue
		}
	}

	logger.Info("Completed extractBusinessGoals: processed %d initiatives", len(epics))
	return nil
}

func mapStatus(jiraStatus string) string {
	statusLower := strings.ToLower(jiraStatus)
	switch {
	case strings.Contains(statusLower, "done") || strings.Contains(statusLower, "closed"):
		return "completed"
	case strings.Contains(statusLower, "progress") || strings.Contains(statusLower, "review"):
		return "active"
	case strings.Contains(statusLower, "cancel"):
		return "cancelled"
	default:
		return "planned"
	}
}
