# Metrics Investigation Findings
**Date:** 2026-01-30
**Dashboards Reviewed:** AI Detection, Business Metrics, Capacity Planning, DORA, Engineering Overview, Engineering Throughput

## Executive Summary

Investigation revealed **4 critical issues** causing metrics to show incorrectly in production:

1. **Schema mismatch**: Time-series dashboards using string date columns instead of timestamps
2. **Missing data dependencies**: AI Code Churn calculation fails when `merge_commit_sha` is not populated
3. **Hardcoded constants**: Multiple plugins use stubbed values instead of real configuration
4. **Incomplete plugin registration**: Need to verify custom plugins are compiled and loaded

---

## 🚨 Critical Issues

### Issue #1: FinDevOps Dashboard - Time Column Error

**Problem:** `monthly_cost_summaries` table cannot be used in Grafana time-series panels

**Root Cause:**
```sql
-- Dashboard query (INCORRECT):
SELECT fiscal_month, SUM(capitalizable_cost) as capitalizable_cost,
       SUM(expense_cost) as expense_cost
FROM monthly_cost_summaries
WHERE project_name in (${project:sqlstring})
  AND $__timeFilter(fiscal_month)  -- <-- fiscal_month is VARCHAR, not TIMESTAMP
```

**Schema Issue:**
```go
// backend/plugins/findevops/models/cost_allocation.go:73
type MonthlyCostSummary struct {
    FiscalMonth    string    `gorm:"type:varchar(10);index"` // "2026-01" format
    CalculatedAt   time.Time
    // ...
}
```

**Error:** `db has no time column: time column is missing; make sure your data includes a time column`

**Solution Options:**
1. **Change dashboard query** to use `calculated_at` instead of `fiscal_month`
2. **Add a timestamp column** `fiscal_month_date` to the schema
3. **Convert string to date** in Grafana: `STR_TO_DATE(CONCAT(fiscal_month, '-01'), '%Y-%m-%d')`

**Files to Fix:**
- `grafana/dashboards/FinDevOps.json` - Update all queries using `fiscal_month` with `$__timeFilter()`
- `backend/plugins/findevops/models/cost_allocation.go` - Consider adding `FiscalMonthDate time.Time`

---

### Issue #2: AI Code Churn Shows 0% ⚠️ UPDATED ROOT CAUSE

**Problem:** AI vs Non-AI code churn metrics show zero or no data

**Original Hypothesis (❌ INCORRECT):** `merge_commit_sha` not populated
- Validation showed: `merge_commit_sha` is **100% populated** (3029/3029 merged PRs)

**Actual Root Cause (✅ CONFIRMED):** Schema mismatch - ratio columns missing from database

**Evidence:**
1. Migration defines ratio columns (`backend/plugins/aidetector/models/migrationscripts/20260131_add_code_churn.go:64-65`):
   ```go
   ChurnRatio7Days   float64    `gorm:"type:decimal(8,4)"`
   ChurnRatio30Days  float64    `gorm:"type:decimal(8,4)"`
   ```

2. Dashboard queries expect these columns:
   ```sql
   SELECT AVG(churn_ratio30_days) FROM ai_churn_metrics WHERE is_ai_assisted = true
   ```

3. Database schema is **missing** these columns:
   ```
   churn_within7_days    bigint   ✅ Present
   churn_within30_days   bigint   ✅ Present
   churn_ratio7_days     decimal  ❌ MISSING
   churn_ratio30_days    decimal  ❌ MISSING
   ```

4. Migration history shows it ran: `20260131000004` on `2026-01-29 16:17:19`
   - But GORM's `AutoMigrateTables` failed to add the new columns to existing table

**Impact:**
- Dashboard queries fail because columns don't exist
- Existing data has absolute churn values (359, 201 lines) but no ratios
- 78 rows of data exist but are unusable by dashboards

**Dashboard Panels Affected:**
- "AI Code Churn (30d)" - Query fails: column `churn_ratio30_days` doesn't exist
- "Non-AI Code Churn (30d)" - Query fails
- "AI vs Non-AI Difference %" - Query fails

