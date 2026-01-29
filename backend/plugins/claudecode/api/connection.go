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
	"net/http"

	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/claudecode/models"
)

var basicRes context.BasicRes

func Init(br context.BasicRes) {
	basicRes = br
}

// @Summary Create Claude Code connection
// @Description Create a new Claude Code Admin API connection
// @Tags plugins/claudecode
// @Param connection body models.ClaudeCodeConnection true "Connection data"
// @Success 200 {object} models.ClaudeCodeConnection
// @Router /plugins/claudecode/connections [post]
func PostConnections(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	var connection models.ClaudeCodeConnection
	if err := helper.Decode(input.Body, &connection, nil); err != nil {
		return nil, errors.BadInput.Wrap(err, "invalid request body")
	}

	db := basicRes.GetDal()
	if err := db.Create(&connection); err != nil {
		return nil, errors.Default.Wrap(err, "failed to create connection")
	}

	return &plugin.ApiResourceOutput{
		Body:   connection,
		Status: http.StatusCreated,
	}, nil
}

// @Summary Get all Claude Code connections
// @Description Get all Claude Code Admin API connections
// @Tags plugins/claudecode
// @Success 200 {array} models.ClaudeCodeConnection
// @Router /plugins/claudecode/connections [get]
func GetConnections(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	db := basicRes.GetDal()
	var connections []models.ClaudeCodeConnection
	if err := db.All(&connections, dal.From(&models.ClaudeCodeConnection{})); err != nil {
		return nil, errors.Default.Wrap(err, "failed to query connections")
	}

	// Mask API keys
	for i := range connections {
		connections[i].AdminApiKey = "********"
	}

	return &plugin.ApiResourceOutput{
		Body:   connections,
		Status: http.StatusOK,
	}, nil
}

// @Summary Get a Claude Code connection
// @Description Get a Claude Code Admin API connection by ID
// @Tags plugins/claudecode
// @Param connectionId path int true "Connection ID"
// @Success 200 {object} models.ClaudeCodeConnection
// @Router /plugins/claudecode/connections/{connectionId} [get]
func GetConnection(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	connectionId := input.Params["connectionId"]
	if connectionId == "" {
		return nil, errors.BadInput.New("connectionId is required")
	}

	db := basicRes.GetDal()
	var connection models.ClaudeCodeConnection
	if err := db.First(&connection, dal.Where("id = ?", connectionId)); err != nil {
		return nil, errors.NotFound.Wrap(err, "connection not found")
	}

	// Mask API key
	connection.AdminApiKey = "********"

	return &plugin.ApiResourceOutput{
		Body:   connection,
		Status: http.StatusOK,
	}, nil
}

// @Summary Update a Claude Code connection
// @Description Update a Claude Code Admin API connection
// @Tags plugins/claudecode
// @Param connectionId path int true "Connection ID"
// @Param connection body models.ClaudeCodeConnection true "Connection data"
// @Success 200 {object} models.ClaudeCodeConnection
// @Router /plugins/claudecode/connections/{connectionId} [patch]
func PatchConnection(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	connectionId := input.Params["connectionId"]
	if connectionId == "" {
		return nil, errors.BadInput.New("connectionId is required")
	}

	db := basicRes.GetDal()
	var connection models.ClaudeCodeConnection
	if err := db.First(&connection, dal.Where("id = ?", connectionId)); err != nil {
		return nil, errors.NotFound.Wrap(err, "connection not found")
	}

	if err := helper.Decode(input.Body, &connection, nil); err != nil {
		return nil, errors.BadInput.Wrap(err, "invalid request body")
	}

	if err := db.Update(&connection); err != nil {
		return nil, errors.Default.Wrap(err, "failed to update connection")
	}

	return &plugin.ApiResourceOutput{
		Body:   connection,
		Status: http.StatusOK,
	}, nil
}

// @Summary Delete a Claude Code connection
// @Description Delete a Claude Code Admin API connection
// @Tags plugins/claudecode
// @Param connectionId path int true "Connection ID"
// @Success 200 {object} map[string]string
// @Router /plugins/claudecode/connections/{connectionId} [delete]
func DeleteConnection(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	connectionId := input.Params["connectionId"]
	if connectionId == "" {
		return nil, errors.BadInput.New("connectionId is required")
	}

	db := basicRes.GetDal()
	if err := db.Delete(&models.ClaudeCodeConnection{}, dal.Where("id = ?", connectionId)); err != nil {
		return nil, errors.Default.Wrap(err, "failed to delete connection")
	}

	return &plugin.ApiResourceOutput{
		Body:   map[string]string{"status": "deleted"},
		Status: http.StatusOK,
	}, nil
}

// GetConnectionForTask loads a connection for task execution
func GetConnectionForTask(connectionId uint64) (*models.ClaudeCodeConnection, error) {
	db := basicRes.GetDal()
	var connection models.ClaudeCodeConnection
	if err := db.First(&connection, dal.Where("id = ?", connectionId)); err != nil {
		return nil, err
	}
	return &connection, nil
}
