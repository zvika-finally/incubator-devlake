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

var _ plugin.MigrationScript = (*changeIssueCodeBlockComponentType)(nil)

type changeIssueCodeBlockComponentType struct{}

func (script *changeIssueCodeBlockComponentType) Up(basicRes context.BasicRes) errors.Error {
	db := basicRes.GetDal()
	// Best-effort drop: the index only exists on databases created from an older
	// schema where `component` was an indexed varchar. Databases whose table was
	// created after the column became TEXT never had it, so an unconditional DROP
	// fails with "index doesn't exist" (MySQL 1091). Ignoring a missing-index error
	// is safe — if the index genuinely still exists and can't be dropped, the
	// ModifyColumnType below fails loudly (MySQL can't convert an indexed column to
	// TEXT), so a real problem is never hidden.
	_ = db.DropIndexes("_tool_sonarqube_issue_code_blocks", "idx__tool_sonarqube_issue_code_blocks_component")
	return db.ModifyColumnType("_tool_sonarqube_issue_code_blocks", "component", "text")
}

func (*changeIssueCodeBlockComponentType) Version() uint64 {
	return 20260701000000
}

func (*changeIssueCodeBlockComponentType) Name() string {
	return "change _tool_sonarqube_issue_code_blocks.component type to text"
}
