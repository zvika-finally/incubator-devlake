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

	processedCount := 0

	// Query Epics from the raw Jira data (_tool_jira_issues) where original type = 'Epic'
	// DevLake maps Epic -> REQUIREMENT in the domain layer, so we need to check the tool layer
	var jiraEpics []struct {
		IssueKey  string
		Summary   string
		Status    string
		Labels    string
		CreatedAt *time.Time
	}

	err := db.All(&jiraEpics,
		dal.Select("ji.issue_key, ji.summary, ji.std_status as status, ji.labels, ji.created_date as created_at"),
		dal.From("_tool_jira_issues ji"),
		dal.Join("LEFT JOIN _tool_jira_board_issues jbi ON jbi.issue_id = ji.issue_id AND jbi.connection_id = ji.connection_id"),
		dal.Join("LEFT JOIN _tool_jira_boards jb ON jb.board_id = jbi.board_id AND jb.connection_id = jbi.connection_id"),
		dal.Join("LEFT JOIN project_mapping pm ON pm.`table` = 'boards' AND pm.row_id = CONCAT('jira:JiraBoard:', ji.connection_id, ':', jb.board_id)"),
		dal.Where("pm.project_name = ? AND ji.type = ?", data.Options.ProjectName, "Epic"),
	)
	if err != nil {
		logger.Error(err, "failed to query epics from tool layer")
	} else {
		logger.Info("Found %d Epics from Jira tool layer", len(jiraEpics))

		for _, epic := range jiraEpics {
			initiative := &models.BusinessInitiative{
				Id:          "epic:" + epic.IssueKey,
				Name:        epic.Summary,
				JiraEpicKey: epic.IssueKey,
				Status:      mapStatus(epic.Status),
				CreatedAt:   time.Now(),
			}

			if epic.Labels != "" {
				parseLabelsIntoInitiative(initiative, epic.Labels, data)
			}

			if err := db.CreateOrUpdate(initiative); err != nil {
				logger.Error(err, "failed to save initiative for epic %s", epic.IssueKey)
				continue
			}
			processedCount++
		}
	}

	// Also extract unique epic_key references from issues for Epics that weren't directly synced
	// but are referenced by other issues
	var epicRefs []struct {
		EpicKey    string
		IssueCount int
	}
	err = db.All(&epicRefs,
		dal.Select("issues.epic_key, COUNT(*) as issue_count"),
		dal.From(&ticket.Issue{}),
		dal.Join("LEFT JOIN board_issues bi ON bi.issue_id = issues.id"),
		dal.Join("LEFT JOIN project_mapping pm ON pm.`table` = 'boards' AND pm.row_id = bi.board_id"),
		dal.Where("pm.project_name = ? AND issues.epic_key IS NOT NULL AND issues.epic_key != ''", data.Options.ProjectName),
		dal.Groupby("issues.epic_key"),
	)
	if err != nil {
		logger.Error(err, "failed to query epic references")
	} else {
		logger.Info("Found %d unique epic_key references from issues", len(epicRefs))

		for _, ref := range epicRefs {
			// Check if we already have this initiative from the direct Epic query
			existingId := "epic:" + ref.EpicKey
			var existing models.BusinessInitiative
			findErr := db.First(&existing, dal.Where("id = ?", existingId))
			if findErr == nil {
				// Already exists, skip
				continue
			}

			initiative := &models.BusinessInitiative{
				Id:          existingId,
				Name:        ref.EpicKey, // Use key as name since we don't have the title
				JiraEpicKey: ref.EpicKey,
				Status:      "active", // Default to active since issues reference it
				CreatedAt:   time.Now(),
			}

			// Try to get details from _tool_jira_issues if the epic exists there
			var epicDetails struct {
				Summary string
				Status  string
				Labels  string
			}
			detailErr := db.First(&epicDetails,
				dal.Select("summary, std_status as status, labels"),
				dal.From("_tool_jira_issues"),
				dal.Where("issue_key = ?", ref.EpicKey),
			)
			if detailErr == nil && epicDetails.Summary != "" {
				initiative.Name = epicDetails.Summary
				initiative.Status = mapStatus(epicDetails.Status)
				if epicDetails.Labels != "" {
					parseLabelsIntoInitiative(initiative, epicDetails.Labels, data)
				}
			}

			if err := db.CreateOrUpdate(initiative); err != nil {
				logger.Error(err, "failed to save initiative for epic reference %s", ref.EpicKey)
				continue
			}
			processedCount++
		}
	}

	logger.Info("Completed extractBusinessGoals: processed %d initiatives", processedCount)
	return nil
}

func parseLabelsIntoInitiative(initiative *models.BusinessInitiative, labels string, data *BusinessMetricsTaskData) {
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
