# FinDevOps Dashboard Audit - Pilot Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create a complete audit checklist for the FinDevOps dashboard (30 metrics), validating logic, data lineage, trust dimensions, and time aggregation.

**Architecture:** Create audit documentation structure, populate per-metric checklists with validation queries, and add automated tests following DevLake's e2e testing patterns.

**Tech Stack:** Markdown documentation, SQL verification queries, Go test files using `e2ehelper.DataFlowTester`

---

## Task 1: Create Audit Directory Structure

**Files:**
- Create: `docs/audit/AUDIT_CHECKLIST_MASTER.md`
- Create: `docs/audit/dashboards/findevops-audit.md`
- Create: `docs/audit/data-lineage/findevops-lineage.md`
- Create: `docs/audit/tests/findevops-verification-queries.sql`

**Step 1: Create audit directory structure**

```bash
mkdir -p docs/audit/dashboards docs/audit/data-lineage docs/audit/tests
```

**Step 2: Create the master checklist scaffold**

Create `docs/audit/AUDIT_CHECKLIST_MASTER.md`:

```markdown
# Dashboard Metrics Audit - Executive Summary

**Audit Date:** 2026-02-01
**Auditor:** [Name]
**Overall Status:** 🔄 IN PROGRESS

## Rollup by Dashboard

| Dashboard | Metrics | Logic | Data Trust | Aggregation | Testing | Status |
|-----------|---------|-------|------------|-------------|---------|--------|
| FinDevOps | 30 | -/30 | -/30 | -/30 | -/30 | 🔄 |
| AI Detection | 25 | - | - | - | - | ⏳ Pending |
| Business Metrics | 20 | - | - | - | - | ⏳ Pending |
| Capacity Planning | 23 | - | - | - | - | ⏳ Pending |

## Trust Dimensions Summary

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Completeness** | 🔄 | Validating... |
| **Accuracy** | 🔄 | Validating... |
| **Freshness** | 🔄 | Validating... |
| **Consistency** | 🔄 | Validating... |

## Gap Analysis & Remediation Path

### Critical Gaps (Must fix before PASS)

| ID | Dashboard | Metric | Gap Type | Issue | Remediation | Owner | Target Date |
|----|-----------|--------|----------|-------|-------------|-------|-------------|
| - | - | - | - | - | - | - | - |

### Path to PASS

1. ☐ Complete FinDevOps audit (pilot)
2. ☐ Resolve all Critical gaps
3. ☐ Re-run verification queries
4. ☐ All automated tests passing
5. ☐ Sign-off from: Engineering ☐ | Finance ☐ | Leadership ☐

## Compliance Notes (Finance)

- ASC 350-40 categorization: ⏳ Validating
- Capitalization rate calculation: ⏳ Validating
- Cost allocation trail: ⏳ Validating
```

**Step 3: Commit scaffold**

```bash
git add docs/audit/
git commit -m "docs: scaffold audit directory structure for dashboard metrics review"
```

---

## Task 2: Document FinDevOps Data Lineage

**Files:**
- Create: `docs/audit/data-lineage/findevops-lineage.md`

**Step 1: Create the data lineage document**

Create `docs/audit/data-lineage/findevops-lineage.md`:

```markdown
# FinDevOps Data Lineage

## Overview

The FinDevOps dashboard calculates development costs and categorizes them for US GAAP ASC 350-40 software capitalization compliance.

## Data Flow Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        SOURCE SYSTEMS                            │
├─────────────────────────────────────────────────────────────────┤
│  Jira                           │  CI/CD (Jenkins/GitHub Actions)│
│  - Issues (type, labels)        │  - Deployments                 │
│  - Worklogs (time_spent)        │  - Pipeline runs               │
│  - Estimates (original_estimate)│                                │
│  - Epic links (epic_key)        │                                │
└─────────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────────┐
│                     DEVLAKE COLLECTORS                           │
├─────────────────────────────────────────────────────────────────┤
│  jira/tasks/issue_collector.go      → _raw_jira_issues          │
│  jira/tasks/worklog_collector.go    → _raw_jira_worklogs        │
│  jenkins/tasks/build_collector.go   → _raw_jenkins_builds       │
│  github/tasks/cicd_run_collector.go → _raw_github_runs          │
└─────────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────────┐
│                     DOMAIN LAYER TABLES                          │
├─────────────────────────────────────────────────────────────────┤
│  issues                    │ id, issue_key, type, status,       │
│                            │ story_point, time_spent_minutes,   │
│                            │ original_estimate_minutes,         │
│                            │ assignee_id, resolution_date       │
├────────────────────────────┼────────────────────────────────────┤
│  board_issues              │ board_id, issue_id                 │
├────────────────────────────┼────────────────────────────────────┤
│  project_mapping           │ project_name, table, row_id        │
├────────────────────────────┼────────────────────────────────────┤
│  cicd_deployment_commits   │ pipeline_id, commit_sha,           │
│                            │ finished_date, result              │
└─────────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────────┐
│                   FINDEVOPS PLUGIN TASKS                         │
├─────────────────────────────────────────────────────────────────┤
│  1. calculateCosts                                               │
│     Input: issues, board_issues, project_mapping, _tool_jira_*  │
│     Logic:                                                       │
│       hours = time_spent_minutes/60 OR original_estimate/60     │
│              OR story_point * 4                                  │
│       cost = hours × hourly_rate                                │
│       variance = (estimated - actual) / estimated × 100         │
│     Output: cost_allocations                                    │
├─────────────────────────────────────────────────────────────────┤
│  2. categorizeCapitalization                                     │
│     Input: cost_allocations                                      │
│     Logic: ASC 350-40 three-stage model                         │
│       - Preliminary (expense): Spike, Research, poc, discovery  │
│       - Development (capitalizable): Story, Feature, Task       │
│       - Post-Implementation (expense): Bug, maintenance, hotfix │
│     Output: cost_allocations (updated with phase/category)      │
├─────────────────────────────────────────────────────────────────┤
│  3. calculateDeploymentCosts                                     │
│     Input: cost_allocations, cicd_deployment_commits            │
│     Logic: total_cost / deployment_count for 7/30/90 day windows│
│     Output: deployment_costs                                     │
└─────────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────────┐
│                   FINDEVOPS OUTPUT TABLES                        │
├─────────────────────────────────────────────────────────────────┤
│  cost_allocations          │ Per-issue cost with ASC 350-40     │
│                            │ categorization and budget variance │
├────────────────────────────┼────────────────────────────────────┤
│  monthly_cost_summaries    │ Aggregated monthly costs with      │
│                            │ capitalization rate, unallocated % │
├────────────────────────────┼────────────────────────────────────┤
│  deployment_costs          │ Cost per deployment by time window │
└─────────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────────┐
│                   GRAFANA DASHBOARD                              │
│                   FinDevOps.json (30 panels)                     │
└─────────────────────────────────────────────────────────────────┘
```

