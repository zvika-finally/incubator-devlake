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

package tasks

import (
	"github.com/apache/incubator-devlake/core/errors"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/aidetector/models"
)

type AIDetectorOptions struct {
	ProjectName string `json:"projectName"`
	// Minimum confidence score to flag as AI-assisted (0-100)
	ConfidenceThreshold int `json:"confidenceThreshold"`
	// Whether to analyze historical PRs (default: true)
	AnalyzeHistorical bool `json:"analyzeHistorical"`
}

type AIDetectorTaskData struct {
	Options  *AIDetectorOptions
	Settings *models.AIDetectorSettings
}

func DecodeAndValidateTaskOptions(options map[string]interface{}) (*AIDetectorOptions, errors.Error) {
	var op AIDetectorOptions
	err := helper.Decode(options, &op, nil)
	if err != nil {
		return nil, errors.Default.Wrap(err, "error decoding aidetector task options")
	}
	// Set defaults
	if op.ConfidenceThreshold == 0 {
		op.ConfidenceThreshold = 70 // Default: 70% confidence to flag as AI-assisted
	}
	return &op, nil
}
