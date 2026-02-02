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
	"math"
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/findevops/models"
)

var CollectDeveloperActivityMeta = plugin.SubTaskMeta{
	Name:             "collectDeveloperActivity",
	EntryPoint:       CollectDeveloperActivity,
	EnabledByDefault: true,
	Description:      "Collect developer activity signals for FTE calculation (Swarmia model)",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET, plugin.DOMAIN_TYPE_CODE},
}

type ActivityWeights struct {
	PrAuthored     float64
	PrReviewed     float64
	CommitAuthored float64
	IssueUpdated   float64
	CommentAdded   float64
}

type DeveloperActivity struct {
	DeveloperId     string
	FiscalMonth     string
	PrsAuthored     int
	PrsReviewed     int
	CommitsAuthored int
	IssuesUpdated   int
	CommentsAdded   int
}

func CollectDeveloperActivity(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*FinDevOpsTaskData)
	logger := taskCtx.GetLogger()
	settings := data.Settings

	if !settings.EnableFteNormalization {
		logger.Info("FTE normalization disabled, skipping collectDeveloperActivity")
		return nil
	}

	logger.Info("Starting collectDeveloperActivity for project: %s", data.Options.ProjectName)

	weights := &ActivityWeights{
		PrAuthored:     settings.ActivityWeightPrAuthored,
		PrReviewed:     settings.ActivityWeightPrReviewed,
		CommitAuthored: settings.ActivityWeightCommitAuthored,
		IssueUpdated:   settings.ActivityWeightIssueUpdated,
		CommentAdded:   settings.ActivityWeightCommentAdded,
	}

	months, err := getDistinctFiscalMonths(db, data.Options.ProjectName)
	if err != nil {
		return err
	}

	for _, month := range months {
		activities, err := collectMonthlyActivities(db, data.Options.ProjectName, month)
		if err != nil {
			logger.Error(err, "failed to collect activities for month %s", month)
			continue
		}

		baselineScore := calculateBaselineScore(activities, weights, settings.FteBaselineMultiplier)

		for _, activity := range activities {
			rawScore := calculateActivityScore(
				activity.PrsAuthored,
				activity.PrsReviewed,
				activity.CommitsAuthored,
				activity.IssuesUpdated,
				activity.CommentsAdded,
				weights,
			)

			rawFte := calculateFte(rawScore, baselineScore, settings.FteMaxPerMonth)

			inactiveDays := 0
			adjustedFte := adjustFteForInactivity(rawFte, inactiveDays, settings.FteInactivityThresholdDays)

			fte := &models.DeveloperMonthlyFte{
				Id:               fmt.Sprintf("%s:%s", activity.DeveloperId, month),
				DeveloperId:      activity.DeveloperId,
				FiscalMonth:      month,
				ProjectName:      data.Options.ProjectName,
				PrsAuthored:      activity.PrsAuthored,
				PrsReviewed:      activity.PrsReviewed,
				CommitsAuthored:  activity.CommitsAuthored,
				IssuesUpdated:    activity.IssuesUpdated,
				CommentsAdded:    activity.CommentsAdded,
				RawActivityScore: rawScore,
				BaselineScore:    baselineScore,
				RawFte:           rawFte,
				InactiveDays:     inactiveDays,
				AdjustedFte:      adjustedFte,
				CalculatedAt:     time.Now(),
			}

			if err := db.CreateOrUpdate(fte); err != nil {
				logger.Error(err, "failed to save FTE for developer %s month %s", activity.DeveloperId, month)
			}
		}
	}

	logger.Info("Completed collectDeveloperActivity")
	return nil
}

