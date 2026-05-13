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
	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	bmModels "github.com/apache/incubator-devlake/plugins/businessmetrics/models"
)

func getProjectInitiatives(db dal.Dal, projectName string) ([]bmModels.BusinessInitiative, errors.Error) {
	var initiatives []bmModels.BusinessInitiative
	err := db.All(&initiatives,
		dal.Select("DISTINCT business_initiatives.*"),
		dal.From(&bmModels.BusinessInitiative{}),
		dal.Join("JOIN issues ON issues.epic_key = business_initiatives.jira_epic_key"),
		dal.Join("JOIN board_issues bi ON bi.issue_id = issues.id"),
		dal.Join("JOIN project_mapping pm ON pm.table = 'boards' AND pm.row_id = bi.board_id"),
		dal.Where("pm.project_name = ? AND business_initiatives.status IN ('active', 'planned')", projectName),
	)
	if err != nil {
		return nil, errors.Default.Wrap(err, "failed to query project initiatives")
	}
	return initiatives, nil
}
