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
	"strings"

	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
)

var _ plugin.MigrationScript = (*addDetectedAtIndex)(nil)

type addDetectedAtIndex struct{}

func (script *addDetectedAtIndex) Up(basicRes context.BasicRes) errors.Error {
	db := basicRes.GetDal()

	// Table may not exist in partially configured environments.
	if !db.HasColumn("ai_usage_signals", "detected_at") {
		return nil
	}

	err := db.Exec(
		"CREATE INDEX idx_ai_usage_signals_detected_at ON ? (?)",
		dal.ClauseTable{Name: "ai_usage_signals"},
		dal.ClauseColumn{Name: "detected_at"},
	)
	if err != nil {
		// Keep migration idempotent if the index already exists.
		msg := strings.ToLower(err.Error())
		if db.IsDuplicationError(err) || strings.Contains(msg, "already exists") {
			return nil
		}
		return errors.Default.Wrap(err, "failed to create idx_ai_usage_signals_detected_at")
	}

	return nil
}

func (script *addDetectedAtIndex) Version() uint64 {
	return 20260217000009
}

func (script *addDetectedAtIndex) Name() string {
	return "aidetector: add index on ai_usage_signals.detected_at"
}

