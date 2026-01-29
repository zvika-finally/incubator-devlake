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
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/log"
	"github.com/apache/incubator-devlake/core/models/domainlayer/code"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/aidetector/models"
)

var AnalyzeCodeChurnMeta = plugin.SubTaskMeta{
	Name:             "analyzeCodeChurn",
	EntryPoint:       AnalyzeCodeChurn,
	EnabledByDefault: true,
	Description:      "Analyze code churn for AI-detected PRs vs non-AI PRs",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CODE},
}

func AnalyzeCodeChurn(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*AIDetectorTaskData)
	logger := taskCtx.GetLogger()

	logger.Info("Starting analyzeCodeChurn for project: %s", data.Options.ProjectName)

	// Get merged PRs with AI signals
	type PRWithSignal struct {
		code.PullRequest
		AIConfidenceScore int
	}

	var prs []PRWithSignal
	err := db.All(&prs,
		dal.Select("pull_requests.*, COALESCE(ai_usage_signals.ai_confidence_score, 0) as ai_confidence_score"),
		dal.From(&code.PullRequest{}),
		dal.Join("LEFT JOIN ai_usage_signals ON ai_usage_signals.pull_request_id = pull_requests.id"),
		dal.Join("LEFT JOIN project_mapping pm ON pm.row_id = pull_requests.base_repo_id AND pm.table = 'repos'"),
		dal.Where("pm.project_name = ? AND pull_requests.merged_date IS NOT NULL", data.Options.ProjectName),
		dal.Orderby("pull_requests.merged_date DESC"),
	)
	if err != nil {
		return errors.Default.Wrap(err, "failed to query merged PRs")
	}

	logger.Info("Analyzing code churn for %d merged PRs", len(prs))

	confidenceThreshold := data.Options.ConfidenceThreshold
	if confidenceThreshold == 0 {
		confidenceThreshold = 65
	}

	var allMetrics []models.AIChurnMetric

	for _, pr := range prs {
		if pr.MergedDate == nil {
			continue
		}

		// Get files changed in this PR
		files := getPRFiles(db, pr.Id)
		if len(files) == 0 {
			continue
		}

		// Calculate churn for these files after merge
		churn7, commits7 := calculateChurnAfterMerge(db, files, *pr.MergedDate, 7)
		churn30, commits30 := calculateChurnAfterMerge(db, files, *pr.MergedDate, 30)

		initialAdditions := pr.Additions
		if initialAdditions == 0 {
			initialAdditions = 1 // Avoid division by zero
		}

		filesJSON, _ := json.Marshal(files)

		metric := &models.AIChurnMetric{
			Id:                fmt.Sprintf("%s:%s", data.Options.ProjectName, pr.Id),
			PullRequestId:     pr.Id,
			PullRequestKey:    pr.PullRequestKey,
			ProjectName:       data.Options.ProjectName,
			AuthorId:          pr.AuthorId,
			AuthorName:        pr.AuthorName,
			AIConfidenceScore: pr.AIConfidenceScore,
			IsAIAssisted:      pr.AIConfidenceScore >= confidenceThreshold,
			InitialAdditions:  pr.Additions,
			InitialDeletions:  pr.Deletions,
			MergedAt:          pr.MergedDate,
			ChurnWithin7Days:  churn7,
			ChurnWithin30Days: churn30,
			FollowUpCommits7:  commits7,
			FollowUpCommits30: commits30,
			ChurnRatio7Days:   float64(churn7) / float64(initialAdditions),
			ChurnRatio30Days:  float64(churn30) / float64(initialAdditions),
			FilePaths:         string(filesJSON),
			CalculatedAt:      time.Now(),
		}

		if err := db.CreateOrUpdate(metric); err != nil {
			logger.Error(err, "failed to save churn metric for PR %d", pr.PullRequestKey)
		}
		allMetrics = append(allMetrics, *metric)
	}

	// Generate project churn summary
	if len(allMetrics) > 0 {
		generateChurnSummary(db, data.Options.ProjectName, allMetrics, confidenceThreshold, logger)
	}

	logger.Info("Completed analyzeCodeChurn")
	return nil
}

// getPRFiles returns the file paths modified in a PR
func getPRFiles(db dal.Dal, prId string) []string {
	var files []struct {
		FilePath string
	}

	// Query commit_files using the merge commit SHA
	// Note: We use merge_commit_sha because individual PR commits get rewritten during
	// squash/rebase merges, but the merge commit remains on the main branch
	_ = db.All(&files,
		dal.Select("DISTINCT commit_files.file_path"),
		dal.From("commit_files"),
		dal.Join("INNER JOIN pull_requests pr ON pr.merge_commit_sha = commit_files.commit_sha"),
		dal.Where("pr.id = ?", prId),
	)

	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = f.FilePath
	}
	return paths
}

