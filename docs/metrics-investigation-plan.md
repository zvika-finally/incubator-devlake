# Comprehensive Metrics Investigation Plan
**Date:** 2026-01-30
**Status:** In Progress

## Executive Summary

This plan outlines a systematic approach to investigate all metrics across 6 dashboards, identify issues, create validation queries, and provide fixes.

## Investigation Strategy

### Phase 1: Dashboard Analysis (Parallel Execution)
Each dashboard will be analyzed by a specialized agent to:
1. Extract ALL panel queries from the JSON file
2. Identify data dependencies (tables, columns, joins)
3. Document expected behavior for each metric
4. Create validation SQL queries
5. Identify potential issues (missing tables, schema mismatches, hardcoded values)
6. Provide fixes for identified issues

### Phase 2: Cross-Dashboard Validation
1. Build comprehensive test suite
2. Identify shared dependencies
3. Create data dependency matrix
4. Test all queries against production schema

### Phase 3: Root Cause Analysis
1. Investigate stubbed/hardcoded data
2. Verify plugin registration and execution
3. Check migration history
4. Validate task execution order

### Phase 4: Fix Implementation
1. Prioritize fixes by impact
2. Implement dashboard query corrections
3. Add missing fallback logic in tasks
4. Populate configuration tables
5. Create unit tests

---

## Dashboards to Analyze

| # | Dashboard | Agent | Priority | Complexity | Est. Panels |
|---|-----------|-------|----------|------------|-------------|
| 1 | AI-Assisted Development | Explore | HIGH | High | 25+ panels |
| 2 | Business Alignment & Team Health | Explore | HIGH | Medium | 15-20 panels |
| 3 | Capacity Planning & Forecasting | Explore | HIGH | High | 20+ panels |
| 4 | DORA | Explore | CRITICAL | Medium | 10-15 panels |
| 5 | Engineering Overview | Explore | MEDIUM | Low | 8-12 panels |
| 6 | Engineering Throughput & Cycle Time | Explore | MEDIUM | Low | 10-15 panels |

---

## Agent Task Assignments

### Agent 1: AI Detection Dashboard Analyzer
**File:** `grafana/dashboards/AIDetection.json`
**Focus Areas:**
- ai_usage_signals table queries
- ai_churn_metrics (YOUR ZERO PERCENT ISSUE)
- ai_impact_metrics
- Cursor/Claude Code integration
- Score calculation logic

**Deliverables:**
- Panel-by-panel analysis
- Validation queries for each metric
- Root cause analysis for zero churn issue
- Fix recommendations

### Agent 2: Business Metrics Dashboard Analyzer
**File:** `grafana/dashboards/BusinessMetrics.json`
**Focus Areas:**
- team_health_scores (DORA-based scoring)
- business_initiatives
- work_allocations
- Health score algorithm validation
- Business value scoring

**Deliverables:**
- Algorithm validation
- Hardcoded threshold identification
- Data source verification
- Expected vs actual behavior

### Agent 3: Capacity Planning Dashboard Analyzer
**File:** `grafana/dashboards/CapacityPlanning.json`
**Focus Areas:**
- team_velocities
- monte_carlo_forecasts
- Throughput vs velocity (Kanban vs Scrum)
- ROI calculations (STUBBED DATA SUSPECTED)
- Brooks's Law model

**Deliverables:**
- Monte Carlo simulation validation
- Identify hardcoded constants
- Forecast accuracy checks
- ROI parameter verification

### Agent 4: DORA Dashboard Analyzer
**File:** `grafana/dashboards/DORA.json`
**Focus Areas:**
- Deployment Frequency
- Lead Time for Changes
- Change Failure Rate
- Failed Deployment Recovery Time
- project_pr_metrics table

**Deliverables:**
- Standard DORA metric validation
- Benchmark threshold verification
- Dependencies on other plugins
- Data transformation requirements

