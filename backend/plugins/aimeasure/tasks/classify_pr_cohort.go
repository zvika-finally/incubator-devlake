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
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/aimeasure/models"
)

var ClassifyPRCohortMeta = plugin.SubTaskMeta{
	Name:             "classifyPRCohort",
	EntryPoint:       ClassifyPRCohort,
	EnabledByDefault: true,
	Description:      "Classify each merged PR into NONE/LOW/MEDIUM/HIGH AI-assistance cohort",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CODE},
}

// ClassifyInput is the pure-function input for cohort classification.
// Extracted so the rule is unit-testable without DB access.
type ClassifyInput struct {
	ConfidenceScore   int
	HasExplicitMarker bool
	HasCommitTrailer  bool
	HighThreshold     int
	LowThreshold      int
}

// Classify applies the cohort decision rules. Pure function.
func Classify(in ClassifyInput) models.AICohort {
	if in.HasExplicitMarker || in.HasCommitTrailer {
		return models.CohortHigh
	}
	if in.ConfidenceScore >= in.HighThreshold {
		return models.CohortMedium
	}
	if in.ConfidenceScore >= in.LowThreshold {
		return models.CohortLow
	}
	return models.CohortNone
}

// commitTrailerRE matches the `Co-authored-by: <tool>` trailer for common AI agents.
var commitTrailerRE = regexp.MustCompile(`(?im)^Co-authored-by:.*(claude|copilot|cursor|devin|github\s*copilot|anthropic)`)

// HasAITrailer returns true if any of the supplied commit messages contains an AI trailer.
func HasAITrailer(commitMessages []string) bool {
	for _, msg := range commitMessages {
		if commitTrailerRE.MatchString(msg) {
			return true
		}
	}
	return false
}

// prCohortInput is the row shape returned by the classify cursor.
type prCohortInput struct {
	PRId              string `gorm:"column:pr_id"`
	ConfidenceScore   int    `gorm:"column:confidence_score"`
	HasExplicitMarker bool   `gorm:"column:has_explicit_marker"`
}

// ResolveClassifiedAt picks the classified_at value to persist for a row.
// If there is no prior row, or any classification dimension has changed
// (cohort, confidence, evidence flags, or classifier version), the
// timestamp is bumped to `now`. Otherwise the existing timestamp is
// preserved so idempotent re-runs don't churn the column — which lets
// downstream consumers filter on classified_at as a real event time
// rather than a heartbeat from the last batch run.
func ResolveClassifiedAt(existing *models.PRAICohort, fresh models.PRAICohort, now time.Time) time.Time {
	if existing == nil {
		return now
	}
	if existing.AICohort != fresh.AICohort ||
		existing.ConfidenceScore != fresh.ConfidenceScore ||
		existing.HasExplicitMarker != fresh.HasExplicitMarker ||
		existing.HasCommitTrailer != fresh.HasCommitTrailer ||
		existing.ClassifierVersion != fresh.ClassifierVersion {
		return now
	}
	return existing.ClassifiedAt
}

// ClassifyPRCohort is the subtask entrypoint. Reads ai_usage_signals + commits,
// writes one pr_ai_cohort row per merged PR. Re-runs are idempotent and
// preserve classified_at when the classification hasn't materially changed
// — see ResolveClassifiedAt.
func ClassifyPRCohort(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*AIMeasureTaskData)
	db := taskCtx.GetDal()
	logger := taskCtx.GetLogger()

	cursor, err := db.Cursor(
		dal.Select("pull_requests.id AS pr_id, COALESCE(ai_usage_signals.ai_confidence_score, 0) AS confidence_score, COALESCE(ai_usage_signals.explicit_tool_detected, FALSE) AS has_explicit_marker"),
		dal.From("pull_requests"),
		dal.Join("LEFT JOIN ai_usage_signals ON ai_usage_signals.pull_request_id = pull_requests.id"),
		dal.Where("pull_requests.merged_date IS NOT NULL"),
	)
	if err != nil {
		return errors.Default.Wrap(err, "failed to query PRs for classification")
	}
	defer cursor.Close()

	now := time.Now().UTC()
	count := 0
	for cursor.Next() {
		var in prCohortInput
		if err := db.Fetch(cursor, &in); err != nil {
			return errors.Default.Wrap(err, "row scan failed")
		}

		// Pull commit messages for the PR
		messages, err := loadCommitMessages(db, in.PRId)
		if err != nil {
			logger.Warn(err, "could not load commits for PR %s; treating as no-trailer", in.PRId)
		}
		hasTrailer := HasAITrailer(messages)

		cohort := Classify(ClassifyInput{
			ConfidenceScore:   in.ConfidenceScore,
			HasExplicitMarker: in.HasExplicitMarker,
			HasCommitTrailer:  hasTrailer,
			HighThreshold:     data.Options.HighCohortThreshold,
			LowThreshold:      data.Options.LowCohortThreshold,
		})

		row := &models.PRAICohort{
			PRId:              in.PRId,
			AICohort:          cohort,
			ConfidenceScore:   in.ConfidenceScore,
			HasExplicitMarker: in.HasExplicitMarker,
			HasCommitTrailer:  hasTrailer,
			ClassifierVersion: ClassifierVersion,
		}

		var existingPtr *models.PRAICohort
		var existing models.PRAICohort
		if err := db.First(&existing, dal.Where("pr_id = ?", in.PRId)); err != nil {
			if !db.IsErrorNotFound(err) {
				return errors.Default.Wrap(err, "failed to load existing pr_ai_cohort row")
			}
		} else {
			existingPtr = &existing
		}
		row.ClassifiedAt = ResolveClassifiedAt(existingPtr, *row, now)

		if err := db.CreateOrUpdate(row); err != nil {
			return errors.Default.Wrap(err, "failed to upsert pr_ai_cohort row")
		}
		count++
	}
	logger.Info("classifyPRCohort processed %d PRs", count)
	return nil
}

// commitMessage is a scan target for the trailer query.
type commitMessage struct {
	Message string `gorm:"column:message"`
}

func loadCommitMessages(db dal.Dal, prId string) ([]string, errors.Error) {
	var rows []commitMessage
	err := db.All(&rows,
		dal.Select("commits.message"),
		dal.From("commits"),
		dal.Join("INNER JOIN pull_request_commits prc ON prc.commit_sha = commits.sha"),
		dal.Where("prc.pull_request_id = ?", prId),
	)
	if err != nil {
		return nil, err
	}
	msgs := make([]string, 0, len(rows))
	for _, r := range rows {
		msgs = append(msgs, r.Message)
	}
	return msgs, nil
}
