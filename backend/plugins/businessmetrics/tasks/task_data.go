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
	"github.com/apache/incubator-devlake/plugins/businessmetrics/models"
)

type BusinessMetricsOptions struct {
	ProjectName string `json:"projectName"`
	// Label prefix for investment category (e.g., "investment:" matches "investment:ktlo")
	InvestmentLabelPrefix string `json:"investmentLabelPrefix"`
	// Label prefix for development stage (e.g., "stage:" matches "stage:development")
	StageLabelPrefix string `json:"stageLabelPrefix"`
}

type BusinessMetricsTaskData struct {
	Options  *BusinessMetricsOptions
	Settings *models.BusinessMetricsSettings
}

func DecodeAndValidateTaskOptions(options map[string]interface{}) (*BusinessMetricsOptions, errors.Error) {
	var op BusinessMetricsOptions
	err := helper.Decode(options, &op, nil)
	if err != nil {
		return nil, errors.Default.Wrap(err, "error decoding businessmetrics task options")
	}
	// Set defaults
	if op.InvestmentLabelPrefix == "" {
		op.InvestmentLabelPrefix = "investment:"
	}
	if op.StageLabelPrefix == "" {
		op.StageLabelPrefix = "stage:"
	}
	return &op, nil
}