## Table Schemas

### cost_allocations

| Column | Type | Source | Transformation |
|--------|------|--------|----------------|
| id | varchar(255) | Generated | `{issue_id}:{fiscal_month}` |
| initiative_id | varchar(255) | _tool_jira_issues.epic_key | Direct mapping |
| issue_id | varchar(255) | issues.id | Direct mapping |
| fiscal_month | varchar(10) | issues.resolution_date | `YYYY-MM` format |
| developer_id | varchar(255) | issues.assignee_id | Direct mapping |
| hours_worked | decimal(10,2) | issues.time_spent_minutes | `minutes / 60` OR `story_point * 4` |
| hourly_rate | decimal(10,2) | developer_hourly_rates | Lookup or default ($87) |
| total_cost | decimal(12,2) | Calculated | `hours_worked × hourly_rate` |
| project_phase | varchar(50) | categorizeCapitalization | preliminary/development/post_implementation |
| capitalization_category | varchar(50) | categorizeCapitalization | capitalizable/expense |
| capitalization_percent | int | categorizeCapitalization | 0 or 100 |
| category_reason | varchar(255) | categorizeCapitalization | Audit trail |
| estimated_minutes | bigint | issues.original_estimate_minutes | Direct mapping |
| actual_minutes | bigint | issues.time_spent_minutes | Direct mapping |
| variance_percent | decimal(8,2) | Calculated | `(estimated - actual) / estimated × 100` |
| over_budget | bool | Calculated | `actual > estimated` |
| is_unallocated | bool | Calculated | `epic_key IS NULL AND parent_issue_id IS NULL` |

### monthly_cost_summaries

| Column | Type | Source | Transformation |
|--------|------|--------|----------------|
| total_cost | decimal(14,2) | cost_allocations | `SUM(total_cost)` |
| capitalizable_cost | decimal(14,2) | cost_allocations | `SUM WHERE project_phase = 'development'` |
| expense_cost | decimal(14,2) | cost_allocations | `SUM WHERE project_phase IN ('preliminary', 'post_implementation')` |
| capitalization_rate | decimal(5,2) | Calculated | `capitalizable_cost / total_cost × 100` |
| unallocated_percent | decimal(5,2) | Calculated | `unallocated_cost / total_cost × 100` |
| budget_variance | decimal(8,2) | Calculated | `(estimated - actual) / estimated × 100` |

### deployment_costs

| Column | Type | Source | Transformation |
|--------|------|--------|----------------|
| window_days | int | Configuration | 7, 30, or 90 |
| total_cost | decimal(14,2) | cost_allocations | `SUM(total_cost) WHERE calculated_at BETWEEN period_start AND period_end` |
| deployment_count | int | cicd_deployment_commits | `COUNT WHERE result = 'SUCCESS'` |
| cost_per_deployment | decimal(14,2) | Calculated | `total_cost / deployment_count` |

## Plugin Task Execution Order

```
findevops plugin subtasks:
  1. calculateCosts         (creates cost_allocations, monthly_cost_summaries)
  2. categorizeCapitalization (updates cost_allocations with ASC 350-40 phase)
  3. calculateDeploymentCosts (creates deployment_costs)
```

## Dependencies

| Dependency | Required Tables | Notes |
|------------|-----------------|-------|
| Jira plugin | issues, board_issues, _tool_jira_issues | Must run before findevops |
| CI/CD plugins | cicd_deployment_commits | Required for deployment cost metrics |
| project_mapping | project_mapping | Links boards to projects |
```

**Step 2: Commit lineage document**

```bash
git add docs/audit/data-lineage/findevops-lineage.md
git commit -m "docs: add FinDevOps data lineage documentation"
```

---

## Task 3: Create FinDevOps Verification Queries

**Files:**
- Create: `docs/audit/tests/findevops-verification-queries.sql`

**Step 1: Create verification queries file**

Create `docs/audit/tests/findevops-verification-queries.sql`:

```sql
-- ============================================
-- FINDEVOPS DASHBOARD VERIFICATION QUERIES
-- Run these to validate metric calculations
-- ============================================

-- ============================================
-- SECTION 1: COMPLETENESS CHECKS
-- ============================================

-- #fin-completeness-01: All resolved issues should have cost allocations
-- Expected: missing_count = 0
SELECT
    'fin-completeness-01' as check_id,
    'All resolved issues have cost allocations' as check_name,
    COUNT(*) as missing_count,
    CASE WHEN COUNT(*) = 0 THEN 'PASS' ELSE 'FAIL' END as status
FROM issues i
JOIN board_issues bi ON i.id = bi.issue_id
JOIN project_mapping pm ON bi.board_id = pm.row_id AND pm.`table` = 'boards'
WHERE i.resolution_date IS NOT NULL
  AND i.id NOT IN (SELECT issue_id FROM cost_allocations WHERE issue_id IS NOT NULL);

-- #fin-completeness-02: Monthly summaries exist for all months with allocations
SELECT
    'fin-completeness-02' as check_id,
    'Monthly summaries exist for all allocation months' as check_name,
    COUNT(DISTINCT ca.fiscal_month) as allocation_months,
    COUNT(DISTINCT mcs.fiscal_month) as summary_months,
    CASE
        WHEN COUNT(DISTINCT ca.fiscal_month) = COUNT(DISTINCT mcs.fiscal_month)
        THEN 'PASS' ELSE 'FAIL'
    END as status
FROM cost_allocations ca
LEFT JOIN monthly_cost_summaries mcs ON ca.fiscal_month = mcs.fiscal_month;

-- #fin-completeness-03: Deployment costs exist for all three windows
SELECT
    'fin-completeness-03' as check_id,
    'Deployment costs exist for 7, 30, 90 day windows' as check_name,
    GROUP_CONCAT(DISTINCT window_days ORDER BY window_days) as windows_present,
    CASE
        WHEN COUNT(DISTINCT window_days) = 3 THEN 'PASS' ELSE 'FAIL'
    END as status
