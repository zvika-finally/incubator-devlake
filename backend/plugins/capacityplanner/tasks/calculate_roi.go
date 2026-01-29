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
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/capacityplanner/models"
)

var CalculateROIMeta = plugin.SubTaskMeta{
	Name:             "calculateROI",
	EntryPoint:       CalculateROI,
	EnabledByDefault: true,
	Description:      "Calculate ROI for development investments (AI tools, hiring, tech debt)",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

// Constants from eng-product-metrics
const (
	DefaultHourlyCost       = 75.0  // USD per hour
	WeeksPerYear            = 52
	HoursPerWeek            = 40
	BugFixTimePercent       = 0.20  // 20% of time spent on bugs
	AIProductivityGainPer10 = 0.03  // 3% productivity gain per 10% AI adoption
	AIHoursSavedPerUser     = 2.0   // Hours saved per week per AI tool user
	AIQualityImprovement    = 5.0   // 5% quality improvement from AI
)

// ROIParameters holds input parameters for ROI calculation
type ROIParameters struct {
	TeamSize           int     `json:"teamSize"`
	HourlyCost         float64 `json:"hourlyCost"`
	AIAdoptionPercent  float64 `json:"aiAdoptionPercent"`  // 0-100
	ProductivityGain   float64 `json:"productivityGain"`   // Percentage
	QualityImprovement float64 `json:"qualityImprovement"` // Percentage
	HoursSavedPerWeek  float64 `json:"hoursSavedPerWeek"`
}

func CalculateROI(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*CapacityPlannerTaskData)
	logger := taskCtx.GetLogger()

	logger.Info("Starting calculateROI for project: %s", data.Options.ProjectName)

	// Get team size from velocity data
	var velocity models.TeamVelocity
	velocityClauses := []dal.Clause{
		dal.From(&models.TeamVelocity{}),
		dal.Where("project_name = ?", data.Options.ProjectName),
		dal.Orderby("calculated_at DESC"),
		dal.Limit(1),
	}

	teamSize := 5 // Default
	if err := db.First(&velocity, velocityClauses...); err == nil && velocity.TeamSize > 0 {
		teamSize = velocity.TeamSize
	}

	// Calculate ROI for AI tools investment scenario
	aiToolsROI := calculateAIToolsROI(teamSize, DefaultHourlyCost, data.Options.ProjectName)
	if err := db.CreateOrUpdate(&aiToolsROI); err != nil {
		logger.Error(err, "failed to save AI tools ROI")
	} else {
		logger.Info("AI Tools ROI calculated: payback=%.1f months, 3-year ROI=%.1f%%",
			aiToolsROI.PaybackMonths, aiToolsROI.ThreeYearROI)
	}

	logger.Info("Completed ROI calculations for project %s", data.Options.ProjectName)
	return nil
}

// calculateAIToolsROI calculates ROI for AI coding tools investment
func calculateAIToolsROI(teamSize int, hourlyCost float64, projectName string) models.InvestmentROI {
	// Typical AI tool costs (Copilot, Cursor, etc.)
	monthlyPerUserCost := 20.0 // USD per user per month
	upfrontCost := 0.0         // No significant upfront cost for SaaS tools

	// Costs
	monthlyCost := monthlyPerUserCost * float64(teamSize)
	annualCost := upfrontCost + (monthlyCost * 12)

	// Benefits calculation
	params := ROIParameters{
		TeamSize:           teamSize,
		HourlyCost:         hourlyCost,
		AIAdoptionPercent:  80.0, // Assume 80% adoption
		ProductivityGain:   AIProductivityGainPer10 * (80.0 / 10) * 100, // Scale to 80% adoption
		QualityImprovement: AIQualityImprovement,
		HoursSavedPerWeek:  AIHoursSavedPerUser,
	}

	// Direct benefit: hours saved per week * 52 weeks * hourly cost
	directBenefit := params.HoursSavedPerWeek * float64(teamSize) * WeeksPerYear * hourlyCost

	// Productivity benefit: team_size * hours/week * weeks/year * (gain%/100) * hourly_cost
	teamAnnualHours := float64(teamSize) * HoursPerWeek * WeeksPerYear
	productivityBenefit := teamAnnualHours * (params.ProductivityGain / 100) * hourlyCost

	// Quality benefit: team_hours * bug_fix_time_percent * (improvement%/100) * hourly_cost
	qualityBenefit := teamAnnualHours * BugFixTimePercent * (params.QualityImprovement / 100) * hourlyCost

	totalAnnualBenefit := directBenefit + productivityBenefit + qualityBenefit

	// ROI calculations
	paybackMonths := 0.0
	if totalAnnualBenefit > 0 {
		paybackMonths = (annualCost / totalAnnualBenefit) * 12
	}

	// 3-year ROI
	threeYearCost := upfrontCost + (monthlyCost * 36)
	threeYearBenefit := totalAnnualBenefit * 3
	threeYearROI := 0.0
	if threeYearCost > 0 {
		threeYearROI = ((threeYearBenefit - threeYearCost) / threeYearCost) * 100
	}

	// Serialize parameters
	paramsJSON, _ := json.Marshal(params)

	return models.InvestmentROI{
		Id:             fmt.Sprintf("%s:ai_tools:%d", projectName, time.Now().Unix()),
		InvestmentName: "AI Coding Tools (Copilot/Cursor)",
		InvestmentType: "ai_tools",

		UpfrontCost: upfrontCost,
		MonthlyCost: monthlyCost,
		AnnualCost:  annualCost,

		DirectBenefit:       directBenefit,
		ProductivityBenefit: productivityBenefit,
		QualityBenefit:      qualityBenefit,
		TotalAnnualBenefit:  totalAnnualBenefit,

		PaybackMonths: paybackMonths,
		ThreeYearROI:  threeYearROI,

		Parameters: string(paramsJSON),

		CalculatedAt: time.Now(),
	}
}

// CalculateROIFromParams calculates ROI from explicit parameters
// Exported for testing and API usage
func CalculateROIFromParams(params ROIParameters, upfrontCost, monthlyCost float64) (annualBenefit, paybackMonths, threeYearROI float64) {
	// Direct benefit
	directBenefit := params.HoursSavedPerWeek * float64(params.TeamSize) * WeeksPerYear * params.HourlyCost

	// Productivity benefit
	teamAnnualHours := float64(params.TeamSize) * HoursPerWeek * WeeksPerYear
	productivityBenefit := teamAnnualHours * (params.ProductivityGain / 100) * params.HourlyCost

	// Quality benefit
	qualityBenefit := teamAnnualHours * BugFixTimePercent * (params.QualityImprovement / 100) * params.HourlyCost

	annualBenefit = directBenefit + productivityBenefit + qualityBenefit

	// Payback period
	annualCost := upfrontCost + (monthlyCost * 12)
	if annualBenefit > 0 {
		paybackMonths = (annualCost / annualBenefit) * 12
	}

	// 3-year ROI
	threeYearCost := upfrontCost + (monthlyCost * 36)
	threeYearBenefitTotal := annualBenefit * 3
	if threeYearCost > 0 {
		threeYearROI = ((threeYearBenefitTotal - threeYearCost) / threeYearCost) * 100
	}

	return annualBenefit, paybackMonths, threeYearROI
}
