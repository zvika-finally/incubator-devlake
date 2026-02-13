# Comprehensive Metrics Analysis - All Dashboards
**Date:** 2026-01-30
**Purpose:** Systematic investigation of all metrics across 6 key dashboards

## Table of Contents
1. [AI-Assisted Development Dashboard](#ai-assisted-development-dashboard)
2. [Business Alignment & Team Health](#business-alignment--team-health)
3. [Capacity Planning & Forecasting](#capacity-planning--forecasting)
4. [DORA Metrics](#dora-metrics)
5. [Engineering Overview](#engineering-overview)
6. [Engineering Throughput & Cycle Time](#engineering-throughput--cycle-time)
7. [Validation Test Suite](#validation-test-suite)
8. [Data Dependency Matrix](#data-dependency-matrix)

---

# AI-Assisted Development Dashboard

## Dashboard Overview
- **File:** `grafana/dashboards/AIDetection.json`
- **Primary Plugin:** `aidetector`
- **Secondary Plugins:** `cursor`, `claudecode`
- **Dependencies:** Pull requests, commits, ai_usage_signals, ai_churn_metrics

## Panel Analysis

### Section 1: AI Detection Overview (Rows 5-17)

#### Panel 1.1: Explicit AI Markers (ID: 10)
**Type:** Stat
**Query:**
```sql
SELECT COUNT(*) as explicit_detections
FROM ai_usage_signals s
JOIN pull_requests pr ON s.pull_request_id = pr.id
JOIN repos r ON pr.base_repo_id = r.id
JOIN project_mapping pm ON r.id = pm.row_id AND pm.`table` = 'repos'
WHERE s.explicit_tool_detected = true
  AND pm.project_name in (${project:sqlstring})
  AND $__timeFilter(s.detected_at)
```

**Data Dependencies:**
- ✅ `ai_usage_signals` table
- ✅ `pull_requests` table
- ✅ `repos` table
- ✅ `project_mapping` table

**Fields Required:**
- `ai_usage_signals.explicit_tool_detected` (boolean)
- `ai_usage_signals.detected_at` (timestamp)
- `ai_usage_signals.pull_request_id` (FK)

**Expected Behavior:** Count of PRs with explicit AI markers (Co-Authored-By, tool signatures in commits)

**Potential Issues:**
- ⚠️ Requires `aidetector` plugin tasks to run: `detectExplicitSignals`
- ⚠️ Depends on PR collection being complete
- ⚠️ If `detected_at` is NULL, panel will show no data

**Validation Query:**
```sql
-- Check if explicit markers are being detected
SELECT
  COUNT(*) as total_signals,
  COUNT(CASE WHEN explicit_tool_detected = true THEN 1 END) as explicit_count,
  COUNT(CASE WHEN explicit_tool_detected = true THEN 1 END) * 100.0 / COUNT(*) as explicit_percent
FROM ai_usage_signals;

-- Check detected_at population
SELECT
  COUNT(*) as total,
  COUNT(detected_at) as with_timestamp
FROM ai_usage_signals;
```

---

#### Panel 1.2: Avg AI Confidence (ID: 2)
**Type:** Gauge
**Query:**
```sql
SELECT AVG(s.ai_confidence_score) as avg_confidence
FROM ai_usage_signals s
JOIN pull_requests pr ON s.pull_request_id = pr.id
JOIN repos r ON pr.base_repo_id = r.id
JOIN project_mapping pm ON r.id = pm.row_id AND pm.`table` = 'repos'
WHERE s.ai_confidence_score > 0
  AND pm.project_name in (${project:sqlstring})
  AND $__timeFilter(s.detected_at)
```

**Data Dependencies:**
- ✅ `ai_usage_signals.ai_confidence_score` (int 0-100)

**Expected Range:** 0-100 (percentage)

**Color Thresholds:**
- Green: < 40
- Yellow: 40-60
- Orange: 60-80
- Red: > 80

**Potential Issues:**
- ⚠️ Requires `scoreAIConfidence` task to run after all signal detection tasks
- ⚠️ Will show NULL if no PRs have confidence > 0

**Validation Query:**
```sql
SELECT
  COUNT(*) as total_prs,
  AVG(ai_confidence_score) as avg_score,
  MIN(ai_confidence_score) as min_score,
  MAX(ai_confidence_score) as max_score,
  COUNT(CASE WHEN ai_confidence_score = 0 THEN 1 END) as zero_score_count
FROM ai_usage_signals;
```

---

#### Panel 1.3: Total PRs Analyzed (ID: 3)
**Type:** Stat
**Query:**
```sql
SELECT COUNT(*) as total_prs
FROM ai_usage_signals s
JOIN pull_requests pr ON s.pull_request_id = pr.id
JOIN repos r ON pr.base_repo_id = r.id
JOIN project_mapping pm ON r.id = pm.row_id AND pm.`table` = 'repos'
WHERE pm.project_name in (${project:sqlstring})
  AND $__timeFilter(s.detected_at)
```

**Expected Behavior:** Total count of PRs that have been analyzed by aidetector plugin

**Validation Query:**
```sql
-- Compare against total PRs in system
SELECT
  (SELECT COUNT(*) FROM pull_requests) as total_prs_in_system,
  (SELECT COUNT(*) FROM ai_usage_signals) as total_analyzed,
  ROUND((SELECT COUNT(*) FROM ai_usage_signals) * 100.0 / (SELECT COUNT(*) FROM pull_requests), 2) as coverage_percent;
```

---

#### Panel 1.4: High Confidence (≥70%) (ID: 4)
**Type:** Stat
**Query:**
```sql
SELECT COUNT(*) as high_confidence
FROM ai_usage_signals s
...
WHERE s.ai_confidence_score >= 70
```

**Threshold:** 70% (hardcoded in query)

**Validation Query:**
```sql
SELECT
  COUNT(CASE WHEN ai_confidence_score >= 70 THEN 1 END) as high,
  COUNT(CASE WHEN ai_confidence_score >= 40 AND ai_confidence_score < 70 THEN 1 END) as medium,
  COUNT(CASE WHEN ai_confidence_score < 40 THEN 1 END) as low
FROM ai_usage_signals;
```

---

#### Panel 1.5: Medium Confidence (40-69%) (ID: 5)
**Type:** Stat
**Query:**
```sql
SELECT COUNT(*) as medium_confidence
...
WHERE s.ai_confidence_score >= 40 AND s.ai_confidence_score < 70
```

**Note:** Thresholds match scoring algorithm in `aidetector/tasks/score_ai_confidence.go`

---

#### Panel 1.6: AI Tools Detected (ID: 12)
**Type:** Pie Chart
**Query:**
```sql
SELECT
  COALESCE(NULLIF(s.detected_tool, ''), 'unknown') as tool,
  COUNT(*) as count
FROM ai_usage_signals s
...
WHERE s.ai_confidence_score >= 40
GROUP BY s.detected_tool
ORDER BY count DESC
```

**Data Dependencies:**
- ✅ `ai_usage_signals.detected_tool` (varchar: copilot, cursor, claude, unknown)

**Expected Values:** copilot, cursor, claude, codepilot, unknown

**Potential Issues:**
- ⚠️ Tool detection happens in `detectExplicitSignals` task
- ⚠️ Empty string converted to "unknown"
- ⚠️ Only counts PRs with confidence >= 40

**Validation Query:**
```sql
SELECT
  COALESCE(NULLIF(detected_tool, ''), 'unknown') as tool,
  COUNT(*) as count,
  AVG(ai_confidence_score) as avg_confidence
FROM ai_usage_signals
WHERE ai_confidence_score >= 40
GROUP BY detected_tool
ORDER BY count DESC;
```

---

#### Panel 1.7: AI Detection Trend Over Time (ID: 8)
**Type:** Time Series
**Query:**
```sql
SELECT
  MIN(s.detected_at) as time,
  AVG(s.ai_confidence_score) as avg_score,
  SUM(CASE WHEN s.explicit_tool_detected = true THEN 1 ELSE 0 END) as explicit_count
FROM ai_usage_signals s
...
GROUP BY DATE(s.detected_at)
ORDER BY time
```

**Chart Type:** Time series (2 series)
- Series 1: Average confidence score
- Series 2: Explicit detections count

**Potential Issues:**
- ⚠️ Uses `MIN(detected_at)` for time aggregation
- ⚠️ Groups by DATE() which may have timezone issues

---

### Section 2: Code Churn Analysis (Rows 18-33)

#### Panel 2.1: AI Code Churn (30d) (ID: 51)
**Type:** Stat (Gauge)
**Query:**
```sql
SELECT AVG(churn_ratio30_days) as ai_churn
FROM ai_churn_metrics
WHERE is_ai_assisted = true
  AND project_name in (${project:sqlstring})
  AND $__timeFilter(merged_at)
```

**Data Dependencies:**
- ❌ `ai_churn_metrics` table (populated by `analyzeCodeChurn` task)
- ❌ Requires `merge_commit_sha` to be populated in `pull_requests`

**Expected Range:** 0.0 - 1.0 (ratio, displayed as percent)

**Color Thresholds:**
- Green: < 0.15 (15%)
- Yellow: 0.15-0.25
- Orange: 0.25-0.40
- Red: > 0.40

**CRITICAL ISSUE:** This is the metric showing 0% in production!

**Root Cause Analysis:**
1. `analyzeCodeChurn` task calls `getPRFiles()`
2. `getPRFiles()` joins on `merge_commit_sha`:
   ```go
   dal.Join("INNER JOIN pull_requests pr ON pr.merge_commit_sha = commit_files.commit_sha")
   ```
3. If `merge_commit_sha` is NULL → empty files array → zero churn

**Validation Queries:**
```sql
-- Check ai_churn_metrics population
SELECT
  project_name,
  COUNT(*) as total_metrics,
  AVG(churn_ratio30_days) as avg_churn_30d,
  COUNT(CASE WHEN churn_ratio30_days = 0 THEN 1 END) as zero_churn_count,
  COUNT(CASE WHEN is_ai_assisted = true THEN 1 END) as ai_assisted_count,
  COUNT(CASE WHEN is_ai_assisted = false THEN 1 END) as non_ai_count
FROM ai_churn_metrics
GROUP BY project_name;

-- Check merge_commit_sha population (ROOT CAUSE CHECK)
SELECT
  COUNT(*) as total_merged_prs,
  COUNT(merge_commit_sha) as with_merge_sha,
  COUNT(merge_commit_sha) * 100.0 / NULLIF(COUNT(*), 0) as percent_populated,
  COUNT(CASE WHEN merge_commit_sha IS NULL THEN 1 END) as missing_count
FROM pull_requests
WHERE merged_date IS NOT NULL;

-- Check if commit_files table has data
SELECT
  COUNT(DISTINCT commit_sha) as unique_commits,
  COUNT(DISTINCT file_path) as unique_files,
  COUNT(*) as total_file_changes
FROM commit_files;
```

---

#### Panel 2.2: Non-AI Code Churn (30d) (ID: 52)
**Type:** Stat
**Query:**
```sql
SELECT AVG(churn_ratio30_days) as non_ai_churn
FROM ai_churn_metrics
WHERE is_ai_assisted = false
  AND project_name in (${project:sqlstring})
  AND $__timeFilter(merged_at)
```

**Same issues as Panel 2.1**

---

#### Panel 2.3: AI vs Non-AI Difference % (ID: 53)
**Type:** Stat
**Query:**
```sql
SELECT churn_difference_percent
FROM project_churn_summaries
WHERE project_name in (${project:sqlstring})
  AND $__timeFilter(calculated_at)
ORDER BY calculated_at DESC
LIMIT 1
```

**Data Dependencies:**
- ❌ `project_churn_summaries` table
- ❌ Populated by `generateChurnSummary()` function in `analyze_code_churn.go`

**Calculation:**
```go
// From analyze_code_churn.go:229
churnDiff = (aiAvg30 - nonAIAvg30) / nonAIAvg30 * 100
```

**Expected:** +41% (based on GitClear research)

**Validation Query:**
```sql
SELECT
  project_name,
  total_prs_analyzed,
  ai_pr_count,
  non_ai_pr_count,
  ai_avg_churn_ratio30,
  non_ai_avg_churn_ratio30,
  churn_difference_percent,
  calculated_at
FROM project_churn_summaries
ORDER BY calculated_at DESC
LIMIT 5;
```

---

#### Panel 2.4: PRs with Churn Data (ID: 54)
**Type:** Stat
**Query:**
```sql
SELECT COUNT(*) as total
FROM ai_churn_metrics
WHERE project_name in (${project:sqlstring})
  AND $__timeFilter(merged_at)
```

**Purpose:** Verify churn analysis is running

**Expected:** Should match or be close to total merged PRs

**Validation Query:**
```sql
SELECT
  (SELECT COUNT(*) FROM pull_requests WHERE merged_date IS NOT NULL) as total_merged_prs,
  (SELECT COUNT(*) FROM ai_churn_metrics) as prs_with_churn,
  ROUND((SELECT COUNT(*) FROM ai_churn_metrics) * 100.0 /
        NULLIF((SELECT COUNT(*) FROM pull_requests WHERE merged_date IS NOT NULL), 0), 2) as coverage_percent;
```

---

#### Panel 2.5: Code Churn Over Time (ID: 55)
**Type:** Time Series
**Query:**
```sql
SELECT
  MIN(merged_at) as time,
  AVG(CASE WHEN is_ai_assisted = true THEN churn_ratio30_days END) as ai_churn,
  AVG(CASE WHEN is_ai_assisted = false THEN churn_ratio30_days END) as non_ai_churn
FROM ai_churn_metrics
WHERE merged_at IS NOT NULL
  AND project_name in (${project:sqlstring})
  AND $__timeFilter(merged_at)
GROUP BY DATE(merged_at)
ORDER BY time
```

**Chart Type:** Dual series time chart
- Blue line: AI-assisted code churn
- Orange line: Non-AI code churn

---

### Section 3: AI Tool Usage - Direct Metrics (Rows 34-56)

**Note:** These panels require separate plugin connections (`cursor` and `claudecode`)

#### Panel 3.1-3.3: Cursor Metrics (IDs: 61-63)
**Tables:**
- `cursor_usage_metrics` (org-level daily aggregates)
- `cursor_user_metrics` (user-level daily data)

**Queries:**
```sql
-- Suggestions
SELECT SUM(total_suggestions) as suggestions
FROM cursor_usage_metrics
WHERE $__timeFilter(date)

-- Acceptances
SELECT SUM(total_acceptances) as acceptances
FROM cursor_usage_metrics
WHERE $__timeFilter(date)

-- Accept Rate
SELECT AVG(acceptance_rate) as rate
FROM cursor_usage_metrics
WHERE $__timeFilter(date)
```

**Data Source:** Cursor Business API
- Requires connection configuration
- Data collected by `cursor` plugin

**Potential Issues:**
- ⚠️ Panels show "No data" if Cursor connection not configured
- ⚠️ Organization-wide metrics (not filtered by project)
- ⚠️ Dashboard note clearly states this

---

#### Panel 3.4-3.6: Claude Code Metrics (IDs: 64-66)
**Tables:**
- `claude_code_usage_metrics` (org-level)
- `claude_code_user_metrics` (user-level)

**Queries:**
```sql
-- Tool Uses
SELECT SUM(total_tool_uses) as tool_uses
FROM claude_code_usage_metrics
WHERE $__timeFilter(date)

-- Sessions
SELECT SUM(total_sessions) as sessions
FROM claude_code_usage_metrics
WHERE $__timeFilter(date)

-- Lines Added
SELECT SUM(lines_added) as lines
FROM claude_code_usage_metrics
WHERE $__timeFilter(date)
```

**Data Source:** Claude Code Admin API
- Requires connection configuration
- Data collected by `claudecode` plugin

---

#### Panel 3.7-3.8: Usage Trends (IDs: 67-68)
**Type:** Time Series

**Cursor Trend:**
```sql
SELECT
  date as time,
  total_suggestions as suggestions,
  total_acceptances as acceptances,
  daily_active_users as active_users
FROM cursor_usage_metrics
WHERE $__timeFilter(date)
ORDER BY date
```

**Claude Code Trend:**
```sql
SELECT
  date as time,
  total_tool_uses as tool_uses,
  total_sessions as sessions,
  lines_added
FROM claude_code_usage_metrics
WHERE $__timeFilter(date)
ORDER BY date
```

---

#### Panel 3.9-3.10: Top Users Tables (IDs: 69-70)

**Top Cursor Users:**
```sql
SELECT
  user_email,
  SUM(tab_acceptances + composer_acceptances) as acceptances,
  AVG(acceptance_rate) as avg_rate,
  COUNT(DISTINCT date) as active_days
FROM cursor_user_metrics
WHERE $__timeFilter(date)
GROUP BY user_email
ORDER BY acceptances DESC
LIMIT 20
```

**Top Claude Code Users:**
```sql
SELECT
  user_email,
  SUM(total_tool_uses) as tool_uses,
  SUM(lines_written) as lines_written,
  COUNT(DISTINCT date) as active_days
FROM claude_code_user_metrics
WHERE $__timeFilter(date)
GROUP BY user_email
ORDER BY tool_uses DESC
LIMIT 20
```

---

### Section 4: AI Impact Analysis (Rows 57-65)

#### Panel 4.1: PR Throughput Change (ID: 21)
**Type:** Stat
**Query:**
```sql
SELECT pr_throughput_change
FROM ai_impact_metrics
WHERE project_name in (${project:sqlstring})
  AND $__timeFilter(calculated_at)
ORDER BY calculated_at DESC
LIMIT 1
```

**Data Dependencies:**
- ❌ `ai_impact_metrics` table
- ❌ Populated by `calculateAIImpact` task

**Methodology:** Compare 90-day baseline vs 30-day current period

**Calculation Logic:**
```go
// From calculate_ai_impact.go
baselinePRs := countPRsInPeriod(now.AddDate(0, 0, -120), now.AddDate(0, 0, -30))
currentPRs := countPRsInPeriod(now.AddDate(0, 0, -30), now)
throughputChange := ((currentPRs - baselinePRs) / baselinePRs) * 100
```

**Expected:** Positive number = improvement

**Color Thresholds:**
- Red: < 0 (slower)
- Yellow: 0-10
- Green: > 10

**Validation Query:**
```sql
SELECT
  project_name,
  baseline_period_start,
  baseline_period_end,
  baseline_pr_count,
  current_period_start,
  current_period_end,
  current_pr_count,
  pr_throughput_change,
  calculated_at
FROM ai_impact_metrics
ORDER BY calculated_at DESC
LIMIT 5;
```

---

#### Panel 4.2: Review Time Change (ID: 22)
**Type:** Stat
**Query:**
```sql
SELECT review_time_change
FROM ai_impact_metrics
WHERE project_name in (${project:sqlstring})
  AND $__timeFilter(calculated_at)
ORDER BY calculated_at DESC
LIMIT 1
```

**Note:** Sign is inverted (faster review time = positive change)

**Calculation:**
```go
// Inverted: faster = positive
reviewTimeChange := ((baselineReviewTime - currentReviewTime) / baselineReviewTime) * 100
```

---

#### Panel 4.3: Lead Time Change (ID: 23)
**Type:** Stat
**Query:**
```sql
SELECT lead_time_change
FROM ai_impact_metrics
WHERE project_name in (${project:sqlstring})
  AND $__timeFilter(calculated_at)
ORDER BY calculated_at DESC
LIMIT 1
```

**Note:** Sign is inverted (faster lead time = positive change)

**Dependencies:**
- ❌ Requires DORA plugin to populate `project_pr_metrics` table
- ❌ aidetector depends on dora (specified in `RunAfter()`)

---

### Section 5: Detailed Analysis (Rows 66-98)

#### Panel 5.1: PRs with Explicit AI Tool Markers (ID: 11)
**Type:** Table
**Query:**
```sql
SELECT
  s.explicit_tools as ai_tool,
  pr.title as pr_title,
  pr.author_name,
  s.explicit_signal_score,
  s.ai_confidence_score as total_score,
  DATE(s.detected_at) as detected_date
FROM ai_usage_signals s
JOIN pull_requests pr ON pr.id = s.pull_request_id
JOIN repos r ON pr.base_repo_id = r.id
JOIN project_mapping pm ON r.id = pm.row_id AND pm.`table` = 'repos'
WHERE s.explicit_tool_detected = true
  AND pm.project_name in (${project:sqlstring})
  AND $__timeFilter(s.detected_at)
ORDER BY s.detected_at DESC
LIMIT 25
```

**Columns:**
- ai_tool: Comma-separated list of detected tools
- pr_title: PR title
- author_name: Developer name
- explicit_signal_score: Score from explicit markers (max 70)
- total_score: Overall AI confidence score
- detected_date: When analysis ran

---

#### Panel 5.2: Distribution by Detection Confidence (ID: 7)
**Type:** Pie Chart
**Query:**
```sql
SELECT
  CASE
    WHEN s.explicit_tool_detected = true THEN 'Explicit (Confirmed)'
    WHEN s.ai_confidence_score >= 70 THEN 'High (≥70%)'
    WHEN s.ai_confidence_score >= 40 THEN 'Medium (40-69%)'
    ELSE 'Low (<40%)'
  END as confidence_level,
  COUNT(*) as count
FROM ai_usage_signals s
...
GROUP BY confidence_level
ORDER BY count DESC
```

**Categories:**
1. Explicit (Confirmed) - Has git trailers/markers
2. High (≥70%) - Strong behavioral signals
3. Medium (40-69%) - Moderate signals
4. Low (<40%) - Weak or no signals

---

#### Panel 5.3: All PRs by AI Confidence Score (ID: 6)
**Type:** Table (50 rows)
**Query:**
```sql
SELECT
  s.id,
  pr.title as pr_title,
  pr.author_name,
  s.detected_tool,
  s.explicit_tool_detected as explicit,
  s.explicit_signal_score,
  s.rapid_commit_score,
  s.pr_size_score,
  s.lines_per_minute_score,
  s.ai_confidence_score,
  DATE(s.detected_at) as detected_date
FROM ai_usage_signals s
JOIN pull_requests pr ON pr.id = s.pull_request_id
JOIN repos r ON pr.base_repo_id = r.id
JOIN project_mapping pm ON r.id = pm.row_id AND pm.`table` = 'repos'
WHERE pm.project_name in (${project:sqlstring})
  AND $__timeFilter(s.detected_at)
ORDER BY s.ai_confidence_score DESC
LIMIT 50
```

**Purpose:** Debug view showing score breakdown
- explicit_signal_score (max 70)
- rapid_commit_score (max 30)
- pr_size_score (max 20)
- lines_per_minute_score (max 25)

**Total:** Sum of all scores = ai_confidence_score

---

## AI Detection Dashboard - Summary

### Tables Required

| Table | Plugin | Status | Purpose |
|-------|--------|--------|---------|
| `ai_usage_signals` | aidetector | ✅ Schema OK | AI detection signals per PR |
| `ai_churn_metrics` | aidetector | ⚠️ Data issue | Code churn metrics per PR |
| `project_churn_summaries` | aidetector | ⚠️ Data issue | Aggregated churn by project |
| `ai_impact_metrics` | aidetector | ❓ Unknown | Productivity impact analysis |
| `cursor_usage_metrics` | cursor | ❓ Unknown | Cursor daily metrics |
| `cursor_user_metrics` | cursor | ❓ Unknown | Cursor per-user metrics |
| `claude_code_usage_metrics` | claudecode | ❓ Unknown | Claude Code daily metrics |
| `claude_code_user_metrics` | claudecode | ❓ Unknown | Claude Code per-user metrics |

### Task Execution Order

1. `detectExplicitSignals` - Must run first
2. `analyzeCommitPatterns` - Behavioral analysis
3. `analyzePRCharacteristics` - PR size/structure analysis
4. `scoreAIConfidence` - Combine all scores
5. `calculateAIImpact` - Productivity impact (requires DORA)
6. `analyzeCodeChurn` - Churn analysis (requires merge_commit_sha)

### Critical Issues Found

1. **AI Code Churn shows 0%**
   - Root cause: `merge_commit_sha` not populated
   - Impact: Panels 2.1, 2.2, 2.3, 2.5 show no/wrong data
   - Priority: HIGH

2. **AI Impact Analysis may not work**
   - Table `ai_impact_metrics` existence unknown
   - Panels 4.1, 4.2, 4.3 may show "No data"
   - Priority: HIGH

3. **Cursor/Claude Code metrics optional**
   - Require separate API connections
   - Panels 3.1-3.10 will show "No data" without config
   - Priority: LOW (documented in dashboard)

---

# TO BE CONTINUED...

Next sections to analyze:
2. Business Alignment & Team Health
3. Capacity Planning & Forecasting
4. DORA Metrics
5. Engineering Overview
6. Engineering Throughput & Cycle Time

---

## Validation Test Suite

### Test Script 1: AI Detection Dashboard Data Check

```sql
-- Test 1: Check ai_usage_signals population
SELECT
  'ai_usage_signals' as table_name,
  COUNT(*) as row_count,
  COUNT(DISTINCT pull_request_id) as unique_prs,
  AVG(ai_confidence_score) as avg_confidence,
  COUNT(CASE WHEN explicit_tool_detected = true THEN 1 END) as explicit_count
FROM ai_usage_signals;

-- Test 2: Check ai_churn_metrics population
SELECT
  'ai_churn_metrics' as table_name,
  COUNT(*) as row_count,
  AVG(churn_ratio30_days) as avg_churn,
  COUNT(CASE WHEN is_ai_assisted = true THEN 1 END) as ai_count,
  COUNT(CASE WHEN is_ai_assisted = false THEN 1 END) as non_ai_count
FROM ai_churn_metrics;

-- Test 3: Check project_churn_summaries
SELECT * FROM project_churn_summaries
ORDER BY calculated_at DESC LIMIT 3;

-- Test 4: Check ai_impact_metrics (may not exist)
SELECT * FROM ai_impact_metrics
ORDER BY calculated_at DESC LIMIT 3;

-- Test 5: CRITICAL - Check merge_commit_sha population
SELECT
  COUNT(*) as total_merged,
  COUNT(merge_commit_sha) as with_sha,
  ROUND(COUNT(merge_commit_sha) * 100.0 / COUNT(*), 2) as percent_populated
FROM pull_requests
WHERE merged_date IS NOT NULL;

-- Test 6: Check if aidetector tasks have run
SELECT script, executed_at
FROM _devlake_migration_history
WHERE script LIKE '%aidetector%'
ORDER BY executed_at DESC;
```

### Expected Results

**Test 1:** Should return rows > 0, avg_confidence between 0-100
**Test 2:** Should return rows > 0, avg_churn between 0-1
**Test 3:** Should return at least 1 row with non-null churn metrics
**Test 4:** May return error if table doesn't exist
**Test 5:** **CRITICAL** - percent_populated should be > 90%
**Test 6:** Should show aidetector migrations executed

---

