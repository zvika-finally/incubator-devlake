# Dashboard Metrics Audit - Design Document

**Date:** 2026-02-01
**Status:** Approved
**Dashboards:** AI Detection, Business Metrics, Capacity Planning, FinDevOps

---

## Overview

This document defines the audit framework for validating 98 metrics across 4 custom Grafana dashboards. The audit ensures:

1. **Logic correctness** - formulas match documented intent
2. **Data trust** - completeness, accuracy, freshness, consistency
3. **Time aggregation** - daily tracking and quarterly reporting
4. **Testing** - verification queries and automated tests

## Audience

| Audience | Primary Concern | Key Deliverables |
|----------|-----------------|------------------|
| Engineering | Data lineage, SQL correctness, test coverage | Per-metric checklists, verification queries, automated tests |
| Finance | ASC 350-40 compliance, audit trail | FinDevOps audit, capitalization validation |
| Leadership | Overall confidence, remediation path | Executive summary, rollup status |

---

## Deliverables

### Directory Structure

```
docs/audit/
├── AUDIT_CHECKLIST_MASTER.md      ← Executive summary + rollup status
├── dashboards/
│   ├── ai-detection-audit.md       ← 25 metrics
│   ├── business-metrics-audit.md   ← 20 metrics
│   ├── capacity-planning-audit.md  ← 23 metrics
│   └── findevops-audit.md          ← 30 metrics
├── data-lineage/
│   ├── source-to-domain.md         ← Jira/GitHub → DevLake domain tables
│   ├── domain-to-plugin.md         ← Domain → custom plugin transformations
│   └── plugin-dependencies.md      ← aidetector→DORA, findevops→businessmetrics
└── tests/
    ├── verification-queries.sql    ← Manual validation queries
    └── metrics_audit_test.go       ← Automated test suite
```

---

## Executive Summary Format

**File:** `AUDIT_CHECKLIST_MASTER.md`

```markdown
# Dashboard Metrics Audit - Executive Summary

**Audit Date:** YYYY-MM-DD
**Auditor:** [Name]
**Overall Status:** 🟢 PASS | 🟡 PARTIAL | 🔴 FAIL

## Rollup by Dashboard

| Dashboard | Metrics | Logic | Data Trust | Aggregation | Testing | Status |
|-----------|---------|-------|------------|-------------|---------|--------|
| AI Detection | 25 | 25/25 | 24/25 | 25/25 | 20/25 | 🟡 |
| Business Metrics | 20 | 20/20 | 20/20 | 20/20 | 18/20 | 🟢 |
| Capacity Planning | 23 | 23/23 | 23/23 | 22/23 | 19/23 | 🟡 |
| FinDevOps | 30 | 30/30 | 30/30 | 30/30 | 28/30 | 🟢 |

## Trust Dimensions Summary

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Completeness** | 🟢 | All expected data flowing |
| **Accuracy** | 🟢 | Formulas match documentation |
| **Freshness** | 🟡 | 2 metrics have stale data |
| **Consistency** | 🟢 | Cross-metric validation passed |

## Compliance Notes (Finance)
- ASC 350-40 categorization: ✅ Validated
- Capitalization rate calculation: ✅ Auditable
- Cost allocation trail: ✅ Complete
```

---

## Gap Analysis & Remediation Path

**Included in:** `AUDIT_CHECKLIST_MASTER.md`

