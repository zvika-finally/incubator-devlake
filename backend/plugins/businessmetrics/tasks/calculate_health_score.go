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
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/businessmetrics/models"
)

var CalculateHealthScoreMeta = plugin.SubTaskMeta{
	Name:             "calculateHealthScore",
	EntryPoint:       CalculateHealthScore,
	EnabledByDefault: true,
	Description:      "Calculate team health score based on DORA metrics against elite benchmarks",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CICD},
}

// DORA Elite Benchmarks from eng-product-metrics
const (
	EliteDeployFreq    = 1.0  // 1 deploy per day (7 per week)
	EliteLeadTimeHours = 24.0 // 1 day
	EliteCFR           = 5.0  // 5% change failure rate
	EliteMTTRHours     = 1.0  // 1 hour
	MaxScore           = 25   // Max score per metric
)

func CalculateHealthScore(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*BusinessMetricsTaskData)
	logger := taskCtx.GetLogger()

	logger.Info("Starting calculateHealthScore for project: %s", data.Options.ProjectName)

	// Calculate for last 30 days
	now := time.Now()
	periodEnd := now
	periodStart := now.AddDate(0, 0, -30)

	// Query DORA metrics from project_pr_metrics and cicd tables
	doraMetrics, err := queryDORAMetrics(db, data.Options.ProjectName, periodStart, periodEnd)
	if err != nil {
		return errors.Default.Wrap(err, "failed to query DORA metrics")
	}

	// Calculate individual scores
	eliteDeployFreq, eliteLeadTimeHours, eliteCFR, eliteMTTRHours := getEffectiveEliteBenchmarks(data.Settings)
	deployFreqScore := calculateDeployFreqScore(doraMetrics.deployFrequency, eliteDeployFreq)
	leadTimeScore := calculateLeadTimeScore(doraMetrics.leadTimeHours, eliteLeadTimeHours)
	cfrScore := calculateCFRScore(doraMetrics.changeFailureRate, eliteCFR)
	mttrScore := calculateMTTRScore(doraMetrics.mttrHours, eliteMTTRHours)

	// Total score
	totalScore := deployFreqScore + leadTimeScore + cfrScore + mttrScore

	// Determine health level
	eliteThreshold, highThreshold, mediumThreshold := getEffectiveHealthThresholds(data.Settings)
	healthLevel := determineHealthLevel(totalScore, eliteThreshold, highThreshold, mediumThreshold)

	// Create health score record
	healthScore := models.TeamHealthScore{
		Id:          fmt.Sprintf("%s:%s", data.Options.ProjectName, periodEnd.Format("20060102")),
		ProjectName: data.Options.ProjectName,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,

		DeployFrequency:   doraMetrics.deployFrequency,
		LeadTimeHours:     doraMetrics.leadTimeHours,
		ChangeFailureRate: doraMetrics.changeFailureRate,
		MTTRHours:         doraMetrics.mttrHours,

		DeployFreqScore: deployFreqScore,
		LeadTimeScore:   leadTimeScore,
		CFRScore:        cfrScore,
		MTTRScore:       mttrScore,

		TotalScore:  totalScore,
		HealthLevel: healthLevel,

		CalculatedAt: now,
	}

	if err := db.CreateOrUpdate(&healthScore); err != nil {
		return errors.Default.Wrap(err, "failed to save health score")
	}

	logger.Info("Health score for project %s: %d (%s) - Deploy=%d, LeadTime=%d, CFR=%d, MTTR=%d",
		data.Options.ProjectName, totalScore, healthLevel,
		deployFreqScore, leadTimeScore, cfrScore, mttrScore)

	return nil
}

// doraMetrics holds raw DORA metric values
type doraMetrics struct {
	deployFrequency   float64
	leadTimeHours     float64
	changeFailureRate float64
	mttrHours         float64
}

