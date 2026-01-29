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
	"github.com/apache/incubator-devlake/plugins/capacityplanner/models"
)

type CapacityPlannerOptions struct {
	ProjectName string `json:"projectName"`
	// Number of sprints to use for velocity calculation (default: 6)
	VelocitySprintCount int `json:"velocitySprintCount"`
	// Sprint duration in weeks (default: 2)
	SprintDurationWeeks int `json:"sprintDurationWeeks"`
}

type CapacityPlannerTaskData struct {
	Options  *CapacityPlannerOptions
	Settings *models.CapacityPlannerSettings
}

func DecodeAndValidateTaskOptions(options map[string]interface{}) (*CapacityPlannerOptions, errors.Error) {
	var op CapacityPlannerOptions
	err := helper.Decode(options, &op, nil)
	if err != nil {
		return nil, errors.Default.Wrap(err, "error decoding capacityplanner task options")
	}
	// Set defaults
	if op.VelocitySprintCount == 0 {
		op.VelocitySprintCount = 6
	}
	if op.SprintDurationWeeks == 0 {
		op.SprintDurationWeeks = 2
	}
	return &op, nil
}