```markdown
## Gap Analysis & Remediation Path

### Critical Gaps (Must fix before PASS)

| ID | Dashboard | Metric | Gap Type | Issue | Remediation | Owner | Target Date |
|----|-----------|--------|----------|-------|-------------|-------|-------------|
| G-01 | AI Detection | Code Churn | Testing | No automated test | Add e2e test for churn calculation | - | - |
| G-02 | Capacity | Monte Carlo P90 | Aggregation | Quarterly rollup undefined | Define quarterly aggregation logic | - | - |

### Warnings (Recommended but not blocking)

| ID | Dashboard | Metric | Gap Type | Issue | Remediation |
|----|-----------|--------|----------|-------|-------------|
| W-01 | FinDevOps | Cost Per Deploy | Freshness | Data 3 days old | Verify collection job schedule |

### Remediation Progress

| Priority | Total Gaps | Resolved | Remaining | % Complete |
|----------|-----------|----------|-----------|------------|
| Critical | 5 | 3 | 2 | 60% |
| Warning | 8 | 6 | 2 | 75% |

### Path to PASS

1. ☐ Resolve all Critical gaps (G-01 through G-05)
2. ☐ Re-run verification queries
3. ☐ All automated tests passing
4. ☐ Sign-off from: Engineering ☐ | Finance ☐ | Leadership ☐
```

---

## Per-Metric Audit Template

**Files:** `dashboards/*.md`

Each metric follows this template:

```markdown
### Metric: [Metric Name]

**Panel:** [Panel Title] | **Dashboard:** [Dashboard Name]

#### 1. Logic Validation
- ☐ **Outcome defined:** [What this metric measures and why]
- ☐ **Formula documented:**
  ```
  [Mathematical formula or algorithm]
  ```
- ☐ **Edge cases identified:**
  - [Edge case 1] → [Handling]
  - [Edge case 2] → [Handling]

#### 2. Data Lineage
| Layer | Table/Source | Transformation |
|-------|--------------|----------------|
| Source | [External API] | Raw API data |
| Domain | [domain_table] | [Collector/Extractor] |
| Plugin | [plugin_table] | [Task name] |
| Dashboard | [SQL aggregation] | Grafana SQL |

#### 3. Trust Validation
- ☐ **Completeness:** [Validation query]
- ☐ **Accuracy:** [Sample verification method]
- ☐ **Freshness:** [Timestamp check]
- ☐ **Consistency:** [Cross-metric validation]

#### 4. Time Aggregation
- ☐ **Daily:** [Daily calculation method]
- ☐ **Quarterly:** [Quarterly rollup method]

#### 5. Testing
- ☐ **Verification query:** See `verification-queries.sql#[anchor]`
- ☐ **Automated test:** `[test file path]`
- ☐ **Test data:** `[test data path]`

#### Status: 🟢 PASS | 🟡 GAPS | 🔴 FAIL
**Gaps:** [List any gaps or "None"]
```

---

## Data Lineage Documentation

### Source to Domain (source-to-domain.md)

```markdown
# Data Lineage: Source → Domain Layer

## GitHub → DevLake Domain

| Source Entity | API Endpoint | Collector | Domain Table | Key Fields |
|---------------|--------------|-----------|--------------|------------|
| Pull Requests | `/repos/{}/pulls` | github/tasks/pr_collector.go | `pull_requests` | id, merged_date, author_name, additions, deletions |
| Commits | `/repos/{}/commits` | github/tasks/commit_collector.go | `commits` | sha, authored_date, message, additions, deletions |
| PR Comments | `/repos/{}/pulls/{}/comments` | github/tasks/pr_comment_collector.go | `pull_request_comments` | id, pull_request_id, body, created_date |

## Jira → DevLake Domain

| Source Entity | API Endpoint | Collector | Domain Table | Key Fields |
|---------------|--------------|-----------|--------------|------------|
| Issues | `/rest/api/3/search` | jira/tasks/issue_collector.go | `issues` | id, issue_key, type, status, story_point, resolution_date |
| Sprints | `/rest/agile/1.0/board/{}/sprint` | jira/tasks/sprint_collector.go | `sprints` | id, name, start_date, end_date |
| Worklogs | `/rest/api/3/issue/{}/worklog` | jira/tasks/worklog_collector.go | `issue_worklogs` | issue_id, time_spent_seconds, author_id |

## CI/CD → DevLake Domain

