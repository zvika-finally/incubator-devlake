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

var _ plugin.MigrationScript = (*convertPrMetricsToMilliseconds)(nil)

type convertPrMetricsToMilliseconds struct{}

func (*convertPrMetricsToMilliseconds) Up(basicRes context.BasicRes) errors.Error {
	// Convert pr_cycle_time and related fields from minutes to milliseconds
	// The change_lead_time_calculator.go now outputs milliseconds instead of minutes
	// This migration converts existing data to match the new format
	// Conversion: minutes * 60 * 1000 = milliseconds (multiply by 60000)
	db := basicRes.GetDal()

	// Safety check: Only convert values that are clearly in minutes (< 1 million)
	// If a value is already in milliseconds (>= 1 million), it won't be converted
	// This prevents double-conversion if the migration runs on already-converted data
	return db.Exec(`
		UPDATE project_pr_metrics
		SET pr_coding_time = CASE WHEN pr_coding_time < 1000000 THEN pr_coding_time * 60000 ELSE pr_coding_time END,
		    pr_pickup_time = CASE WHEN pr_pickup_time < 1000000 THEN pr_pickup_time * 60000 ELSE pr_pickup_time END,
		    pr_review_time = CASE WHEN pr_review_time < 1000000 THEN pr_review_time * 60000 ELSE pr_review_time END,
		    pr_deploy_time = CASE WHEN pr_deploy_time < 1000000 THEN pr_deploy_time * 60000 ELSE pr_deploy_time END,
		    pr_cycle_time = CASE WHEN pr_cycle_time < 1000000 THEN pr_cycle_time * 60000 ELSE pr_cycle_time END
		WHERE (pr_coding_time IS NOT NULL AND pr_coding_time < 1000000)
		   OR (pr_pickup_time IS NOT NULL AND pr_pickup_time < 1000000)
		   OR (pr_review_time IS NOT NULL AND pr_review_time < 1000000)
		   OR (pr_deploy_time IS NOT NULL AND pr_deploy_time < 1000000)
		   OR (pr_cycle_time IS NOT NULL AND pr_cycle_time < 1000000)
	`)
}

func (*convertPrMetricsToMilliseconds) Version() uint64 {
	return 20260129150000
}

func (*convertPrMetricsToMilliseconds) Name() string {
	return "convert project_pr_metrics time fields from minutes to milliseconds"
}
