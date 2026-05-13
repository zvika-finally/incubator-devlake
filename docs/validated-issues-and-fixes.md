# Validated Metrics Issues and Fix Plan
**Date:** 2026-01-30
**Status:** Validation Complete - Ready for Implementation
**Source:** Production database validation queries

---

## Executive Summary

Ran comprehensive validation queries against production MySQL database to confirm investigation hypotheses. **4 critical issues confirmed**, **1 hypothesis disproven**.

### Validation Results

| Issue | Hypothesis | Validation Result | Severity | Fix Effort |
|-------|-----------|------------------|----------|------------|
| AI Code Churn 0% | merge_commit_sha missing | ❌ **DISPROVEN** - 100% populated<br>✅ **ACTUAL**: Schema mismatch | CRITICAL | LOW (SQL ALTER) |
| FinDevOps time column | fiscal_month is VARCHAR | ✅ **CONFIRMED** | HIGH | LOW (Dashboard edit) |
| Hardcoded $87/hour | developer_hourly_rates empty | ✅ **CONFIRMED** - exactly 87.0 avg | HIGH | MEDIUM (Populate data) |
| Monte Carlo forecasts | Unknown | ✅ **CONFIRMED** - 0 rows despite velocity data | HIGH | MEDIUM (Investigate task) |
| Settings tables empty | Suspected | ✅ **CONFIRMED** - All 3 tables empty | MEDIUM | LOW (Document defaults) |

---

## Issue #1: AI Code Churn Schema Mismatch 🔴 CRITICAL

### The Problem
Dashboard shows zero/no data for AI Code Churn metrics despite having 78 rows of churn data.

### Root Cause Analysis

**Original Hypothesis:** `merge_commit_sha` field not populated
**Status:** ❌ **DISPROVEN**

**Validation Results:**
```sql
total_merged_prs: 3029
with_merge_sha: 3029
percent_populated: 100%
```

**Actual Root Cause:** Schema mismatch - ratio columns missing from database

**Evidence:**

1. **Migration defines the columns:**
   ```go
   // File: backend/plugins/aidetector/models/migrationscripts/20260131_add_code_churn.go:64-65
   ChurnRatio7Days   float64    `gorm:"type:decimal(8,4)"`
   ChurnRatio30Days  float64    `gorm:"type:decimal(8,4)"`
   ```

2. **Dashboard queries expect them:**
   ```sql
   SELECT AVG(churn_ratio30_days) FROM ai_churn_metrics
   WHERE is_ai_assisted = true
   ```

3. **Database schema missing them:**
   ```sql
   -- DESCRIBE ai_churn_metrics shows:
   churn_within7_days    bigint       ✅ Present (absolute values)
   churn_within30_days   bigint       ✅ Present (absolute values)
   follow_up_commits7    bigint       ✅ Present
   churn_ratio7_days     decimal(8,4) ❌ MISSING
   churn_ratio30_days    decimal(8,4) ❌ MISSING
   follow_up_commits30   int          ❌ MISSING
   file_paths            text         ❌ MISSING
   ```

4. **Current data (unusable by dashboards):**
   ```
   | project_name       | churn_metrics_count | avg_churn_7d | avg_churn_30d |
   |--------------------|---------------------|--------------|---------------|
   | Expense Management | 15                  | 359          | 360           |
   | finally-DevEx      | 39                  | 201          | 239           |
   | SMB Platform       | 24                  | 103          | 163           |
   ```
   These are **absolute line counts**, not ratios!

### Why This Happened
GORM's `AutoMigrateTables()` failed to add new columns to existing table. Migration ran successfully (`20260131000004` on `2026-01-29 16:17:19`) but columns weren't created.

### Impact
- **3 dashboard panels fail:** AI Code Churn, Non-AI Code Churn, AI vs Non-AI Difference
- **78 existing data rows** cannot be displayed
- **Key metric unavailable** for engineering teams

### Fix Implementation

**File:** `scripts/fix-ai-churn-schema.sql` (already created)

**Steps:**
1. Run ALTER TABLE to add missing columns
2. Backfill ratios from existing absolute values
3. Verify dashboard queries work

**SQL Fix:**
```sql
ALTER TABLE ai_churn_metrics
ADD COLUMN churn_ratio7_days DECIMAL(8,4) AFTER follow_up_commits7,
ADD COLUMN churn_ratio30_days DECIMAL(8,4) AFTER churn_ratio7_days,
ADD COLUMN follow_up_commits30 INT AFTER churn_within30_days,
ADD COLUMN file_paths TEXT AFTER churn_ratio30_days;

-- Backfill ratios
UPDATE ai_churn_metrics
SET
    churn_ratio7_days = CASE
        WHEN initial_additions > 0 THEN churn_within7_days / initial_additions
        ELSE 0
    END,
    churn_ratio30_days = CASE
        WHEN initial_additions > 0 THEN churn_within30_days / initial_additions
        ELSE 0
    END;
```

