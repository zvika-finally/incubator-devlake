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
	"fmt"
	"strings"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/findevops/models"
)

var CategorizeCapitalizationMeta = plugin.SubTaskMeta{
	Name:             "categorizeCapitalization",
	EntryPoint:       CategorizeCapitalization,
	EnabledByDefault: true,
	Description:      "Categorize costs per US GAAP ASC 350-40 three-stage model",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

// ASC 350-40 Stage Categories
const (
	StagePreliminary       = "preliminary"
	StageDevelopment       = "development"
	StagePostImplementation = "post_implementation"

	CategoryCapitalizable = "capitalizable"
	CategoryExpense       = "expense"
)

// Labels that indicate preliminary stage (research/planning)
var preliminaryLabels = []string{
	"research", "spike", "investigation", "feasibility",
	"discovery", "poc", "proof-of-concept", "planning",
}

// Labels that indicate post-implementation stage (maintenance)
var postImplLabels = []string{
	"bug", "hotfix", "maintenance", "ktlo", "support",
	"incident", "fix", "patch", "tech-debt",
}

// Issue types that indicate different stages
var preliminaryTypes = []string{"Spike", "Research", "Discovery"}
var postImplTypes = []string{"Bug", "Defect", "Hotfix", "Support", "Incident"}
var developmentTypes = []string{
	"Story", "Feature", "Enhancement", "Epic",
	"Requirement", "REQUIREMENT", "Task", "Sub-task", "Subtask",
	"Improvement", "New Feature", "Change Request",
}

func CategorizeCapitalization(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*FinDevOpsTaskData)
	logger := taskCtx.GetLogger()

	logger.Info("Starting categorizeCapitalization for project: %s", data.Options.ProjectName)

	// Get all cost allocations
	var allocations []models.CostAllocation
	if err := db.All(&allocations, dal.From(&models.CostAllocation{})); err != nil {
		return errors.Default.Wrap(err, "failed to query cost allocations")
	}

	logger.Info("Categorizing %d cost allocations", len(allocations))

	// Track totals for summary
	var totalCost, capitalizableCost, expenseCost float64
	var preliminaryCost, developmentCost, postImplCost float64

	for i := range allocations {
		allocation := &allocations[i]

		// Determine stage and capitalization based on ASC 350-40 rules
		stage, category, reason := categorizeWork(allocation.IssueType, allocation.IssueLabels)

		allocation.ProjectPhase = stage
		allocation.CapitalizationCategory = category
		allocation.CategoryReason = reason

		if category == CategoryCapitalizable {
			allocation.CapitalizationPercent = 100
			capitalizableCost += allocation.TotalCost
		} else {
			allocation.CapitalizationPercent = 0
			expenseCost += allocation.TotalCost
		}

		totalCost += allocation.TotalCost

		// Track by phase
		switch stage {
		case StagePreliminary:
			preliminaryCost += allocation.TotalCost
		case StageDevelopment:
			developmentCost += allocation.TotalCost
		case StagePostImplementation:
			postImplCost += allocation.TotalCost
		}

		if err := db.Update(allocation); err != nil {
			logger.Error(err, "failed to update allocation %s", allocation.Id)
		}
	}

	// Regenerate all monthly summaries now that categorization is complete
	// This recalculates capitalizable/expense costs based on updated ProjectPhase values
	if err := generateMonthlySummaries(db, data.Options.ProjectName, logger); err != nil {
		return errors.Default.Wrap(err, "failed to regenerate monthly summaries after categorization")
	}

	logger.Info("Completed categorizeCapitalization: Total=$%.2f, Capitalizable=$%.2f, Expense=$%.2f",
		totalCost, capitalizableCost, expenseCost)
	return nil
}

func categorizeWork(issueType, labels string) (stage, category, reason string) {
	labelsLower := strings.ToLower(labels)

	// Rule 1: Check for preliminary stage indicators (EXPENSE)
	for _, label := range preliminaryLabels {
		if strings.Contains(labelsLower, label) {
			return StagePreliminary, CategoryExpense,
				fmt.Sprintf("Preliminary stage: label contains '%s'", label)
		}
	}
	for _, t := range preliminaryTypes {
		if strings.EqualFold(issueType, t) {
			return StagePreliminary, CategoryExpense,
				fmt.Sprintf("Preliminary stage: issue type is '%s'", issueType)
		}
	}

	// Rule 2: Check for post-implementation stage indicators (EXPENSE)
	for _, label := range postImplLabels {
		if strings.Contains(labelsLower, label) {
			return StagePostImplementation, CategoryExpense,
				fmt.Sprintf("Post-implementation: label contains '%s'", label)
		}
	}
	for _, t := range postImplTypes {
		if strings.EqualFold(issueType, t) {
			return StagePostImplementation, CategoryExpense,
				fmt.Sprintf("Post-implementation: issue type is '%s'", issueType)
		}
	}

	// Rule 3: Check for development stage indicators (CAPITALIZABLE)
	for _, t := range developmentTypes {
		if strings.EqualFold(issueType, t) {
			return StageDevelopment, CategoryCapitalizable,
				fmt.Sprintf("Application development: issue type is '%s' (new functionality)", issueType)
		}
	}

	// Rule 4: Check for stage labels (user-defined)
	if strings.Contains(labelsLower, "stage:development") {
		return StageDevelopment, CategoryCapitalizable,
			"Application development: stage:development label"
	}
	if strings.Contains(labelsLower, "stage:maintenance") {
		return StagePostImplementation, CategoryExpense,
			"Post-implementation: stage:maintenance label"
	}
	if strings.Contains(labelsLower, "stage:research") {
		return StagePreliminary, CategoryExpense,
			"Preliminary: stage:research label"
	}

	// Default: Unclassified - flag for review (default to expense to be conservative)
	return StagePostImplementation, CategoryExpense,
		"Unclassified work: defaulting to expense (review recommended)"
}