**Fix Required:**
- Run manual `ALTER TABLE` to add missing columns (see `scripts/fix-ai-churn-schema.sql`)
- Backfill ratio calculations from existing absolute values
- Alternative: Investigate why GORM AutoMigrate didn't add columns

**Validation Results:**
```
✅ merge_commit_sha: 100% populated (3029/3029)
✅ ai_churn_metrics: 78 rows with churn data
❌ churn_ratio columns: Missing from schema
✅ Migration script: Ran successfully but didn't create columns
```

---

### Issue #3: Hardcoded Constants (Stubbed Data)

**Problem:** Multiple plugins use placeholder values instead of real configuration

#### FinDevOps Plugin

**File:** `backend/plugins/findevops/tasks/calculate_costs.go`

```go
// Line 258 - Hardcoded hourly rate
defaultHourlyRate := 87.0 // TODO: get from settings

// Line 304 - Hardcoded hours per story point
return *issue.StoryPoint * 4.0  // Assumes 4 hours per story point
```

**Impact:**
- Cost calculations use $87/hour default when `developer_hourly_rates` table is empty
- Story points converted to hours using fixed 4:1 ratio
- Actual developer rates and team velocity not reflected

**Solution:**
- Populate `developer_hourly_rates` table with real rates
- Make story point conversion configurable via `findevops_settings` table
- Update docs to guide users to configure these settings

#### Capacity Planner Plugin

**File:** `backend/plugins/capacityplanner/tasks/calculate_roi.go`

```go
// Lines 40-48 - Industry benchmarks (acceptable)
const (
    DefaultHourlyCost       = 75.0  // USD per hour
    AIHoursSavedPerUser     = 2.0   // Hours saved per week
    AIQualityImprovement    = 5.0   // 5% quality improvement
)

// Line 76 - Hardcoded team size fallback
teamSize := 5 // Default

// Line 97 - Hardcoded tool cost
monthlyPerUserCost := 20.0 // USD per user per month

// Line 108 - Assumed adoption rate
AIAdoptionPercent: 80.0, // Assume 80% adoption
```

**Impact:**
- ROI calculations use industry averages, not org-specific data
- Team size defaults to 5 if velocity data is missing
- AI tool costs and adoption rates are assumptions

**Solution:**
- Document that these are industry benchmarks
- Allow overriding via `capacity_planner_settings` table
- Add dashboard annotations explaining assumptions

---

## 📊 Dashboard Query Compatibility Issues

### FinDevOps Dashboard

**All queries using `fiscal_month` with `$__timeFilter()` will fail:**

```json
// grafana/dashboards/FinDevOps.json
// Example panel query:
"rawSql": "SELECT fiscal_month, SUM(capitalizable_cost) as capitalizable_cost,
           SUM(expense_cost) as expense_cost
           FROM monthly_cost_summaries
           WHERE project_name in (${project:sqlstring})
           AND $__timeFilter(fiscal_month)"  // <-- ERROR HERE
```

**Fix:** Replace `$__timeFilter(fiscal_month)` with `$__timeFilter(calculated_at)`

### AI Detection Dashboard

**Code churn queries are correct but depend on data population:**

```sql
-- Panel: AI Code Churn (30d)
SELECT AVG(churn_ratio30_days) as ai_churn
FROM ai_churn_metrics
WHERE is_ai_assisted = true
  AND project_name in (${project:sqlstring})
  AND $__timeFilter(merged_at)  -- ✅ merged_at is TIMESTAMP
```