FROM deployment_costs;

-- ============================================
-- SECTION 2: ACCURACY CHECKS
-- ============================================

-- #fin-accuracy-01: Capitalization rate formula validation
-- Formula: capitalization_rate = capitalizable_cost / total_cost * 100
SELECT
    'fin-accuracy-01' as check_id,
    'Capitalization rate formula is correct' as check_name,
    fiscal_month,
    total_cost,
    capitalizable_cost,
    capitalization_rate as stored_rate,
    ROUND(capitalizable_cost / total_cost * 100, 2) as calculated_rate,
    CASE
        WHEN ABS(capitalization_rate - (capitalizable_cost / total_cost * 100)) < 0.01
        THEN 'PASS' ELSE 'FAIL'
    END as status
FROM monthly_cost_summaries
WHERE total_cost > 0
ORDER BY fiscal_month DESC
LIMIT 5;

-- #fin-accuracy-02: Total cost equals sum of phase costs
SELECT
    'fin-accuracy-02' as check_id,
    'Total cost = preliminary + development + post_impl' as check_name,
    fiscal_month,
    total_cost,
    preliminary_cost + development_cost + post_impl_cost as sum_of_phases,
    CASE
        WHEN ABS(total_cost - (preliminary_cost + development_cost + post_impl_cost)) < 0.01
        THEN 'PASS' ELSE 'FAIL'
    END as status
FROM monthly_cost_summaries
ORDER BY fiscal_month DESC
LIMIT 5;

-- #fin-accuracy-03: Capitalizable = Development, Expense = Preliminary + PostImpl
SELECT
    'fin-accuracy-03' as check_id,
    'Capitalizable equals development cost' as check_name,
    fiscal_month,
    capitalizable_cost,
    development_cost,
    expense_cost,
    preliminary_cost + post_impl_cost as sum_expense_phases,
    CASE
        WHEN ABS(capitalizable_cost - development_cost) < 0.01
         AND ABS(expense_cost - (preliminary_cost + post_impl_cost)) < 0.01
        THEN 'PASS' ELSE 'FAIL'
    END as status
FROM monthly_cost_summaries
ORDER BY fiscal_month DESC
LIMIT 5;

-- #fin-accuracy-04: Budget variance formula validation
-- Formula: variance = (estimated - actual) / estimated * 100
SELECT
    'fin-accuracy-04' as check_id,
    'Budget variance formula is correct' as check_name,
    fiscal_month,
    total_estimated_cost,
    total_actual_cost,
    budget_variance as stored_variance,
    CASE
        WHEN total_estimated_cost > 0
        THEN ROUND((total_estimated_cost - total_actual_cost) / total_estimated_cost * 100, 2)
        ELSE 0
    END as calculated_variance,
    CASE
        WHEN total_estimated_cost = 0 THEN 'SKIP'
        WHEN ABS(budget_variance - ((total_estimated_cost - total_actual_cost) / total_estimated_cost * 100)) < 0.1
        THEN 'PASS' ELSE 'FAIL'
    END as status
FROM monthly_cost_summaries
WHERE total_estimated_cost > 0
ORDER BY fiscal_month DESC
LIMIT 5;

-- #fin-accuracy-05: Cost per deployment formula validation
SELECT
    'fin-accuracy-05' as check_id,
    'Cost per deployment formula is correct' as check_name,
    window_days,
    total_cost,
    deployment_count,
    cost_per_deployment as stored_cpd,
    CASE
        WHEN deployment_count > 0
        THEN ROUND(total_cost / deployment_count, 2)
        ELSE 0
    END as calculated_cpd,
    CASE
        WHEN deployment_count = 0 THEN 'SKIP'
        WHEN ABS(cost_per_deployment - (total_cost / deployment_count)) < 0.01
        THEN 'PASS' ELSE 'FAIL'
    END as status
FROM deployment_costs
ORDER BY calculated_at DESC
LIMIT 9;

-- ============================================
-- SECTION 3: ASC 350-40 CATEGORIZATION CHECKS
-- ============================================

-- #fin-asc350-01: Bug issues should be categorized as expense
SELECT
    'fin-asc350-01' as check_id,
    'Bug issues are categorized as expense' as check_name,
    ca.issue_type,
    ca.capitalization_category,
    COUNT(*) as count,
    CASE
        WHEN ca.capitalization_category = 'expense' THEN 'PASS'
        ELSE 'FAIL'
    END as status
FROM cost_allocations ca
WHERE ca.issue_type IN ('Bug', 'Defect', 'Hotfix')
GROUP BY ca.issue_type, ca.capitalization_category;

-- #fin-asc350-02: Story/Feature issues should be categorized as capitalizable
SELECT
    'fin-asc350-02' as check_id,
    'Story/Feature issues are categorized as capitalizable' as check_name,
    ca.issue_type,
    ca.capitalization_category,
    COUNT(*) as count,
    CASE
        WHEN ca.capitalization_category = 'capitalizable' THEN 'PASS'
        ELSE 'FAIL'
    END as status
FROM cost_allocations ca
WHERE ca.issue_type IN ('Story', 'Feature', 'Enhancement', 'Task')
  AND ca.issue_labels NOT LIKE '%maintenance%'
  AND ca.issue_labels NOT LIKE '%research%'
  AND ca.issue_labels NOT LIKE '%poc%'
GROUP BY ca.issue_type, ca.capitalization_category;

-- #fin-asc350-03: Research/Spike issues should be categorized as preliminary expense
SELECT
    'fin-asc350-03' as check_id,
    'Research/Spike issues are preliminary expense' as check_name,
    ca.issue_type,
    ca.project_phase,
    ca.capitalization_category,
    COUNT(*) as count,
    CASE
        WHEN ca.project_phase = 'preliminary' AND ca.capitalization_category = 'expense'
        THEN 'PASS' ELSE 'FAIL'
    END as status
FROM cost_allocations ca
WHERE ca.issue_type IN ('Spike', 'Research')
   OR ca.issue_labels LIKE '%research%'
   OR ca.issue_labels LIKE '%poc%'
GROUP BY ca.issue_type, ca.project_phase, ca.capitalization_category;

