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
	"github.com/apache/incubator-devlake/core/models/domainlayer/code"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/aidetector/models"
)

var CalculateAIImpactMeta = plugin.SubTaskMeta{
	Name:             "calculateAIImpact",
	EntryPoint:       CalculateAIImpact,
	EnabledByDefault: true,
	Description:      "Calculate productivity impact by comparing metrics before and after AI adoption",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CODE},
}

// Constants from eng-product-metrics
const (
	BaselineDays = 90 // Days before adoption for baseline
	CurrentDays  = 30 // Days after adoption for current period
)

func CalculateAIImpact(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*AIDetectorTaskData)
	logger := taskCtx.GetLogger()

	logger.Info("Starting calculateAIImpact for project: %s", data.Options.ProjectName)

	// Find AI adoption date - first explicit AI signal detection for this project
	adoptionDate, err := findAIAdoptionDate(db, data.Options.ProjectName)
	if err != nil {
		return errors.Default.Wrap(err, "failed to find AI adoption date")
	}

	if adoptionDate == nil {
		logger.Info("No AI adoption detected for project %s, skipping impact calculation", data.Options.ProjectName)
		return nil
	}

	logger.Info("AI adoption date for project %s: %s", data.Options.ProjectName, adoptionDate.Format("2006-01-02"))

	// Calculate baseline period (90 days before adoption)
	baselineEnd := *adoptionDate
	baselineStart := baselineEnd.AddDate(0, 0, -BaselineDays)

	// Calculate current period (30 days after adoption, or up to now if less time has passed)
	currentStart := *adoptionDate
	currentEnd := adoptionDate.AddDate(0, 0, CurrentDays)
	now := time.Now()
	if currentEnd.After(now) {
		currentEnd = now
	}

	// Calculate metrics for both periods
	baselineMetrics, err := calculatePeriodMetrics(db, data.Options.ProjectName, baselineStart, baselineEnd)
	if err != nil {
		return errors.Default.Wrap(err, "failed to calculate baseline metrics")
	}

	currentMetrics, err := calculatePeriodMetrics(db, data.Options.ProjectName, currentStart, currentEnd)
	if err != nil {
		return errors.Default.Wrap(err, "failed to calculate current metrics")
	}

	// Calculate percentage changes
	// Formula: ((current - baseline) / baseline) * 100
	// For throughput: higher is better, positive change = improvement
	// For time metrics: lower is better, so we invert the sign (faster = positive)
	prThroughputChange := CalculatePercentChange(baselineMetrics.prThroughput, currentMetrics.prThroughput)
	reviewTimeChange := -CalculatePercentChange(baselineMetrics.reviewTime, currentMetrics.reviewTime)   // Inverted
	leadTimeChange := -CalculatePercentChange(baselineMetrics.leadTime, currentMetrics.leadTime)         // Inverted

	// Create or update impact metric
	impactMetric := models.AIImpactMetric{
		Id:              fmt.Sprintf("%s:%s", data.Options.ProjectName, adoptionDate.Format("20060102")),
		ProjectName:     data.Options.ProjectName,
		AIAdoptionDate:  adoptionDate,

		BaselinePRThroughput: baselineMetrics.prThroughput,
		BaselineReviewTime:   baselineMetrics.reviewTime,
		BaselineLeadTime:     baselineMetrics.leadTime,

		CurrentPRThroughput: currentMetrics.prThroughput,
		CurrentReviewTime:   currentMetrics.reviewTime,
		CurrentLeadTime:     currentMetrics.leadTime,

		PRThroughputChange: prThroughputChange,
		ReviewTimeChange:   reviewTimeChange,
		LeadTimeChange:     leadTimeChange,

		CalculatedAt: time.Now(),
	}

	if err := db.CreateOrUpdate(&impactMetric); err != nil {
		return errors.Default.Wrap(err, "failed to save AI impact metric")
	}

	logger.Info("AI Impact calculated for project %s: PR throughput change: %.1f%%, Review time change: %.1f%%, Lead time change: %.1f%%",
		data.Options.ProjectName, prThroughputChange, reviewTimeChange, leadTimeChange)

	return nil
}