**Expected Results After Fix:**
- `churn_ratio30_days` values between 0.0 - 1.0 (e.g., 0.15 = 15% churn)
- Dashboard panels show percentages instead of zero

**Effort:** 🟢 LOW (5 minutes - run SQL script)
**Priority:** 🔴 CRITICAL
**Testing:** Query dashboard after fix to verify panels display data

---

## Issue #2: FinDevOps Time Column Error 🟡 HIGH

### The Problem
Grafana error: `db has no time column: time column is missing`

### Root Cause
✅ **CONFIRMED** - `fiscal_month` stored as `VARCHAR(10)` instead of timestamp

**Validation Results:**
```sql
-- DESCRIBE monthly_cost_summaries
fiscal_month      varchar(10)    ❌ Cannot use with $__timeFilter()
calculated_at     datetime(3)    ✅ Can use with $__timeFilter()
```

**Current Data Sample:**
```
| project_name       | fiscal_month | total_cost | calculated_at       |
|--------------------|--------------|------------|---------------------|
| Expense Management | 2025-12      | 15312      | 2026-01-29 22:59:55 |
| finally-DevEx      | 2025-12      | 21576      | 2026-01-29 21:01:15 |
```

### Impact
- **All FinDevOps time-series panels fail** with this error
- Monthly cost summaries exist (98 rows) but cannot be visualized
- Fiscal month trends unavailable

### Fix Implementation

**Option 1: Update Dashboard Queries (RECOMMENDED)**
Replace all instances of:
```sql
-- OLD (broken):
WHERE $__timeFilter(fiscal_month)

-- NEW (working):
WHERE $__timeFilter(calculated_at)
```

**Option 2: Add Timestamp Column**
```sql
ALTER TABLE monthly_cost_summaries
ADD COLUMN fiscal_month_date DATE
  GENERATED ALWAYS AS (STR_TO_DATE(CONCAT(fiscal_month, '-01'), '%Y-%m-%d')) STORED;
```

**Files to Update:** `grafana/dashboards/FinDevOps.json`

**Effort:** 🟢 LOW (15 minutes - search and replace)
**Priority:** 🟡 HIGH
**Testing:** Load FinDevOps dashboard after fix

---

## Issue #3: Hardcoded $87/Hour Developer Rate 🟡 HIGH

### The Problem
All cost calculations use $87/hour regardless of actual developer rates

### Root Cause
✅ **CONFIRMED** - `developer_hourly_rates` table is empty

**Validation Results:**
```sql
-- developer_hourly_rates
total_rate_entries: 0
avg_rate: NULL

-- cost_allocations
total_allocations: 719
avg_rate: 87.0  (exactly!)
```

**Code Reference:**
```go
// backend/plugins/findevops/tasks/calculate_costs.go:258
defaultHourlyRate := 87.0 // TODO: get from settings
```

### Impact
- **Inaccurate cost calculations** if actual rates differ
- **No visibility** into rate variations by seniority/location
- Cost reports misleading if assuming default rate

### Fix Implementation

**Option 1: Populate developer_hourly_rates Table (RECOMMENDED)**
```sql
INSERT INTO developer_hourly_rates (developer_id, developer_name, hourly_rate, effective_date)
VALUES
  ('user:123', 'John Doe', 125.00, '2026-01-01'),
  ('user:456', 'Jane Smith', 95.00, '2026-01-01'),
  -- ... add all developers
;
```

**Option 2: Document $87 as Default**
Add dashboard annotation explaining this is industry average and actual costs may vary.

**Effort:** 🟡 MEDIUM (1-2 hours - gather real rates, populate table)
**Priority:** 🟡 HIGH
**Testing:** Verify new rates used in cost calculations

---

## Issue #4: Monte Carlo Forecasts Not Generated 🟡 HIGH

### The Problem
Capacity Planning dashboard shows no forecast data

### Root Cause
✅ **CONFIRMED** - Table empty despite velocity data and successful plugin runs

**Validation Results:**
```sql
-- monte_carlo_forecasts
row_count: 0

-- team_velocities (input data exists!)
row_count: 24
avg_throughput: 4-15 issues per week

-- Task execution
plugin: capacityplanner
task_executions: 65
completed: 65
failed: 0
```

### Why This is Odd
- ✅ Plugin compiling and running successfully
- ✅ Velocity data available (24 weeks across 4 projects)
- ✅ No task failures logged
- ❌ But forecast table is empty!

### Possible Causes
1. Forecast task not included in pipeline/blueprint
2. Task runs but hits silent error/early return
3. Conditional logic preventing forecast generation
4. Velocity data doesn't meet minimum thresholds

### Fix Implementation

**Step 1: Check if task is in pipeline**
```sql
SELECT plugin, subtask_name, COUNT(*) as executions
FROM _devlake_subtasks
WHERE plugin = 'capacityplanner'
GROUP BY plugin, subtask_name;
```