func getDistinctFiscalMonths(db dal.Dal, projectName string) ([]string, errors.Error) {
	var months []string
	err := db.All(&months,
		dal.Select("DISTINCT DATE_FORMAT(resolution_date, '%Y-%m') as fiscal_month"),
		dal.From("issues"),
		dal.Join("LEFT JOIN board_issues bi ON bi.issue_id = issues.id"),
		dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'boards' AND pm.row_id = bi.board_id"),
		dal.Where("pm.project_name = ? AND resolution_date IS NOT NULL", projectName),
	)
	if err != nil {
		return nil, errors.Default.Wrap(err, "failed to get distinct fiscal months")
	}
	return months, nil
}

func collectMonthlyActivities(db dal.Dal, projectName string, fiscalMonth string) ([]DeveloperActivity, errors.Error) {
	var activities []DeveloperActivity

	type commitCount struct {
		AuthorId string
		Count    int
	}
	var commits []commitCount
	_ = db.All(&commits,
		dal.Select("author_id, COUNT(*) as count"),
		dal.From("commits"),
		dal.Where("DATE_FORMAT(authored_date, '%Y-%m') = ?", fiscalMonth),
		dal.Groupby("author_id"),
	)

	type prCount struct {
		AuthorId string
		Count    int
	}
	var prsAuthored []prCount
	_ = db.All(&prsAuthored,
		dal.Select("author_id, COUNT(*) as count"),
		dal.From("pull_requests"),
		dal.Where("DATE_FORMAT(created_date, '%Y-%m') = ?", fiscalMonth),
		dal.Groupby("author_id"),
	)

	activityMap := make(map[string]*DeveloperActivity)
	for _, c := range commits {
		if c.AuthorId == "" {
			continue
		}
		if _, exists := activityMap[c.AuthorId]; !exists {
			activityMap[c.AuthorId] = &DeveloperActivity{
				DeveloperId: c.AuthorId,
				FiscalMonth: fiscalMonth,
			}
		}
		activityMap[c.AuthorId].CommitsAuthored = c.Count
	}

	for _, pr := range prsAuthored {
		if pr.AuthorId == "" {
			continue
		}
		if _, exists := activityMap[pr.AuthorId]; !exists {
			activityMap[pr.AuthorId] = &DeveloperActivity{
				DeveloperId: pr.AuthorId,
				FiscalMonth: fiscalMonth,
			}
		}
		activityMap[pr.AuthorId].PrsAuthored = pr.Count
	}

	for _, activity := range activityMap {
		activities = append(activities, *activity)
	}

	return activities, nil
}

func calculateActivityScore(prsAuthored, prsReviewed, commits, issuesUpdated, comments int, weights *ActivityWeights) float64 {
	return float64(prsAuthored)*weights.PrAuthored +
		float64(prsReviewed)*weights.PrReviewed +
		float64(commits)*weights.CommitAuthored +
		float64(issuesUpdated)*weights.IssueUpdated +
		float64(comments)*weights.CommentAdded
}

func calculateBaselineScore(activities []DeveloperActivity, weights *ActivityWeights, multiplier float64) float64 {
	if len(activities) == 0 {
		return 25.0
	}

	scores := make([]float64, len(activities))
	for i, activity := range activities {
		scores[i] = calculateActivityScore(
			activity.PrsAuthored,
			activity.PrsReviewed,
			activity.CommitsAuthored,
			activity.IssuesUpdated,
			activity.CommentsAdded,
			weights,
		)
	}

	median := calculateMedian(scores)
	return median * multiplier
}

func calculateMedian(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j] > sorted[j+1] {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}

func calculateFte(rawScore, baselineScore, maxFte float64) float64 {
	if baselineScore <= 0 {
		return 0
	}
	fte := rawScore / baselineScore
	return math.Min(fte, maxFte)
}

func adjustFteForInactivity(rawFte float64, inactiveDays, threshold int) float64 {
	if inactiveDays <= threshold {
		return rawFte
	}
	workingDays := 20
	activeDays := workingDays - inactiveDays
	if activeDays <= 0 {
		return 0
	}
	return rawFte * float64(activeDays) / float64(workingDays)
}