### Agent 5: Engineering Overview Dashboard Analyzer
**File:** `grafana/dashboards/EngineeringOverview.json`
**Focus Areas:**
- PR metrics
- Issue metrics
- Jira + GitHub data integration
- Aggregation queries

**Deliverables:**
- Data source requirements
- Join validation
- Aggregation correctness

### Agent 6: Engineering Throughput Dashboard Analyzer
**File:** `grafana/dashboards/EngineeringThroughputAndCycleTime.json`
**Focus Areas:**
- PR throughput
- Cycle time calculations
- Story points vs issue count
- Team view vs project view

**Deliverables:**
- Cycle time algorithm validation
- Throughput calculation checks
- Story point configuration verification

---

## Parallel Execution Plan

```
Start Investigation
│
├─> Agent 1: AIDetection.json ─────────┐
├─> Agent 2: BusinessMetrics.json ─────┤
├─> Agent 3: CapacityPlanning.json ────┤───> Consolidate Results
├─> Agent 4: DORA.json ────────────────┤
├─> Agent 5: EngineeringOverview.json ─┤
└─> Agent 6: EngineeringThroughput.json┘
```

**Estimated Time:** 10-15 minutes (parallel execution)

---

## Success Criteria

For each dashboard, agents must provide:

### 1. Complete Panel Inventory
```
Panel ID | Panel Title | Query Type | Tables Used | Status
---------|-------------|------------|-------------|--------
10       | Explicit AI | Stat       | ai_usage_signals | ✅ OK
51       | AI Churn    | Gauge      | ai_churn_metrics | ❌ Zero data
```

### 2. Validation Query Suite
```sql
-- For each panel, provide:
-- Query to check if data exists
-- Query to validate expected ranges
-- Query to identify root cause if broken
```

### 3. Issue Classification
- **CRITICAL:** Blocks dashboard from working
- **HIGH:** Metric shows wrong/zero data
- **MEDIUM:** Metric uses assumptions/defaults
- **LOW:** Documentation/clarity issues

### 4. Fix Recommendations
```markdown
Issue: AI Code Churn shows 0%
Root Cause: merge_commit_sha not populated
Fix Options:
  1. Immediate: Update GitHub plugin to populate field
  2. Workaround: Add fallback logic in analyze_code_churn.go
  3. Dashboard: Change query to use alternative join
Priority: HIGH
Effort: Medium (4-6 hours)
```

---

## Consolidated Output Format

After all agents complete, create:

### 1. Master Validation Script
**File:** `scripts/validate-all-metrics.sql`
- One SQL file with all validation queries
- Organized by dashboard
- Includes expected results
- Flags issues with comments

### 2. Issue Tracker
**File:** `docs/metrics-issues-tracker.md`
- All identified issues
- Priority and severity
- Root cause
- Fix recommendations
- Effort estimates
- Dependencies

### 3. Data Dependency Matrix
**File:** `docs/metrics-data-dependencies.md`
```
Table Name          | Plugin          | Dashboards Using | Status
--------------------|-----------------|------------------|--------
ai_usage_signals    | aidetector      | AI Detection     | ✅ OK
ai_churn_metrics    | aidetector      | AI Detection     | ⚠️ Data issue
monthly_cost_summaries | findevops    | FinDevOps       | ❌ Schema issue
```

### 4. Fix Implementation Plan
**File:** `docs/metrics-fix-plan.md`
- Prioritized list of fixes
- Step-by-step instructions
- SQL migration scripts if needed
- Dashboard JSON patches
- Code changes required

---

## Monitoring & Validation

### After Fix Implementation:
1. Run master validation script
2. Check all dashboard panels load
3. Verify metrics show expected values
4. Document any remaining assumptions
5. Create E2E tests for critical metrics

---

## Next Steps

1. ✅ Launch 6 agents in parallel
2. ⏳ Wait for agents to complete analysis
3. ⏳ Consolidate results
4. ⏳ Create master validation script
5. ⏳ Build fix implementation plan
6. ⏳ Execute fixes
7. ⏳ Validate all metrics working

