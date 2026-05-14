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

var ComputeSentimentProxyMeta = plugin.SubTaskMeta{
	Name:             "computeSentimentProxy",
	EntryPoint:       ComputeSentimentProxy,
	EnabledByDefault: true,
	Description:      "Derive behavioral sentiment proxy from per-engineer Slack and verification signals",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

// SentimentScore returns a 0-100 score (higher = healthier) from the three
// behavioral inputs. Each penalty is bounded to its weight; output is clamped
// to [0, 100].
func SentimentScore(afterHoursRatio, reviewToAuthorRatio, messageDropPct float64) float64 {
	score := 100.0
	score -= 40.0 * clamp01(afterHoursRatio)
	score -= 30.0 * clamp01((reviewToAuthorRatio-1.5)/1.5)
	if messageDropPct > 0.30 {
		score -= 10.0
	}
	if score < 0 {
		score = 0
	}
	return score
}

// BadDeveloperDayFlag returns true when score < 50 OR (afterHoursRatio > 0.15 AND messageDropPct > 0.5).
func BadDeveloperDayFlag(score, afterHoursRatio, messageDropPct float64) bool {
	if score < 50 {
		return true
	}
	if afterHoursRatio > 0.15 && messageDropPct > 0.5 {
		return true
	}
	return false
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

// sentimentInput is the per-engineer per-week aggregate joining the two upstream tables.
type sentimentInput struct {
	EngineerId      string    `gorm:"column:engineer_id"`
	PeriodWeek      time.Time `gorm:"column:period_week"`
	AfterHoursRatio float64   `gorm:"column:after_hours_ratio"` // taken from the GENERAL category bucket as a baseline
	ReviewToAuthor  float64   `gorm:"column:review_to_author_ratio"`
	MessageCount    int       `gorm:"column:message_count"`
}

type engWeek struct {
	Week  time.Time
	Input sentimentInput
}

func ComputeSentimentProxy(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	logger := taskCtx.GetLogger()
	now := time.Now().UTC()

	// Pull per-engineer per-week aggregates. Join effort + slack (general category)
	// so we have all three inputs in one cursor.
	var rows []sentimentInput
	if err := db.All(&rows,
		dal.Select("e.engineer_id AS engineer_id, e.period_week AS period_week, COALESCE(s.after_hours_ratio, 0) AS after_hours_ratio, e.review_to_author_ratio AS review_to_author_ratio, COALESCE(s.message_count, 0) AS message_count"),
		dal.From("engineer_verification_effort e"),
		dal.Join("LEFT JOIN engineer_slack_signals s ON s.engineer_id = e.engineer_id AND s.period_week = e.period_week AND s.channel_category = ?", "general"),
	); err != nil {
		return errors.Default.Wrap(err, "failed to join verification+slack")
	}

	// Group by engineer so we can compute WoW message drop.
	byEng := map[string][]engWeek{}
	for _, r := range rows {
		byEng[r.EngineerId] = append(byEng[r.EngineerId], engWeek{Week: r.PeriodWeek, Input: r})
	}

	count := 0
	for eng, weeks := range byEng {
		// Sort ascending by week to compute WoW.
		sortWeeksAsc(weeks)
		var prev *engWeek
		for i := range weeks {
			w := weeks[i]
			drop := 0.0
			if prev != nil {
				drop = wowDrop(prev.Input.MessageCount, w.Input.MessageCount)
			}
			score := SentimentScore(w.Input.AfterHoursRatio, w.Input.ReviewToAuthor, drop)
			flag := BadDeveloperDayFlag(score, w.Input.AfterHoursRatio, drop)
			row := &models.EngineerDxiProxy{
				EngineerId:          eng,
				PeriodWeek:          w.Week,
				SentimentScore:      score,
				BadDeveloperDayFlag: flag,
				ComputedAt:          now,
			}
			if err := db.CreateOrUpdate(row); err != nil {
				return errors.Default.Wrap(err, "failed to upsert engineer_dxi_proxy row")
			}
			count++
			prev = &weeks[i]
		}
	}
	logger.Info("computeSentimentProxy wrote %d (engineer, week) rows", count)
	return nil
}

func sortWeeksAsc(weeks []engWeek) {
	// simple insertion sort; the slice is small per-engineer
	for i := 1; i < len(weeks); i++ {
		for j := i; j > 0 && weeks[j-1].Week.After(weeks[j].Week); j-- {
			weeks[j-1], weeks[j] = weeks[j], weeks[j-1]
		}
	}
}

func wowDrop(prev, curr int) float64 {
	if prev <= 0 {
		return 0
	}
	if curr >= prev {
		return 0
	}
	return float64(prev-curr) / float64(prev)
}
