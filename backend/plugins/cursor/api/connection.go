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
	"github.com/apache/incubator-devlake/plugins/cursor/models"
)

var basicRes context.BasicRes

func Init(br context.BasicRes) {
	basicRes = br
}

// @Summary Create Cursor connection
// @Description Create a new Cursor API connection
// @Tags plugins/cursor
// @Param connection body models.CursorConnection true "Connection data"
// @Success 200 {object} models.CursorConnection
// @Router /plugins/cursor/connections [post]
func PostConnections(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	var connection models.CursorConnection
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

// @Summary Get all Cursor connections
// @Description Get all Cursor API connections
// @Tags plugins/cursor
// @Success 200 {array} models.CursorConnection
// @Router /plugins/cursor/connections [get]
func GetConnections(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	db := basicRes.GetDal()
	var connections []models.CursorConnection
	if err := db.All(&connections, dal.From(&models.CursorConnection{})); err != nil {
		return nil, errors.Default.Wrap(err, "failed to query connections")
	}

	// Mask API keys
	for i := range connections {
		connections[i].ApiKey = "********"
	}

	return &plugin.ApiResourceOutput{
		Body:   connections,
		Status: http.StatusOK,
	}, nil
}

// @Summary Get a Cursor connection
// @Description Get a Cursor API connection by ID
// @Tags plugins/cursor
// @Param connectionId path int true "Connection ID"
// @Success 200 {object} models.CursorConnection
// @Router /plugins/cursor/connections/{connectionId} [get]
func GetConnection(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	connectionId := input.Params["connectionId"]
	if connectionId == "" {
		return nil, errors.BadInput.New("connectionId is required")
	}

	db := basicRes.GetDal()
	var connection models.CursorConnection
	if err := db.First(&connection, dal.Where("id = ?", connectionId)); err != nil {
		return nil, errors.NotFound.Wrap(err, "connection not found")
	}

	// Mask API key
	connection.ApiKey = "********"

	return &plugin.ApiResourceOutput{
		Body:   connection,
		Status: http.StatusOK,
	}, nil
}

// @Summary Update a Cursor connection
// @Description Update a Cursor API connection
// @Tags plugins/cursor
// @Param connectionId path int true "Connection ID"
// @Param connection body models.CursorConnection true "Connection data"
// @Success 200 {object} models.CursorConnection
// @Router /plugins/cursor/connections/{connectionId} [patch]
func PatchConnection(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	connectionId := input.Params["connectionId"]
	if connectionId == "" {
		return nil, errors.BadInput.New("connectionId is required")
	}

	db := basicRes.GetDal()
	var connection models.CursorConnection
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

// @Summary Delete a Cursor connection
// @Description Delete a Cursor API connection
// @Tags plugins/cursor
// @Param connectionId path int true "Connection ID"
// @Success 200 {object} map[string]string
// @Router /plugins/cursor/connections/{connectionId} [delete]
func DeleteConnection(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	connectionId := input.Params["connectionId"]
	if connectionId == "" {
		return nil, errors.BadInput.New("connectionId is required")
	}

	db := basicRes.GetDal()
	if err := db.Delete(&models.CursorConnection{}, dal.Where("id = ?", connectionId)); err != nil {
		return nil, errors.Default.Wrap(err, "failed to delete connection")
	}

	return &plugin.ApiResourceOutput{
		Body:   map[string]string{"status": "deleted"},
		Status: http.StatusOK,
	}, nil
}

// GetConnectionForTask loads a connection for task execution
func GetConnectionForTask(connectionId uint64) (*models.CursorConnection, error) {
	db := basicRes.GetDal()
	var connection models.CursorConnection
	if err := db.First(&connection, dal.Where("id = ?", connectionId)); err != nil {
		return nil, err
	}
	return &connection, nil
}