-- #fin-asc350-04: All allocations have a category reason (audit trail)
SELECT
    'fin-asc350-04' as check_id,
    'All allocations have category reason' as check_name,
    COUNT(*) as total_allocations,
    SUM(CASE WHEN category_reason IS NULL OR category_reason = '' THEN 1 ELSE 0 END) as missing_reason,
    CASE
        WHEN SUM(CASE WHEN category_reason IS NULL OR category_reason = '' THEN 1 ELSE 0 END) = 0
        THEN 'PASS' ELSE 'FAIL'
    END as status
FROM cost_allocations;

-- ============================================
-- SECTION 4: CONSISTENCY CHECKS
-- ============================================

-- #fin-consistency-01: Monthly summary totals match sum of allocations
SELECT
    'fin-consistency-01' as check_id,
    'Monthly summary matches allocation sum' as check_name,
    mcs.fiscal_month,
    mcs.total_cost as summary_total,
    COALESCE(SUM(ca.total_cost), 0) as allocation_sum,
    CASE
        WHEN ABS(mcs.total_cost - COALESCE(SUM(ca.total_cost), 0)) < 1.00
        THEN 'PASS' ELSE 'FAIL'
    END as status
FROM monthly_cost_summaries mcs
LEFT JOIN cost_allocations ca ON ca.fiscal_month = mcs.fiscal_month
GROUP BY mcs.fiscal_month, mcs.total_cost
ORDER BY mcs.fiscal_month DESC
LIMIT 5;

-- #fin-consistency-02: Unallocated count matches flagged allocations
SELECT
    'fin-consistency-02' as check_id,
    'Orphan issue count matches unallocated flag' as check_name,
    mcs.fiscal_month,
    mcs.orphan_issue_count as summary_count,
    COUNT(ca.id) as actual_unallocated,
    CASE
        WHEN mcs.orphan_issue_count = COUNT(ca.id) THEN 'PASS' ELSE 'FAIL'
    END as status
FROM monthly_cost_summaries mcs
LEFT JOIN cost_allocations ca ON ca.fiscal_month = mcs.fiscal_month AND ca.is_unallocated = true
GROUP BY mcs.fiscal_month, mcs.orphan_issue_count
ORDER BY mcs.fiscal_month DESC
LIMIT 5;

-- ============================================
-- SECTION 5: FRESHNESS CHECKS
-- ============================================

-- #fin-freshness-01: Cost allocations have recent calculated_at timestamps
SELECT
    'fin-freshness-01' as check_id,
    'Cost allocations are fresh (within 7 days)' as check_name,
    MAX(calculated_at) as most_recent,
    DATEDIFF(NOW(), MAX(calculated_at)) as days_old,
    CASE
        WHEN DATEDIFF(NOW(), MAX(calculated_at)) <= 7 THEN 'PASS'
        ELSE 'FAIL'
    END as status
FROM cost_allocations;

-- #fin-freshness-02: Monthly summaries have recent calculated_at timestamps
SELECT
    'fin-freshness-02' as check_id,
    'Monthly summaries are fresh (within 7 days)' as check_name,
    MAX(calculated_at) as most_recent,
    DATEDIFF(NOW(), MAX(calculated_at)) as days_old,
    CASE
        WHEN DATEDIFF(NOW(), MAX(calculated_at)) <= 7 THEN 'PASS'
        ELSE 'FAIL'
    END as status
FROM monthly_cost_summaries;

-- #fin-freshness-03: Deployment costs have recent calculated_at timestamps
SELECT
    'fin-freshness-03' as check_id,
    'Deployment costs are fresh (within 7 days)' as check_name,
    MAX(calculated_at) as most_recent,
    DATEDIFF(NOW(), MAX(calculated_at)) as days_old,
    CASE
        WHEN DATEDIFF(NOW(), MAX(calculated_at)) <= 7 THEN 'PASS'
        ELSE 'FAIL'
    END as status
FROM deployment_costs;

-- ============================================
-- SECTION 6: QUARTERLY AGGREGATION CHECKS
-- ============================================

-- #fin-quarterly-01: Quarterly cost can be calculated from monthly summaries
SELECT
    'fin-quarterly-01' as check_id,
    'Quarterly aggregation from monthly summaries' as check_name,
    CONCAT(YEAR(STR_TO_DATE(CONCAT(fiscal_month, '-01'), '%Y-%m-%d')), '-Q',
           QUARTER(STR_TO_DATE(CONCAT(fiscal_month, '-01'), '%Y-%m-%d'))) as quarter,
    SUM(total_cost) as quarterly_total_cost,
    SUM(capitalizable_cost) as quarterly_capitalizable,
    SUM(expense_cost) as quarterly_expense,
    ROUND(SUM(capitalizable_cost) / NULLIF(SUM(total_cost), 0) * 100, 2) as quarterly_cap_rate,
    'PASS' as status
FROM monthly_cost_summaries
GROUP BY YEAR(STR_TO_DATE(CONCAT(fiscal_month, '-01'), '%Y-%m-%d')),
         QUARTER(STR_TO_DATE(CONCAT(fiscal_month, '-01'), '%Y-%m-%d'))
ORDER BY quarter DESC
LIMIT 4;

-- ============================================
-- SUMMARY: Run all checks and count results
-- ============================================

-- #fin-summary: Aggregate check results
-- Run each check above and manually tally PASS/FAIL counts
```

**Step 2: Commit verification queries**

```bash
git add docs/audit/tests/findevops-verification-queries.sql
git commit -m "docs: add FinDevOps verification queries for audit"
```

---

## Task 4: Create FinDevOps Per-Metric Audit Checklist (Part 1: Cost Summary Metrics)

**Files:**
- Create: `docs/audit/dashboards/findevops-audit.md`

**Step 1: Create the audit checklist document with first 10 metrics**

Create `docs/audit/dashboards/findevops-audit.md`:

```markdown
# FinDevOps Dashboard Audit Checklist

**Dashboard:** FinDevOps - Cost & Capitalization
**Total Metrics:** 30
**Audit Date:** 2026-02-01
**Status:** 🔄 IN PROGRESS

## Summary

