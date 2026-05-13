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

var ComputeQualityCohortMeta = plugin.SubTaskMeta{
	Name:             "computeQualityCohort",
	EntryPoint:       ComputeQualityCohort,
	EnabledByDefault: true,
	Description:      "Compute defect signals (revert/hotfix/incident) within the configured window per merged PR",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CODE},
}

// hotfixTitleRE matches PR titles that indicate emergency fixes.
var hotfixTitleRE = regexp.MustCompile(`(?i)\b(hotfix|urgent|emergency|emergency-rollback)\b`)

// IsHotfixTitle returns true if a PR title matches the hotfix pattern.
func IsHotfixTitle(title string) bool {
	return hotfixTitleRE.MatchString(title)
}

// FileOverlapRatio returns |a ∩ b| / max(|a|, 1). Used to decide whether a candidate
// hotfix PR touches "enough" of the original PR's files (Phase A threshold: ≥ 0.5).
func FileOverlapRatio(a, b []string) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}
	set := make(map[string]struct{}, len(b))
	for _, f := range b {
		set[f] = struct{}{}
	}
	overlap := 0
	for _, f := range a {
		if _, ok := set[f]; ok {
			overlap++
		}
	}
	return float64(overlap) / float64(len(a))
}

func ComputeQualityCohort(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*AIMeasureTaskData)
	db := taskCtx.GetDal()
	logger := taskCtx.GetLogger()

	// Detect whether an incident table is present. If not, mark IncidentDataAvailable=false.
	incidentDataAvailable, err := tableExists(db, "issues") // 'issues' carries incidents when type='INCIDENT'
	if err != nil {
		logger.Warn(err, "could not detect incident table; assuming unavailable")
	}

	windowDays := data.Options.DefectWindowDays

	type prRow struct {
		Id             string    `gorm:"column:id"`
		MergedDate     time.Time `gorm:"column:merged_date"`
		Title          string    `gorm:"column:title"`
		MergeCommitSha string    `gorm:"column:merge_commit_sha"`
	}
	var prs []prRow
	if err := db.All(&prs,
		dal.Select("id, merged_date, title, merge_commit_sha"),
		dal.From("pull_requests"),
		dal.Where("merged_date IS NOT NULL"),
	); err != nil {
		return errors.Default.Wrap(err, "failed to query merged PRs")
	}

	now := time.Now().UTC()
	count := 0
	for _, pr := range prs {
		windowEnd := pr.MergedDate.Add(time.Duration(windowDays) * 24 * time.Hour)

		hasRevert, err := detectRevert(db, pr.MergeCommitSha, pr.MergedDate, windowEnd)
		if err != nil {
			logger.Warn(err, "revert detection failed for PR %s", pr.Id)
		}
		hasHotfix, err := detectHotfix(db, pr.Id, pr.MergedDate, windowEnd)
		if err != nil {
			logger.Warn(err, "hotfix detection failed for PR %s", pr.Id)
		}
		hasIncident := false
		if incidentDataAvailable {
			hasIncident, err = detectIncident(db, pr.Id, pr.MergedDate, windowEnd)
			if err != nil {
				logger.Warn(err, "incident detection failed for PR %s", pr.Id)
			}
		}

		out := &models.PRDefectSignals{
			PRId:                  pr.Id,
			HasRevert14d:          hasRevert,
			HasHotfix14d:          hasHotfix,
			HasIncident14d:        hasIncident,
			IncidentDataAvailable: incidentDataAvailable,
			TotalDefectCount:      boolToInt(hasRevert) + boolToInt(hasHotfix) + boolToInt(hasIncident),
			WindowCloseDate:       windowEnd,
			ComputedAt:            now,
		}
		if err := db.CreateOrUpdate(out); err != nil {
			return errors.Default.Wrap(err, "failed to upsert pr_defect_signals row")
		}
		count++
	}
	logger.Info("computeQualityCohort processed %d PRs", count)
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func tableExists(db dal.Dal, tableName string) (bool, errors.Error) {
	count, err := db.Count(
		dal.From("information_schema.tables"),
		dal.Where("table_name = ?", tableName),
	)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func detectRevert(db dal.Dal, mergeSha string, after, before time.Time) (bool, errors.Error) {
	if mergeSha == "" {
		return false, nil
	}
	count, err := db.Count(
		dal.From("commits"),
		dal.Where("authored_date >= ? AND authored_date < ? AND message LIKE ?", after, before, "%"+mergeSha+"%"),
	)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func detectHotfix(db dal.Dal, prId string, after, before time.Time) (bool, errors.Error) {
	// Get the original PR's file list
	type filePath struct{ FilePath string }
	var originalFiles []filePath
	err := db.All(&originalFiles,
		dal.Select("DISTINCT cf.file_path AS file_path"),
		dal.From("commit_files cf"),
		dal.Join("INNER JOIN pull_request_commits prc ON prc.commit_sha = cf.commit_sha"),
		dal.Where("prc.pull_request_id = ?", prId),
	)
	if err != nil {
		return false, err
	}
	if len(originalFiles) == 0 {
		return false, nil
	}
	originalSet := make([]string, len(originalFiles))
	for i, f := range originalFiles {
		originalSet[i] = f.FilePath
	}

	// Candidate hotfix PRs: hotfix-titled, merged within window
	type candidate struct {
		Id    string
		Title string
	}
	var candidates []candidate
	err = db.All(&candidates,
		dal.Select("id, title"),
		dal.From("pull_requests"),
		dal.Where("merged_date >= ? AND merged_date < ? AND id != ?", after, before, prId),
	)
	if err != nil {
		return false, err
	}

	for _, c := range candidates {
		if !IsHotfixTitle(c.Title) {
			continue
		}
		var hotfixFiles []filePath
		err = db.All(&hotfixFiles,
			dal.Select("DISTINCT cf.file_path AS file_path"),
			dal.From("commit_files cf"),
			dal.Join("INNER JOIN pull_request_commits prc ON prc.commit_sha = cf.commit_sha"),
			dal.Where("prc.pull_request_id = ?", c.Id),
		)
		if err != nil {
			continue
		}
		hotfixSet := make([]string, len(hotfixFiles))
		for i, f := range hotfixFiles {
			hotfixSet[i] = f.FilePath
		}
		if FileOverlapRatio(originalSet, hotfixSet) >= 0.5 {
			return true, nil
		}
	}
	return false, nil
}

func detectIncident(db dal.Dal, prId string, after, before time.Time) (bool, errors.Error) {
	// Phase A heuristic: any issue of type INCIDENT created within the window is
	// treated as a related incident. Proper PR↔incident linkage (via commit
	// references or branch metadata) is future work in Phase B.
	count, err := db.Count(
		dal.From("issues"),
		dal.Where("type = ? AND created_date >= ? AND created_date < ?", "INCIDENT", after, before),
	)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
