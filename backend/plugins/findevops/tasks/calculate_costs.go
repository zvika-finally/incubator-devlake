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

// EffortResult holds the result of effort calculation with source tracking
type EffortResult struct {
	Hours          float64
	Source         string
	Confidence     string
	GitCodingHours float64
	GitReviewHours float64
	GitComplexity  float64
	GitActiveDays  int
	Validated      bool
	VariancePct    float64
	CommitShas     string
	PrIds          string
}

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
		effortResult := getHoursWorkedMultiSource(issue, data.Settings)
		if effortResult.Hours == 0 {
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

		// Calculate budget variance from Jira estimates
		var estimatedMinutes, actualMinutes int64
		if issue.OriginalEstimateMinutes != nil {
			estimatedMinutes = *issue.OriginalEstimateMinutes
		}
		if issue.TimeSpentMinutes != nil {
			actualMinutes = *issue.TimeSpentMinutes
		}

		varianceMinutes := estimatedMinutes - actualMinutes
		var variancePercent float64
		if estimatedMinutes > 0 {
			variancePercent = float64(varianceMinutes) / float64(estimatedMinutes) * 100
		}
		overBudget := actualMinutes > estimatedMinutes && estimatedMinutes > 0

		// Check if issue is unallocated (no epic/initiative)
		isUnallocated := checkIsUnallocated(db, issue)

		// Get initiative/epic ID for attribution
		initiativeId := getInitiativeId(db, issue)

		// Create cost allocation record
		// Note: ProjectPhase, CapitalizationCategory, and CategoryReason
		// are set by the categorizeCapitalization subtask which runs after this
		allocation := &models.CostAllocation{
			Id:                    fmt.Sprintf("%s:%s", issue.Id, fiscalMonth),
			InitiativeId:          initiativeId,
			IssueId:               issue.Id,
			FiscalMonth:           fiscalMonth,
			DeveloperId:           issue.AssigneeId,
			HoursWorked:           effortResult.Hours,
			HourlyRate:            hourlyRate,
			DeveloperCost:         effortResult.Hours * hourlyRate,
			TotalCost:             effortResult.Hours * hourlyRate,
			IssueType:             issue.Type,
			IssueLabels:           labels,
			EstimatedMinutes:      estimatedMinutes,
			ActualMinutes:         actualMinutes,
			VarianceMinutes:       varianceMinutes,
			VariancePercent:       variancePercent,
			OverBudget:            overBudget,
			IsUnallocated:         isUnallocated,
			EffortSource:          effortResult.Source,
			ConfidenceLevel:       effortResult.Confidence,
			GitCodingHours:        effortResult.GitCodingHours,
			GitReviewHours:        effortResult.GitReviewHours,
			GitComplexityFactor:   effortResult.GitComplexity,
			GitActiveDays:         effortResult.GitActiveDays,
			EffortValidated:       effortResult.Validated,
			ValidationVariancePct: effortResult.VariancePct,
			LinkedCommitShas:      effortResult.CommitShas,
			LinkedPrIds:           effortResult.PrIds,
			CalculatedAt:          time.Now(),
			CreatedAt:             time.Now(),
		}

		if err := db.CreateOrUpdate(allocation); err != nil {
			logger.Error(err, "failed to save cost allocation for issue %s", issue.IssueKey)
		}
	}

	// Generate monthly cost summaries
	if err := generateMonthlySummaries(db, data.Options.ProjectName, logger); err != nil {
		return errors.Default.Wrap(err, "failed to generate monthly summaries")
	}

	logger.Info("Completed calculateCosts")
	return nil
}

// checkIsUnallocated determines if an issue has no epic or initiative attribution
func checkIsUnallocated(db dal.Dal, issue ticket.Issue) bool {
	// Check for epic_key in Jira issues
	var epicKey string
	_ = db.First(&epicKey,
		dal.Select("epic_key"),
		dal.From("_tool_jira_issues"),
		dal.Where("issue_key = ?", issue.IssueKey),
	)

	if epicKey != "" {
		return false
	}

	// Check for parent_issue_id in domain layer
	if issue.ParentIssueId != "" {
		return false
	}

	return true
}

// generateMonthlySummaries aggregates cost allocations into monthly summaries
func generateMonthlySummaries(db dal.Dal, projectName string, logger log.Logger) errors.Error {
	logger.Info("Generating monthly cost summaries for project: %s", projectName)

	// Get distinct fiscal months with allocations
	var months []string
	err := db.All(&months,
		dal.Select("DISTINCT fiscal_month"),
		dal.From(&models.CostAllocation{}),
		dal.Join("LEFT JOIN issues ON issues.id = cost_allocations.issue_id"),
		dal.Join("LEFT JOIN board_issues bi ON bi.issue_id = issues.id"),
		dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'boards' AND pm.row_id = bi.board_id"),
		dal.Where("pm.project_name = ?", projectName),
	)
	if err != nil {
		return errors.Default.Wrap(err, "failed to get distinct fiscal months")
	}

	for _, month := range months {
		summary := calculateMonthlySummary(db, projectName, month, logger)
		if err := db.CreateOrUpdate(summary); err != nil {
			logger.Error(err, "failed to save monthly summary for %s", month)
		}
	}

	return nil
}

