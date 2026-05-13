# P2 Dashboard Audit: DORA & Engineering Suites

**Audit Date:** 2026-02-02
**Scope:** DORA Suite (8 dashboards) + Engineering Suite (3 dashboards)
**Priority Level:** P2 (Lighter touch - formula accuracy + visualization quality)

---

## Executive Summary

| Suite | Dashboards | Formula Accuracy | Visualization | Status |
|-------|------------|------------------|---------------|--------|
| DORA | 8 | ✅ 8/8 | ✅ Good | ✅ PASS |
| Engineering | 3 | ✅ 3/3 | ✅ Good | ✅ PASS |
| **TOTAL** | **11** | **✅ 11/11** | **✅ Good** | **✅ PASS** |

---

## DORA Suite (8 Dashboards)

### Dashboards Audited

| Dashboard | UID | Primary Metric | Status |
|-----------|-----|----------------|--------|
| DORA | qNo8_0M4z | All 4 DORA metrics | ✅ PASS |
| DORAByTeam | - | Team breakdown | ✅ PASS |
| DORADebug | KGkUnV-Vz | Validation data | ✅ PASS |
| DORADetails-DeploymentFrequency | Deployment-frequency | DF drill-down | ✅ PASS |
| DORADetails-LeadTimeforChanges | Lead-time-for-changes | LTC drill-down | ✅ PASS |
| DORADetails-ChangeFailureRate | Change-failure-rate | CFR drill-down | ✅ PASS |
| DORADetails-TimetoRestoreService | Time-to-restore-service | MTTR drill-down | ✅ PASS |
| DORADetails-FailedDeploymentRecoveryTime | - | FDRT (2023 report) | ✅ PASS |

### Formula Validation

#### Deployment Frequency
```sql
-- Median deployment days per week/month/6-months
WITH _production_deployment_days AS (
  SELECT cicd_deployment_id, MAX(DATE(finished_date)) as day
  FROM cicd_deployment_commits cdc
  JOIN project_mapping pm ON cdc.cicd_scope_id = pm.row_id
  WHERE result = 'SUCCESS' AND environment = 'PRODUCTION'
  GROUP BY 1
)
-- Uses percent_rank() for median calculation
```
**Status:** ✅ Correct - Uses proper median calculation via `percent_rank()`

#### Lead Time for Changes
```sql
-- Median PR cycle time from deployment commits
SELECT MAX(pr_cycle_time) as median_change_lead_time
FROM _median_change_lead_time_ranks
WHERE ranks <= 0.5
```
**Tables:** `pull_requests`, `project_pr_metrics`, `cicd_deployment_commits`
**Status:** ✅ Correct - Uses PR cycle time from `project_pr_metrics`

#### Change Failure Rate
```sql
SELECT SUM(has_incident) / COUNT(deployment_id) as change_failure_rate
FROM _failure_caused_by_deployments
```
**Tables:** `cicd_deployment_commits`, `project_incident_deployment_relationships`, `incidents`
**Status:** ✅ Correct - Standard CFR formula

#### Time to Restore Service / Failed Deployment Recovery Time
```sql
-- Median lead_time_minutes from incidents
SELECT MAX(lead_time_minutes) as median_time_to_resolve
FROM _median_mttr_ranks
WHERE ranks <= 0.5
```
**Tables:** `incidents`, `cicd_deployment_commits`
**Status:** ✅ Correct - Uses proper median calculation

### DORA Benchmark Compliance

| Benchmark | 2023 Report | 2021 Report | Implementation |
|-----------|-------------|-------------|----------------|
| **DF Elite** | ≥7 days/week | ≥7 days/week | ✅ Matches |
| **DF High** | ≥1 day/week | ≥1 day/month | ✅ Matches |
| **DF Medium** | ≥1 day/month | ≥1 day/6 months | ✅ Matches |
| **DF Low** | <1 day/month | <1 day/6 months | ✅ Matches |
| **LTC Elite** | <1 day | <1 hour | ✅ Matches |
| **LTC High** | 1 day - 1 week | <1 week | ✅ Matches |
| **CFR Elite** | 0-5% | 0-15% | ✅ Matches |
| **CFR High** | 5-10% | 16-20% | ✅ Matches |
| **MTTR Elite** | <1 hour | <1 hour | ✅ Matches |
| **MTTR High** | <1 day | <1 day | ✅ Matches |

### DORA Visualization Quality

| Element | Implementation | Assessment |
|---------|----------------|------------|
| **Performance Colors** | Elite=Purple, High=Green, Medium=Yellow, Low=Red | ✅ Standard DORA |
| **Stat Panels** | Used for KPIs with regex color mapping | ✅ Appropriate |
| **Time Series** | Trend charts for historical analysis | ✅ Appropriate |
| **Tables** | Detail drill-downs with sortable columns | ✅ Appropriate |
| **Links** | Cross-dashboard navigation maintained | ✅ Good UX |
| **Thresholds** | Numeric thresholds for MTTR (24h, 168h, 720h) | ✅ Correct |

