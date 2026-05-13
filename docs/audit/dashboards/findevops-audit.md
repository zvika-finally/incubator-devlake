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

### Metric 11: Monthly Cost Breakdown (Capitalizable vs Expensed)

**Panel ID:** 7 | **Type:** Time Series

#### 1. Logic Validation
- ☐ **Outcome defined:** Monthly trend showing capitalizable vs expensed costs over time
- ☐ **Formula:**
  ```sql
  SELECT calculated_at as time,
         capitalizable_cost as "Capitalizable",
         expense_cost as "Expensed"
  FROM monthly_cost_summaries
  WHERE project_name IN (selected_projects)
  ORDER BY calculated_at
  ```

#### Status: ☐ PASS | ☐ GAPS | ☐ FAIL

---

### Metric 12: Cost by Capitalization Category

**Panel ID:** 25 | **Type:** Pie Chart

#### 1. Logic Validation
- ☐ **Outcome defined:** Distribution of costs across detailed ASC 350-40 categories
- ☐ **Formula:**
  ```sql
  SELECT capitalization_category, SUM(total_cost)
  FROM cost_allocations
  WHERE project_name IN (selected_projects)
  GROUP BY capitalization_category

  Categories: feature_development, bug_fix, maintenance, research, support
  ```

#### Status: ☐ PASS | ☐ GAPS | ☐ FAIL

---

### Metric 13: Cost Allocations Detail

**Panel ID:** 8 | **Type:** Table

#### 1. Logic Validation
- ☐ **Outcome defined:** Detailed view of individual cost allocation records
- ☐ **Formula:**
  ```sql
  SELECT issue_key, developer_name, hours_worked,
         hourly_rate, total_cost, project_phase,
         capitalization_category, work_date
  FROM cost_allocations
  WHERE project_name IN (selected_projects)
  ORDER BY work_date DESC
  LIMIT 100
  ```

#### Status: ☐ PASS | ☐ GAPS | ☐ FAIL

---

## Budget Variance Metrics

### Metric 14: Current Month Budget Variance

**Panel ID:** 102 | **Type:** Stat

#### 1. Logic Validation
- ☐ **Outcome defined:** Difference between estimated and actual costs for current month
- ☐ **Formula:**
  ```
  Budget Variance = estimated_cost - actual_cost
  WHERE calculated_at = CURRENT_MONTH

  Positive = under budget (good)
  Negative = over budget (concern)
  ```
- ☐ **Edge cases identified:**
  - No estimate → variance undefined (NULL)
  - No actual costs yet → variance = estimated_cost

#### Status: ☐ PASS | ☐ GAPS | ☐ FAIL

---

### Metric 15: Estimated Cost (This Month)

**Panel ID:** 103 | **Type:** Stat

#### 1. Logic Validation
- ☐ **Outcome defined:** Projected total cost for current month based on velocity
- ☐ **Formula:**
  ```
  Estimated Cost = SUM(issue.story_points × avg_cost_per_point)
  WHERE issue.sprint = CURRENT_SPRINT
  AND issue.status != 'Done'

  avg_cost_per_point = rolling 3-month average from historical data
  ```

#### Status: ☐ PASS | ☐ GAPS | ☐ FAIL

---

### Metric 16: Actual Cost (This Month)

**Panel ID:** 104 | **Type:** Stat

#### 1. Logic Validation
- ☐ **Outcome defined:** Sum of actual costs incurred to date in current month
- ☐ **Formula:**
  ```
  Actual Cost = SUM(cost_allocations.total_cost)
  WHERE work_date >= FIRST_DAY_OF_MONTH
  AND work_date <= TODAY
  ```

#### Status: ☐ PASS | ☐ GAPS | ☐ FAIL

---

### Metric 17: Over Budget Issues

**Panel ID:** 105 | **Type:** Stat

#### 1. Logic Validation
- ☐ **Outcome defined:** Count of issues where actual cost exceeds estimate
- ☐ **Formula:**
  ```
  Over Budget Issues = COUNT(DISTINCT issue_key)
  WHERE actual_cost > estimated_cost
  AND actual_cost > 0

  Shows poor estimation or scope creep
  ```

#### Status: ☐ PASS | ☐ GAPS | ☐ FAIL

---

### Metric 18: Estimated vs Actual Cost Trend

**Panel ID:** 106 | **Type:** Time Series

#### 1. Logic Validation
- ☐ **Outcome defined:** Monthly comparison of estimated vs actual costs over time
- ☐ **Formula:**
  ```sql
  SELECT calculated_at as time,
         estimated_cost as "Estimated",
         actual_cost as "Actual"
  FROM monthly_budget_variance
  ORDER BY calculated_at

  Gap between lines shows estimation accuracy
  ```

#### Status: ☐ PASS | ☐ GAPS | ☐ FAIL

---

### Metric 19: Budget Variance % Over Time

**Panel ID:** 107 | **Type:** Time Series

