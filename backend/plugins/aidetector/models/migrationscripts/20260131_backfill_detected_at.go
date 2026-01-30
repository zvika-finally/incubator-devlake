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

var _ plugin.MigrationScript = (*backfillDetectedAt)(nil)

type backfillDetectedAt struct{}

func (script *backfillDetectedAt) Up(basicRes context.BasicRes) errors.Error {
	db := basicRes.GetDal()

	// Backfill detected_at timestamps to match PR dates instead of batch processing date
	// This enables historical trend analysis in dashboards
	// Only updates records where detected_at is within the last 2 days (recent batch processing)
	err := db.Exec(`
		UPDATE ai_usage_signals s
		JOIN pull_requests pr ON s.pull_request_id = pr.id
		SET s.detected_at = COALESCE(pr.merged_date, pr.created_date)
		WHERE s.detected_at > NOW() - INTERVAL 2 DAY
			AND COALESCE(pr.merged_date, pr.created_date) IS NOT NULL
	`)
	if err != nil {
		return errors.Default.Wrap(err, "failed to backfill detected_at timestamps")
	}

	return nil
}

func (script *backfillDetectedAt) Version() uint64 {
	return 20260131000005
}

func (script *backfillDetectedAt) Name() string {
	return "aidetector: backfill detected_at with PR dates for historical analysis"
}
