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
	"regexp"
	"strings"
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/domainlayer/code"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/aidetector/models"
)

var AnalyzeCommitPatternsMeta = plugin.SubTaskMeta{
	Name:             "analyzeCommitPatterns",
	EntryPoint:       AnalyzeCommitPatterns,
	EnabledByDefault: true,
	Description:      "Analyze commit patterns to detect AI-assisted development (rapid commits, generic messages)",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CODE},
}

// Generic commit message patterns that might indicate AI-generated commits
var genericMessagePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^(fix|add|update|refactor|improve|clean)\s+\w+$`),
	regexp.MustCompile(`(?i)^(wip|work in progress|checkpoint)$`),
	regexp.MustCompile(`(?i)^(minor|small)\s+(fix|change|update)s?$`),
	regexp.MustCompile(`(?i)^changes?$`),
	regexp.MustCompile(`(?i)^updates?$`),
}

func AnalyzeCommitPatterns(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*AIDetectorTaskData)
	logger := taskCtx.GetLogger()

	logger.Info("Starting analyzeCommitPatterns for project: %s", data.Options.ProjectName)

	// Get all PRs for this project
	var prs []code.PullRequest
	clauses := []dal.Clause{
		dal.From(&code.PullRequest{}),
		dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'repos' AND pm.row_id = pull_requests.base_repo_id"),
		dal.Where("pm.project_name = ?", data.Options.ProjectName),
	}

	err := db.All(&prs, clauses...)
	if err != nil {
		return errors.Default.Wrap(err, "failed to query pull requests")
	}

	logger.Info("Analyzing commit patterns for %d PRs", len(prs))

	for _, pr := range prs {
		// Get commits for this PR
		var commits []code.Commit
		commitClauses := []dal.Clause{
			dal.From(&code.Commit{}),
			dal.Join("LEFT JOIN pull_request_commits prc ON prc.commit_sha = commits.sha"),
			dal.Where("prc.pull_request_id = ?", pr.Id),
			dal.Orderby("commits.authored_date ASC"),
		}

		if err := db.All(&commits, commitClauses...); err != nil {
			logger.Error(err, "failed to get commits for PR %s", pr.Id)
			continue
		}

		if len(commits) == 0 {
			continue
		}

		// Calculate commit pattern metrics
		signal := analyzeCommitsForPR(&pr, commits)
		signal.PullRequestId = pr.Id
		signal.Id = pr.Id // Use PR ID as signal ID
		signal.DetectedAt = time.Now()
		signal.CreatedAt = time.Now()

		// Save the partial signal (will be completed by other tasks)
		if err := db.CreateOrUpdate(signal); err != nil {
			logger.Error(err, "failed to save signal for PR %s", pr.Id)
		}
	}

	logger.Info("Completed analyzeCommitPatterns")
	return nil
}

func analyzeCommitsForPR(pr *code.PullRequest, commits []code.Commit) *models.AIUsageSignal {
	signal := &models.AIUsageSignal{
		CommitCount:   len(commits),
		PRAdditions:   pr.Additions,
		PRDeletions:   pr.Deletions,
	}

	// Calculate average time between commits
	if len(commits) > 1 {
		var totalMinutes float64
		for i := 1; i < len(commits); i++ {
			diff := commits[i].AuthoredDate.Sub(commits[i-1].AuthoredDate)
			totalMinutes += diff.Minutes()
		}
		signal.AvgTimeBetweenCommits = totalMinutes / float64(len(commits)-1)
	}

	// Score rapid commits (AI generates code fast, leading to quick successive commits)
	// Max 30 points for rapid commits
	signal.RapidCommitScore = calculateRapidCommitScore(signal.AvgTimeBetweenCommits)

	// Score generic commit messages
	// Max 10 points for generic messages
	genericCount := 0
	for _, commit := range commits {
		if isGenericMessage(commit.Message) {
			genericCount++
		}
	}
	if len(commits) > 0 {
		genericRatio := float64(genericCount) / float64(len(commits))
		signal.GenericMessageScore = int(genericRatio * 10)
	}

	// Calculate cycle time
	if pr.MergedDate != nil && !pr.CreatedDate.IsZero() {
		signal.CycleTimeHours = pr.MergedDate.Sub(pr.CreatedDate).Hours()
	}

	return signal
}

func calculateRapidCommitScore(avgMinutes float64) int {
	if avgMinutes == 0 {
		return 0
	}
	// Very rapid commits (< 5 min avg) = 30 points
	// Rapid commits (5-15 min avg) = 20 points
	// Moderate (15-30 min avg) = 10 points
	// Normal (> 30 min) = 0 points
	switch {
	case avgMinutes < 5:
		return 30
	case avgMinutes < 15:
		return 20
	case avgMinutes < 30:
		return 10
	default:
		return 0
	}
}

func isGenericMessage(message string) bool {
	// Trim and get first line
	message = strings.TrimSpace(message)
	if idx := strings.Index(message, "\n"); idx > 0 {
		message = message[:idx]
	}

	// Check against generic patterns
	for _, pattern := range genericMessagePatterns {
		if pattern.MatchString(message) {
			return true
		}
	}

	// Also check for very short messages (< 10 chars)
	if len(message) < 10 {
		return true
	}

	return false
}
