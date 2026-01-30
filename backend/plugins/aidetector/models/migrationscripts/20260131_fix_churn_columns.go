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

package migrationscripts

import (
	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
)

var _ plugin.MigrationScript = (*fixChurnColumns)(nil)

type fixChurnColumns struct{}

func (*fixChurnColumns) Up(basicRes context.BasicRes) errors.Error {
	db := basicRes.GetDal()

	// Add missing columns that AutoMigrate failed to create
	err := db.Exec(`
		ALTER TABLE ai_churn_metrics
		ADD COLUMN IF NOT EXISTS churn_ratio7_days DECIMAL(8,4) DEFAULT NULL
	`)
	if err != nil {
		_ = db.Exec(`ALTER TABLE ai_churn_metrics ADD COLUMN churn_ratio7_days DECIMAL(8,4) DEFAULT NULL`)
	}

	err = db.Exec(`
		ALTER TABLE ai_churn_metrics
		ADD COLUMN IF NOT EXISTS churn_ratio30_days DECIMAL(8,4) DEFAULT NULL
	`)
	if err != nil {
		_ = db.Exec(`ALTER TABLE ai_churn_metrics ADD COLUMN churn_ratio30_days DECIMAL(8,4) DEFAULT NULL`)
	}

	err = db.Exec(`
		ALTER TABLE ai_churn_metrics
		ADD COLUMN IF NOT EXISTS follow_up_commits30 INT DEFAULT 0
	`)
	if err != nil {
		_ = db.Exec(`ALTER TABLE ai_churn_metrics ADD COLUMN follow_up_commits30 INT DEFAULT 0`)
	}

	err = db.Exec(`
		ALTER TABLE ai_churn_metrics
		ADD COLUMN IF NOT EXISTS file_paths TEXT DEFAULT NULL
	`)
	if err != nil {
		_ = db.Exec(`ALTER TABLE ai_churn_metrics ADD COLUMN file_paths TEXT DEFAULT NULL`)
	}

	// Backfill calculated ratios from existing data
	err = db.Exec(`
		UPDATE ai_churn_metrics
		SET churn_ratio7_days = CASE
			WHEN initial_additions > 0 THEN CAST(churn_within7_days AS DECIMAL(8,4)) / initial_additions
			ELSE NULL
		END,
		churn_ratio30_days = CASE
			WHEN initial_additions > 0 THEN CAST(churn_within30_days AS DECIMAL(8,4)) / initial_additions
			ELSE NULL
		END
		WHERE churn_ratio7_days IS NULL OR churn_ratio30_days IS NULL
	`)
	if err != nil {
		return errors.Default.Wrap(err, "failed to backfill churn ratios")
	}

	return nil
}

func (*fixChurnColumns) Version() uint64 {
	return 20260131000006
}

func (*fixChurnColumns) Name() string {
	return "aidetector: fix missing columns in ai_churn_metrics table"
}
