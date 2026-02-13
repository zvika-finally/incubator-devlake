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
	"github.com/apache/incubator-devlake/core/log"
	"github.com/apache/incubator-devlake/core/models/domainlayer/code"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/aidetector/models"
)

var AnalyzePRCharacteristicsMeta = plugin.SubTaskMeta{
	Name:             "analyzePRCharacteristics",
	EntryPoint:       AnalyzePRCharacteristics,
	EnabledByDefault: true,
	Description:      "Analyze PR size and velocity characteristics to detect AI patterns",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CODE},
}

func AnalyzePRCharacteristics(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*AIDetectorTaskData)
	logger := taskCtx.GetLogger()

	logger.Info("Starting analyzePRCharacteristics for project: %s", data.Options.ProjectName)

	// First, calculate developer baselines
	baselines, err := calculateDeveloperBaselines(db, data.Options.ProjectName, logger)
	if err != nil {
		return errors.Default.Wrap(err, "failed to calculate developer baselines")
	}

	logger.Info("Calculated baselines for %d developers", len(baselines))

	// Now analyze each PR against its author's baseline
	var signals []models.AIUsageSignal
	err = db.All(&signals,
		dal.Select("ai_usage_signals.*"),
		dal.From(&models.AIUsageSignal{}),
		dal.Join("LEFT JOIN pull_requests pr ON pr.id = ai_usage_signals.pull_request_id"),
		dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'repos' AND pm.row_id = pr.base_repo_id"),
		dal.Where("pm.project_name = ?", data.Options.ProjectName),
	)
	if err != nil {
		return errors.Default.Wrap(err, "failed to query existing signals")
	}

	for _, signal := range signals {
		// Get the PR
		var pr code.PullRequest
		if err := db.First(&pr, dal.Where("id = ?", signal.PullRequestId)); err != nil {
			continue
		}

		// Get author baseline
		baseline, exists := baselines[pr.AuthorId]
		if !exists {
			continue
		}

		// Calculate PR size score (AI-assisted PRs are typically 18% larger)
		// Max 20 points
		signal.PRSizeScore = calculatePRSizeScore(pr.Additions, baseline.AvgPRAdditions)

		// Calculate lines per minute (if we have cycle time)
		if signal.CycleTimeHours > 0 {
			signal.LinesPerMinute = float64(pr.Additions) / (signal.CycleTimeHours * 60)
			// Max 25 points for high lines/minute
			signal.LinesPerMinuteScore = calculateLinesPerMinuteScore(signal.LinesPerMinute)
		}

		// Calculate velocity multiplier
		if baseline.AvgCycleTimeHours > 0 && signal.CycleTimeHours > 0 {
			signal.VelocityMultiplier = baseline.AvgCycleTimeHours / signal.CycleTimeHours
		}

		// Update the signal
		if err := db.Update(&signal); err != nil {
			logger.Error(err, "failed to update signal for PR %s", signal.PullRequestId)
		}
	}

	logger.Info("Completed analyzePRCharacteristics")
	return nil
}

func calculateDeveloperBaselines(db dal.Dal, projectName string, logger log.Logger) (map[string]*models.DeveloperBaseline, error) {
	baselines := make(map[string]*models.DeveloperBaseline)

	// Query aggregated stats per developer
	// Use PostgreSQL-compatible syntax for time difference
	rows, err := db.Cursor(
		dal.Select(`
			pr.author_id,
			AVG(pr.additions) as avg_additions,
			AVG(pr.deletions) as avg_deletions,
			AVG((UNIX_TIMESTAMP(pr.merged_date) - UNIX_TIMESTAMP(pr.created_date)) / 3600) as avg_cycle_hours,
			COUNT(*) as pr_count
		`),
		dal.From("pull_requests pr"),
		dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'repos' AND pm.row_id = pr.base_repo_id"),
		dal.Where("pm.project_name = ? AND pr.merged_date IS NOT NULL", projectName),
		dal.Groupby("pr.author_id"),
		dal.Having("COUNT(*) >= 5"), // Need at least 5 PRs for meaningful baseline
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var authorId string
		var avgAdditions, avgDeletions, avgCycleHours float64
		var prCount int

		if err := rows.Scan(&authorId, &avgAdditions, &avgDeletions, &avgCycleHours, &prCount); err != nil {
			logger.Error(errors.Convert(err), "failed to scan baseline row")
			continue
		}

		baseline := &models.DeveloperBaseline{
			Id:                authorId,
			DeveloperId:       authorId,
			AvgPRAdditions:    avgAdditions,
			AvgPRDeletions:    avgDeletions,
			AvgCycleTimeHours: avgCycleHours,
			PRCount:           prCount,
			CalculatedAt:      time.Now(),
		}
		baselines[authorId] = baseline

		// Save baseline
		if err := db.CreateOrUpdate(baseline); err != nil {
			logger.Error(err, "failed to save baseline for developer %s", authorId)
		}
	}

	return baselines, nil
}

func calculatePRSizeScore(additions int, baselineAdditions float64) int {
	if baselineAdditions == 0 {
		return 0
	}

	ratio := float64(additions) / baselineAdditions

	// AI-assisted PRs are typically 15-20% larger (Jellyfish data shows 18%)
	// Score based on how much larger than baseline
	switch {
	case ratio > 1.5: // 50%+ larger
		return 20
	case ratio > 1.25: // 25%+ larger
		return 15
	case ratio > 1.15: // 15%+ larger (the AI threshold)
		return 10
	default:
		return 0
	}
}

func calculateLinesPerMinuteScore(linesPerMin float64) int {
	// High code production rate suggests AI assistance
	// Based on research, AI-assisted coding can produce 20+ lines/min
	switch {
	case linesPerMin > 30:
		return 25
	case linesPerMin > 20:
		return 20
	case linesPerMin > 10:
		return 10
	default:
		return 0
	}
}
