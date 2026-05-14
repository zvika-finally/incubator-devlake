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
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/aimeasure/models"
)

var ComputeVerificationEffortMeta = plugin.SubTaskMeta{
	Name:             "computeVerificationEffort",
	EntryPoint:       ComputeVerificationEffort,
	EnabledByDefault: true,
	Description:      "Aggregate per-engineer per-week authoring vs reviewing effort and AI-cohort comment density",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CODE},
}

// EstimateAuthorMinutes returns the proxy authoring minutes for a PR of the given LOC.
// Proxy: 5 LOC/min, clipped to [10, 240]. Documented in the README.
func EstimateAuthorMinutes(loc int) int {
	m := loc / 5
	if m < 10 {
		return 10
	}
	if m > 240 {
		return 240
	}
	return m
}

// EstimateReviewerMinutes returns the proxy review minutes given the number of
// review comments on the PR. Proxy: 15 + 2*comments, clipped to [10, 120].
func EstimateReviewerMinutes(numComments int) int {
	m := 15 + 2*numComments
	if m < 10 {
		return 10
	}
	if m > 120 {
		return 120
	}
	return m
}

// SafeRatio returns num/denom or 0.0 when denom is 0.
func SafeRatio(num, denom int) float64 {
	if denom == 0 {
		return 0.0
	}
	return float64(num) / float64(denom)
}

// authoredPR is one row from the authored-PR scan.
type authoredPR struct {
	PRId       string    `gorm:"column:pr_id"`
	AuthorId   string    `gorm:"column:author_id"`
	MergedDate time.Time `gorm:"column:merged_date"`
	Additions  int       `gorm:"column:additions"`
	Deletions  int       `gorm:"column:deletions"`
	AICohort   string    `gorm:"column:ai_cohort"`
}

// reviewerActivity is one row from the reviewer-comment scan.
type reviewerActivity struct {
	PRId        string    `gorm:"column:pr_id"`
	ReviewerId  string    `gorm:"column:reviewer_id"`
	CommentedAt time.Time `gorm:"column:created_date"`
	AICohort    string    `gorm:"column:ai_cohort"`
}

// effortBucket is the aggregator key (engineer, week).
type effortBucket struct {
	EngineerId string
	WeekStart  time.Time
}

// effortAgg is the running aggregate for a (engineer, week) bucket.
type effortAgg struct {
	AuthorMinutes            int
	ReviewerMinutes          int
	AuthorLOC                int
	ReviewedLOC              int
	ReviewCommentsTotal      int
	ReviewCommentsHighCohort int
	ReviewedLOCHigh          int
}