**These queries are correct** - issue is in data collection (Issue #2)

### Business Metrics Dashboard

**Queries reference tables that exist:**

```sql
-- Panel: Team Health Score
SELECT total_score
FROM team_health_scores
WHERE project_name in (${project:sqlstring})
  AND $__timeFilter(calculated_at)  -- ✅ Correct

-- Panel: Total Initiatives
SELECT COUNT(*) as total
FROM business_initiatives  -- ✅ Table exists
```

**These queries should work** - need to verify data is being populated

---

## 🔧 Plugin Architecture Verification

### Custom Plugins Discovered

All plugins are properly structured and implement required interfaces:

1. **aidetector** (`backend/plugins/aidetector/`)
   - Implements: `MetricPluginBlueprintV200`
   - Depends on: `dora` (must run after DORA)
   - Tables: `ai_usage_signals`, `ai_churn_metrics`, `project_churn_summaries`
   - Status: ✅ Code structure correct

2. **businessmetrics** (`backend/plugins/businessmetrics/`)
   - Implements: `MetricPluginBlueprintV200`
   - Depends on: `dora`
   - Tables: `team_health_scores`, `business_initiatives`, `work_allocations`
   - Status: ✅ Code structure correct

3. **capacityplanner** (`backend/plugins/capacityplanner/`)
   - Implements: `MetricPluginBlueprintV200`
   - Depends on: none
   - Tables: `team_velocities`, `monte_carlo_forecasts`
   - Status: ✅ Code structure correct

4. **findevops** (`backend/plugins/findevops/`)
   - Implements: `MetricPluginBlueprintV200`
   - Depends on: `businessmetrics`
   - Tables: `cost_allocations`, `monthly_cost_summaries`, `developer_hourly_rates`
   - Status: ✅ Code structure correct

### Plugin Entry Points

Each plugin has a `PluginEntry` variable exported in its main file:

```go
// backend/plugins/aidetector/aidetector.go:27
var PluginEntry impl.AIDetector //nolint
```

### Migration Scripts

All plugins have migration scripts registered:

- ✅ `aidetector`: 20260129_init_schema.go, 20260129_add_explicit_signals.go
- ✅ `businessmetrics`: 20260130_add_health_and_capability.go
- ✅ `capacityplanner`: 20260130_add_advanced_planning.go
- ✅ `findevops`: 20260129_init_schema.go, 20260130_add_settings.go

**Status: All migration scripts exist and follow naming conventions**

---

## ✅ Action Items (UPDATED after validation - 2026-01-30)

### Priority 1: Critical Fixes (Blocking Production)

1. **✅ VALIDATED - Fix AI Code Churn Schema Mismatch**
   - [x] Confirmed: `merge_commit_sha` is 100% populated (not the issue)
   - [ ] **ACTION REQUIRED**: Run `scripts/fix-ai-churn-schema.sql` to add missing ratio columns
   - [ ] Verify: `churn_ratio7_days` and `churn_ratio30_days` columns exist after ALTER TABLE
   - [ ] Backfill: Calculate ratios for existing 78 rows
   - [ ] Test: AI Code Churn dashboard panels should show percentages (0.15 = 15% churn)
   - **Files:** `scripts/fix-ai-churn-schema.sql`

2. **✅ VALIDATED - Fix FinDevOps Dashboard Time Column Issue**
   - [x] Confirmed: `fiscal_month` is VARCHAR(10), `calculated_at` is datetime(3)
   - [ ] Update all FinDevOps dashboard queries to use `calculated_at` instead of `fiscal_month` with `$__timeFilter()`
   - [ ] File: `grafana/dashboards/FinDevOps.json`
   - [ ] Test queries in Grafana after fix

3. **✅ VALIDATED - Plugin Execution Working**
   - [x] All 4 custom plugins running successfully (65 executions each, 0 failures)
   - [x] All migrations have run correctly (verified in migration history)
   - [x] Plugin registration verified
   - No action needed - plugins are working correctly

### Priority 2: Configuration & Data Quality

4. **✅ VALIDATED - Populate Required Configuration Tables**
   - [x] Confirmed: `developer_hourly_rates` table is empty (0 rows)
   - [x] Confirmed: `cost_allocations.avg_rate` = 87.0 exactly (hardcoded default in use)
   - [x] Confirmed: All settings tables empty (`_tool_findevops_settings`, `_tool_capacityplanner_settings`, `_tool_aidetector_settings`)
   - [ ] **ACTION REQUIRED**: Populate `developer_hourly_rates` with real rates per developer
   - [ ] Optional: Add project-specific settings if defaults don't work
   - **Impact**: Until populated, all costs use $87/hour default

5. **✅ VALIDATED - Investigate Monte Carlo Forecast Generation**
   - [x] Confirmed: `monte_carlo_forecasts` table is empty (0 rows)
   - [x] Confirmed: `team_velocities` has 24 rows of good data
   - [x] Confirmed: Plugin running successfully (65 executions, 0 failures)
   - [ ] **ACTION REQUIRED**: Investigate why forecast task not generating data
   - [ ] Check if forecast task is in pipeline configuration
   - [ ] Review task logs for errors
   - **Files:** `backend/plugins/capacityplanner/tasks/monte_carlo_forecast.go`

6. **Document Assumptions in Dashboards**
   - [ ] Add annotations to FinDevOps ROI panels explaining $87/hour default
   - [ ] Add notes to Capacity Planning dashboard about industry benchmarks
   - [ ] Document that AI tool cost assumes $20/user/month

### Priority 3: Testing & Validation

6. **Create Test Queries for Each Dashboard**
   - [ ] Write SQL queries to validate data exists for each panel
   - [ ] Check row counts in custom tables by project
   - [ ] Verify time ranges have data

7. **Add E2E Tests for Custom Plugins**
   - [ ] Create test data for aidetector code churn calculation
   - [ ] Test FinDevOps cost calculation with sample issues
   - [ ] Verify BusinessMetrics health score calculation

---

## 📝 Verification Queries

Run these queries in production to diagnose issues:

```sql
-- Check custom plugin tables exist
SHOW TABLES LIKE '%ai_%';
SHOW TABLES LIKE '%cost_%';
SHOW TABLES LIKE '%monte_carlo%';

-- Check aidetector data population
SELECT
    COUNT(*) as total_prs,
    COUNT(DISTINCT pull_request_id) as prs_with_signals,
    AVG(ai_confidence_score) as avg_confidence
FROM ai_usage_signals;

SELECT project_name, COUNT(*) as churn_metrics_count
FROM ai_churn_metrics
GROUP BY project_name;

-- Check findevops data
SELECT project_name, fiscal_month, total_cost, capitalizable_cost, expense_cost
FROM monthly_cost_summaries
ORDER BY fiscal_month DESC
LIMIT 10;

-- Check business metrics
SELECT project_name, total_score, health_level, calculated_at
FROM team_health_scores
ORDER BY calculated_at DESC
LIMIT 10;

-- Check capacity planner
SELECT project_name, AVG(issues_completed) as avg_throughput
FROM team_velocities
GROUP BY project_name;

-- Verify merge_commit_sha population
SELECT
    COUNT(*) as total_merged_prs,
    COUNT(merge_commit_sha) as with_merge_sha,
    ROUND(COUNT(merge_commit_sha) * 100.0 / COUNT(*), 2) as percent_populated
FROM pull_requests
WHERE merged_date IS NOT NULL;
```

---

## 🎯 Root Cause Summary

| Issue | Root Cause | Severity | Effort to Fix |
|-------|-----------|----------|---------------|
| FinDevOps time column | Schema design - fiscal_month as VARCHAR | High | Low - Dashboard query change |
| AI Code Churn 0% | Missing merge_commit_sha field | High | Medium - Plugin config + fallback logic |
| Stubbed constants | Incomplete configuration setup | Medium | Low - Add settings, document defaults |
| Plugin registration | Unknown - needs verification | Critical | Unknown until verified |

---

## 📚 Next Steps

**Immediate (Today):**
1. Run verification queries to confirm which tables have data
2. Fix FinDevOps dashboard queries (30 min fix)
3. Check plugin compilation status

**Short-term (This Week):**
1. Investigate merge_commit_sha population
2. Add fallback logic for code churn calculation
3. Populate configuration tables with real values

**Medium-term (Next Sprint):**
1. Add dashboard annotations for assumptions
2. Create E2E tests for custom plugins
3. Document plugin configuration in user guide
