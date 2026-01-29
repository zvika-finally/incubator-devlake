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
	"sort"
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/log"
	"github.com/apache/incubator-devlake/core/models/domainlayer/code"
	"github.com/apache/incubator-devlake/core/models/domainlayer/ticket"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/businessmetrics/models"
)

var CheckAgreementsMeta = plugin.SubTaskMeta{
	Name:             "checkAgreements",
	EntryPoint:       CheckAgreements,
	EnabledByDefault: true,
	Description:      "Check working agreements and record violations",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CODE, plugin.DOMAIN_TYPE_TICKET},
}

func CheckAgreements(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*BusinessMetricsTaskData)
	logger := taskCtx.GetLogger()

	logger.Info("Starting checkAgreements for project: %s", data.Options.ProjectName)

	// Load or create default working agreements
	agreements, err := loadOrCreateAgreements(db, data.Options.ProjectName, logger)
	if err != nil {
		return err
	}

	// Check each agreement type
	for _, agreement := range agreements {
		if !agreement.AlertEnabled {
			continue
		}

		switch agreement.AgreementType {
		case models.AgreementPRMergeTime:
			checkPRMergeTime(db, &agreement, data.Options.ProjectName, logger)
		case models.AgreementReviewTurnaround:
			checkReviewTurnaround(db, &agreement, data.Options.ProjectName, logger)
		case models.AgreementWIPLimit:
			checkWIPLimit(db, &agreement, data.Options.ProjectName, logger)
		case models.AgreementIssuesInProgress:
			checkIssuesInProgress(db, &agreement, data.Options.ProjectName, logger)
		}
	}

	// Generate compliance summaries
	generateComplianceSummaries(db, data.Options.ProjectName, agreements, logger)

	logger.Info("Completed checkAgreements")
	return nil
}

func loadOrCreateAgreements(db dal.Dal, projectName string, logger log.Logger) ([]models.WorkingAgreement, errors.Error) {
	var agreements []models.WorkingAgreement
	err := db.All(&agreements,
		dal.From(&models.WorkingAgreement{}),
		dal.Where("project_name = ?", projectName),
	)
	if err != nil {
		return nil, errors.Default.Wrap(err, "failed to load agreements")
	}

	// If no agreements exist, create defaults
	if len(agreements) == 0 {
		logger.Info("No agreements found for project %s, creating defaults", projectName)
		defaults := models.DefaultWorkingAgreements(projectName)
		for _, agreement := range defaults {
			if err := db.Create(&agreement); err != nil {
				logger.Error(err, "failed to create default agreement %s", agreement.Id)
			}
		}
		return defaults, nil
	}

	return agreements, nil
}

// checkPRMergeTime checks if PRs are being merged within the threshold
func checkPRMergeTime(db dal.Dal, agreement *models.WorkingAgreement, projectName string, logger log.Logger) {
	// Get open PRs and recently merged PRs
	var prs []code.PullRequest
	thresholdHours := agreement.ThresholdValue * 24 // Convert days to hours
	if agreement.ThresholdUnit == models.UnitHours {
		thresholdHours = agreement.ThresholdValue
	}

	// Query PRs for this project
	err := db.All(&prs,
		dal.From(&code.PullRequest{}),
		dal.Join("LEFT JOIN repo_mapping rm ON rm.row_id = pull_requests.base_repo_id AND rm.table = 'repos'"),
		dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'cicd_scopes' AND pm.row_id = rm.cicd_scope_id"),
		dal.Where("pm.project_name = ?", projectName),
	)
	if err != nil {
		logger.Error(err, "failed to query PRs for project %s", projectName)
		return
	}

	now := time.Now()
	for _, pr := range prs {
		var cycleTimeHours float64
		if pr.MergedDate != nil {
			// For merged PRs, calculate actual cycle time
			cycleTimeHours = pr.MergedDate.Sub(pr.CreatedDate).Hours()
		} else if pr.Status == "OPEN" {
			// For open PRs, check age so far
			cycleTimeHours = now.Sub(pr.CreatedDate).Hours()
		} else {
			continue // Skip closed but not merged PRs
		}

		if cycleTimeHours > thresholdHours {
			recordViolation(db, agreement, "pull_request", pr.Id, fmt.Sprintf("#%d", pr.PullRequestKey), cycleTimeHours/24, agreement.ThresholdValue, logger)
		}
	}
}

