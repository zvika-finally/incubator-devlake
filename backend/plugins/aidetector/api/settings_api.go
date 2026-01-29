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

package api

import (
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
)

// @Summary Get AI Detector settings for a project
// @Description Get the AI detector configuration for a specific project. Returns defaults if not configured.
// @Tags plugins/aidetector
// @Param projectName path string true "Project Name"
// @Success 200 {object} models.AIDetectorSettings
// @Failure 400 {object} shared.ApiBody "Bad Request"
// @Router /plugins/aidetector/settings/{projectName} [get]
func GetSettings(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return settingsHelper.Get(input)
}

// @Summary Create or update AI Detector settings for a project
// @Description Set the AI detector configuration for a specific project
// @Tags plugins/aidetector
// @Param projectName path string true "Project Name"
// @Param body body models.AIDetectorSettings true "Settings"
// @Success 200 {object} models.AIDetectorSettings
// @Failure 400 {object} shared.ApiBody "Bad Request"
// @Router /plugins/aidetector/settings/{projectName} [put]
func PutSettings(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return settingsHelper.CreateOrUpdate(input)
}

// @Summary Delete AI Detector settings for a project
// @Description Delete custom settings for a project, reverting to defaults
// @Tags plugins/aidetector
// @Param projectName path string true "Project Name"
// @Success 200 {object} map[string]string
// @Failure 404 {object} shared.ApiBody "Not Found"
// @Router /plugins/aidetector/settings/{projectName} [delete]
func DeleteSettings(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return settingsHelper.Delete(input)
}

// @Summary List all AI Detector settings
// @Description Get all configured AI detector settings across projects
// @Tags plugins/aidetector
// @Success 200 {array} models.AIDetectorSettings
// @Router /plugins/aidetector/settings [get]
func ListSettings(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return settingsHelper.List(input)
}