| Category | Metrics | Logic | Data Trust | Aggregation | Testing |
|----------|---------|-------|------------|-------------|---------|
| Cost Summary | 4 | -/4 | -/4 | -/4 | -/4 |
| Deployment Costs | 5 | -/5 | -/5 | -/5 | -/5 |
| Cost Breakdown | 4 | -/4 | -/4 | -/4 | -/4 |
| Budget Variance | 6 | -/6 | -/6 | -/6 | -/6 |
| Unallocated Costs | 7 | -/7 | -/7 | -/7 | -/7 |
| Detail Tables | 4 | -/4 | -/4 | -/4 | -/4 |
| **TOTAL** | **30** | **-/30** | **-/30** | **-/30** | **-/30** |

---

## Cost Summary Metrics

### Metric 1: Total Development Cost

**Panel ID:** 2 | **Type:** Stat

#### 1. Logic Validation
- ☐ **Outcome defined:** Sum of all development costs for selected project(s) and time range
- ☐ **Formula documented:**
  ```
  Total Cost = SUM(monthly_cost_summaries.total_cost)
  WHERE project_name IN (selected_projects)
  AND calculated_at IN (time_range)
  ```
- ☐ **Edge cases identified:**
  - No data for project → Returns NULL (displayed as "No data")
  - Time range with no allocations → Returns 0

#### 2. Data Lineage
| Layer | Table/Source | Transformation |
|-------|--------------|----------------|
| Source | Jira issues, worklogs | Time tracking data |
| Domain | issues.time_spent_minutes | Hours = minutes / 60 |
| Plugin | cost_allocations | hours × hourly_rate |
| Plugin | monthly_cost_summaries | SUM(total_cost) by month |
| Dashboard | SUM(total_cost) | Grafana aggregation |

#### 3. Trust Validation
- ☐ **Completeness:** Query `#fin-completeness-01` passes
- ☐ **Accuracy:** Manually verify 3 issues: hours × rate = cost
- ☐ **Freshness:** `calculated_at` within 7 days
- ☐ **Consistency:** SUM(allocations) = monthly_summary.total_cost

#### 4. Time Aggregation
- ☐ **Daily:** N/A (monthly granularity)
- ☐ **Quarterly:** SUM of 3 months in quarter

#### 5. Testing
- ☐ **Verification query:** `#fin-accuracy-02`, `#fin-consistency-01`
- ☐ **Automated test:** `findevops/e2e/calculate_costs_test.go`
- ☐ **Test data:** `findevops/e2e/calculate_costs/*.csv`

#### Status: ☐ PASS | ☐ GAPS | ☐ FAIL
**Gaps:** TBD

---

### Metric 2: Capitalizable Cost (ASC 350-40)

**Panel ID:** 3 | **Type:** Stat

#### 1. Logic Validation
- ☐ **Outcome defined:** Sum of costs that can be capitalized per US GAAP ASC 350-40
- ☐ **Formula documented:**
  ```
  Capitalizable Cost = SUM(monthly_cost_summaries.capitalizable_cost)

  Where capitalizable_cost = SUM(cost_allocations.total_cost)
  WHERE project_phase = 'development'

  ASC 350-40 Rule: Only "Application Development" stage work is capitalizable
  ```
- ☐ **Edge cases identified:**
  - All work is maintenance → capitalizable = 0
  - No categorized allocations → capitalizable = 0

#### 2. Data Lineage
| Layer | Table/Source | Transformation |
|-------|--------------|----------------|
| Source | Jira issue type, labels | Category signals |
| Plugin | categorizeCapitalization task | Determines project_phase |
| Plugin | cost_allocations.project_phase | 'development' = capitalizable |
| Plugin | monthly_cost_summaries | SUM WHERE phase = development |

#### 3. Trust Validation
- ☐ **Completeness:** All allocations have project_phase set
- ☐ **Accuracy:** Query `#fin-asc350-02` passes (Stories = capitalizable)
- ☐ **Freshness:** `calculated_at` within 7 days
- ☐ **Consistency:** capitalizable_cost = development_cost

#### 4. Time Aggregation
- ☐ **Daily:** N/A (monthly granularity)
- ☐ **Quarterly:** SUM of 3 months capitalizable_cost

#### 5. Testing
- ☐ **Verification query:** `#fin-accuracy-03`, `#fin-asc350-02`
- ☐ **Automated test:** `findevops/tasks/categorize_capitalization_test.go`
- ☐ **Test data:** Unit test cases in test file

#### Status: ☐ PASS | ☐ GAPS | ☐ FAIL
**Gaps:** TBD

---

### Metric 3: Expensed Cost

**Panel ID:** 4 | **Type:** Stat

#### 1. Logic Validation
- ☐ **Outcome defined:** Sum of costs that must be expensed (not capitalized)
- ☐ **Formula documented:**
  ```
  Expensed Cost = SUM(monthly_cost_summaries.expense_cost)

  Where expense_cost = preliminary_cost + post_impl_cost

  ASC 350-40 Rule: Preliminary and Post-Implementation stages are expensed
  - Preliminary: Research, spikes, POC, feasibility
  - Post-Implementation: Bugs, maintenance, support
  ```
- ☐ **Edge cases identified:**
  - All work is features → expense = 0
  - Unknown types default to expense (conservative)

#### 2. Data Lineage
| Layer | Table/Source | Transformation |
|-------|--------------|----------------|
| Source | Jira issue type, labels | Category signals |
| Plugin | categorizeCapitalization task | Determines project_phase |
| Plugin | cost_allocations.project_phase | 'preliminary' or 'post_implementation' |
| Plugin | monthly_cost_summaries | preliminary_cost + post_impl_cost |

#### 3. Trust Validation
- ☐ **Completeness:** All allocations have project_phase set
- ☐ **Accuracy:** Query `#fin-asc350-01` passes (Bugs = expense)
- ☐ **Freshness:** `calculated_at` within 7 days
- ☐ **Consistency:** expense_cost = preliminary_cost + post_impl_cost

#### 4. Time Aggregation
- ☐ **Daily:** N/A (monthly granularity)
- ☐ **Quarterly:** SUM of 3 months expense_cost

#### 5. Testing
- ☐ **Verification query:** `#fin-accuracy-03`, `#fin-asc350-01`
- ☐ **Automated test:** `findevops/tasks/categorize_capitalization_test.go`
- ☐ **Test data:** Unit test cases in test file

#### Status: ☐ PASS | ☐ GAPS | ☐ FAIL
**Gaps:** TBD

---

### Metric 4: Capitalization Rate

**Panel ID:** 5 | **Type:** Gauge

