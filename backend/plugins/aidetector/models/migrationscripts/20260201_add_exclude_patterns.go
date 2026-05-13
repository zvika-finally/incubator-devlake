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

var _ plugin.MigrationScript = (*addExcludePatterns)(nil)

type addExcludePatterns struct{}

func (*addExcludePatterns) Up(basicRes context.BasicRes) errors.Error {
	db := basicRes.GetDal()

	// Add exclude_file_patterns column to settings table
	err := db.Exec(`
		ALTER TABLE _tool_aidetector_settings
		ADD COLUMN IF NOT EXISTS exclude_file_patterns TEXT DEFAULT '[".github/","package.json","package-lock.json","yarn.lock","pnpm-lock.yaml","Dockerfile","docker-compose.yml",".gitignore",".eslintrc","tsconfig.json"]'
	`)
	if err != nil {
		// Try without IF NOT EXISTS for databases that don't support it
		_ = db.Exec(`
			ALTER TABLE _tool_aidetector_settings
			ADD COLUMN exclude_file_patterns TEXT DEFAULT '[".github/","package.json","package-lock.json","yarn.lock","pnpm-lock.yaml","Dockerfile","docker-compose.yml",".gitignore",".eslintrc","tsconfig.json"]'
		`)
	}

	return nil
}

func (*addExcludePatterns) Version() uint64 {
	return 20260201000001
}

func (*addExcludePatterns) Name() string {
	return "aidetector: add exclude_file_patterns to settings table"
}
