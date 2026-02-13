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
	"math"
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/log"
	"github.com/apache/incubator-devlake/core/plugin"
	bmModels "github.com/apache/incubator-devlake/plugins/businessmetrics/models"
	"github.com/apache/incubator-devlake/plugins/capacityplanner/models"
)

var ForecastCompletionMeta = plugin.SubTaskMeta{
	Name:             "forecastCompletion",
	EntryPoint:       ForecastCompletion,
	EnabledByDefault: true,
	Description:      "Forecast initiative completion dates based on velocity trends",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

type ScenarioData struct {
	BestCase   ForecastScenario `json:"best_case"`
	WorstCase  ForecastScenario `json:"worst_case"`
	MostLikely ForecastScenario `json:"most_likely"`
}

type ForecastScenario struct {
	Sprints        int     `json:"sprints"`
	CompletionDate string  `json:"completion_date"`
	Velocity       float64 `json:"velocity"`
}

func ForecastCompletion(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*CapacityPlannerTaskData)
	logger := taskCtx.GetLogger()

	logger.Info("Starting forecastCompletion for project: %s", data.Options.ProjectName)

	// Calculate average velocity from recent sprints
	avgVelocity, stdDev := calculateAverageVelocity(db, data.Options.ProjectName)
	logger.Info("Average velocity: %.1f story points/sprint (std dev: %.1f)", avgVelocity, stdDev)

	if avgVelocity == 0 {
		logger.Warn(nil, "No velocity data available, skipping forecasts")
		return nil
	}

	// Get all initiatives
	initiatives, err := getProjectInitiatives(db, data.Options.ProjectName)
	if err != nil {
		return err
	}

	logger.Info("Forecasting completion for %d initiatives", len(initiatives))

	for _, initiative := range initiatives {
		forecast := forecastInitiative(db, initiative, avgVelocity, stdDev, data.Options.SprintDurationWeeks, logger)
		if forecast == nil {
			continue
		}

		if err := db.CreateOrUpdate(forecast); err != nil {
			logger.Error(err, "failed to save forecast for initiative %s", initiative.Id)
		}
	}

	logger.Info("Completed forecastCompletion")
	return nil
}

func calculateAverageVelocity(db dal.Dal, projectName string) (avg float64, stdDev float64) {
	var velocities []models.TeamVelocity
	err := db.All(&velocities,
		dal.From(&models.TeamVelocity{}),
		dal.Where("project_name = ?", projectName),
		dal.Orderby("sprint_end_date DESC"),
		dal.Limit(6),
	)
	if err != nil || len(velocities) == 0 {
		return 0, 0
	}

	// Calculate average
	var sum float64
	for _, v := range velocities {
		sum += float64(v.StoryPointsCompleted)
	}
	avg = sum / float64(len(velocities))

	// Calculate standard deviation
	var sqDiffSum float64
	for _, v := range velocities {
		diff := float64(v.StoryPointsCompleted) - avg
		sqDiffSum += diff * diff
	}
	stdDev = math.Sqrt(sqDiffSum / float64(len(velocities)))

	return avg, stdDev
}

func forecastInitiative(db dal.Dal, initiative bmModels.BusinessInitiative, avgVelocity, stdDev float64, sprintWeeks int, logger log.Logger) *models.InitiativeForecast {
	// Get story points for this initiative
	var totalPoints, completedPoints int64

	// Total story points (from work allocations)
	db.First(&totalPoints,
		dal.Select("COALESCE(SUM(story_points), 0)"),
		dal.From("work_allocations"),
		dal.Where("initiative_id = ?", initiative.Id),
	)

	// Completed story points (issues that are done)
	db.First(&completedPoints,
		dal.Select("COALESCE(SUM(wa.story_points), 0)"),
		dal.From("work_allocations wa"),
		dal.Join("LEFT JOIN issues i ON i.id = wa.entity_id"),
		dal.Where("wa.initiative_id = ? AND wa.entity_type = ? AND i.status = ?",
			initiative.Id, "issue", "Done"),
	)

	remaining := int(totalPoints - completedPoints)
	if remaining <= 0 {
		// Already complete
		return nil
	}

	// Calculate forecast
	estimatedSprints := int(math.Ceil(float64(remaining) / avgVelocity))
	completionDate := time.Now().AddDate(0, 0, estimatedSprints*sprintWeeks*7)

	// Determine confidence level based on std dev
	var confidence string
	if stdDev < avgVelocity*0.2 {
		confidence = "high"
	} else if stdDev < avgVelocity*0.4 {
		confidence = "medium"
	} else {
		confidence = "low"
	}

	// Build scenario data
	bestCaseVelocity := avgVelocity + stdDev
	worstCaseVelocity := avgVelocity - stdDev
	if worstCaseVelocity < 1 {
		worstCaseVelocity = 1
	}

	scenarios := ScenarioData{
		BestCase: ForecastScenario{
			Sprints:  int(math.Ceil(float64(remaining) / bestCaseVelocity)),
			Velocity: bestCaseVelocity,
		},
		WorstCase: ForecastScenario{
			Sprints:  int(math.Ceil(float64(remaining) / worstCaseVelocity)),
			Velocity: worstCaseVelocity,
		},
		MostLikely: ForecastScenario{
			Sprints:  estimatedSprints,
			Velocity: avgVelocity,
		},
	}

	// Calculate dates for scenarios
	scenarios.BestCase.CompletionDate = time.Now().AddDate(0, 0, scenarios.BestCase.Sprints*sprintWeeks*7).Format("2006-01-02")
	scenarios.WorstCase.CompletionDate = time.Now().AddDate(0, 0, scenarios.WorstCase.Sprints*sprintWeeks*7).Format("2006-01-02")
	scenarios.MostLikely.CompletionDate = completionDate.Format("2006-01-02")

	scenarioJSON, _ := json.Marshal(scenarios)

	var percentComplete float64
	if totalPoints > 0 {
		percentComplete = float64(completedPoints) * 100.0 / float64(totalPoints)
	}

	forecast := &models.InitiativeForecast{
		Id:                      initiative.Id,
		InitiativeId:            initiative.Id,
		InitiativeName:          initiative.Name,
		TotalStoryPoints:        int(totalPoints),
		CompletedStoryPoints:    int(completedPoints),
		RemainingStoryPoints:    remaining,
		PercentComplete:         percentComplete,
		AvgVelocity:             avgVelocity,
		EstimatedSprints:        estimatedSprints,
		EstimatedCompletionDate: &completionDate,
		ConfidenceLevel:         confidence,
		VelocityStdDev:          stdDev,
		ScenarioData:            string(scenarioJSON),
		CalculatedAt:            time.Now(),
	}

	logger.Info("Initiative %s: %d/%d points (%.0f%%), %d sprints remaining, completion: %s (%s confidence)",
		initiative.JiraEpicKey, completedPoints, totalPoints, percentComplete,
		estimatedSprints, completionDate.Format("2006-01-02"), confidence)

	return forecast
}
