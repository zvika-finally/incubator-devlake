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

type addSettings struct{}

type aiDetectorSettings20260130 struct {
	Id          uint64 `gorm:"primaryKey;autoIncrement"`
	ProjectName string `gorm:"type:varchar(255);uniqueIndex"`

	// Confidence thresholds (0-100)
	ConfidenceTrailer int `gorm:"default:95"`
	ConfidenceBody    int `gorm:"default:85"`
	ConfidenceGeneric int `gorm:"default:60"`
	ConfidenceEmail   int `gorm:"default:90"`

	// Detection threshold
	DetectionThreshold int `gorm:"default:70"`

	// Scoring weights
	ExplicitSignalWeight   int `gorm:"default:50"`
	BehavioralSignalWeight int `gorm:"default:30"`
	PRPatternWeight        int `gorm:"default:20"`

	// Custom patterns (JSON)
	CustomToolPatterns string `gorm:"type:text"`

	// Analysis options
	AnalyzeHistorical bool `gorm:"default:true"`
}

func (aiDetectorSettings20260130) TableName() string {
	return "_tool_aidetector_settings"
}

func (u *addSettings) Up(baseRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		baseRes,
		&aiDetectorSettings20260130{},
	)
}

func (*addSettings) Version() uint64 {
	return 20260130000001
}

func (*addSettings) Name() string {
	return "aidetector: add settings table for project-scoped configuration"
}