| Source | Collector | Domain Table | Key Fields |
|--------|-----------|--------------|------------|
| Jenkins Builds | jenkins/tasks/build_collector.go | `cicd_pipelines` | id, name, result, finished_date |
| GitHub Actions | github/tasks/cicd_run_collector.go | `cicd_deployment_commits` | pipeline_id, commit_sha, finished_date |
```

### Domain to Plugin (domain-to-plugin.md)

```markdown
# Data Lineage: Domain → Custom Plugins

## aidetector Plugin

| Domain Input | Task | Output Table | Transformation |
|--------------|------|--------------|----------------|
| `pull_requests` | detectExplicitSignals | `ai_usage_signals` | Scan PR title, body, commits for AI markers |
| `commits` | analyzeCommitPatterns | `ai_usage_signals` | Calculate rapid_commit_score, message patterns |
| `ai_usage_signals` | scoreAIConfidence | `ai_usage_signals` | Combine all scores into 0-100 confidence |
| `pull_requests` + `commits` | analyzeCodeChurn | `ai_churn_metrics` | Track file changes 30 days post-merge |

## businessmetrics Plugin

| Domain Input | Task | Output Table | Transformation |
|--------------|------|--------------|----------------|
| `issues` (Epics) | extractBusinessGoals | `business_initiatives` | Parse Jira Epics as initiatives |
| `issues`, `pull_requests`, `commits` | calculateAlignment | `work_allocations` | Map work to initiatives |
| DORA metrics | calculateHealthScore | `team_health_scores` | Score against elite benchmarks |
| `working_agreements` | checkAgreements | `agreement_violations` | Check SLA compliance |

## capacityplanner Plugin

| Domain Input | Task | Output Table | Transformation |
|--------------|------|--------------|----------------|
| `issues`, `sprints` | calculateVelocity | `team_velocities` | Story points per sprint |
| `team_velocities` | monteCarloForecast | `monte_carlo_forecasts` | 1000 simulation runs |
| `issues` | calculateFlowEfficiency | `issue_flow_metrics` | Active vs waiting time |

## findevops Plugin

| Domain Input | Task | Output Table | Transformation |
|--------------|------|--------------|----------------|
| `issues`, `issue_worklogs` | calculateCosts | `cost_allocations` | Hours × hourly rate |
| `cost_allocations` | categorizeCapitalization | `cost_allocations` | ASC 350-40 classification |
| `cicd_deployment_commits` | calculateDeploymentCosts | `deployment_costs` | Cost per deployment |
```

### Plugin Dependencies (plugin-dependencies.md)

```markdown
# Plugin Execution Order & Dependencies

## Dependency Graph

```
Source APIs (GitHub, Jira, Jenkins)
    ↓
[github, jira, jenkins collectors]
    ↓
Domain Layer (pull_requests, commits, issues, cicd_*)
    ↓
[dora plugin] ──→ project_pr_metrics, deployment metrics
    ↓
┌───────────────────┬────────────────────┐
│                   │                    │
▼                   ▼                    │
[aidetector]    [businessmetrics]        │
    │               │                    │
    ▼               ▼                    │
ai_usage_signals  team_health_scores     │
ai_churn_metrics  business_initiatives   │
                  work_allocations       │
                        │                │
                        ▼                │
                  [findevops] ◄──────────┘
                        │
                        ▼
                  cost_allocations
                  monthly_cost_summaries
                        │
┌───────────────────────┘
│
▼
[capacityplanner] (no dependencies, can run parallel)
    │
    ▼
monte_carlo_forecasts
team_velocities
investment_rois
    │
    ▼
Grafana Dashboards
```

## Execution Order

1. **Domain collection** - github, jira, jenkins (parallel)
2. **DORA metrics** - dora plugin
3. **Analysis plugins** - aidetector, businessmetrics (parallel)
4. **Cost tracking** - findevops (after businessmetrics)
5. **Forecasting** - capacityplanner (independent)
```

---

## Testing Artifacts

### Verification Queries (verification-queries.sql)

```sql
-- ============================================
-- VERIFICATION QUERIES FOR DASHBOARD METRICS
-- Run these manually to validate calculations
-- ============================================

