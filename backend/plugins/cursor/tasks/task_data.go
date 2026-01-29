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
	"github.com/apache/incubator-devlake/plugins/cursor/models"
)

type CursorOptions struct {
	ConnectionId uint64 `json:"connectionId"`
	// Number of days of history to collect (default: 30)
	DaysBack int `json:"daysBack"`
}

type CursorTaskData struct {
	Options    *CursorOptions
	Connection *models.CursorConnection
}

func DecodeAndValidateTaskOptions(options map[string]interface{}) (*CursorOptions, errors.Error) {
	var op CursorOptions
	err := helper.Decode(options, &op, nil)
	if err != nil {
		return nil, errors.Default.Wrap(err, "error decoding cursor task options")
	}
	if op.DaysBack == 0 {
		op.DaysBack = 30 // Default: 30 days of history
	}
	return &op, nil
}
