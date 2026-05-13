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
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/log"
	"github.com/apache/incubator-devlake/core/models/domainlayer/ticket"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/capacityplanner/models"
)

var CalculateFlowEfficiencyMeta = plugin.SubTaskMeta{
	Name:             "calculateFlowEfficiency",
	EntryPoint:       CalculateFlowEfficiency,
	EnabledByDefault: true,
	Description:      "Calculate flow efficiency from issue status transitions",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

// Active statuses where work is being done (case-insensitive matching)
var activeStatuses = map[string]bool{
	"in progress":    true,
	"in development": true,
	"developing":     true,
	"coding":         true,
	"implementing":   true,
	"in review":      true,
	"reviewing":      true,
	"testing":        true,
	"in qa":          true,
	"in test":        true,
}

// statusTransition represents a single status change
type statusTransition struct {
	FromValue string
	ToValue   string
	ChangedAt time.Time
}

func CalculateFlowEfficiency(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*CapacityPlannerTaskData)
	logger := taskCtx.GetLogger()

	logger.Info("Starting calculateFlowEfficiency for project: %s", data.Options.ProjectName)

	// Get all completed issues for this project
	var issues []ticket.Issue
	err := db.All(&issues,
		dal.From(&ticket.Issue{}),
		dal.Join("LEFT JOIN board_issues bi ON bi.issue_id = issues.id"),
		dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'boards' AND pm.row_id = bi.board_id"),
		dal.Where("pm.project_name = ? AND issues.status = 'DONE'", data.Options.ProjectName),
	)
	if err != nil {
		return errors.Default.Wrap(err, "failed to query issues")
	}

	logger.Info("Calculating flow efficiency for %d completed issues", len(issues))

	var allMetrics []models.IssueFlowMetric

	for _, issue := range issues {
		metric := calculateIssueFlowMetric(db, data.Options.ProjectName, issue, logger)
		if metric != nil {
			if err := db.CreateOrUpdate(metric); err != nil {
				logger.Error(err, "failed to save flow metric for issue %s", issue.IssueKey)
			}
			allMetrics = append(allMetrics, *metric)
		}
	}

	// Generate project flow summary
	if len(allMetrics) > 0 {
		generateFlowSummary(db, data.Options.ProjectName, allMetrics, logger)
	}

	logger.Info("Completed calculateFlowEfficiency")
	return nil
}

func calculateIssueFlowMetric(db dal.Dal, projectName string, issue ticket.Issue, logger log.Logger) *models.IssueFlowMetric {
	// Get all status transitions for this issue from issue_changelogs
	var changelogs []struct {
		FieldName   string
		FromValue   string
		ToValue     string
		CreatedDate time.Time
	}

	err := db.All(&changelogs,
		dal.Select("field_name, from_value, to_value, created_date"),
		dal.From("issue_changelogs"),
		dal.Where("issue_id = ? AND field_name = 'status'", issue.Id),
		dal.Orderby("created_date ASC"),
	)
	if err != nil || len(changelogs) == 0 {
		// No transitions found - can't calculate flow efficiency
		return nil
	}

	// Build transition history
	transitions := make([]statusTransition, len(changelogs))
	for i, cl := range changelogs {
		transitions[i] = statusTransition{
			FromValue: cl.FromValue,
			ToValue:   cl.ToValue,
			ChangedAt: cl.CreatedDate,
		}
	}

	// Calculate time in each status.
	// Always bound durations to the issue completion timestamp so reopened/reprocessed
	// transitions after completion don't inflate active/waiting time.
	statusTime := make(map[string]float64) // status -> days
	startedAt := transitions[0].ChangedAt
	completedAt := determineCompletionTime(issue, transitions)
	if startedAt.IsZero() || completedAt.IsZero() || !completedAt.After(startedAt) {
		return nil
	}

	var totalActiveDays, totalWaitingDays float64

	addDuration := func(status string, intervalStart, intervalEnd time.Time) {
		if !intervalEnd.After(intervalStart) {
			return
		}
		daysInStatus := intervalEnd.Sub(intervalStart).Hours() / 24
		statusTime[status] += daysInStatus
		if isActiveStatus(status) {
			totalActiveDays += daysInStatus
		} else {
			totalWaitingDays += daysInStatus
		}
	}

	for i := 1; i < len(transitions); i++ {
		prev := transitions[i-1]
		curr := transitions[i]
		if !prev.ChangedAt.Before(completedAt) {
			break
		}
		intervalEnd := curr.ChangedAt
		if intervalEnd.After(completedAt) {
			intervalEnd = completedAt
		}
		addDuration(prev.ToValue, prev.ChangedAt, intervalEnd)
	}

	// Add the tail from the last transition to completion.
	last := transitions[len(transitions)-1]
	if last.ChangedAt.Before(completedAt) {
		addDuration(last.ToValue, last.ChangedAt, completedAt)
	}

	totalDays := completedAt.Sub(startedAt).Hours() / 24
	if totalDays <= 0 {
		return nil
	}

	flowEfficiency := (totalActiveDays / totalDays) * 100
	if flowEfficiency < 0 {
		flowEfficiency = 0
	}
	if flowEfficiency > 100 {
		flowEfficiency = 100
	}

	// Convert status breakdown to JSON
	statusBreakdownJSON, _ := json.Marshal(statusTime)

	return &models.IssueFlowMetric{
		Id:              fmt.Sprintf("%s:%s", projectName, issue.Id),
		ProjectName:     projectName,
		IssueId:         issue.Id,
		IssueKey:        issue.IssueKey,
		IssueType:       issue.Type,
		TotalDays:       totalDays,
		ActiveDays:      totalActiveDays,
		WaitingDays:     totalWaitingDays,
		FlowEfficiency:  flowEfficiency,
		StartedAt:       &startedAt,
		CompletedAt:     &completedAt,
		StatusBreakdown: string(statusBreakdownJSON),
		TransitionCount: len(transitions),
		CalculatedAt:    time.Now(),
	}
}