// queryDORAMetrics retrieves DORA metrics from the database
func queryDORAMetrics(db dal.Dal, projectName string, start, end time.Time) (*doraMetrics, errors.Error) {
	metrics := &doraMetrics{}

	// Deploy frequency: count deployments / days
	deploymentClauses := []dal.Clause{
		dal.From("cicd_deployment_commits"),
		dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'cicd_scopes' AND pm.row_id = cicd_deployment_commits.cicd_scope_id"),
		dal.Where("pm.project_name = ? AND cicd_deployment_commits.finished_date >= ? AND cicd_deployment_commits.finished_date < ? AND cicd_deployment_commits.result = ?",
			projectName, start, end, "SUCCESS"),
	}
	deployCount, err := db.Count(deploymentClauses...)
	if err != nil {
		return nil, errors.Default.Wrap(err, "failed to count deployments")
	}

	days := end.Sub(start).Hours() / 24
	if days > 0 {
		metrics.deployFrequency = float64(deployCount) / days
	}

	// Lead time: average from project_pr_metrics
	type leadTimeResult struct {
		AvgLeadTime *float64 `gorm:"column:avg_lead_time"`
	}
	var results []leadTimeResult
	leadTimeClauses := []dal.Clause{
		dal.Select("AVG(pr_coding_time + pr_pickup_time + pr_review_time + pr_deploy_time) / 3600000 as avg_lead_time"),
		dal.From("project_pr_metrics"),
		dal.Where("project_name = ? AND pr_merged_date >= ? AND pr_merged_date < ?",
			projectName, start, end),
	}
	// Use db.All() instead of db.First() to avoid automatic ORDER BY on aggregate queries
	if err := db.All(&results, leadTimeClauses...); err != nil {
		return nil, errors.Default.Wrap(err, "failed to query lead time")
	}
	if len(results) > 0 && results[0].AvgLeadTime != nil {
		metrics.leadTimeHours = *results[0].AvgLeadTime
	}

	// Change failure rate: failed deployments / total deployments
	failedDeployClauses := []dal.Clause{
		dal.From("cicd_deployment_commits"),
		dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'cicd_scopes' AND pm.row_id = cicd_deployment_commits.cicd_scope_id"),
		dal.Where("pm.project_name = ? AND cicd_deployment_commits.finished_date >= ? AND cicd_deployment_commits.finished_date < ? AND cicd_deployment_commits.result = ?",
			projectName, start, end, "FAILURE"),
	}
	failedCount, err := db.Count(failedDeployClauses...)
	if err != nil {
		return nil, errors.Default.Wrap(err, "failed to count failed deployments")
	}

	totalDeploys := deployCount + failedCount
	if totalDeploys > 0 {
		metrics.changeFailureRate = (float64(failedCount) / float64(totalDeploys)) * 100
	}

	// MTTR/FDRT: use DORA's incident-to-deployment relationship first.
	type mttrResult struct {
		AvgMTTRHours *float64 `gorm:"column:avg_mttr_hours"`
	}
	var mttrResults []mttrResult
	mttrClauses := []dal.Clause{
		dal.Select("AVG(TIMESTAMPDIFF(MINUTE, d.deployment_finished_date, i.resolution_date)) / 60 as avg_mttr_hours"),
		dal.From("project_incident_deployment_relationships pidr"),
		dal.Join("JOIN incidents i ON i.id = pidr.id"),
		dal.Join(`JOIN (
			SELECT
				cdc.cicd_deployment_id,
				MAX(cdc.finished_date) AS deployment_finished_date
			FROM cicd_deployment_commits cdc
			JOIN project_mapping pm ON pm.table = 'cicd_scopes' AND pm.row_id = cdc.cicd_scope_id
			WHERE
				pm.project_name = ? AND
				cdc.result = 'SUCCESS' AND
				cdc.environment = 'PRODUCTION'
			GROUP BY cdc.cicd_deployment_id
		) d ON d.cicd_deployment_id = pidr.deployment_id`, projectName),
		dal.Where("pidr.project_name = ? AND i.resolution_date >= ? AND i.resolution_date < ? AND i.resolution_date IS NOT NULL",
			projectName, start, end),
	}
	if err := db.All(&mttrResults, mttrClauses...); err == nil && len(mttrResults) > 0 && mttrResults[0].AvgMTTRHours != nil {
		metrics.mttrHours = *mttrResults[0].AvgMTTRHours
	} else {
		// Fallback: use incident lead_time_minutes / created->resolved if relationship data is absent.
		mttrFallbackClauses := []dal.Clause{
			dal.Select("AVG(COALESCE(incidents.lead_time_minutes, TIMESTAMPDIFF(MINUTE, incidents.created_date, incidents.resolution_date))) / 60 as avg_mttr_hours"),
			dal.From("incidents"),
			dal.Join("LEFT JOIN project_mapping pm ON pm.table = incidents.`table` AND pm.row_id = incidents.scope_id"),
			dal.Where("pm.project_name = ? AND incidents.resolution_date >= ? AND incidents.resolution_date < ? AND incidents.resolution_date IS NOT NULL",
				projectName, start, end),
		}
		if err := db.All(&mttrResults, mttrFallbackClauses...); err == nil && len(mttrResults) > 0 && mttrResults[0].AvgMTTRHours != nil {
			metrics.mttrHours = *mttrResults[0].AvgMTTRHours
		} else {
			metrics.mttrHours = 4.0
		}
	}

	return metrics, nil
}