---

## Engineering Suite (3 Dashboards)

### Dashboards Audited

| Dashboard | UID | Primary Metrics | Status |
|-----------|-----|-----------------|--------|
| EngineeringOverview | - | Defects, Commits, PRs | ✅ PASS |
| EngineeringThroughputAndCycleTime | - | PRs, Issues, Cycle Time | ✅ PASS |
| EngineeringThroughputAndCycleTimeTeamView | - | Team-level metrics | ✅ PASS |

### Formula Validation

#### Critical Defects
```sql
SELECT COUNT(DISTINCT i.id)
FROM issues i
JOIN board_issues bi ON i.id = bi.issue_id
WHERE i.type = 'BUG' AND i.priority IN (${priority})
```
**Status:** ✅ Correct - Standard bug count from issues

#### PRs Opened/Merged
```sql
SELECT
  COUNT(DISTINCT pr.id) as "PR: Opened",
  COUNT(DISTINCT CASE WHEN pr.merged_date IS NOT NULL THEN id END) as "PR: Merged"
FROM pull_requests pr
JOIN project_mapping pm ON pr.base_repo_id = pm.row_id
```
**Status:** ✅ Correct - Standard PR metrics

#### Issue Throughput
```sql
SELECT
  COUNT(DISTINCT i.id) as issue_count
FROM issues i
WHERE i.resolution_date IS NOT NULL
```
**Status:** ✅ Correct - Standard issue resolution count

### Engineering Visualization Quality

| Element | Implementation | Assessment |
|---------|----------------|------------|
| **Stat Panels** | Bug counts with thresholds (0/10/20) | ✅ Appropriate |
| **Bar Charts** | Monthly defect trends | ✅ Appropriate |
| **Time Series** | PR throughput over time | ✅ Appropriate |
| **Color Coding** | Green (good) → Orange → Red (alert) | ✅ Intuitive |
| **Row Organization** | Throughput/Quality sections | ✅ Clear layout |

---

## Data Source Dependencies

### DORA Suite
| Table | Required For | Plugin |
|-------|--------------|--------|
| `cicd_deployment_commits` | DF, CFR, FDRT | CI/CD plugins |
| `pull_requests` | LTC | Git plugins |
| `project_pr_metrics` | LTC | DORA plugin |
| `incidents` | CFR, MTTR | Issue tracker plugins |
| `project_mapping` | All | Core |
| `dora_benchmarks` | Classification | DORA plugin |

### Engineering Suite
| Table | Required For | Plugin |
|-------|--------------|--------|
| `issues` | Bug counts, throughput | Issue tracker |
| `pull_requests` | PR metrics | Git plugins |
| `boards` | Issue filtering | Issue tracker |
| `project_mapping` | All | Core |

---

## Visualization Review Summary

### Chart Type Assessment

| Panel Type | DORA | Engineering | Assessment |
|------------|------|-------------|------------|
| `stat` | 8 | 4 | ✅ Appropriate for KPIs |
| `table` | 12 | 2 | ✅ Appropriate for details |
| `timeseries` | 6 | 6 | ✅ Appropriate for trends |
| `barchart` | 2 | 3 | ✅ Appropriate for comparisons |
| `text` | 8 | 3 | ✅ Good documentation |

### Threshold Consistency

| Suite | Threshold Logic | Consistency |
|-------|-----------------|-------------|
| DORA | Regex-based performance level colors | ✅ Consistent across dashboards |
| Engineering | Numeric thresholds for alerts | ✅ Consistent |

---

## Issues Found

**None.** All P2 dashboards pass formula accuracy and visualization quality checks.

---

## Recommendations

1. **DORA Suite**: No changes required - implements official DORA methodology correctly
2. **Engineering Suite**: No changes required - standard metrics with appropriate visualizations
3. **Documentation**: Consider adding benchmark year selector documentation for DORA dashboards

---

## Verification Checklist

| Check | DORA | Engineering | Status |
|-------|------|-------------|--------|
| Formulas match DORA/industry standards | ✅ | ✅ | ✅ PASS |
| Table references valid | ✅ | ✅ | ✅ PASS |
| Column names correct | ✅ | ✅ | ✅ PASS |
| Filters work correctly | ✅ | ✅ | ✅ PASS |
| Chart types appropriate | ✅ | ✅ | ✅ PASS |
| Colors intuitive | ✅ | ✅ | ✅ PASS |
| Thresholds meaningful | ✅ | ✅ | ✅ PASS |
| Layout organized | ✅ | ✅ | ✅ PASS |

---

**Report Generated:** 2026-02-02
**Auditor:** Claude Code (Automated)
