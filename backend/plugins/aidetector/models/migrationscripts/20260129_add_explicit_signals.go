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

var _ plugin.MigrationScript = (*addExplicitSignals)(nil)

type addExplicitSignals struct{}

func (script *addExplicitSignals) Up(basicRes context.BasicRes) errors.Error {
	db := basicRes.GetDal()

	// Add explicit signal columns to ai_usage_signals table
	err := db.AutoMigrate(&aiUsageSignalExplicit20260129{})
	if err != nil {
		return errors.Default.Wrap(err, "failed to add explicit signal columns")
	}

	return nil
}

func (script *addExplicitSignals) Version() uint64 {
	return 20260129000003
}

func (script *addExplicitSignals) Name() string {
	return "aidetector: add explicit AI tool detection columns"
}

// Model with new columns for migration
type aiUsageSignalExplicit20260129 struct {
	ExplicitToolDetected bool   `gorm:"type:bool"`
	ExplicitTools        string `gorm:"type:varchar(255)"`
	ExplicitPatterns     string `gorm:"type:text"`
	ExplicitSignalScore  int    `gorm:"type:int"`
}

func (aiUsageSignalExplicit20260129) TableName() string {
	return "ai_usage_signals"
}
