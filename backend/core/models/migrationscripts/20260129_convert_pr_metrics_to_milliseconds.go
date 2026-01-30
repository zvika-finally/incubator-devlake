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

	return db.Exec(`
		UPDATE project_pr_metrics
		SET pr_coding_time = pr_coding_time * 60000,
		    pr_pickup_time = pr_pickup_time * 60000,
		    pr_review_time = pr_review_time * 60000,
		    pr_deploy_time = pr_deploy_time * 60000,
		    pr_cycle_time = pr_cycle_time * 60000
		WHERE pr_coding_time IS NOT NULL
		   OR pr_pickup_time IS NOT NULL
		   OR pr_review_time IS NOT NULL
		   OR pr_deploy_time IS NOT NULL
		   OR pr_cycle_time IS NOT NULL
	`)
}

func (*convertPrMetricsToMilliseconds) Version() uint64 {
	return 20260129150000
}

func (*convertPrMetricsToMilliseconds) Name() string {
	return "convert project_pr_metrics time fields from minutes to milliseconds"
}
