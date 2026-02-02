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