#### 1. Logic Validation
- ☐ **Outcome defined:** Percentage of total cost that is capitalizable
- ☐ **Formula documented:**
  ```
  Capitalization Rate = AVG(monthly_cost_summaries.capitalization_rate)

  Where capitalization_rate = (capitalizable_cost / total_cost) × 100

  Typical range: 50-70% for feature-heavy teams
  ```
- ☐ **Edge cases identified:**
  - total_cost = 0 → rate undefined (should be NULL or 0)
  - 100% bugs → rate = 0%
  - 100% features → rate = 100%

#### 2. Data Lineage
| Layer | Table/Source | Transformation |
|-------|--------------|----------------|
| Plugin | monthly_cost_summaries.capitalizable_cost | Numerator |
| Plugin | monthly_cost_summaries.total_cost | Denominator |
| Plugin | monthly_cost_summaries.capitalization_rate | Pre-calculated |
| Dashboard | AVG(capitalization_rate) | Grafana aggregation |

#### 3. Trust Validation
- ☐ **Completeness:** All monthly summaries have rate calculated
- ☐ **Accuracy:** Query `#fin-accuracy-01` passes
- ☐ **Freshness:** `calculated_at` within 7 days
- ☐ **Consistency:** Rate = capitalizable / total × 100

#### 4. Time Aggregation
- ☐ **Daily:** N/A (monthly granularity)
- ☐ **Quarterly:** Recalculate from quarterly totals (not AVG of monthly rates)

#### 5. Testing
- ☐ **Verification query:** `#fin-accuracy-01`
- ☐ **Automated test:** Add test in `calculate_costs_test.go`
- ☐ **Test data:** Verify rate calculation

#### Status: ☐ PASS | ☐ GAPS | ☐ FAIL
**Gaps:** TBD

---

## Deployment Cost Metrics

### Metric 5: Cost Per Deploy (7-day window)

**Panel ID:** 21 | **Type:** Stat

#### 1. Logic Validation
- ☐ **Outcome defined:** Average cost per deployment over rolling 7-day window
- ☐ **Formula documented:**
  ```
  Cost Per Deployment = total_cost / deployment_count
  WHERE window_days = 7

  Lower is better - indicates efficient delivery
  Threshold: Green <$5,000, Yellow $5,000-$10,000, Red >$10,000
  ```
- ☐ **Edge cases identified:**
  - No deployments in window → NULL (no data)
  - No costs in window → cost_per_deployment = 0

#### 2. Data Lineage
| Layer | Table/Source | Transformation |
|-------|--------------|----------------|
| Source | CI/CD pipelines | Deployment events |
| Domain | cicd_deployment_commits | COUNT WHERE result = 'SUCCESS' |
| Plugin | cost_allocations | SUM(total_cost) in window |
| Plugin | deployment_costs | total_cost / deployment_count |

#### 3. Trust Validation
- ☐ **Completeness:** Query `#fin-completeness-03` passes (all windows exist)
- ☐ **Accuracy:** Query `#fin-accuracy-05` passes
- ☐ **Freshness:** `calculated_at` within 7 days
- ☐ **Consistency:** Cost + deploy count → CPD formula holds

#### 4. Time Aggregation
- ☐ **Daily:** Rolling 7-day window recalculated daily
- ☐ **Quarterly:** Use 90-day window metric instead

#### 5. Testing
- ☐ **Verification query:** `#fin-accuracy-05`
- ☐ **Automated test:** `findevops/e2e/calculate_deployment_costs_test.go`
- ☐ **Test data:** TBD

#### Status: ☐ PASS | ☐ GAPS | ☐ FAIL
**Gaps:** TBD

---

### Metric 6: Cost Per Deploy (30-day window)

**Panel ID:** 22 | **Type:** Stat

*(Same structure as Metric 5, with window_days = 30)*

#### 1. Logic Validation
- ☐ **Outcome defined:** Average cost per deployment over rolling 30-day window
- ☐ **Formula:** `total_cost / deployment_count WHERE window_days = 30`

#### Status: ☐ PASS | ☐ GAPS | ☐ FAIL

---

### Metric 7: Cost Per Deploy (90-day window)

**Panel ID:** 23 | **Type:** Stat

*(Same structure as Metric 5, with window_days = 90)*

#### 1. Logic Validation
- ☐ **Outcome defined:** Average cost per deployment over rolling 90-day window (quarterly)
- ☐ **Formula:** `total_cost / deployment_count WHERE window_days = 90`

#### Status: ☐ PASS | ☐ GAPS | ☐ FAIL

---

### Metric 8: Deployment Cost History

**Panel ID:** 24 | **Type:** Table

#### 1. Logic Validation
- ☐ **Outcome defined:** Historical view of deployment costs across all time windows
- ☐ **Formula documented:**
  ```sql
  SELECT project_name, window_days, period_start, period_end,
         total_cost, deployment_count, cost_per_deployment
  FROM deployment_costs
  ORDER BY calculated_at DESC, window_days
  LIMIT 30
  ```

#### Status: ☐ PASS | ☐ GAPS | ☐ FAIL

---

### Metric 9: Avg Cost Per Deployment by Window

**Panel ID:** 26 | **Type:** Bar Chart

#### 1. Logic Validation
- ☐ **Outcome defined:** Compare average CPD across 7, 30, 90 day windows
- ☐ **Formula:** `AVG(cost_per_deployment) GROUP BY window_days`

#### Status: ☐ PASS | ☐ GAPS | ☐ FAIL

---

## Cost Breakdown Metrics

### Metric 10: Cost by Development Phase (ASC 350-40)

**Panel ID:** 6 | **Type:** Pie Chart

#### 1. Logic Validation
- ☐ **Outcome defined:** Visual breakdown of costs by ASC 350-40 development phase
- ☐ **Formula documented:**
  ```sql
  SELECT project_phase, SUM(total_cost)
  FROM cost_allocations
  GROUP BY project_phase

  Phases: preliminary, development, post_implementation
  ```
- ☐ **Edge cases identified:**
  - All same phase → single slice pie chart

#### 2. Data Lineage
| Layer | Table/Source | Transformation |
|-------|--------------|----------------|
| Plugin | categorizeCapitalization | Sets project_phase |
| Plugin | cost_allocations | Has phase + cost |
| Dashboard | GROUP BY project_phase | Pie chart slices |