func getEffectiveEliteBenchmarks(settings *models.BusinessMetricsSettings) (deployFreq, leadTimeHours, cfr, mttrHours float64) {
	deployFreq = EliteDeployFreq
	leadTimeHours = EliteLeadTimeHours
	cfr = EliteCFR
	mttrHours = EliteMTTRHours

	if settings == nil {
		return
	}
	if settings.EliteDeployFreq > 0 {
		deployFreq = settings.EliteDeployFreq
	}
	if settings.EliteLeadTimeHours > 0 {
		leadTimeHours = settings.EliteLeadTimeHours
	}
	if settings.EliteCFR > 0 {
		cfr = settings.EliteCFR
	}
	if settings.EliteMTTRHours > 0 {
		mttrHours = settings.EliteMTTRHours
	}
	return
}

func getEffectiveHealthThresholds(settings *models.BusinessMetricsSettings) (elite, high, medium int) {
	elite = 80
	high = 60
	medium = 40
	if settings == nil {
		return
	}
	if settings.EliteThreshold > 0 {
		elite = settings.EliteThreshold
	}
	if settings.HighThreshold > 0 {
		high = settings.HighThreshold
	}
	if settings.MediumThreshold > 0 {
		medium = settings.MediumThreshold
	}
	return
}

// CalculateDeployFreqScore calculates score for deploy frequency (0-25)
// Higher frequency = better score
// Exported for testing
func CalculateDeployFreqScore(deployFreq float64) int {
	return calculateDeployFreqScore(deployFreq, EliteDeployFreq)
}

func calculateDeployFreqScore(deployFreq float64, eliteBenchmark float64) int {
	if deployFreq <= 0 {
		return 0
	}
	if eliteBenchmark <= 0 {
		eliteBenchmark = EliteDeployFreq
	}
	// Score = min(25, (deployFreq / eliteBenchmark) * 25)
	score := int((deployFreq / eliteBenchmark) * float64(MaxScore))
	if score > MaxScore {
		return MaxScore
	}
	return score
}

// CalculateLeadTimeScore calculates score for lead time (0-25)
// Lower lead time = better score (inverted)
// Exported for testing
func CalculateLeadTimeScore(leadTimeHours float64) int {
	return calculateLeadTimeScore(leadTimeHours, EliteLeadTimeHours)
}

func calculateLeadTimeScore(leadTimeHours float64, eliteBenchmark float64) int {
	if leadTimeHours <= 0 {
		return MaxScore // Best possible if lead time is 0 or negative (shouldn't happen)
	}
	if eliteBenchmark <= 0 {
		eliteBenchmark = EliteLeadTimeHours
	}
	// Score = min(25, (eliteBenchmark / leadTime) * 25)
	score := int((eliteBenchmark / leadTimeHours) * float64(MaxScore))
	if score > MaxScore {
		return MaxScore
	}
	if score < 0 {
		return 0
	}
	return score
}

// CalculateCFRScore calculates score for change failure rate (0-25)
// Lower CFR = better score (inverted)
// Exported for testing
func CalculateCFRScore(cfr float64) int {
	return calculateCFRScore(cfr, EliteCFR)
}

func calculateCFRScore(cfr float64, eliteBenchmark float64) int {
	if cfr <= 0 {
		return MaxScore // Perfect if no failures
	}
	if eliteBenchmark <= 0 {
		eliteBenchmark = EliteCFR
	}
	// Score = min(25, (eliteBenchmark / cfr) * 25)
	score := int((eliteBenchmark / cfr) * float64(MaxScore))
	if score > MaxScore {
		return MaxScore
	}
	if score < 0 {
		return 0
	}
	return score
}

// CalculateMTTRScore calculates score for MTTR (0-25)
// Lower MTTR = better score (inverted)
// Exported for testing
func CalculateMTTRScore(mttrHours float64) int {
	return calculateMTTRScore(mttrHours, EliteMTTRHours)
}

func calculateMTTRScore(mttrHours float64, eliteBenchmark float64) int {
	if mttrHours <= 0 {
		return MaxScore // Best possible if MTTR is 0
	}
	if eliteBenchmark <= 0 {
		eliteBenchmark = EliteMTTRHours
	}
	// Score = min(25, (eliteBenchmark / mttr) * 25)
	score := int((eliteBenchmark / mttrHours) * float64(MaxScore))
	if score > MaxScore {
		return MaxScore
	}
	if score < 0 {
		return 0
	}
	return score
}

// DetermineHealthLevel returns the health level based on total score
// Exported for testing
func DetermineHealthLevel(totalScore int) string {
	return determineHealthLevel(totalScore, 80, 60, 40)
}

func determineHealthLevel(totalScore, eliteThreshold, highThreshold, mediumThreshold int) string {
	if totalScore >= eliteThreshold {
		return models.HealthLevelElite
	}
	if totalScore >= highThreshold {
		return models.HealthLevelHigh
	}
	if totalScore >= mediumThreshold {
		return models.HealthLevelMedium
	}
	return models.HealthLevelLow
}
