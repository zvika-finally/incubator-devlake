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
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/log"
	"github.com/apache/incubator-devlake/core/models/domainlayer/ticket"
	"github.com/apache/incubator-devlake/core/plugin"
	bmModels "github.com/apache/incubator-devlake/plugins/businessmetrics/models"
	"github.com/apache/incubator-devlake/plugins/capacityplanner/models"
)

var ForecastCompletionKanbanMeta = plugin.SubTaskMeta{
	Name:             "forecastCompletionKanban",
	EntryPoint:       ForecastCompletionKanban,
	EnabledByDefault: true,
	Description:      "Forecast initiative completion using Kanban throughput (issue count based)",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

func ForecastCompletionKanban(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*CapacityPlannerTaskData)
	logger := taskCtx.GetLogger()

	logger.Info("Starting forecastCompletionKanban for project: %s", data.Options.ProjectName)

	// Calculate average throughput from recent periods (issues per week)
	avgThroughput, stdDev := calculateAverageThroughput(db, data.Options.ProjectName, logger)
	logger.Info("Average throughput: %.1f issues/week (std dev: %.1f)", avgThroughput, stdDev)

	if avgThroughput == 0 {
		logger.Warn(nil, "No throughput data available, skipping forecasts")
		return nil
	}

	// Get project-scoped initiatives
	initiatives, err := getProjectInitiatives(db, data.Options.ProjectName)
	if err != nil {
		return err
	}

	logger.Info("Forecasting completion for %d initiatives using Kanban metrics", len(initiatives))

	for _, initiative := range initiatives {
		forecast := forecastInitiativeKanban(db, data.Options.ProjectName, initiative, avgThroughput, stdDev, data.Options.SprintDurationWeeks, logger)
		if forecast == nil {
			continue
		}

		if err := db.CreateOrUpdate(forecast); err != nil {
			logger.Error(err, "failed to save forecast for initiative %s", initiative.Name)
		}
	}

	logger.Info("Completed forecastCompletionKanban")
	return nil
}

func calculateAverageThroughput(db dal.Dal, projectName string, logger log.Logger) (float64, float64) {
	// Get throughput data from team_velocities table
	var velocities []struct {
		IssuesCompleted int
	}

	err := db.All(&velocities,
		dal.Select("issues_completed"),
		dal.From("team_velocities"),
		dal.Where("project_name = ? AND issues_completed > 0", projectName),
		dal.Orderby("sprint_end_date DESC"),
		dal.Limit(6), // Last 6 periods
	)

	if err != nil || len(velocities) == 0 {
		logger.Warn(err, "No throughput data found")
		return 0, 0
	}

	// Calculate average and standard deviation
	var sum, sumSquares float64
	for _, v := range velocities {
		throughput := float64(v.IssuesCompleted)
		sum += throughput
		sumSquares += throughput * throughput
	}

	count := float64(len(velocities))
	avg := sum / count
	variance := 0.0
	if len(velocities) > 1 {
		variance = (sumSquares - (sum*sum)/count) / (count - 1)
	}
	stdDev := 0.0
	if variance > 0 {
		stdDev = math.Sqrt(variance)
	}

	return avg, stdDev
}