func ComputeVerificationEffort(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	logger := taskCtx.GetLogger()
	now := time.Now().UTC()

	buckets := make(map[effortBucket]*effortAgg)
	getOrCreate := func(eng string, week time.Time) *effortAgg {
		key := effortBucket{EngineerId: eng, WeekStart: week}
		if a, ok := buckets[key]; ok {
			return a
		}
		a := &effortAgg{}
		buckets[key] = a
		return a
	}

	// 1. Author-side: walk all merged PRs with author, join to pr_ai_cohort.
	var authored []authoredPR
	if err := db.All(&authored,
		dal.Select("pr.id AS pr_id, pr.author_id AS author_id, pr.merged_date AS merged_date, pr.additions AS additions, pr.deletions AS deletions, COALESCE(c.ai_cohort,'NONE') AS ai_cohort"),
		dal.From("pull_requests pr"),
		dal.Join("LEFT JOIN pr_ai_cohort c ON c.pr_id = pr.id"),
		dal.Where("pr.merged_date IS NOT NULL AND pr.author_id IS NOT NULL"),
	); err != nil {
		return errors.Default.Wrap(err, "failed to load authored PRs")
	}
	for _, a := range authored {
		loc := a.Additions + a.Deletions
		bucket := getOrCreate(a.AuthorId, PeriodWeekStart(a.MergedDate))
		bucket.AuthorMinutes += EstimateAuthorMinutes(loc)
		bucket.AuthorLOC += loc
	}

	// 2. Reviewer-side: walk pull_request_comments with type=REVIEW or DIFF,
	//    grouped per (reviewer, pr) so reviewer effort gets the per-PR comment-count proxy.
	type prc struct {
		PRId        string    `gorm:"column:pr_id"`
		AccountId   string    `gorm:"column:account_id"`
		CreatedDate time.Time `gorm:"column:created_date"`
		AICohort    string    `gorm:"column:ai_cohort"`
		Additions   int       `gorm:"column:additions"`
		Deletions   int       `gorm:"column:deletions"`
	}
	var comments []prc
	if err := db.All(&comments,
		dal.Select("c.pull_request_id AS pr_id, c.account_id AS account_id, c.created_date AS created_date, COALESCE(coh.ai_cohort,'NONE') AS ai_cohort, pr.additions AS additions, pr.deletions AS deletions"),
		dal.From("pull_request_comments c"),
		dal.Join("INNER JOIN pull_requests pr ON pr.id = c.pull_request_id"),
		dal.Join("LEFT JOIN pr_ai_cohort coh ON coh.pr_id = c.pull_request_id"),
		dal.Where("c.account_id IS NOT NULL AND c.account_id != pr.author_id"),
	); err != nil {
		return errors.Default.Wrap(err, "failed to load PR comments")
	}

	// Per (reviewer, pr) aggregation first, so review-minutes uses the per-PR count.
	type rpKey struct {
		Reviewer string
		PR       string
	}
	type rpAgg struct {
		Comments  int
		FirstSeen time.Time
		AICohort  string
		LOC       int
	}
	rp := map[rpKey]*rpAgg{}
	for _, c := range comments {
		k := rpKey{Reviewer: c.AccountId, PR: c.PRId}
		a, ok := rp[k]
		if !ok {
			a = &rpAgg{FirstSeen: c.CreatedDate, AICohort: c.AICohort, LOC: c.Additions + c.Deletions}
			rp[k] = a
		}
		if c.CreatedDate.Before(a.FirstSeen) {
			a.FirstSeen = c.CreatedDate
		}
		a.Comments++
	}
	for k, a := range rp {
		bucket := getOrCreate(k.Reviewer, PeriodWeekStart(a.FirstSeen))
		bucket.ReviewerMinutes += EstimateReviewerMinutes(a.Comments)
		bucket.ReviewedLOC += a.LOC
		bucket.ReviewCommentsTotal += a.Comments
		if a.AICohort == "HIGH" {
			bucket.ReviewCommentsHighCohort += a.Comments
			bucket.ReviewedLOCHigh += a.LOC
		}
	}

	// 3. Write rows.
	count := 0
	for bk, agg := range buckets {
		row := &models.EngineerVerificationEffort{
			EngineerId:               bk.EngineerId,
			PeriodWeek:               bk.WeekStart,
			AuthorMinutes:            agg.AuthorMinutes,
			ReviewerMinutes:          agg.ReviewerMinutes,
			ReviewToAuthorRatio:      SafeRatio(agg.ReviewerMinutes, agg.AuthorMinutes),
			ReviewCommentsTotal:      agg.ReviewCommentsTotal,
			ReviewCommentsPerLoc:     SafeRatio(agg.ReviewCommentsTotal, agg.ReviewedLOC),
			ReviewCommentsHighCohort: agg.ReviewCommentsHighCohort,
			ReviewCommentsPerLocHigh: SafeRatio(agg.ReviewCommentsHighCohort, agg.ReviewedLOCHigh),
			ComputedAt:               now,
		}
		if err := db.CreateOrUpdate(row); err != nil {
			return errors.Default.Wrap(err, "failed to upsert engineer_verification_effort row")
		}
		count++
	}
	logger.Info("computeVerificationEffort wrote %d (engineer, week) buckets", count)
	return nil
}
