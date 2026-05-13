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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCategorizeWork(t *testing.T) {
	tests := []struct {
		name         string
		issueType    string
		labels       string
		wantStage    string
		wantCategory string
	}{
		// POST-IMPLEMENTATION cases (Not capitalizable)
		{
			name:         "Bug issue type",
			issueType:    "Bug",
			labels:       "",
			wantStage:    StagePostImplementation,
			wantCategory: CategoryExpense,
		},
		{
			name:         "Defect issue type",
			issueType:    "Defect",
			labels:       "",
			wantStage:    StagePostImplementation,
			wantCategory: CategoryExpense,
		},
		{
			name:         "Story with maintenance label",
			issueType:    "Story",
			labels:       "maintenance,backend",
			wantStage:    StagePostImplementation,
			wantCategory: CategoryExpense,
		},
		{
			name:         "Task with support label",
			issueType:    "Task",
			labels:       "support",
			wantStage:    StagePostImplementation,
			wantCategory: CategoryExpense,
		},
		{
			name:         "Story with hotfix label",
			issueType:    "Story",
			labels:       "hotfix,urgent",
			wantStage:    StagePostImplementation,
			wantCategory: CategoryExpense,
		},
		{
			name:         "Task with bug label",
			issueType:    "Task",
			labels:       "bug,fix",
			wantStage:    StagePostImplementation,
			wantCategory: CategoryExpense,
		},
		{
			name:         "Incident issue type",
			issueType:    "Incident",
			labels:       "",
			wantStage:    StagePostImplementation,
			wantCategory: CategoryExpense,
		},

		// PRELIMINARY cases (Not capitalizable)
		{
			name:         "Spike issue type",
			issueType:    "Spike",
			labels:       "",
			wantStage:    StagePreliminary,
			wantCategory: CategoryExpense,
		},
		{
			name:         "Research issue type",
			issueType:    "Research",
			labels:       "",
			wantStage:    StagePreliminary,
			wantCategory: CategoryExpense,
		},
		{
			name:         "Story with research label",
			issueType:    "Story",
			labels:       "research",
			wantStage:    StagePreliminary,
			wantCategory: CategoryExpense,
		},
		{
			name:         "Task with poc label",
			issueType:    "Task",
			labels:       "poc",
			wantStage:    StagePreliminary,
			wantCategory: CategoryExpense,
		},
		{
			name:         "Story with discovery label",
			issueType:    "Story",
			labels:       "discovery",
			wantStage:    StagePreliminary,
			wantCategory: CategoryExpense,
		},
		{
			name:         "Task with feasibility label",
			issueType:    "Task",
			labels:       "feasibility",
			wantStage:    StagePreliminary,
			wantCategory: CategoryExpense,
		},

		// DEVELOPMENT cases (Capitalizable)
		{
			name:         "Story issue type",
			issueType:    "Story",
			labels:       "backend,api",
			wantStage:    StageDevelopment,
			wantCategory: CategoryCapitalizable,
		},
		{
			name:         "Feature issue type",
			issueType:    "Feature",
			labels:       "",
			wantStage:    StageDevelopment,
			wantCategory: CategoryCapitalizable,
		},
		{
			name:         "Enhancement issue type",
			issueType:    "Enhancement",
			labels:       "",
			wantStage:    StageDevelopment,
			wantCategory: CategoryCapitalizable,
		},
		{
			name:         "Epic issue type",
			issueType:    "Epic",
			labels:       "",
			wantStage:    StageDevelopment,
			wantCategory: CategoryCapitalizable,
		},
		{
			name:         "Requirement issue type",
			issueType:    "Requirement",
			labels:       "",
			wantStage:    StageDevelopment,
			wantCategory: CategoryCapitalizable,
		},
		{
			name:         "REQUIREMENT uppercase",
			issueType:    "REQUIREMENT",
			labels:       "",
			wantStage:    StageDevelopment,
			wantCategory: CategoryCapitalizable,
		},
		{
			name:         "Task issue type",
			issueType:    "Task",
			labels:       "backend",
			wantStage:    StageDevelopment,
			wantCategory: CategoryCapitalizable,
		},
		{
			name:         "Sub-task issue type",
			issueType:    "Sub-task",
			labels:       "",
			wantStage:    StageDevelopment,
			wantCategory: CategoryCapitalizable,
		},
		{
			name:         "Improvement issue type",
			issueType:    "Improvement",
			labels:       "",
			wantStage:    StageDevelopment,
			wantCategory: CategoryCapitalizable,
		},

		// Stage label overrides
		{
			name:         "Stage development label",
			issueType:    "Task",
			labels:       "stage:development",
			wantStage:    StageDevelopment,
			wantCategory: CategoryCapitalizable,
		},
		{
			name:         "Stage maintenance label",
			issueType:    "Story",
			labels:       "stage:maintenance",
			wantStage:    StagePostImplementation,
			wantCategory: CategoryExpense,
		},
		{
			name:         "Stage research label",
			issueType:    "Task",
			labels:       "stage:research",
			wantStage:    StagePreliminary,
			wantCategory: CategoryExpense,
		},

		// Unclassified defaults to expense
		{
			name:         "Unknown issue type",
			issueType:    "CustomType",
			labels:       "",
			wantStage:    StagePostImplementation,
			wantCategory: CategoryExpense,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStage, gotCategory, gotReason := categorizeWork(tt.issueType, tt.labels)
			assert.Equal(t, tt.wantStage, gotStage, "Stage mismatch for %s", tt.name)
			assert.Equal(t, tt.wantCategory, gotCategory, "Category mismatch for %s", tt.name)
			assert.NotEmpty(t, gotReason, "Reason should not be empty")
		})
	}
}

// Integration test for complete categorization workflow
func TestCompleteCategorization(t *testing.T) {
	testCases := []struct {
		name                  string
		issueType             string
		labels                string
		expectedCapitalizable bool
	}{
		{
			name:                  "New feature story - capitalizable",
			issueType:             "Story",
			labels:                "feature,backend",
			expectedCapitalizable: true,
		},
		{
			name:                  "Requirement - capitalizable",
			issueType:             "REQUIREMENT",
			labels:                "",
			expectedCapitalizable: true,
		},
		{
			name:                  "Bug fix - not capitalizable",
			issueType:             "Bug",
			labels:                "",
			expectedCapitalizable: false,
		},
		{
			name:                  "Research spike - not capitalizable",
			issueType:             "Spike",
			labels:                "research",
			expectedCapitalizable: false,
		},
		{
			name:                  "Maintenance task - not capitalizable",
			issueType:             "Task",
			labels:                "maintenance",
			expectedCapitalizable: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, category, _ := categorizeWork(tc.issueType, tc.labels)
			isCapitalizable := (category == CategoryCapitalizable)
			assert.Equal(t, tc.expectedCapitalizable, isCapitalizable, "Capitalizable flag mismatch")
		})
	}
}