-- #ai-confidence-completeness: All merged PRs should have signals
SELECT
    'ai-confidence-completeness' as check_name,
    (SELECT COUNT(*) FROM pull_requests WHERE merged_date IS NOT NULL) as expected_prs,
    (SELECT COUNT(DISTINCT pull_request_id) FROM ai_usage_signals) as actual_signals,
    CASE
        WHEN (SELECT COUNT(*) FROM pull_requests WHERE merged_date IS NOT NULL)
             = (SELECT COUNT(DISTINCT pull_request_id) FROM ai_usage_signals)
        THEN 'PASS' ELSE 'FAIL'
    END as status;

-- #ai-confidence-accuracy: Sample verification
SELECT pr.id, pr.title, s.ai_confidence_score, s.explicit_tool_detected, s.explicit_tools
FROM pull_requests pr
JOIN ai_usage_signals s ON pr.id = s.pull_request_id
WHERE s.explicit_tool_detected = true
LIMIT 5;

-- #findevops-capitalization: ASC 350-40 categorization
SELECT
    i.type as issue_type,
    ca.capitalization_category,
    COUNT(*) as count,
    CASE
        WHEN i.type IN ('Bug', 'Defect') AND ca.capitalization_category = 'expense' THEN 'PASS'
        WHEN i.type IN ('Story', 'Task', 'Feature') AND ca.capitalization_category = 'capitalizable' THEN 'PASS'
        ELSE 'REVIEW'
    END as status
FROM cost_allocations ca
JOIN issues i ON ca.issue_id = i.id
GROUP BY i.type, ca.capitalization_category;

-- #health-score-range: All scores should be 0-100
SELECT
    'health-score-range' as check_name,
    MIN(total_score) as min_score,
    MAX(total_score) as max_score,
    CASE
        WHEN MIN(total_score) >= 0 AND MAX(total_score) <= 100 THEN 'PASS'
        ELSE 'FAIL'
    END as status
FROM team_health_scores;

-- #quarterly-aggregation: Verify quarterly matches daily sum
-- (Example for deployment count)
SELECT
    'quarterly-deployment-count' as check_name,
    (SELECT SUM(deployment_count) FROM deployment_costs
     WHERE window_days = 7
     AND period_start >= '2026-01-01' AND period_end < '2026-04-01') as sum_weekly,
    (SELECT deployment_count FROM deployment_costs
     WHERE window_days = 90
     AND period_end = (SELECT MAX(period_end) FROM deployment_costs WHERE window_days = 90)) as quarterly_value;
```

### Automated Tests (metrics_audit_test.go)

```go
// backend/test/audit/metrics_audit_test.go
package audit

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestAIConfidenceCompleteness(t *testing.T) {
    // Every merged PR should have a signal record
    // Query: COUNT(merged PRs) == COUNT(DISTINCT pull_request_id in ai_usage_signals)
}

func TestAIConfidenceScoreRange(t *testing.T) {
    // All scores should be 0-100
    // Query: MIN(score) >= 0 AND MAX(score) <= 100
}

func TestChurnRatioConsistency(t *testing.T) {
    // AI + Non-AI churn records should cover all PRs with churn data
}

func TestCapitalizationCategorization(t *testing.T) {
    // Bugs → expense, Stories/Features → capitalizable
}

func TestHealthScoreCalculation(t *testing.T) {
    // deploy_freq_score + lead_time_score + cfr_score + mttr_score == total_score
}

func TestQuarterlyAggregationConsistency(t *testing.T) {
    // Quarterly rollups match sum/avg of daily values where applicable
}

func TestDataFreshness(t *testing.T) {
    // All time-based tables have records within expected freshness window
}