func forecastInitiativeKanban(db dal.Dal, projectName string, initiative bmModels.BusinessInitiative, avgThroughput, stdDev float64, weeksPerPeriod int, logger log.Logger) *models.InitiativeForecast {
	// Count total and completed issues for this initiative (epic)
	var totalIssues, completedIssues int64

	// Total issues linked to this epic
	err := db.First(&totalIssues,
		dal.Select("COUNT(DISTINCT issues.id)"),
		dal.From(&ticket.Issue{}),
		dal.Join("LEFT JOIN board_issues bi ON bi.issue_id = issues.id"),
		dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'boards' AND pm.row_id = bi.board_id"),
		dal.Where("pm.project_name = ? AND (issues.parent_issue_id = ? OR issues.epic_key = ?)", projectName, initiative.Id, initiative.JiraEpicKey),
	)
	if err != nil {
		logger.Debug("Could not count total issues for initiative %s: %v", initiative.Name, err)
		return nil
	}

	// Completed issues
	err = db.First(&completedIssues,
		dal.Select("COUNT(DISTINCT issues.id)"),
		dal.From(&ticket.Issue{}),
		dal.Join("LEFT JOIN board_issues bi ON bi.issue_id = issues.id"),
		dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'boards' AND pm.row_id = bi.board_id"),
		dal.Where("pm.project_name = ? AND (issues.parent_issue_id = ? OR issues.epic_key = ?) AND issues.status = 'DONE'",
			projectName, initiative.Id, initiative.JiraEpicKey),
	)
	if err != nil {
		completedIssues = 0
	}

	if totalIssues == 0 {
		logger.Debug("Initiative %s has no linked issues, skipping", initiative.Name)
		return nil
	}

	remainingIssues := totalIssues - completedIssues
	if remainingIssues <= 0 {
		remainingIssues = 0
	}

	percentComplete := float64(completedIssues) / float64(totalIssues) * 100

	// Calculate estimated weeks to completion
	weeksToComplete := float64(remainingIssues) / avgThroughput
	if weeksToComplete < 0 {
		weeksToComplete = 0
	}

	// Calculate periods (treating each week as a "period")
	if weeksPerPeriod <= 0 {
		weeksPerPeriod = 1
	}
	periodsToComplete := int(math.Ceil(weeksToComplete / float64(weeksPerPeriod)))
	if periodsToComplete < 1 && remainingIssues > 0 {
		periodsToComplete = 1
	}

	// Calculate estimated completion date
	completionDate := time.Now().AddDate(0, 0, int(weeksToComplete*7))

	// Determine confidence level based on standard deviation
	confidenceLevel := "medium"
	if stdDev < avgThroughput*0.2 {
		confidenceLevel = "high"
	} else if stdDev > avgThroughput*0.5 {
		confidenceLevel = "low"
	}

	bestCasePeriods := int(math.Ceil(float64(periodsToComplete) * 0.8))
	if bestCasePeriods < 1 && remainingIssues > 0 {
		bestCasePeriods = 1
	}
	worstCasePeriods := int(math.Ceil(float64(periodsToComplete) * 1.3))
	if worstCasePeriods < 1 && remainingIssues > 0 {
		worstCasePeriods = 1
	}

	// Create scenario data (best/worst/likely)
	scenarioData := ScenarioData{
		BestCase: ForecastScenario{
			Sprints:        bestCasePeriods,
			CompletionDate: time.Now().AddDate(0, 0, int(weeksToComplete*0.8*7)).Format("2006-01-02"),
			Velocity:       avgThroughput * 1.2,
		},
		WorstCase: ForecastScenario{
			Sprints:        worstCasePeriods,
			CompletionDate: time.Now().AddDate(0, 0, int(weeksToComplete*1.3*7)).Format("2006-01-02"),
			Velocity:       avgThroughput * 0.7,
		},
		MostLikely: ForecastScenario{
			Sprints:        periodsToComplete,
			CompletionDate: completionDate.Format("2006-01-02"),
			Velocity:       avgThroughput,
		},
	}

	scenarioJSON, _ := json.Marshal(scenarioData)

	forecast := &models.InitiativeForecast{
		Id:                      fmt.Sprintf("%s:%s", initiative.Id, time.Now().Format("20060102")),
		InitiativeId:            initiative.Id,
		InitiativeName:          initiative.Name,
		TotalStoryPoints:        int(totalIssues), // Using issue count instead of story points
		CompletedStoryPoints:    int(completedIssues),
		RemainingStoryPoints:    int(remainingIssues),
		PercentComplete:         percentComplete,
		AvgVelocity:             avgThroughput,
		EstimatedSprints:        int(periodsToComplete),
		EstimatedCompletionDate: &completionDate,
		ConfidenceLevel:         confidenceLevel,
		VelocityStdDev:          stdDev,
		ScenarioData:            string(scenarioJSON),
		CalculatedAt:            time.Now(),
	}

	logger.Info("Initiative '%s': %d/%d issues (%.0f%%), estimated %d weeks (%.1f issues/week)",
		initiative.Name, completedIssues, totalIssues, percentComplete, int(weeksToComplete), avgThroughput)

	return forecast
}
