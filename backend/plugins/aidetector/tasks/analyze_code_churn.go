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
	"path"
	"strings"
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

	confidenceThreshold := GetEffectiveConfidenceThreshold(data)

	var allMetrics []models.AIChurnMetric

	for _, pr := range prs {
		if pr.MergedDate == nil {
			continue
		}
		baselineLines := pr.Additions + pr.Deletions
		if baselineLines <= 0 {
			// No changed lines in the original PR means churn ratio is undefined.
			continue
		}

		// Get files changed in this PR (excluding infrastructure files)
		files := getPRFiles(db, pr.Id)
		files = filterInfrastructureFiles(files, data.Settings, logger)
		if len(files) == 0 {
			continue
		}

		// Calculate churn for these files after merge
		churn7, commits7 := calculateChurnAfterMerge(db, files, *pr.MergedDate, 7)
		churn30, commits30 := calculateChurnAfterMerge(db, files, *pr.MergedDate, 30)

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
			ChurnRatio7Days:   float64(churn7) / float64(baselineLines),
			ChurnRatio30Days:  float64(churn30) / float64(baselineLines),
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
	var aiCount, nonAICount int
	var aiTotalChurn7, nonAITotalChurn7 int
	var aiTotalChurn30, nonAITotalChurn30 int
	var aiTotalBaselineLines, nonAITotalBaselineLines int
	var aiTotalAdditions, nonAITotalAdditions int

	for _, m := range metrics {
		baselineLines := m.InitialAdditions + m.InitialDeletions
		if baselineLines <= 0 {
			continue
		}
		if m.IsAIAssisted {
			aiCount++
			aiTotalChurn7 += m.ChurnWithin7Days
			aiTotalChurn30 += m.ChurnWithin30Days
			aiTotalBaselineLines += baselineLines
			aiTotalAdditions += m.InitialAdditions
		} else {
			nonAICount++
			nonAITotalChurn7 += m.ChurnWithin7Days
			nonAITotalChurn30 += m.ChurnWithin30Days
			nonAITotalBaselineLines += baselineLines
			nonAITotalAdditions += m.InitialAdditions
		}
	}

	var aiAvg7, aiAvg30, nonAIAvg7, nonAIAvg30, churnDiff float64

	if aiTotalBaselineLines > 0 {
		aiAvg7 = float64(aiTotalChurn7) / float64(aiTotalBaselineLines)
		aiAvg30 = float64(aiTotalChurn30) / float64(aiTotalBaselineLines)
	}
	if nonAITotalBaselineLines > 0 {
		nonAIAvg7 = float64(nonAITotalChurn7) / float64(nonAITotalBaselineLines)
		nonAIAvg30 = float64(nonAITotalChurn30) / float64(nonAITotalBaselineLines)
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

// filterInfrastructureFiles removes infrastructure/config files from analysis
// Infrastructure files (.github/, package.json, etc.) often have high churn but aren't application code
func filterInfrastructureFiles(files []string, settings *models.AIDetectorSettings, logger log.Logger) []string {
	if settings == nil || settings.ExcludeFilePatterns == "" {
		return files
	}

	// Parse exclusion patterns from JSON
	var excludePatterns []string
	if err := json.Unmarshal([]byte(settings.ExcludeFilePatterns), &excludePatterns); err != nil {
		logger.Warn(err, "Failed to parse exclude file patterns, using all files")
		return files
	}

	if len(excludePatterns) == 0 {
		return files
	}

	// Filter files
	filtered := make([]string, 0, len(files))
	excludedCount := 0

	for _, file := range files {
		if shouldExcludeFile(file, excludePatterns) {
			excludedCount++
			continue
		}
		filtered = append(filtered, file)
	}

	if excludedCount > 0 {
		logger.Debug("Excluded %d infrastructure files from churn analysis", excludedCount)
	}

	return filtered
}

// shouldExcludeFile checks if a file path matches any exclusion pattern
func shouldExcludeFile(filePath string, patterns []string) bool {
	filePath = strings.ReplaceAll(strings.TrimSpace(filePath), "\\", "/")
	base := path.Base(filePath)
	for _, pattern := range patterns {
		pattern = strings.ReplaceAll(strings.TrimSpace(pattern), "\\", "/")
		if pattern == "" {
			continue
		}

		// Directory prefix match (e.g., ".github/" matches ".github/workflows/ci.yml")
		if strings.HasSuffix(pattern, "/") {
			if strings.HasPrefix(filePath, pattern) {
				return true
			}
			continue
		}

		// Glob support (e.g., "*.lock", "**/*.md" style path globs supported by path.Match semantics).
		if strings.ContainsAny(pattern, "*?[") {
			if matched, _ := path.Match(pattern, filePath); matched {
				return true
			}
			// For basename-only globs like "*.lock", match against filename too.
			if !strings.Contains(pattern, "/") {
				if matched, _ := path.Match(pattern, base); matched {
					return true
				}
			}
			continue
		}

		// Exact full path match for path-based patterns.
		if strings.Contains(pattern, "/") {
			if filePath == pattern {
				return true
			}
			continue
		}

		// Basename match for filename-only patterns (e.g., "Dockerfile", "package.json").
		if base == pattern {
			return true
		}
	}
	return false
}