#### 1. Logic Validation
- ☐ **Outcome defined:** Percentage variance from budget tracked monthly
- ☐ **Formula:**
  ```
  Variance % = ((actual_cost - estimated_cost) / estimated_cost) × 100

  Positive % = over budget
  Negative % = under budget
  Target: -5% to +5% (good estimation)
  ```

#### Status: ☐ PASS | ☐ GAPS | ☐ FAIL

---

## Unallocated Costs Metrics

### Metric 20: Unallocated Cost %

**Panel ID:** 202 | **Type:** Gauge

#### 1. Logic Validation
- ☐ **Outcome defined:** Percentage of total costs not allocated to epics or initiatives
- ☐ **Formula:**
  ```
  Unallocated % = (unallocated_cost / total_cost) × 100

  Where unallocated_cost = SUM(cost_allocations.total_cost)
  WHERE epic_key IS NULL OR initiative_id IS NULL

  Target: <10% (most work should be allocated)
  ```

#### Status: ☐ PASS | ☐ GAPS | ☐ FAIL

---

### Metric 21: Unallocated Cost ($)

**Panel ID:** 203 | **Type:** Stat

#### 1. Logic Validation
- ☐ **Outcome defined:** Dollar amount of costs not allocated to strategic initiatives
- ☐ **Formula:**
  ```
  Unallocated Cost = SUM(cost_allocations.total_cost)
  WHERE epic_key IS NULL OR initiative_id IS NULL
  ```

#### Status: ☐ PASS | ☐ GAPS | ☐ FAIL

---

### Metric 22: Orphan Issues (No Epic)

**Panel ID:** 204 | **Type:** Stat

#### 1. Logic Validation
- ☐ **Outcome defined:** Count of issues with work logged but no epic assignment
- ☐ **Formula:**
  ```
  Orphan Issues = COUNT(DISTINCT ca.issue_key)
  FROM cost_allocations ca
  LEFT JOIN issues i ON ca.issue_key = i.issue_key
  WHERE i.epic_key IS NULL
  AND ca.total_cost > 0
  ```

#### Status: ☐ PASS | ☐ GAPS | ☐ FAIL

---

### Metric 23: Total Unallocated Issues

**Panel ID:** 205 | **Type:** Stat

#### 1. Logic Validation
- ☐ **Outcome defined:** Count of all issues missing initiative or epic assignment
- ☐ **Formula:**
  ```
  Unallocated Issues = COUNT(DISTINCT issue_key)
  FROM cost_allocations
  WHERE (epic_key IS NULL OR initiative_id IS NULL)
  AND total_cost > 0
  ```

#### Status: ☐ PASS | ☐ GAPS | ☐ FAIL

---

### Metric 24: Unallocated % Trend

**Panel ID:** 206 | **Type:** Time Series

#### 1. Logic Validation
- ☐ **Outcome defined:** Monthly trend of unallocated cost percentage
- ☐ **Formula:**
  ```sql
  SELECT calculated_at as time,
         (unallocated_cost / total_cost) × 100 as "Unallocated %"
  FROM monthly_cost_summaries
  WHERE unallocated_cost IS NOT NULL
  ORDER BY calculated_at

  Downward trend is good (better allocation)
  ```

#### Status: ☐ PASS | ☐ GAPS | ☐ FAIL

---

### Metric 25: Orphan Issue Count Trend

**Panel ID:** 207 | **Type:** Time Series

#### 1. Logic Validation
- ☐ **Outcome defined:** Monthly count of orphan issues over time
- ☐ **Formula:**
  ```sql
  SELECT DATE_TRUNC('month', work_date) as time,
         COUNT(DISTINCT ca.issue_key) as "Orphan Issues"
  FROM cost_allocations ca
  LEFT JOIN issues i ON ca.issue_key = i.issue_key
  WHERE i.epic_key IS NULL
  GROUP BY DATE_TRUNC('month', work_date)
  ORDER BY time
  ```

#### Status: ☐ PASS | ☐ GAPS | ☐ FAIL

---

### Metric 26: Unallocated Issues Detail

**Panel ID:** 208 | **Type:** Table

#### 1. Logic Validation
- ☐ **Outcome defined:** List of specific issues that lack proper allocation
- ☐ **Formula:**
  ```sql
  SELECT ca.issue_key, i.issue_type, i.summary,
         SUM(ca.total_cost) as cost,
         i.epic_key, i.initiative_id
  FROM cost_allocations ca
  LEFT JOIN issues i ON ca.issue_key = i.issue_key
  WHERE i.epic_key IS NULL OR i.initiative_id IS NULL
  GROUP BY ca.issue_key, i.issue_type, i.summary, i.epic_key, i.initiative_id
  ORDER BY cost DESC
  LIMIT 50
  ```

#### Status: ☐ PASS | ☐ GAPS | ☐ FAIL

---

## Detail Tables

### Metric 27: Cost Allocations Table

**Panel ID:** 9 | **Type:** Table

