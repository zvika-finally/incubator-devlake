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
	"github.com/apache/incubator-devlake/helpers/migrationhelper"
)

// addClaudeCodeTokenCost adds token usage and cost columns to the user activity table.
// These are populated only on the console/usage_report collection path.
type addClaudeCodeTokenCost struct{}

func (*addClaudeCodeTokenCost) Up(basicRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&claudeCodeUserActivity20260721{},
	)
}

type claudeCodeUserActivity20260721 struct {
	InputTokens         int64  `json:"inputTokens"`
	OutputTokens        int64  `json:"outputTokens"`
	CacheReadTokens     int64  `json:"cacheReadTokens"`
	CacheCreationTokens int64  `json:"cacheCreationTokens"`
	EstimatedCostCents  int64  `json:"estimatedCostCents"`
	CostCurrency        string `gorm:"type:varchar(10)" json:"costCurrency"`
}

func (claudeCodeUserActivity20260721) TableName() string {
	return "_tool_claude_code_user_activity"
}

func (*addClaudeCodeTokenCost) Version() uint64 {
	return 20260721000000
}

func (*addClaudeCodeTokenCost) Name() string {
	return "add token usage and cost columns to claude_code user activity"
}
