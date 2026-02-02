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
	"math"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
)

var InferGitEffortMeta = plugin.SubTaskMeta{
	Name:             "inferGitEffort",
	EntryPoint:       InferGitEffort,
	EnabledByDefault: true,
	Description:      "Infer effort from Git activity (git2effort methodology)",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CODE},
	Dependencies:     []*plugin.SubTaskMeta{&CollectDeveloperActivityMeta},
}

type GitInferenceConfig struct {
	ProductiveHoursPerActiveDay float64
	ReviewHoursPerCycle         float64
	CommentsPerReviewCycle      int
	MinHoursPerIssue            float64
	MaxHoursPerIssue            float64
}

type commitInfo struct {
	CommitSha    string
	AuthoredDate string
	Additions    int
	Deletions    int
}

type prInfo struct {
	PullRequestId string
	MergedDate    string
	Additions     int
	Deletions     int
}

type GitEffortResult struct {
	IssueId          string
	CodingHours      float64
	ReviewHours      float64
	ComplexityFactor float64
	ActiveDays       int
	TotalHours       float64
	CommitShas       []string
	PrIds            []string
}

var gitEffortCache = make(map[string]*GitEffortResult)

func InferGitEffort(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*FinDevOpsTaskData)
	logger := taskCtx.GetLogger()
	settings := data.Settings

	if !settings.EnableGitEffortInference {
		logger.Info("Git effort inference disabled, skipping")
		return nil
	}

	logger.Info("Starting inferGitEffort for project: %s", data.Options.ProjectName)

	config := &GitInferenceConfig{
		ProductiveHoursPerActiveDay: settings.GitProductiveHoursPerActiveDay,
		ReviewHoursPerCycle:         settings.GitReviewHoursPerCycle,
		CommentsPerReviewCycle:      settings.GitCommentsPerReviewCycle,
		MinHoursPerIssue:            settings.GitMinHoursPerIssue,
		MaxHoursPerIssue:            settings.GitMaxHoursPerIssue,
	}

	gitEffortCache = make(map[string]*GitEffortResult)

	var issueIds []string
	err := db.All(&issueIds,
		dal.Select("issues.id"),
		dal.From("issues"),
		dal.Join("LEFT JOIN board_issues bi ON bi.issue_id = issues.id"),
		dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'boards' AND pm.row_id = bi.board_id"),
		dal.Where("pm.project_name = ?", data.Options.ProjectName),
	)
	if err != nil {
		return errors.Default.Wrap(err, "failed to get issues")
	}

	logger.Info("Inferring git effort for %d issues", len(issueIds))

	for _, issueId := range issueIds {
		result := inferEffortForIssue(db, issueId, config)
		if result != nil {
			gitEffortCache[issueId] = result
		}
	}

	logger.Info("Completed inferGitEffort, cached %d results", len(gitEffortCache))
	return nil
}

func inferEffortForIssue(db dal.Dal, issueId string, config *GitInferenceConfig) *GitEffortResult {
	result := &GitEffortResult{
		IssueId:    issueId,
		CommitShas: []string{},
		PrIds:      []string{},
	}

	var commits []commitInfo
	_ = db.All(&commits,
		dal.Select("c.sha as commit_sha, c.authored_date, c.additions, c.deletions"),
		dal.From("issue_commits ic"),
		dal.Join("JOIN commits c ON c.sha = ic.commit_sha"),
		dal.Where("ic.issue_id = ?", issueId),
	)

	var prs []prInfo
	_ = db.All(&prs,
		dal.Select("pr.id as pull_request_id, pr.merged_date, pr.additions, pr.deletions"),
		dal.From("pull_request_issues pri"),
		dal.Join("JOIN pull_requests pr ON pr.id = pri.pull_request_id"),
		dal.Where("pri.issue_id = ?", issueId),
	)

	activeDays := countActiveDays(commits)
	reviewComments := countReviewComments(db, prs)
	linesChanged, filesChanged := sumChanges(commits, prs, db)

	for _, c := range commits {
		result.CommitShas = append(result.CommitShas, c.CommitSha)
	}
	for _, pr := range prs {
		result.PrIds = append(result.PrIds, pr.PullRequestId)
	}

	result.ActiveDays = activeDays
	result.ComplexityFactor = calculateComplexityFactor(linesChanged, filesChanged)
	result.CodingHours = float64(activeDays) * config.ProductiveHoursPerActiveDay
	result.ReviewHours = float64(reviewComments/config.CommentsPerReviewCycle) * config.ReviewHoursPerCycle
	result.TotalHours = calculateGitInferredHours(activeDays, reviewComments, linesChanged, filesChanged, config)

	return result
}

func countActiveDays(commits []commitInfo) int {
	dates := make(map[string]bool)
	for _, c := range commits {
		if len(c.AuthoredDate) >= 10 {
			dates[c.AuthoredDate[:10]] = true
		}
	}
	return len(dates)
}

func countReviewComments(db dal.Dal, prs []prInfo) int {
	if len(prs) == 0 {
		return 0
	}

	var prIds []string
	for _, pr := range prs {
		prIds = append(prIds, pr.PullRequestId)
	}

	var count int
	_ = db.First(&count,
		dal.Select("COUNT(*)"),
		dal.From("pull_request_comments"),
		dal.Where("pull_request_id IN ?", prIds),
	)
	return count
}

func sumChanges(commits []commitInfo, prs []prInfo, db dal.Dal) (int, int) {
	totalLines := 0
	for _, c := range commits {
		totalLines += c.Additions + c.Deletions
	}

	var filesChanged int
	for _, pr := range prs {
		var count int
		_ = db.First(&count,
			dal.Select("COUNT(DISTINCT file_path)"),
			dal.From("pull_request_commits prc"),
			dal.Join("JOIN commit_files cf ON cf.commit_sha = prc.commit_sha"),
			dal.Where("prc.pull_request_id = ?", pr.PullRequestId),
		)
		filesChanged += count
	}

	return totalLines, filesChanged
}

func calculateGitInferredHours(activeDays, reviewComments, linesChanged, filesChanged int, config *GitInferenceConfig) float64 {
	codingHours := float64(activeDays) * config.ProductiveHoursPerActiveDay

	reviewCycles := float64(reviewComments) / float64(config.CommentsPerReviewCycle)
	reviewHours := reviewCycles * config.ReviewHoursPerCycle

	complexity := calculateComplexityFactor(linesChanged, filesChanged)

	totalHours := (codingHours + reviewHours) * complexity

	totalHours = math.Max(totalHours, config.MinHoursPerIssue)
	totalHours = math.Min(totalHours, config.MaxHoursPerIssue)

	return totalHours
}

func calculateComplexityFactor(linesChanged, filesChanged int) float64 {
	if linesChanged == 0 && filesChanged == 0 {
		return 1.0
	}

	linesFactor := math.Log10(float64(linesChanged+1)) / 2
	filesFactor := math.Sqrt(float64(filesChanged)) / 3

	factor := 1.0 + linesFactor + filesFactor

	factor = math.Max(0.5, factor)
	factor = math.Min(5.0, factor)

	return factor
}

func GetGitEffortForIssue(issueId string) *GitEffortResult {
	return gitEffortCache[issueId]
}

func GetGitEffortAuditTrail(result *GitEffortResult) (commitShas, prIds string) {
	if result == nil {
		return "", ""
	}

	commitBytes, _ := json.Marshal(result.CommitShas)
	prBytes, _ := json.Marshal(result.PrIds)

	return string(commitBytes), string(prBytes)
}
