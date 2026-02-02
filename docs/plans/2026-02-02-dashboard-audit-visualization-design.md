# Dashboard Audit & Visualization Review Design

**Date:** 2026-02-02
**Status:** Approved
**Author:** Claude Code

---

## Overview

This design covers the validation of dashboard JSON files against audit documentation and comprehensive visualization quality review.

## Priority Order

| Priority | Dashboards | Focus |
|----------|------------|-------|
| **P1 (Primary)** | FinDevOps, AIDetection, BusinessMetrics, CapacityPlanning | JSON ↔ Audit sync + Visualization quality |
| **P2 (Secondary)** | DORA suite (8) + Engineering (3) | Formula accuracy + Visualization quality |

---

## P1: Existing Dashboard Validation

### Part A: JSON ↔ Audit Sync

For each of the 4 custom dashboards, verify that the SQL queries in the JSON files match the documented formulas.

| Check | What We Verify |
|-------|----------------|
| Query match | SQL in JSON panels matches formulas in `METRICS_ALGORITHMS_VALIDATION.md` |
| Table references | Queries reference correct tables (`cost_allocations`, `ai_usage_signals`, etc.) |
| Column names | GORM column naming matches (e.g., `total_score` not `totalScore`) |
| Filter logic | WHERE clauses apply correct filters (date ranges, status filters) |

### Part B: Visualization Quality

| Check | What We Verify |
|-------|----------------|
| Chart types | Right visualization for the metric (gauge for %, line for trends, table for details) |
| Thresholds | Color bands match business logic (e.g., health score levels) |
| Layout | Related metrics grouped, logical flow, not overcrowded |
| Labels/legends | Clear, accurate, not truncated |

### P1 Dashboards

| Dashboard | JSON File | Audit File |
|-----------|-----------|------------|
| FinDevOps | `grafana/dashboards/FinDevOps.json` | `docs/audit/dashboards/findevops-audit.md` |
| AIDetection | `grafana/dashboards/AIDetection.json` | `docs/audit/dashboards/aidetection-audit.md` |
| BusinessMetrics | `grafana/dashboards/BusinessMetrics.json` | `docs/audit/dashboards/businessmetrics-audit.md` |
| CapacityPlanning | `grafana/dashboards/CapacityPlanning.json` | `docs/audit/dashboards/capacityplanning-audit.md` |

---

## P2: DORA & Engineering Dashboards

### DORA Suite (8 dashboards)

| Dashboard | JSON File | Expected Content |
|-----------|-----------|------------------|
| DORA | `DORA.json` | All 4 metrics overview, team comparison |
| DORAByTeam | `DORAByTeam.json` | Team-level breakdown of all metrics |
| DORADebug | `DORADebug.json` | Diagnostic queries for troubleshooting |
| DORADetails-DeploymentFrequency | `DORADetails-DeploymentFrequency.json` | Deep-dive on deployment patterns |
| DORADetails-LeadTimeforChanges | `DORADetails-LeadTimeforChanges.json` | Commit-to-deploy breakdown |
| DORADetails-ChangeFailureRate | `DORADetails-ChangeFailureRate.json` | Failure analysis and trends |
| DORADetails-TimetoRestoreService | `DORADetails-TimetoRestoreService.json` | Incident recovery analysis |
| DORADetails-FailedDeploymentRecoveryTime | `DORADetails-FailedDeploymentRecoveryTime.json` | Failed deployment recovery |

### DORA Metric Definitions (per DORA Research)

| Metric | Definition | Elite | High | Medium | Low |
|--------|------------|-------|------|--------|-----|
| **Deployment Frequency** | How often code deploys to production | Multiple/day | Daily-Weekly | Weekly-Monthly | <Monthly |
| **Lead Time for Changes** | Time from commit to production | <1 hour | <1 day | <1 week | <1 month |
| **Change Failure Rate** | % of deployments causing incidents | <5% | 5-10% | 10-15% | >15% |
| **Mean Time to Recovery** | Time to restore service after incident | <1 hour | <1 day | <1 week | <1 month |

### Engineering Suite (3 dashboards)

| Dashboard | JSON File | Expected Content |
|-----------|-----------|------------------|
| EngineeringOverview | `EngineeringOverview.json` | High-level summary of all engineering metrics |
| EngineeringThroughputAndCycleTime | `EngineeringThroughputAndCycleTime.json` | Detailed throughput and cycle time breakdown |
| EngineeringThroughputAndCycleTimeTeamView | `EngineeringThroughputAndCycleTimeTeamView.json` | Same metrics but grouped by team |

### Engineering Metric Definitions

| Metric | Definition | Purpose |
|--------|------------|---------|
| **Throughput** | Issues/PRs completed per time period | Measure delivery rate |
| **Cycle Time** | Time from work started to done | Measure delivery speed |
| **PR Cycle Time** | Time from PR open to merge | Code review efficiency |
| **Coding Time** | Time spent actively writing code | Developer productivity |
| **Review Time** | Time PRs spend in review | Review bottleneck detection |
| **Deploy Time** | Time from merge to production | CI/CD efficiency |

---

## Deliverables

### New Files

| File | Purpose |
|------|---------|
| `docs/audit/JSON_AUDIT_REPORT.md` | Documents any drift between JSON queries and audit formulas |
| `docs/audit/VISUALIZATION_REVIEW.md` | Comprehensive viz assessment with recommendations |
| `docs/audit/dashboards/dora-suite-audit.md` | Combined audit for 8 DORA dashboards |
| `docs/audit/dashboards/engineering-suite-audit.md` | Combined audit for 3 Engineering dashboards |

### Enhanced Files

| File | Enhancement |
|------|-------------|
| `docs/audit/dashboards/findevops-audit.md` | Add visualization review section |
| `docs/audit/dashboards/aidetection-audit.md` | Add visualization review section |
| `docs/audit/dashboards/businessmetrics-audit.md` | Add visualization review section |
| `docs/audit/dashboards/capacityplanning-audit.md` | Add visualization review section |
| `docs/audit/AUDIT_CHECKLIST_MASTER.md` | Add new dashboards to tracking |
| `docs/audit/AUDIT_SUMMARY_REPORT.md` | Update with final results |

---

## Execution Phases

### Phase 1: P1 JSON ↔ Audit Sync
1. Read each P1 dashboard JSON file
2. Extract all panel queries
3. Compare against documented formulas in `METRICS_ALGORITHMS_VALIDATION.md`
4. Flag any mismatches in `JSON_AUDIT_REPORT.md`

### Phase 2: P1 Visualization Review
1. Assess each panel for chart type appropriateness
2. Validate thresholds and color coding
3. Evaluate layout and data density
4. Document findings in `VISUALIZATION_REVIEW.md`
5. Enhance existing audit files with viz sections

### Phase 3: P2 DORA & Engineering
1. Read dashboard JSON files
2. Validate formulas against DORA standards
3. Review visualizations (same criteria)
4. Create combined audit docs

### Phase 4: Summary Updates
1. Update `AUDIT_CHECKLIST_MASTER.md`
2. Update `AUDIT_SUMMARY_REPORT.md`

---

## Success Criteria

| Criterion | Target |
|-----------|--------|
| JSON ↔ Audit sync | 100% of P1 queries match documented formulas |
| Visualization issues | All critical issues documented with recommendations |
| P2 formula accuracy | DORA queries match standard definitions |
| Documentation | All artifacts created and linked |

---

**Approved:** 2026-02-02
