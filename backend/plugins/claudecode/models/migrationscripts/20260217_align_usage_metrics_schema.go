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
	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
)

type alignUsageMetricsSchema struct{}

func (*alignUsageMetricsSchema) Up(basicRes context.BasicRes) errors.Error {
	db := basicRes.GetDal()
	tableName := "claude_code_usage_metrics"
	if !db.HasTable(tableName) {
		return nil
	}

	addColumnIfMissing := func(columnName, columnType string) errors.Error {
		if db.HasColumn(tableName, columnName) {
			return nil
		}
		return db.AddColumn(tableName, columnName, dal.ColumnType(columnType))
	}
	backfillColumn := func(targetColumn, sourceColumn string) errors.Error {
		if !db.HasColumn(tableName, targetColumn) || !db.HasColumn(tableName, sourceColumn) {
			return nil
		}
		return db.Exec(
			"UPDATE ? SET ? = ? WHERE ? IS NULL AND ? IS NOT NULL",
			dal.ClauseTable{Name: tableName},
			dal.ClauseColumn{Name: targetColumn},
			dal.ClauseColumn{Name: sourceColumn},
			dal.ClauseColumn{Name: targetColumn},
			dal.ClauseColumn{Name: sourceColumn},
		)
	}

	err := addColumnIfMissing("actor_type", "varchar(50)")
	if err != nil {
		return err
	}
	err = addColumnIfMissing("actor_email", "varchar(255)")
	if err != nil {
		return err
	}
	err = addColumnIfMissing("actor_api_key", "varchar(255)")
	if err != nil {
		return err
	}
	err = addColumnIfMissing("customer_type", "varchar(50)")
	if err != nil {
		return err
	}
	err = addColumnIfMissing("terminal_type", "varchar(100)")
	if err != nil {
		return err
	}
	err = addColumnIfMissing("num_sessions", "int")
	if err != nil {
		return err
	}
	err = addColumnIfMissing("commits_by_claude_code", "int")
	if err != nil {
		return err
	}
	err = addColumnIfMissing("pull_requests_by_claude_code", "int")
	if err != nil {
		return err
	}
	err = addColumnIfMissing("edit_tool_accepted", "int")
	if err != nil {
		return err
	}
	err = addColumnIfMissing("edit_tool_rejected", "int")
	if err != nil {
		return err
	}
	err = addColumnIfMissing("multi_edit_tool_accepted", "int")
	if err != nil {
		return err
	}
	err = addColumnIfMissing("multi_edit_tool_rejected", "int")
	if err != nil {
		return err
	}
	err = addColumnIfMissing("write_tool_accepted", "int")
	if err != nil {
		return err
	}
	err = addColumnIfMissing("write_tool_rejected", "int")
	if err != nil {
		return err
	}
	err = addColumnIfMissing("notebook_edit_accepted", "int")
	if err != nil {
		return err
	}
	err = addColumnIfMissing("notebook_edit_rejected", "int")
	if err != nil {
		return err
	}
	err = addColumnIfMissing("edit_tool_acceptance_rate", "decimal(5,2)")
	if err != nil {
		return err
	}
	err = addColumnIfMissing("write_tool_acceptance_rate", "decimal(5,2)")
	if err != nil {
		return err
	}
	err = addColumnIfMissing("overall_acceptance_rate", "decimal(5,2)")
	if err != nil {
		return err
	}
	err = addColumnIfMissing("cache_read_tokens", "bigint")
	if err != nil {
		return err
	}
	err = addColumnIfMissing("cache_creation_tokens", "bigint")
	if err != nil {
		return err
	}
	err = addColumnIfMissing("estimated_cost_cents", "bigint")
	if err != nil {
		return err
	}
	err = addColumnIfMissing("cost_currency", "varchar(10) DEFAULT 'USD'")
	if err != nil {
		return err
	}

	err = backfillColumn("num_sessions", "total_sessions")
	if err != nil {
		return err
	}
	err = backfillColumn("commits_by_claude_code", "commits_created")
	if err != nil {
		return err
	}
	err = backfillColumn("pull_requests_by_claude_code", "p_rs_created")
	if err != nil {
		return err
	}
	err = backfillColumn("overall_acceptance_rate", "acceptance_rate")
	if err != nil {
		return err
	}
	if db.HasColumn(tableName, "cost_currency") {
		err = db.Exec(
			"UPDATE ? SET ? = 'USD' WHERE ? IS NULL OR ? = ''",
			dal.ClauseTable{Name: tableName},
			dal.ClauseColumn{Name: "cost_currency"},
			dal.ClauseColumn{Name: "cost_currency"},
			dal.ClauseColumn{Name: "cost_currency"},
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (*alignUsageMetricsSchema) Version() uint64 {
	return 20260217000007
}

func (*alignUsageMetricsSchema) Name() string {
	return "claudecode: align usage metrics schema with current model"
}