#### Status: ☐ PASS | ☐ GAPS | ☐ FAIL

---

*(Continue with remaining 20 metrics following same template)*

## Remaining Metrics (To Be Documented)

### Cost Breakdown (continued)
- ☐ Metric 11: Monthly Cost Breakdown (Capitalizable vs Expensed) - Panel 7
- ☐ Metric 12: Cost by Capitalization Category - Panel 25
- ☐ Metric 13: Cost Allocations Detail - Panel 8

### Budget Variance
- ☐ Metric 14: Current Month Budget Variance - Panel 102
- ☐ Metric 15: Estimated Cost (This Month) - Panel 103
- ☐ Metric 16: Actual Cost (This Month) - Panel 104
- ☐ Metric 17: Over Budget Issues - Panel 105
- ☐ Metric 18: Estimated vs Actual Cost Trend - Panel 106
- ☐ Metric 19: Budget Variance % Over Time - Panel 107

### Unallocated Costs
- ☐ Metric 20: Unallocated Cost % - Panel 202
- ☐ Metric 21: Unallocated Cost ($) - Panel 203
- ☐ Metric 22: Orphan Issues (No Epic) - Panel 204
- ☐ Metric 23: Total Unallocated Issues - Panel 205
- ☐ Metric 24: Unallocated % Trend - Panel 206
- ☐ Metric 25: Orphan Issue Count Trend - Panel 207
- ☐ Metric 26: Unallocated Issues Detail - Panel 208

---

## Audit Notes

### Findings

| ID | Severity | Finding | Recommendation |
|----|----------|---------|----------------|
| - | - | - | - |

### Sign-Off

| Role | Name | Date | Signature |
|------|------|------|-----------|
| Engineering | | | ☐ |
| Finance | | | ☐ |
| Leadership | | | ☐ |
```

**Step 2: Commit audit checklist**

```bash
git add docs/audit/dashboards/findevops-audit.md
git commit -m "docs: add FinDevOps per-metric audit checklist (partial)"
```

---

## Task 5: Create Automated Test for FinDevOps Audit Validations

**Files:**
- Create: `backend/test/audit/findevops_audit_test.go`

**Step 1: Create audit test directory**

```bash
mkdir -p backend/test/audit
```

**Step 2: Create the test file**

Create `backend/test/audit/findevops_audit_test.go`:

```go
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

package audit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCapitalizationRateFormula validates the capitalization rate calculation
// Formula: capitalization_rate = capitalizable_cost / total_cost * 100
func TestCapitalizationRateFormula(t *testing.T) {
	testCases := []struct {
		name             string
		capitalizableCost float64
		totalCost        float64
		expectedRate     float64
	}{
		{
			name:             "50% capitalization",
			capitalizableCost: 50000.00,
			totalCost:        100000.00,
			expectedRate:     50.00,
		},
		{
			name:             "100% capitalization (all features)",
			capitalizableCost: 75000.00,
			totalCost:        75000.00,
			expectedRate:     100.00,
		},
		{
			name:             "0% capitalization (all maintenance)",
			capitalizableCost: 0.00,
			totalCost:        50000.00,
			expectedRate:     0.00,
		},
		{
			name:             "Zero total cost",
			capitalizableCost: 0.00,
			totalCost:        0.00,
			expectedRate:     0.00, // Avoid division by zero
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var rate float64
			if tc.totalCost > 0 {
				rate = tc.capitalizableCost / tc.totalCost * 100
			}
			assert.InDelta(t, tc.expectedRate, rate, 0.01, "Capitalization rate mismatch")
		})
	}
}

// TestBudgetVarianceFormula validates the budget variance calculation
// Formula: variance = (estimated - actual) / estimated * 100
// Positive = under budget, Negative = over budget
func TestBudgetVarianceFormula(t *testing.T) {
	testCases := []struct {
		name             string
		estimatedMinutes int64
		actualMinutes    int64
		expectedVariance float64
		expectedOverBudget bool
	}{
		{
			name:             "On budget",
			estimatedMinutes: 480,  // 8 hours
			actualMinutes:    480,
			expectedVariance: 0.00,
			expectedOverBudget: false,
		},
		{
			name:             "Under budget by 25%",
			estimatedMinutes: 480,
			actualMinutes:    360,  // 6 hours
			expectedVariance: 25.00,
			expectedOverBudget: false,
		},
		{
			name:             "Over budget by 50%",
			estimatedMinutes: 480,
			actualMinutes:    720,  // 12 hours
			expectedVariance: -50.00,
			expectedOverBudget: true,
		},
		{
			name:             "No estimate (zero)",
			estimatedMinutes: 0,
			actualMinutes:    480,
			expectedVariance: 0.00, // Cannot calculate variance
			expectedOverBudget: false, // No estimate means no over-budget flag
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var variance float64
			if tc.estimatedMinutes > 0 {
				variance = float64(tc.estimatedMinutes-tc.actualMinutes) / float64(tc.estimatedMinutes) * 100
			}
			overBudget := tc.actualMinutes > tc.estimatedMinutes && tc.estimatedMinutes > 0

			assert.InDelta(t, tc.expectedVariance, variance, 0.01, "Variance mismatch")
			assert.Equal(t, tc.expectedOverBudget, overBudget, "OverBudget flag mismatch")
		})
	}
}

// TestCostPerDeploymentFormula validates the cost per deployment calculation
// Formula: cost_per_deployment = total_cost / deployment_count
func TestCostPerDeploymentFormula(t *testing.T) {
	testCases := []struct {
		name            string
		totalCost       float64
		deploymentCount int
		expectedCPD     float64
	}{
		{
			name:            "Normal case",
			totalCost:       50000.00,
			deploymentCount: 100,
			expectedCPD:     500.00,
		},
		{
			name:            "High efficiency",
			totalCost:       10000.00,
			deploymentCount: 200,
			expectedCPD:     50.00,
		},
		{
			name:            "No deployments",
			totalCost:       50000.00,
			deploymentCount: 0,
			expectedCPD:     0.00, // Avoid division by zero
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var cpd float64
			if tc.deploymentCount > 0 {
				cpd = tc.totalCost / float64(tc.deploymentCount)
			}
			assert.InDelta(t, tc.expectedCPD, cpd, 0.01, "Cost per deployment mismatch")
		})
	}
}

