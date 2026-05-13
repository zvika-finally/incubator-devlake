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

var ComputeChangeCompositionMeta = plugin.SubTaskMeta{
	Name:             "computeChangeComposition",
	EntryPoint:       ComputeChangeComposition,
	EnabledByDefault: true,
	Description:      "Compute per-PR batch size, file count, and refactor ratio for change-composition drift tracking",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CODE},
}

// BucketFor returns the batch-size bucket for a total LOC change count.
func BucketFor(loc int) models.BatchBucket {
	switch {
	case loc < 50:
		return models.BucketXS
	case loc < 200:
		return models.BucketS
	case loc < 500:
		return models.BucketM
	case loc < 1000:
		return models.BucketL
	default:
		return models.BucketXL
	}
}

// ComputeRefactorRatio returns refactor_lines / (additive_lines + refactor_lines).
// Returns 0.0 when both are 0 (avoids div-by-zero on empty PRs).
func ComputeRefactorRatio(additive, refactor int) float64 {
	total := additive + refactor
	if total == 0 {
		return 0.0
	}
	return float64(refactor) / float64(total)
}

// Heuristic for additive vs. refactor classification: a file is treated as
// "additive" if it had no deletions across all the PR's commits (suggesting a
// brand-new file); files with any deletions contribute their additions+deletions
// to refactor_lines. For Phase A this approximates pre-existing-file detection
// without needing a separate base-ref diff query. Refined in a later phase.
//
// PR file-level aggregate. Scanned from the cursor.
type prFileAgg struct {
	PRId     string `gorm:"column:pr_id"`
	FilePath string `gorm:"column:file_path"`
	FileAdd  int    `gorm:"column:file_add"`
	FileDel  int    `gorm:"column:file_del"`
}

func ComputeChangeComposition(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	logger := taskCtx.GetLogger()

	// Step 1: enumerate merged PRs (id + total additions/deletions)
	type prTotals struct {
		Id        string `gorm:"column:id"`
		Additions int    `gorm:"column:additions"`
		Deletions int    `gorm:"column:deletions"`
	}
	var prs []prTotals
	if err := db.All(&prs,
		dal.Select("id, additions, deletions"),
		dal.From("pull_requests"),
		dal.Where("merged_date IS NOT NULL"),
	); err != nil {
		return errors.Default.Wrap(err, "failed to enumerate merged PRs")
	}

	now := time.Now().UTC()
	count := 0
	for _, pr := range prs {
		// Step 2: per-PR per-file aggregate of additions/deletions
		var files []prFileAgg
		if err := db.All(&files,
			dal.Select("? AS pr_id, cf.file_path AS file_path, SUM(cf.additions) AS file_add, SUM(cf.deletions) AS file_del", pr.Id),
			dal.From("commit_files cf"),
			dal.Join("INNER JOIN pull_request_commits prc ON prc.commit_sha = cf.commit_sha"),
			dal.Where("prc.pull_request_id = ?", pr.Id),
			dal.Groupby("cf.file_path"),
		); err != nil {
			return errors.Default.Wrap(err, "failed to aggregate commit_files for PR "+pr.Id)
		}

		additive, refactor, fileCount := 0, 0, len(files)
		for _, f := range files {
			if f.FileDel == 0 {
				additive += f.FileAdd
			} else {
				refactor += f.FileAdd + f.FileDel
			}
		}
		totalLOC := pr.Additions + pr.Deletions

		out := &models.PRChangeComposition{
			PRId:          pr.Id,
			Additions:     pr.Additions,
			Deletions:     pr.Deletions,
			FileCount:     fileCount,
			AdditiveLines: additive,
			RefactorLines: refactor,
			RefactorRatio: ComputeRefactorRatio(additive, refactor),
			BatchBucket:   BucketFor(totalLOC),
			ComputedAt:    now,
		}
		if err := db.CreateOrUpdate(out); err != nil {
			return errors.Default.Wrap(err, "failed to upsert pr_change_composition row")
		}
		count++
	}
	logger.Info("computeChangeComposition processed %d PRs", count)
	return nil
}