// calculateMonthlySummary generates aggregated metrics for a fiscal month
func calculateMonthlySummary(db dal.Dal, projectName string, fiscalMonth string, _ log.Logger) *models.MonthlyCostSummary {
	var allocations []models.CostAllocation
	_ = db.All(&allocations,
		dal.From(&models.CostAllocation{}),
		dal.Join("LEFT JOIN issues ON issues.id = cost_allocations.issue_id"),
		dal.Join("LEFT JOIN board_issues bi ON bi.issue_id = issues.id"),
		dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'boards' AND pm.row_id = bi.board_id"),
		dal.Where("pm.project_name = ? AND cost_allocations.fiscal_month = ?", projectName, fiscalMonth),
	)

	summary := &models.MonthlyCostSummary{
		Id:           fmt.Sprintf("%s:%s", projectName, fiscalMonth),
		ProjectName:  projectName,
		FiscalMonth:  fiscalMonth,
		CalculatedAt: time.Now(),
	}

	var totalEstimatedMinutes, totalActualMinutes int64
	for _, alloc := range allocations {
		summary.TotalCost += alloc.TotalCost

		// Aggregate costs by ASC 350-40 project phase
		switch alloc.ProjectPhase {
		case "preliminary":
			summary.PreliminaryCost += alloc.TotalCost
			summary.ExpenseCost += alloc.TotalCost
		case "development":
			summary.DevelopmentCost += alloc.TotalCost
			summary.CapitalizableCost += alloc.TotalCost
		case "post_implementation":
			summary.PostImplCost += alloc.TotalCost
			summary.ExpenseCost += alloc.TotalCost
		}

		// Track unallocated costs
		if alloc.IsUnallocated {
			summary.UnallocatedCost += alloc.TotalCost
			summary.OrphanIssueCount++
		}

		// Track budget variance
		totalEstimatedMinutes += alloc.EstimatedMinutes
		totalActualMinutes += alloc.ActualMinutes
		if alloc.OverBudget {
			summary.OverBudgetIssueCount++
		}
	}

	// Calculate capitalization rate
	if summary.TotalCost > 0 {
		summary.CapitalizationRate = summary.CapitalizableCost / summary.TotalCost * 100
	}

	// Calculate unallocated percentage
	if summary.TotalCost > 0 {
		summary.UnallocatedPercent = summary.UnallocatedCost / summary.TotalCost * 100
	}

	// Calculate budget variance at monthly level
	// Use default hourly rate for converting minutes to cost (simplified)
	defaultHourlyRate := 87.0 // TODO: get from settings
	summary.TotalEstimatedCost = float64(totalEstimatedMinutes) / 60.0 * defaultHourlyRate
	summary.TotalActualCost = float64(totalActualMinutes) / 60.0 * defaultHourlyRate
	if summary.TotalEstimatedCost > 0 {
		summary.BudgetVariance = (summary.TotalEstimatedCost - summary.TotalActualCost) / summary.TotalEstimatedCost * 100
	}

	return summary
}

func loadHourlyRates(db dal.Dal, _ float64, logger log.Logger) map[string]float64 {
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

func getHoursWorkedMultiSource(issue ticket.Issue, settings *models.FinDevOpsSettings) *EffortResult {
	result := &EffortResult{}

	// Priority 1: Actual time spent (from Jira worklogs) - HIGH confidence
	if issue.TimeSpentMinutes != nil && *issue.TimeSpentMinutes > 0 {
		result.Hours = float64(*issue.TimeSpentMinutes) / 60.0
		result.Source = "jira_time"
		result.Confidence = "high"
	} else if issue.OriginalEstimateMinutes != nil && *issue.OriginalEstimateMinutes > 0 {
		// Priority 2: Original estimate - MEDIUM confidence
		result.Hours = float64(*issue.OriginalEstimateMinutes) / 60.0
		result.Source = "jira_estimate"
		result.Confidence = "medium"
	} else if issue.StoryPoint != nil && *issue.StoryPoint > 0 {
		// Priority 3: Story points - MEDIUM confidence
		result.Hours = *issue.StoryPoint * settings.HoursPerStoryPoint
		result.Source = "story_points"
		result.Confidence = "medium"
	}

	// Try to get Git-inferred effort for validation or as primary source
	gitResult := GetGitEffortForIssue(issue.Id)
	if gitResult != nil && gitResult.TotalHours > 0 {
		result.GitCodingHours = gitResult.CodingHours
		result.GitReviewHours = gitResult.ReviewHours
		result.GitComplexity = gitResult.ComplexityFactor
		result.GitActiveDays = gitResult.ActiveDays
		result.CommitShas, result.PrIds = GetGitEffortAuditTrail(gitResult)

		if result.Hours > 0 {
			// Validate Jira vs Git
			variance := (result.Hours - gitResult.TotalHours) / result.Hours * 100
			result.VariancePct = variance
			result.Validated = true
		} else {
			// Priority 4: Git-inferred - INFERRED confidence
			result.Hours = gitResult.TotalHours
			result.Source = "git_inferred"
			result.Confidence = "inferred"
		}
	}

	return result
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

// getInitiativeId retrieves the epic or parent issue ID for cost attribution
func getInitiativeId(db dal.Dal, issue ticket.Issue) string {
	// Try to get epic_key from Jira issues
	var epicKey string
	_ = db.First(&epicKey,
		dal.Select("epic_key"),
		dal.From("_tool_jira_issues"),
		dal.Where("issue_key = ?", issue.IssueKey),
	)

	if epicKey != "" {
		return epicKey
	}

	// Fall back to parent_issue_id from domain layer
	if issue.ParentIssueId != "" {
		return issue.ParentIssueId
	}

	return ""
}