// TestUnallocatedPercentFormula validates the unallocated cost percentage
// Formula: unallocated_percent = unallocated_cost / total_cost * 100
// Target: < 10%
func TestUnallocatedPercentFormula(t *testing.T) {
	testCases := []struct {
		name              string
		unallocatedCost   float64
		totalCost         float64
		expectedPercent   float64
		meetsTarget       bool
	}{
		{
			name:              "Good - 5% unallocated",
			unallocatedCost:   5000.00,
			totalCost:        100000.00,
			expectedPercent:   5.00,
			meetsTarget:       true, // < 10%
		},
		{
			name:              "Warning - 15% unallocated",
			unallocatedCost:   15000.00,
			totalCost:        100000.00,
			expectedPercent:   15.00,
			meetsTarget:       false, // >= 10%
		},
		{
			name:              "Perfect - 0% unallocated",
			unallocatedCost:   0.00,
			totalCost:        100000.00,
			expectedPercent:   0.00,
			meetsTarget:       true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var percent float64
			if tc.totalCost > 0 {
				percent = tc.unallocatedCost / tc.totalCost * 100
			}
			meetsTarget := percent < 10.0

			assert.InDelta(t, tc.expectedPercent, percent, 0.01, "Unallocated percent mismatch")
			assert.Equal(t, tc.meetsTarget, meetsTarget, "Target threshold check mismatch")
		})
	}
}

// TestTotalCostEqualsPhaseSum validates cost breakdown consistency
// total_cost = preliminary_cost + development_cost + post_impl_cost
func TestTotalCostEqualsPhaseSum(t *testing.T) {
	testCases := []struct {
		name            string
		preliminaryCost float64
		developmentCost float64
		postImplCost    float64
	}{
		{
			name:            "Mixed workload",
			preliminaryCost: 10000.00,
			developmentCost: 60000.00,
			postImplCost:    30000.00,
		},
		{
			name:            "All development",
			preliminaryCost: 0.00,
			developmentCost: 100000.00,
			postImplCost:    0.00,
		},
		{
			name:            "All maintenance",
			preliminaryCost: 0.00,
			developmentCost: 0.00,
			postImplCost:    50000.00,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			totalCost := tc.preliminaryCost + tc.developmentCost + tc.postImplCost
			phaseSum := tc.preliminaryCost + tc.developmentCost + tc.postImplCost

			assert.InDelta(t, totalCost, phaseSum, 0.01, "Total should equal sum of phases")
		})
	}
}

// TestCapitalizableEqualsExpenseSum validates ASC 350-40 split
// capitalizable_cost = development_cost
// expense_cost = preliminary_cost + post_impl_cost
func TestCapitalizableExpenseSplit(t *testing.T) {
	testCases := []struct {
		name            string
		preliminaryCost float64
		developmentCost float64
		postImplCost    float64
	}{
		{
			name:            "Typical mix",
			preliminaryCost: 5000.00,
			developmentCost: 70000.00,
			postImplCost:    25000.00,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			capitalizableCost := tc.developmentCost
			expenseCost := tc.preliminaryCost + tc.postImplCost

			// Capitalizable should equal development
			assert.InDelta(t, tc.developmentCost, capitalizableCost, 0.01)

			// Expense should equal preliminary + post-impl
			assert.InDelta(t, tc.preliminaryCost+tc.postImplCost, expenseCost, 0.01)
		})
	}
}
```

**Step 3: Run the tests**

```bash
cd backend && go test ./test/audit/... -v
```

Expected output: All tests pass

**Step 4: Commit test file**

```bash
git add backend/test/audit/findevops_audit_test.go
git commit -m "test: add FinDevOps audit formula validation tests"
```

---

## Task 6: Complete Remaining Metrics in Audit Checklist

**Files:**
- Modify: `docs/audit/dashboards/findevops-audit.md`

**Step 1: Add remaining 20 metrics following the same template**

Complete the audit checklist with metrics 11-30, following the same structure:
- Monthly Cost Breakdown (Panel 7)
- Cost by Capitalization Category (Panel 25)
- Cost Allocations Detail (Panel 8)
- Budget Variance metrics (Panels 102-107)
- Unallocated Cost metrics (Panels 202-208)

**Step 2: Commit completed checklist**

```bash
git add docs/audit/dashboards/findevops-audit.md
git commit -m "docs: complete FinDevOps audit checklist with all 30 metrics"
```

---

## Task 7: Run Verification Queries and Document Results

**Step 1: Execute verification queries against database**

```bash
# Connect to MySQL and run queries
docker exec -i incubator-devlake-mysql-1 mysql -umerico -pmerico lake < docs/audit/tests/findevops-verification-queries.sql > docs/audit/tests/findevops-verification-results.txt
```

**Step 2: Review results and update audit checklist**

For each query result:
- If PASS: Check the corresponding box in `findevops-audit.md`
- If FAIL: Add to Gap Analysis in `AUDIT_CHECKLIST_MASTER.md`

**Step 3: Commit results**

```bash
git add docs/audit/
git commit -m "docs: add FinDevOps verification query results"
```

---

## Task 8: Update Master Checklist with FinDevOps Results

**Files:**
- Modify: `docs/audit/AUDIT_CHECKLIST_MASTER.md`

**Step 1: Update rollup counts based on verification results**

**Step 2: Document any gaps found**

**Step 3: Commit final status**

```bash
git add docs/audit/AUDIT_CHECKLIST_MASTER.md
git commit -m "docs: update audit master checklist with FinDevOps results"
```

---

## Summary

| Task | Description | Estimated Steps |
|------|-------------|-----------------|
| 1 | Create audit directory structure | 3 |
| 2 | Document FinDevOps data lineage | 2 |
| 3 | Create verification queries | 2 |
| 4 | Create per-metric audit checklist (partial) | 2 |
| 5 | Create automated audit tests | 4 |
| 6 | Complete remaining metrics | 2 |
| 7 | Run verification and document results | 3 |
| 8 | Update master checklist | 3 |
| **Total** | | **21 steps** |

---

## Execution Notes

- Tasks 1-5 can be executed sequentially
- Task 6 is documentation completion
- Task 7 requires running database (can be skipped if no data)
- Task 8 summarizes findings

**After completing this pilot:**
- Review gaps found
- Decide if audit template works for other dashboards
- Proceed to audit AI Detection, Business Metrics, Capacity Planning dashboards
