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
	"github.com/apache/incubator-devlake/plugins/findevops/models"
)

type FinDevOpsOptions struct {
	ProjectName string `json:"projectName"`
	// Default hourly rate if individual rates not available
	DefaultHourlyRate float64 `json:"defaultHourlyRate"`
	// Fiscal month to calculate (e.g., "2026-01")
	FiscalMonth string `json:"fiscalMonth"`
	// Capitalization framework: "asc_350_40_stages" (current) or "asc_350_40_probable" (future)
	CapitalizationFramework string `json:"capitalizationFramework"`
}

// HourlyRateConfig stores developer hourly rates
type HourlyRateConfig struct {
	Engineer       float64 `json:"engineer"`
	SeniorEngineer float64 `json:"seniorEngineer"`
	StaffEngineer  float64 `json:"staffEngineer"`
	Default        float64 `json:"default"`
}

type FinDevOpsTaskData struct {
	Options     *FinDevOpsOptions
	HourlyRates *HourlyRateConfig
	Settings    *models.FinDevOpsSettings
}

func DecodeAndValidateTaskOptions(options map[string]interface{}) (*FinDevOpsOptions, errors.Error) {
	var op FinDevOpsOptions
	err := helper.Decode(options, &op, nil)
	if err != nil {
		return nil, errors.Default.Wrap(err, "error decoding findevops task options")
	}
	// Set defaults
	if op.DefaultHourlyRate == 0 {
		op.DefaultHourlyRate = 87.0 // Default blended rate
	}
	if op.CapitalizationFramework == "" {
		op.CapitalizationFramework = "asc_350_40_stages" // Current standard
	}
	return &op, nil
}