// periodMetrics holds calculated metrics for a time period
type periodMetrics struct {
	prThroughput float64 // PRs merged per week
	reviewTime   float64 // Average hours from PR open to merge
	leadTime     float64 // Average hours from first commit to deploy
}

// findAIAdoptionDate finds the earliest explicit AI signal detection for a project
func findAIAdoptionDate(db dal.Dal, projectName string) (*time.Time, errors.Error) {
	// Query for the earliest PR with explicit AI tool detection
	var signal models.AIUsageSignal
	clauses := []dal.Clause{
		dal.From(&models.AIUsageSignal{}),
		dal.Join("LEFT JOIN pull_requests pr ON pr.id = ai_usage_signals.pull_request_id"),
		dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'repos' AND pm.row_id = pr.base_repo_id"),
		dal.Where("pm.project_name = ? AND ai_usage_signals.explicit_tool_detected = ?", projectName, true),
		dal.Orderby("ai_usage_signals.detected_at ASC"),
		dal.Limit(1),
	}

	err := db.First(&signal, clauses...)
	if err != nil {
		// No explicit AI detection found
		if db.IsErrorNotFound(err) {
			return nil, nil
		}
		return nil, errors.Default.Wrap(err, "failed to query AI signals")
	}

	if signal.DetectedAt.IsZero() {
		return nil, nil
	}

	return &signal.DetectedAt, nil
}

// calculatePeriodMetrics calculates productivity metrics for a given time period
func calculatePeriodMetrics(db dal.Dal, projectName string, start, end time.Time) (*periodMetrics, errors.Error) {
	metrics := &periodMetrics{}

	// Calculate PR throughput (PRs merged per week)
	countClauses := []dal.Clause{
		dal.From(&code.PullRequest{}),
		dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'repos' AND pm.row_id = pull_requests.base_repo_id"),
		dal.Where("pm.project_name = ? AND pull_requests.merged_date >= ? AND pull_requests.merged_date < ?",
			projectName, start, end),
	}
	prCount, err := db.Count(countClauses...)
	if err != nil {
		return nil, errors.Default.Wrap(err, "failed to count PRs")
	}

	days := end.Sub(start).Hours() / 24
	weeks := days / 7
	if weeks > 0 {
		metrics.prThroughput = float64(prCount) / weeks
	}

	// Calculate average review time (hours from PR creation to merge)
	var avgReviewTime float64
	reviewTimeClauses := []dal.Clause{
		dal.Select("AVG(EXTRACT(EPOCH FROM (merged_date - created_date)) / 3600) as avg_review_time"),
		dal.From(&code.PullRequest{}),
		dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'repos' AND pm.row_id = pull_requests.base_repo_id"),
		dal.Where("pm.project_name = ? AND pull_requests.merged_date >= ? AND pull_requests.merged_date < ? AND pull_requests.merged_date IS NOT NULL",
			projectName, start, end),
	}
	err = db.First(&avgReviewTime, reviewTimeClauses...)
	if err != nil && !db.IsErrorNotFound(err) {
		return nil, errors.Default.Wrap(err, "failed to calculate review time")
	}
	metrics.reviewTime = avgReviewTime

	// Calculate lead time using project_pr_metrics if available
	var avgLeadTime float64
	leadTimeClauses := []dal.Clause{
		dal.Select("AVG(pr_coding_time + pr_pickup_time + pr_review_time + pr_deploy_time) / 3600000 as avg_lead_time"),
		dal.From("project_pr_metrics"),
		dal.Where("project_name = ? AND pr_merged_date >= ? AND pr_merged_date < ?",
			projectName, start, end),
	}
	err = db.First(&avgLeadTime, leadTimeClauses...)
	if err != nil && !db.IsErrorNotFound(err) {
		// Fall back to review time if project_pr_metrics not available
		metrics.leadTime = metrics.reviewTime
	} else {
		metrics.leadTime = avgLeadTime
	}

	return metrics, nil
}

// CalculatePercentChange calculates percentage change between baseline and current values
// Returns 0 if baseline is 0 to avoid division by zero
// Exported for testing
func CalculatePercentChange(baseline, current float64) float64 {
	if baseline == 0 {
		return 0
	}
	return ((current - baseline) / baseline) * 100
}