// checkReviewTurnaround checks if first review happens within threshold
func checkReviewTurnaround(db dal.Dal, agreement *models.WorkingAgreement, projectName string, logger log.Logger) {
	thresholdHours := agreement.ThresholdValue
	if agreement.ThresholdUnit == models.UnitDays {
		thresholdHours = agreement.ThresholdValue * 24
	}

	// Query PRs with their first review time
	var results []struct {
		PrId          string
		PrKey         int
		CreatedDate   time.Time
		FirstReviewAt *time.Time
	}

	// This is a simplified query - in production, would need to join with PR reviews
	err := db.All(&results,
		dal.Select("pull_requests.id as pr_id, pull_requests.pull_request_key as pr_key, pull_requests.created_date, MIN(pull_request_comments.created_date) as first_review_at"),
		dal.From(&code.PullRequest{}),
		dal.Join("LEFT JOIN pull_request_comments ON pull_request_comments.pull_request_id = pull_requests.id"),
		dal.Join("LEFT JOIN repo_mapping rm ON rm.row_id = pull_requests.base_repo_id AND rm.table = 'repos'"),
		dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'cicd_scopes' AND pm.row_id = rm.cicd_scope_id"),
		dal.Where("pm.project_name = ? AND pull_requests.status = 'OPEN'", projectName),
		dal.Groupby("pull_requests.id"),
	)
	if err != nil {
		logger.Error(err, "failed to query review turnaround for project %s", projectName)
		return
	}

	now := time.Now()
	for _, result := range results {
		var waitingHours float64
		if result.FirstReviewAt != nil {
			waitingHours = result.FirstReviewAt.Sub(result.CreatedDate).Hours()
		} else {
			// No review yet - check how long it's been waiting
			waitingHours = now.Sub(result.CreatedDate).Hours()
		}

		if waitingHours > thresholdHours {
			recordViolation(db, agreement, "pull_request", result.PrId, fmt.Sprintf("#%d", result.PrKey), waitingHours, thresholdHours, logger)
		}
	}
}

// checkWIPLimit checks if developers have too many open PRs
func checkWIPLimit(db dal.Dal, agreement *models.WorkingAgreement, projectName string, logger log.Logger) {
	// Count open PRs per developer
	var results []struct {
		AuthorId   string
		AuthorName string
		OpenCount  int
	}

	err := db.All(&results,
		dal.Select("pull_requests.author_id, pull_requests.author_name, COUNT(*) as open_count"),
		dal.From(&code.PullRequest{}),
		dal.Join("LEFT JOIN repo_mapping rm ON rm.row_id = pull_requests.base_repo_id AND rm.table = 'repos'"),
		dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'cicd_scopes' AND pm.row_id = rm.cicd_scope_id"),
		dal.Where("pm.project_name = ? AND pull_requests.status = 'OPEN'", projectName),
		dal.Groupby("pull_requests.author_id, pull_requests.author_name"),
	)
	if err != nil {
		logger.Error(err, "failed to query WIP counts for project %s", projectName)
		return
	}

	for _, result := range results {
		if float64(result.OpenCount) > agreement.ThresholdValue {
			recordViolation(db, agreement, "developer", result.AuthorId, result.AuthorName, float64(result.OpenCount), agreement.ThresholdValue, logger)
		}
	}
}

// checkIssuesInProgress checks if developers have too many issues in progress
func checkIssuesInProgress(db dal.Dal, agreement *models.WorkingAgreement, projectName string, logger log.Logger) {
	// Count in-progress issues per developer
	var results []struct {
		AssigneeId   string
		AssigneeName string
		InProgress   int
	}

	err := db.All(&results,
		dal.Select("issues.assignee_id, issues.assignee_name, COUNT(*) as in_progress"),
		dal.From(&ticket.Issue{}),
		dal.Join("LEFT JOIN board_issues bi ON bi.issue_id = issues.id"),
		dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'boards' AND pm.row_id = bi.board_id"),
		dal.Where("pm.project_name = ? AND issues.status = 'IN_PROGRESS'", projectName),
		dal.Groupby("issues.assignee_id, issues.assignee_name"),
	)
	if err != nil {
		logger.Error(err, "failed to query issues in progress for project %s", projectName)
		return
	}

	for _, result := range results {
		if float64(result.InProgress) > agreement.ThresholdValue {
			recordViolation(db, agreement, "developer", result.AssigneeId, result.AssigneeName, float64(result.InProgress), agreement.ThresholdValue, logger)
		}
	}
}

