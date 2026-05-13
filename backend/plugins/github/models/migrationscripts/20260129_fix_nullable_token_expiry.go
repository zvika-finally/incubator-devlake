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
)

type fixNullableTokenExpiry struct{}

func (*fixNullableTokenExpiry) Up(basicRes context.BasicRes) errors.Error {
	db := basicRes.GetDal()

	// For MySQL: Modify columns to allow NULL and convert zero dates to NULL
	// For PostgreSQL: These columns already accept NULL, but we ensure consistency

	// Check if we're using MySQL by trying a MySQL-specific operation
	err := db.Exec(`
		ALTER TABLE _tool_github_connections
		MODIFY COLUMN token_expires_at DATETIME NULL,
		MODIFY COLUMN refresh_token_expires_at DATETIME NULL
	`)
	if err != nil {
		// If MySQL syntax fails, try PostgreSQL syntax
		err = db.Exec(`
			ALTER TABLE _tool_github_connections
			ALTER COLUMN token_expires_at DROP NOT NULL,
			ALTER COLUMN refresh_token_expires_at DROP NOT NULL
		`)
		// Ignore error if columns are already nullable
		if err != nil {
			basicRes.GetLogger().Warn(err, "columns may already be nullable, continuing")
		}
	}

	// Update any zero dates to NULL (MySQL specific)
	_ = db.Exec(`
		UPDATE _tool_github_connections
		SET token_expires_at = NULL
		WHERE token_expires_at = '0000-00-00 00:00:00'
	`)
	_ = db.Exec(`
		UPDATE _tool_github_connections
		SET refresh_token_expires_at = NULL
		WHERE refresh_token_expires_at = '0000-00-00 00:00:00'
	`)

	return nil
}

func (*fixNullableTokenExpiry) Version() uint64 {
	return 20260129000001
}

func (*fixNullableTokenExpiry) Name() string {
	return "fix nullable token_expires_at and refresh_token_expires_at for MySQL compatibility"
}