**Step 2: Review task code for early returns**
- File: `backend/plugins/capacityplanner/tasks/monte_carlo_forecast.go`
- Check: Minimum data requirements
- Look for: Silent failures or skipped logic

**Step 3: Add debug logging**
```go
// Add logging to identify why forecasts not generated
log.Info("Starting Monte Carlo forecast generation for project: %s", projectName)
log.Info("Velocity data points available: %d", len(velocities))
```

**Effort:** 🟡 MEDIUM (2-4 hours - investigation + fix)
**Priority:** 🟡 HIGH
**Testing:** Verify forecasts generated after fix

---

## Issue #5: Settings Tables Empty 🟢 MEDIUM

### The Problem
All plugin settings tables are empty, forcing use of hardcoded defaults

### Root Cause
✅ **CONFIRMED** - Tables exist but no configuration data

**Validation Results:**
```sql
_tool_findevops_settings:        0 rows
_tool_capacityplanner_settings:  0 rows
_tool_aidetector_settings:       0 rows
```

### Impact
- FinDevOps: Uses 4 hours/story point hardcoded assumption
- Capacity Planner: Uses industry benchmarks instead of org-specific data
- AI Detector: Uses 65% confidence threshold default

### Fix Implementation

**Option 1: Populate Settings (if needed)**
Only if defaults don't work for your organization.

**Option 2: Document Defaults (RECOMMENDED)**
Add dashboard annotations explaining what assumptions are being made.

**Examples:**
- "Cost calculations assume 4 hours per story point"
- "ROI uses industry benchmark: $75/hour average"
- "AI detection threshold: 65% confidence"

**Effort:** 🟢 LOW (30 minutes - add dashboard notes)
**Priority:** 🟢 MEDIUM
**Testing:** Review dashboards for clarity

---

## Validation Summary

### What Works ✅
- ✅ All 4 custom plugins compiled and running (65 executions each, 0 failures)
- ✅ All migrations executed successfully
- ✅ DORA metrics working correctly (foundation for other plugins)
- ✅ AI Usage Signals populated (3356 rows - explicit detection working)
- ✅ Business Metrics health scores calculated (5 snapshots)
- ✅ Cost allocations generating data (719 allocations)
- ✅ Team velocities tracked (24 weeks of data)
- ✅ Standard Engineering dashboards functional

### What's Broken ❌
- ❌ AI Code Churn (schema mismatch - ratio columns missing)
- ❌ FinDevOps time-series (wrong column type)
- ❌ Monte Carlo forecasts (not generating despite good input data)

### What's Using Defaults ⚠️
- ⚠️ Developer rates ($87/hour hardcoded)
- ⚠️ Story point hours (4 hours/point hardcoded)
- ⚠️ AI confidence threshold (65% default)
- ⚠️ ROI benchmarks (industry averages)

---

## Implementation Priority

### Immediate (Today) - 30 minutes total
1. ✅ Run `scripts/fix-ai-churn-schema.sql` (5 min)
2. ✅ Update FinDevOps dashboard queries (15 min)
3. ✅ Test both dashboards (10 min)

### Short-term (This Week) - 4-6 hours total
1. Investigate Monte Carlo forecast task (2-4 hours)
2. Populate developer_hourly_rates table (1-2 hours)
3. Add dashboard assumption annotations (30 min)

### Medium-term (Next Sprint) - Optional
1. Make story point conversion configurable
2. Add project-specific AI detection thresholds
3. Override ROI benchmarks with org data

---

## Success Criteria

After implementing fixes:

**AI Code Churn Dashboard:**
- ✅ Panels show percentage values (0.15 = 15% churn)
- ✅ AI vs Non-AI comparison working
- ✅ 78 existing rows displayed correctly

**FinDevOps Dashboard:**
- ✅ Time-series charts load without error
- ✅ Monthly cost trends visible
- ✅ Can filter by time range

**Capacity Planning Dashboard:**
- ✅ Monte Carlo forecasts populated
- ✅ P50/P85/P95 completion dates shown
- ✅ Velocity data driving forecasts

**Cost Accuracy:**
- ✅ Cost calculations use real developer rates
- ✅ Or clearly document $87 default assumption
- ✅ Budget variance tracking accurate

---

## Files Created

1. **scripts/validate-all-metrics.sql** - Validation queries (all sections)
2. **scripts/fix-ai-churn-schema.sql** - ALTER TABLE fix for AI churn
3. **docs/validated-issues-and-fixes.md** - This document
4. **docs/metrics-investigation-findings.md** - Updated with validation results

---

## Next Steps

Run immediate fixes (30 minutes):
```bash
# 1. Fix AI churn schema
mysql -u your_user -p your_database < scripts/fix-ai-churn-schema.sql

# 2. Edit FinDevOps dashboard JSON
# Replace all: $__timeFilter(fiscal_month) → $__timeFilter(calculated_at)

# 3. Test dashboards in Grafana
```

After immediate fixes work, tackle short-term items.