func determineCompletionTime(issue ticket.Issue, transitions []statusTransition) time.Time {
	var completedAt time.Time
	// Prefer the latest explicit completed-status transition.
	for _, trans := range transitions {
		if isCompletedStatus(trans.ToValue) && trans.ChangedAt.After(completedAt) {
			completedAt = trans.ChangedAt
		}
	}
	// Fall back to issue resolution date when transition history is incomplete.
	if issue.ResolutionDate != nil && issue.ResolutionDate.After(completedAt) {
		completedAt = *issue.ResolutionDate
	}
	// Final fallback to the last observed status transition.
	lastTransition := transitions[len(transitions)-1].ChangedAt
	if completedAt.IsZero() || lastTransition.After(completedAt) {
		completedAt = lastTransition
	}
	return completedAt
}

func isActiveStatus(status string) bool {
	// Normalize and check
	normalized := normalizeStatus(status)
	return activeStatuses[normalized]
}

func normalizeStatus(status string) string {
	// Convert to lowercase and replace underscores with spaces
	// Jira uses "IN_PROGRESS" but we match against "in progress"
	normalized := strings.ToLower(status)
	normalized = strings.ReplaceAll(normalized, "_", " ")
	return normalized
}

func isCompletedStatus(status string) bool {
	normalized := normalizeStatus(status)
	completedStatuses := map[string]bool{
		"done":      true,
		"closed":    true,
		"completed": true,
		"resolved":  true,
		"released":  true,
	}
	return completedStatuses[normalized]
}

func generateFlowSummary(db dal.Dal, projectName string, metrics []models.IssueFlowMetric, logger log.Logger) {
	if len(metrics) == 0 {
		return
	}

	// Extract efficiency values for percentile calculations
	efficiencies := make([]float64, len(metrics))
	var totalDays, activeDays, waitingDays float64
	var excellent, good, average, poor int

	for i, m := range metrics {
		efficiencies[i] = m.FlowEfficiency
		totalDays += m.TotalDays
		activeDays += m.ActiveDays
		waitingDays += m.WaitingDays

		switch m.FlowEfficiencyCategory() {
		case "excellent":
			excellent++
		case "good":
			good++
		case "average":
			average++
		case "poor":
			poor++
		}
	}

	// Sort for percentile calculations
	sort.Float64s(efficiencies)

	count := float64(len(metrics))

	// Calculate median (P50)
	p50Index := len(efficiencies) / 2
	var median float64
	if len(efficiencies)%2 == 0 {
		median = (efficiencies[p50Index-1] + efficiencies[p50Index]) / 2
	} else {
		median = efficiencies[p50Index]
	}

	// Calculate P90
	p90Index := int(float64(len(efficiencies)) * 0.9)
	if p90Index >= len(efficiencies) {
		p90Index = len(efficiencies) - 1
	}
	p90 := efficiencies[p90Index]

	// Calculate average
	var sumEff float64
	for _, e := range efficiencies {
		sumEff += e
	}
	avgEff := sumEff / count

	now := time.Now()
	summary := &models.ProjectFlowSummary{
		Id:                   fmt.Sprintf("%s:%s", projectName, now.Format("2006-01")),
		ProjectName:          projectName,
		PeriodStart:          time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:            now,
		IssueCount:           len(metrics),
		AvgFlowEfficiency:    avgEff,
		MedianFlowEfficiency: median,
		P90FlowEfficiency:    p90,
		AvgTotalDays:         totalDays / count,
		AvgActiveDays:        activeDays / count,
		AvgWaitingDays:       waitingDays / count,
		ExcellentCount:       excellent,
		GoodCount:            good,
		AverageCount:         average,
		PoorCount:            poor,
		CalculatedAt:         now,
	}

	if err := db.CreateOrUpdate(summary); err != nil {
		logger.Error(err, "failed to save flow summary")
	}
}