func TestCrossMetricConsistency(t *testing.T) {
    // Related metrics agree (e.g., cost_allocations sum == monthly_cost_summaries.total_cost)
}
```

---

## Metrics Inventory

### AI Detection Dashboard (25 metrics)

| Metric | Table | Aggregation |
|--------|-------|-------------|
| Explicit AI Markers | ai_usage_signals | COUNT |
| Avg AI Confidence | ai_usage_signals | AVG |
| Total PRs Analyzed | ai_usage_signals | COUNT |
| High Confidence (>=70%) | ai_usage_signals | COUNT |
| Medium Confidence (40-69%) | ai_usage_signals | COUNT |
| AI Tools Detected | ai_usage_signals | GROUP BY |
| AI Detection Trend | ai_usage_signals | Daily AVG |
| AI Code Churn (30d) | ai_churn_metrics | AVG |
| Non-AI Code Churn (30d) | ai_churn_metrics | AVG |
| Churn Difference % | project_churn_summaries | Latest |
| Code Churn Over Time | ai_churn_metrics | Daily AVG |
| Cursor Suggestions | cursor_usage_metrics | SUM |
| Cursor Accepted | cursor_usage_metrics | SUM |
| Cursor Accept Rate | cursor_usage_metrics | AVG |
| Claude Code Tool Uses | claude_code_usage_metrics | SUM |
| Claude Code Sessions | claude_code_usage_metrics | SUM |
| Claude Code Lines Added | claude_code_usage_metrics | SUM |
| Cursor Usage Trend | cursor_usage_metrics | Daily |
| Claude Code Usage Trend | claude_code_usage_metrics | Daily |
| Top Cursor Users | cursor_user_metrics | TOP N |
| Top Claude Code Users | claude_code_user_metrics | TOP N |
| PR Throughput Change | ai_impact_metrics | Latest |
| Review Time Change | ai_impact_metrics | Latest |
| Lead Time Change | ai_impact_metrics | Latest |
| Detection Distribution | ai_usage_signals | GROUP BY |

### Business Metrics Dashboard (20 metrics)

| Metric | Table | Aggregation |
|--------|-------|-------------|
| Team Health Score | team_health_scores | Latest |
| Health Level | team_health_scores | Latest |
| DORA Score Breakdown | team_health_scores | Latest |
| Total Initiatives | business_initiatives | COUNT |
| Active Initiatives | business_initiatives | COUNT |
| Avg Business Value Score | business_initiatives | AVG |
| Total Story Points | work_allocations | SUM |
| By Investment Category | business_initiatives | GROUP BY |
| By Business Capability | business_initiatives | GROUP BY |
| By Revenue Impact | business_initiatives | GROUP BY |
| Initiative Summary | business_initiatives + work_allocations | JOIN |
| Health Score History | team_health_scores | TOP N |
| Total Agreements | working_agreements | COUNT |
| Active Violations | agreement_violations | COUNT |
| 30-Day Compliance Rate | agreement_compliance_summaries | AVG |
| Resolved Today | agreement_violations | COUNT |
| Violations by Type | agreement_violations | GROUP BY |
| Violations Over Time | agreement_violations | Daily COUNT |
| Configured Agreements | working_agreements | LIST |
| Compliance Summary | agreement_compliance_summaries | TOP N |

### Capacity Planning Dashboard (23 metrics)

| Metric | Table | Aggregation |
|--------|-------|-------------|
| Avg Throughput | team_velocities | AVG |
| Avg Cycle Time | team_velocities | AVG |
| Monte Carlo Forecasts | monte_carlo_forecasts | COUNT |
| AI Tools Annual Benefit | investment_rois | SUM |
| Recent Throughput | team_velocities | TOP N |
| By Confidence Level | initiative_forecasts | GROUP BY |
| Monte Carlo Percentiles | monte_carlo_forecasts | P50/P75/P90/P95 |
| Brooks's Law Scenarios | capacity_models | LIST |
| AI Tools Payback | investment_rois | Latest |
| AI Tools 3-Year ROI | investment_rois | Latest |
| ROI Details | investment_rois | TOP N |
| Initiative Forecasts | initiative_forecasts | LIST |
| Current Flow Efficiency | project_flow_summaries | Latest |
| Avg Total Cycle Time | project_flow_summaries | AVG |
| Avg Active Time | project_flow_summaries | AVG |
| Avg Waiting Time | project_flow_summaries | AVG |
| By Flow Category | project_flow_summaries | GROUP BY |
| Flow Efficiency Trend | project_flow_summaries | Daily |
| By Issue Type | issue_flow_metrics | GROUP BY |
| Active vs Waiting Trend | project_flow_summaries | Daily |
| Period Flow Summary | project_flow_summaries | Weekly |
| Recent Issues Flow | issue_flow_metrics | TOP N |

### FinDevOps Dashboard (30 metrics)

| Metric | Table | Aggregation |
|--------|-------|-------------|
| Total Development Cost | monthly_cost_summaries | SUM |
| Capitalizable Cost | monthly_cost_summaries | SUM |
| Expensed Cost | monthly_cost_summaries | SUM |
| Capitalization Rate | monthly_cost_summaries | AVG |
| By Development Phase | cost_allocations | GROUP BY |
| Monthly Cost Breakdown | monthly_cost_summaries | Monthly |
| By Capitalization Category | cost_allocations | GROUP BY |
| Avg Cost Per Deploy | deployment_costs | AVG |
| Cost Allocations Detail | cost_allocations | TOP N |
| Cost Per Deploy (7d) | deployment_costs | Latest |
| Cost Per Deploy (30d) | deployment_costs | Latest |
| Cost Per Deploy (90d) | deployment_costs | Latest |
| Deployment Cost History | deployment_costs | LIST |
| Budget Variance | monthly_cost_summaries | Latest |
| Estimated Cost | monthly_cost_summaries | SUM |
| Actual Cost | monthly_cost_summaries | SUM |
| Over Budget Issues | monthly_cost_summaries | COUNT |
| Estimated vs Actual Trend | monthly_cost_summaries | Monthly |
| Budget Variance Trend | monthly_cost_summaries | Monthly |
| Unallocated Cost % | monthly_cost_summaries | Latest |
| Unallocated Cost $ | monthly_cost_summaries | Latest |
| Orphan Issues | monthly_cost_summaries | COUNT |
| Total Unallocated Issues | cost_allocations | COUNT |
| Unallocated % Trend | monthly_cost_summaries | Monthly |
| Orphan Count Trend | monthly_cost_summaries | Monthly |
| Unallocated Issues Detail | cost_allocations | TOP N |

---

## Validation Coverage

| Dimension | What Gets Checked |
|-----------|------------------|
| **Logic** | 98 metrics × formula + outcome + edge cases |
| **Completeness** | Record counts at each pipeline stage |
| **Accuracy** | Sample verification for each metric type |
| **Freshness** | Timestamp checks on all time-filtered tables |
| **Consistency** | Cross-metric validation |
| **Aggregation** | Daily + quarterly patterns for all 98 metrics |

## Out of Scope

- Performance benchmarking (query execution times)
- UI/UX review of Grafana panels
- Source system data quality (Jira/GitHub correctness)

---

## Implementation Notes

### Existing Test Coverage (Validated)

| Plugin | E2E Tests | Test Data |
|--------|-----------|-----------|
| aidetector | explicit_signals_test.go, calculate_ai_impact_test.go, analyze_code_churn_test.go | CSV fixtures |
| businessmetrics | calculate_business_value_test.go, calculate_health_score_test.go, check_agreements_test.go | CSV fixtures |
| capacityplanner | calculate_roi_test.go, monte_carlo_forecast_test.go, calculate_flow_efficiency_test.go, brooks_law_model_test.go | CSV fixtures |
| findevops | categorize_capitalization_test.go | CSV fixtures |
| cursor | collect_metrics_test.go | - |
| claudecode | collect_metrics_test.go | - |

### Plugin Dependencies (Validated)

```
dora → aidetector, businessmetrics
businessmetrics → findevops
(capacityplanner has no dependencies)
```

---

**Document Status:** Approved for implementation
**Next Step:** Create audit directory structure and populate checklists