#### 1. Logic Validation
- ☐ **Outcome defined:** Comprehensive table of all cost allocation records with filters
- ☐ **Formula:**
  ```sql
  SELECT project_name, issue_key, developer_name,
         work_date, hours_worked, hourly_rate,
         total_cost, project_phase, capitalization_category,
         epic_key, initiative_name
  FROM cost_allocations
  WHERE project_name IN (selected_projects)
  AND work_date IN (time_range)
  ORDER BY work_date DESC, total_cost DESC
  ```

#### Status: ☐ PASS | ☐ GAPS | ☐ FAIL

---

### Metric 28: Monthly Cost Summary Table

**Panel ID:** 10 | **Type:** Table

#### 1. Logic Validation
- ☐ **Outcome defined:** Month-by-month rollup of key cost metrics
- ☐ **Formula:**
  ```sql
  SELECT project_name,
         calculated_at as month,
         total_cost,
         capitalizable_cost,
         expense_cost,
         capitalization_rate,
         developer_count,
         issue_count
  FROM monthly_cost_summaries
  WHERE project_name IN (selected_projects)
  ORDER BY calculated_at DESC
  ```

#### Status: ☐ PASS | ☐ GAPS | ☐ FAIL

---

### Metric 29: Developer Cost Breakdown

**Panel ID:** 11 | **Type:** Table

#### 1. Logic Validation
- ☐ **Outcome defined:** Per-developer cost summary with phase breakdown
- ☐ **Formula:**
  ```sql
  SELECT developer_name,
         COUNT(DISTINCT issue_key) as issues_worked,
         SUM(hours_worked) as total_hours,
         AVG(hourly_rate) as avg_rate,
         SUM(total_cost) as total_cost,
         SUM(CASE WHEN project_phase = 'development' THEN total_cost ELSE 0 END) as capitalizable,
         SUM(CASE WHEN project_phase != 'development' THEN total_cost ELSE 0 END) as expense
  FROM cost_allocations
  WHERE project_name IN (selected_projects)
  GROUP BY developer_name
  ORDER BY total_cost DESC
  ```

#### Status: ☐ PASS | ☐ GAPS | ☐ FAIL

---

### Metric 30: Initiative Cost Breakdown

**Panel ID:** 12 | **Type:** Table

#### 1. Logic Validation
- ☐ **Outcome defined:** Per-initiative cost rollup with capitalization split
- ☐ **Formula:**
  ```sql
  SELECT initiative_name,
         epic_key,
         COUNT(DISTINCT issue_key) as issues,
         COUNT(DISTINCT developer_name) as developers,
         SUM(hours_worked) as total_hours,
         SUM(total_cost) as total_cost,
         SUM(CASE WHEN project_phase = 'development' THEN total_cost ELSE 0 END) as capitalizable_cost,
         SUM(CASE WHEN project_phase != 'development' THEN total_cost ELSE 0 END) as expense_cost
  FROM cost_allocations
  WHERE project_name IN (selected_projects)
  AND initiative_name IS NOT NULL
  GROUP BY initiative_name, epic_key
  ORDER BY total_cost DESC
  ```

#### Status: ☐ PASS | ☐ GAPS | ☐ FAIL

---

## Visualization Review

**Review Date:** 2026-02-02

### Chart Type Assessment

| Panel Type | Count | Usage | Assessment |
|------------|-------|-------|------------|
| `stat` | 14 | KPIs, totals, counts | ✅ Appropriate |
| `gauge` | 4 | Bounded percentages | ✅ Appropriate |
| `piechart` | 3 | Category breakdowns | ✅ Appropriate |
| `timeseries` | 4 | Trends over time | ✅ Appropriate |
| `barchart` | 1 | Window comparisons | ✅ Appropriate |
| `table` | 3 | Detailed data | ✅ Appropriate |
| `text` | 5 | Methodology docs | ✅ Appropriate |

### Threshold Validation

| Metric | Thresholds | Business Logic | Status |
|--------|------------|----------------|--------|
| Capitalization Rate | 🔴<30% 🟡30-50% 🟢>50% | Higher = more capitalizable | ✅ Correct |
| Cost Per Deploy | 🟢<$5K 🟡$5-10K 🔴>$10K | Lower = more efficient | ✅ Correct |
| Budget Variance | 🔴<-10% 🟡-10-0% 🟢>0% | Positive = under budget | ✅ Correct |
| Unallocated % | 🟢<10% 🟡10-20% 🔴>20% | Lower = better allocation | ✅ Correct |

### Color Coding

| Element | Color | Assessment |
|---------|-------|------------|
| Capitalizable Cost | Green | ✅ Intuitive (positive) |
| Expensed Cost | Red | ✅ Intuitive (not capitalizable) |
| Estimated Cost | Blue | ✅ Clear (neutral) |
| Unallocated Cost | Orange | ✅ Clear (warning) |

### Layout Assessment

- ✅ Summary KPIs at top
- ✅ Logical grouping by topic (Cost Summary → Deployment → Budget → Unallocated)
- ✅ Progressive disclosure (summary → detail)
- ✅ Collapsible rows for secondary content
- ✅ Methodology text embedded

**Visualization Status:** ✅ GOOD

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