func recordViolation(db dal.Dal, agreement *models.WorkingAgreement, entityType string, entityId string, entityKey string, currentValue float64, thresholdValue float64, logger log.Logger) {
	violation := &models.AgreementViolation{
		Id:             fmt.Sprintf("%s:%s:%s", agreement.Id, entityType, entityId),
		AgreementId:    agreement.Id,
		AgreementType:  agreement.AgreementType,
		ProjectName:    agreement.ProjectName,
		EntityType:     entityType,
		EntityId:       entityId,
		EntityKey:      entityKey,
		CurrentValue:   currentValue,
		ThresholdValue: thresholdValue,
		ExcessValue:    currentValue - thresholdValue,
		ViolatedAt:     time.Now(),
		IsResolved:     false,
		CreatedAt:      time.Now(),
	}

	if err := db.CreateOrUpdate(violation); err != nil {
		logger.Error(err, "failed to record violation for %s", entityKey)
	}
}

func generateComplianceSummaries(db dal.Dal, projectName string, agreements []models.WorkingAgreement, logger log.Logger) {
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC) // Start of current month
	periodEnd := now

	for _, agreement := range agreements {
		// Get all violations for this agreement in the period
		var violations []models.AgreementViolation
		_ = db.All(&violations,
			dal.From(&models.AgreementViolation{}),
			dal.Where("agreement_id = ? AND violated_at >= ? AND violated_at <= ?", agreement.Id, periodStart, periodEnd),
		)

		// Get all checked entities (this would need to be calculated based on type)
		totalChecked := getTotalChecked(db, agreement, projectName)
		totalViolations := len(violations)
		totalCompliant := totalChecked - totalViolations
		if totalCompliant < 0 {
			totalCompliant = 0
		}

		var complianceRate float64
		if totalChecked > 0 {
			complianceRate = float64(totalCompliant) / float64(totalChecked) * 100
		}

		// Calculate value statistics
		values := getValuesForAgreement(db, agreement, projectName)
		avgValue, p50, p90 := calculateStats(values)

		summary := &models.AgreementComplianceSummary{
			Id:              fmt.Sprintf("%s:%s:%s", projectName, agreement.AgreementType, periodStart.Format("2006-01")),
			ProjectName:     projectName,
			AgreementType:   agreement.AgreementType,
			PeriodStart:     periodStart,
			PeriodEnd:       periodEnd,
			TotalChecked:    totalChecked,
			TotalCompliant:  totalCompliant,
			TotalViolations: totalViolations,
			ComplianceRate:  complianceRate,
			AverageValue:    avgValue,
			P50Value:        p50,
			P90Value:        p90,
			CalculatedAt:    now,
		}

		if err := db.CreateOrUpdate(summary); err != nil {
			logger.Error(err, "failed to save compliance summary for %s", agreement.AgreementType)
		}
	}
}

func getTotalChecked(db dal.Dal, agreement models.WorkingAgreement, projectName string) int {
	switch agreement.AgreementType {
	case models.AgreementPRMergeTime, models.AgreementReviewTurnaround:
		count, _ := db.Count(
			dal.From(&code.PullRequest{}),
			dal.Join("LEFT JOIN repo_mapping rm ON rm.row_id = pull_requests.base_repo_id AND rm.table = 'repos'"),
			dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'cicd_scopes' AND pm.row_id = rm.cicd_scope_id"),
			dal.Where("pm.project_name = ?", projectName),
		)
		return int(count)
	case models.AgreementWIPLimit, models.AgreementIssuesInProgress:
		// Count unique developers - simplified approach
		var results []struct {
			AuthorId string
		}
		_ = db.All(&results,
			dal.Select("DISTINCT pull_requests.author_id"),
			dal.From(&code.PullRequest{}),
			dal.Join("LEFT JOIN repo_mapping rm ON rm.row_id = pull_requests.base_repo_id AND rm.table = 'repos'"),
			dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'cicd_scopes' AND pm.row_id = rm.cicd_scope_id"),
			dal.Where("pm.project_name = ?", projectName),
		)
		return len(results)
	}
	return 0
}

func getValuesForAgreement(_ dal.Dal, _ models.WorkingAgreement, _ string) []float64 {
	// This would query actual values based on agreement type
	// Simplified for now - would need proper implementation per agreement type
	return []float64{}
}

func calculateStats(values []float64) (avg float64, p50 float64, p90 float64) {
	if len(values) == 0 {
		return 0, 0, 0
	}

	// Calculate average
	var sum float64
	for _, v := range values {
		sum += v
	}
	avg = sum / float64(len(values))

	// Sort for percentiles
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	// Calculate P50 (median)
	p50Index := len(sorted) / 2
	if len(sorted)%2 == 0 {
		p50 = (sorted[p50Index-1] + sorted[p50Index]) / 2
	} else {
		p50 = sorted[p50Index]
	}

	// Calculate P90
	p90Index := int(float64(len(sorted)) * 0.9)
	if p90Index >= len(sorted) {
		p90Index = len(sorted) - 1
	}
	p90 = sorted[p90Index]

	return avg, p50, p90
}
