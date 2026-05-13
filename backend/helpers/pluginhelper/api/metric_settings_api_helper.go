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
	"fmt"
	"net/http"

	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/log"
	"github.com/apache/incubator-devlake/core/plugin"
)

// MetricSettings is the interface that metric plugin settings must implement
type MetricSettings interface {
	dal.Tabler
	GetProjectName() string
	SetProjectName(name string)
}

// MetricSettingsApiHelper provides CRUD operations for metric plugin settings
// Settings are scoped by project name (not connection ID, since these are metric plugins)
type MetricSettingsApiHelper[S MetricSettings] struct {
	basicRes    context.BasicRes
	db          dal.Dal
	log         log.Logger
	pluginName  string
	newDefaults func() S // Factory function to create settings with default values
}

// NewMetricSettingsApiHelper creates a new helper for metric plugin settings
func NewMetricSettingsApiHelper[S MetricSettings](
	basicRes context.BasicRes,
	pluginName string,
	newDefaults func() S,
) *MetricSettingsApiHelper[S] {
	return &MetricSettingsApiHelper[S]{
		basicRes:    basicRes,
		db:          basicRes.GetDal(),
		log:         basicRes.GetLogger().Nested(fmt.Sprintf("%s_settings", pluginName)),
		pluginName:  pluginName,
		newDefaults: newDefaults,
	}
}

// GetSettings returns settings for a project, or defaults if none exist
func (h *MetricSettingsApiHelper[S]) GetSettings(projectName string) (S, errors.Error) {
	settings := h.newDefaults()
	err := h.db.First(settings, dal.Where("project_name = ?", projectName))
	if err != nil {
		if h.db.IsErrorNotFound(err) {
			// Return defaults with project name set
			settings.SetProjectName(projectName)
			return settings, nil
		}
		return settings, errors.Default.Wrap(err, "failed to get settings")
	}
	return settings, nil
}

// GetOrCreateSettings returns existing settings or creates new ones with defaults
func (h *MetricSettingsApiHelper[S]) GetOrCreateSettings(projectName string) (S, errors.Error) {
	settings := h.newDefaults()
	err := h.db.First(settings, dal.Where("project_name = ?", projectName))
	if err != nil {
		if h.db.IsErrorNotFound(err) {
			// Create new settings with defaults
			settings.SetProjectName(projectName)
			if err := h.db.Create(settings); err != nil {
				return settings, errors.Default.Wrap(err, "failed to create settings")
			}
			h.log.Info("Created default settings for project: %s", projectName)
			return settings, nil
		}
		return settings, errors.Default.Wrap(err, "failed to get settings")
	}
	return settings, nil
}

// Get handles GET /plugins/:plugin/settings/:projectName
func (h *MetricSettingsApiHelper[S]) Get(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	projectName, ok := input.Params["projectName"]
	if !ok || projectName == "" {
		return nil, errors.BadInput.New("projectName is required")
	}

	settings, err := h.GetSettings(projectName)
	if err != nil {
		return nil, err
	}

	return &plugin.ApiResourceOutput{
		Body: settings,
	}, nil
}

// CreateOrUpdate handles PUT /plugins/:plugin/settings/:projectName
func (h *MetricSettingsApiHelper[S]) CreateOrUpdate(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	projectName, ok := input.Params["projectName"]
	if !ok || projectName == "" {
		return nil, errors.BadInput.New("projectName is required")
	}

	// Start with defaults, then apply request body
	settings := h.newDefaults()
	settings.SetProjectName(projectName)

	// Check if settings already exist
	existing := h.newDefaults()
	err := h.db.First(existing, dal.Where("project_name = ?", projectName))
	existingFound := err == nil

	// Decode request body into settings
	if err := DecodeMapStruct(input.Body, settings, false); err != nil {
		return nil, errors.BadInput.Wrap(err, "failed to decode settings")
	}

	// Ensure project name is set correctly (in case body tried to override)
	settings.SetProjectName(projectName)

	if existingFound {
		// Update existing
		if err := h.db.Update(settings); err != nil {
			return nil, errors.Default.Wrap(err, "failed to update settings")
		}
		h.log.Info("Updated settings for project: %s", projectName)
	} else {
		// Create new
		if err := h.db.Create(settings); err != nil {
			return nil, errors.Default.Wrap(err, "failed to create settings")
		}
		h.log.Info("Created settings for project: %s", projectName)
	}

	return &plugin.ApiResourceOutput{
		Status: http.StatusOK,
		Body:   settings,
	}, nil
}

// Delete handles DELETE /plugins/:plugin/settings/:projectName
// This reverts the project to using default settings
func (h *MetricSettingsApiHelper[S]) Delete(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	projectName, ok := input.Params["projectName"]
	if !ok || projectName == "" {
		return nil, errors.BadInput.New("projectName is required")
	}

	settings := h.newDefaults()
	err := h.db.First(settings, dal.Where("project_name = ?", projectName))
	if err != nil {
		if h.db.IsErrorNotFound(err) {
			return nil, errors.NotFound.New("settings not found for project: " + projectName)
		}
		return nil, errors.Default.Wrap(err, "failed to find settings")
	}

	if err := h.db.Delete(settings); err != nil {
		return nil, errors.Default.Wrap(err, "failed to delete settings")
	}

	h.log.Info("Deleted settings for project: %s (reverted to defaults)", projectName)

	return &plugin.ApiResourceOutput{
		Status: http.StatusOK,
		Body:   map[string]string{"message": "Settings deleted, project will use defaults"},
	}, nil
}

// List handles GET /plugins/:plugin/settings
// Returns all configured settings (projects with custom settings)
func (h *MetricSettingsApiHelper[S]) List(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	var settings []S
	if err := h.db.All(&settings); err != nil {
		return nil, errors.Default.Wrap(err, "failed to list settings")
	}

	return &plugin.ApiResourceOutput{
		Body: settings,
	}, nil
}