// calculateChurnAfterMerge calculates lines modified in the given files within N days after merge
func calculateChurnAfterMerge(db dal.Dal, files []string, mergedAt time.Time, days int) (churn int, commitCount int) {
	if len(files) == 0 {
		return 0, 0
	}

	windowEnd := mergedAt.AddDate(0, 0, days)

	// Query modifications to these files after merge
	var results []struct {
		Additions int
		Deletions int
		CommitSha string
	}

	// Build file path filter
	_ = db.All(&results,
		dal.Select("commit_files.additions, commit_files.deletions, commit_files.commit_sha"),
		dal.From("commit_files"),
		dal.Join("INNER JOIN commits ON commits.sha = commit_files.commit_sha"),
		dal.Where("commit_files.file_path IN ? AND commits.authored_date > ? AND commits.authored_date <= ?",
			files, mergedAt, windowEnd),
	)

	uniqueCommits := make(map[string]bool)
	for _, r := range results {
		churn += r.Additions + r.Deletions
		uniqueCommits[r.CommitSha] = true
	}

	return churn, len(uniqueCommits)
}

func generateChurnSummary(db dal.Dal, projectName string, metrics []models.AIChurnMetric, _ int, logger log.Logger) {
	var aiChurnSum7, aiChurnSum30, aiAdditionsSum float64
	var nonAIChurnSum7, nonAIChurnSum30, nonAIAdditionsSum float64
	var aiCount, nonAICount int
	var aiTotalChurn30, nonAITotalChurn30 int
	var aiTotalAdditions, nonAITotalAdditions int

	for _, m := range metrics {
		if m.IsAIAssisted {
			aiCount++
			aiChurnSum7 += m.ChurnRatio7Days
			aiChurnSum30 += m.ChurnRatio30Days
			aiAdditionsSum += float64(m.InitialAdditions)
			aiTotalChurn30 += m.ChurnWithin30Days
			aiTotalAdditions += m.InitialAdditions
		} else {
			nonAICount++
			nonAIChurnSum7 += m.ChurnRatio7Days
			nonAIChurnSum30 += m.ChurnRatio30Days
			nonAIAdditionsSum += float64(m.InitialAdditions)
			nonAITotalChurn30 += m.ChurnWithin30Days
			nonAITotalAdditions += m.InitialAdditions
		}
	}

	var aiAvg7, aiAvg30, nonAIAvg7, nonAIAvg30, churnDiff float64

	if aiCount > 0 {
		aiAvg7 = aiChurnSum7 / float64(aiCount)
		aiAvg30 = aiChurnSum30 / float64(aiCount)
	}
	if nonAICount > 0 {
		nonAIAvg7 = nonAIChurnSum7 / float64(nonAICount)
		nonAIAvg30 = nonAIChurnSum30 / float64(nonAICount)
	}

	// Calculate churn difference percentage
	if nonAIAvg30 > 0 {
		churnDiff = (aiAvg30 - nonAIAvg30) / nonAIAvg30 * 100
	}

	now := time.Now()
	summary := &models.ProjectChurnSummary{
		Id:                     fmt.Sprintf("%s:%s", projectName, now.Format("2006-01")),
		ProjectName:            projectName,
		PeriodStart:            time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:              now,
		TotalPRsAnalyzed:       len(metrics),
		AIPRCount:              aiCount,
		NonAIPRCount:           nonAICount,
		AIAvgChurnRatio7:       aiAvg7,
		AIAvgChurnRatio30:      aiAvg30,
		AITotalChurn30:         aiTotalChurn30,
		AITotalAdditions:       aiTotalAdditions,
		NonAIAvgChurnRatio7:    nonAIAvg7,
		NonAIAvgChurnRatio30:   nonAIAvg30,
		NonAITotalChurn30:      nonAITotalChurn30,
		NonAITotalAdditions:    nonAITotalAdditions,
		ChurnDifferencePercent: churnDiff,
		CalculatedAt:           now,
	}

	if err := db.CreateOrUpdate(summary); err != nil {
		logger.Error(err, "failed to save churn summary")
	}

	logger.Info("Churn analysis: AI PRs=%d (avg 30d ratio: %.2f), Non-AI PRs=%d (avg 30d ratio: %.2f), Diff=%.1f%%",
		aiCount, aiAvg30, nonAICount, nonAIAvg30, churnDiff)
}
