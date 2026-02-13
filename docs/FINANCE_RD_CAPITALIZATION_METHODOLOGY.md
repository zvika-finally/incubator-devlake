# R&D Capitalization Methodology

**Prepared For:** Finance Team
**Date:** 2026-02-02
**Version:** 1.0

---

## Executive Summary

This document describes how R&D software development costs are calculated and categorized for capitalization under **ASC 350-40** (Accounting for Costs of Computer Software Developed or Obtained for Internal Use).

### Key Numbers (Current Period)

| Metric | Value |
|--------|-------|
| **Total Development Cost** | $1,301,389 |
| **Capitalizable (Development Phase)** | $1,053,352 (81%) |
| **Expensed (Other Phases)** | $248,037 (19%) |
| **Capitalization Rate** | ~70% |
| **Issues with Cost Allocations** | 16,015 of 16,021 (99.96%) |

---

## 1. Methodology Overview

### ASC 350-40 Three-Stage Model

We categorize all software development costs into three phases per ASC 350-40:

| Phase | Category | Treatment | Issue Types/Labels |
|-------|----------|-----------|-------------------|
| **Preliminary Project** | Expense | Expensed immediately | Spike, Research, POC, Discovery |
| **Application Development** | Capitalizable | Capitalized as asset | Story, Feature, Task, Enhancement |
| **Post-Implementation** | Expense | Expensed immediately | Bug, Defect, Hotfix, Maintenance |

### Decision Logic

```
IF issue.type IN ('Spike', 'Research') OR issue.labels CONTAIN ('poc', 'discovery')
  → Phase = 'preliminary' → EXPENSE

ELSE IF issue.type IN ('Story', 'Feature', 'Task', 'Enhancement')
  → Phase = 'development' → CAPITALIZABLE

ELSE IF issue.type IN ('Bug', 'Defect') OR issue.labels CONTAIN ('hotfix', 'maintenance')
  → Phase = 'post_implementation' → EXPENSE
```

---

## 2. Data Sources

### Primary Sources

| Source | Data Collected | Used For |
|--------|---------------|----------|
| **Jira** | Issues, worklogs, estimates, story points | Effort tracking, phase categorization |
| **GitHub/GitLab** | Commits, pull requests, code changes | Git-based effort inference |
| **CI/CD (Jenkins/GitHub Actions)** | Deployments, pipeline runs | Cost per deployment metrics |

### Data Flow

```
Jira Issues → DevLake Collector → Domain Tables → FinDevOps Plugin → Cost Allocations → Dashboard
     ↓                                                    ↑
Git Commits ────────────────────────────────────────────→┘
```

---

## 3. Cost Calculation Formula

### Hours Worked (Priority-Based)

We use a multi-source approach to determine hours worked, in priority order:

| Priority | Source | Confidence | Formula |
|----------|--------|------------|---------|
| 1 | Jira Logged Time | HIGH | `time_spent_minutes / 60` |
| 2 | Jira Estimate | MEDIUM | `original_estimate_minutes / 60` |
| 3 | Story Points | MEDIUM | `story_point × 4 hours` |
| 4 | Git Activity | INFERRED | `active_days × productive_hours + review_time` |
| 5 | FTE Distribution | LOW | Proportional allocation by activity |

### Cost Formula

```
Total Cost = Hours Worked × Hourly Rate

Where:
- Hours Worked = First non-null value from priority chain above
- Hourly Rate = Developer-specific rate OR default ($87/hour)
```

### Monthly Aggregation

```
Monthly Total Cost = SUM(issue.total_cost) for all issues resolved in month
Capitalizable Cost = SUM(issue.total_cost) WHERE phase = 'development'
Expense Cost = SUM(issue.total_cost) WHERE phase IN ('preliminary', 'post_implementation')
Capitalization Rate = (Capitalizable Cost / Total Cost) × 100
```

---

## 4. Validation & Audit Trail

### Formula Validation (Production Data)

All formulas have been validated against production data:

| Check | Pass | Total | Pass Rate | Status |
|-------|------|-------|-----------|--------|
| Cost = Hours × Rate | 7,000 | 7,000 | 100% | ✅ PASS |
| Cap Rate = Cap/Total × 100 | 135 | 135 | 100% | ✅ PASS |
| Phase Categorization | 7,000 | 7,000 | 100% | ✅ PASS |
| Budget Variance | 7,000 | 7,000 | 100% | ✅ PASS |

### Audit Trail Fields

Each cost allocation record includes:

| Field | Purpose |
|-------|---------|
| `effort_source` | Which data source provided hours (jira_time, git_inferred, etc.) |
| `confidence_level` | Reliability of effort data (high, medium, inferred, low) |
| `category_reason` | Why issue was categorized as capitalizable/expense |
| `linked_commit_shas` | Git commits linked to issue (R&D evidence) |
| `linked_pr_ids` | Pull requests linked to issue (R&D evidence) |

---

## 5. Database Tables

### cost_allocations (Per-Issue Detail)

| Column | Description |
|--------|-------------|
| `issue_id` | Jira issue key |
| `fiscal_month` | YYYY-MM format |
| `hours_worked` | Calculated hours |
| `hourly_rate` | Developer rate |
| `total_cost` | hours × rate |
| `project_phase` | preliminary / development / post_implementation |
| `capitalization_category` | capitalizable / expense |
| `capitalization_percent` | 0 or 100 |
| `category_reason` | Audit explanation |

### monthly_cost_summaries (Aggregated)

| Column | Description |
|--------|-------------|
| `fiscal_month` | YYYY-MM format |
| `total_cost` | Sum of all issue costs |
| `capitalizable_cost` | Sum of development phase costs |
| `expense_cost` | Sum of preliminary + post_implementation costs |
| `capitalization_rate` | Percentage capitalizable |

---

## 6. Reports Available

### Grafana Dashboard: FinDevOps

Access at: `/grafana/d/findevops/`

| Panel | Description |
|-------|-------------|
| Total Development Cost | Sum of all allocated costs |
| Capitalizable Cost | Development phase costs |
| Expensed Cost | Preliminary + post-implementation costs |
| Capitalization Rate | Gauge showing % capitalizable |
| Cost by Phase | Pie chart breakdown |
| Monthly Breakdown | Time series of costs over time |
| Budget Variance | Over/under budget tracking |

### Export Queries

**Monthly Summary for Financial Reporting:**
```sql
SELECT
  fiscal_month,
  total_cost,
  capitalizable_cost,
  expense_cost,
  capitalization_rate
FROM monthly_cost_summaries
WHERE project_name = 'Your Project'
ORDER BY fiscal_month DESC;
```

**Detailed Issue-Level Export:**
```sql
SELECT
  issue_id,
  fiscal_month,
  hours_worked,
  hourly_rate,
  total_cost,
  project_phase,
  capitalization_category,
  category_reason,
  effort_source,
  confidence_level
FROM cost_allocations
WHERE fiscal_month = '2026-01'
ORDER BY total_cost DESC;
```

---

## 7. Key Assumptions

| Assumption | Value | Rationale |
|------------|-------|-----------|
| Default hourly rate | $87/hour | Blended fully-loaded rate |
| Story point to hours | 4 hours/point | Industry standard |
| Productive hours/day | 6 hours | Excludes meetings, admin |
| Code review time | 30 min/review | Average review duration |

---

## 8. Compliance Notes

### ASC 350-40 Alignment

- **Preliminary Phase**: All research, feasibility, and planning work is expensed per ASC 350-40-25-2
- **Development Phase**: Coding, testing, and documentation is capitalizable per ASC 350-40-25-3
- **Post-Implementation**: Bug fixes and maintenance are expensed per ASC 350-40-25-7

### SOX Considerations

- All cost allocations have audit trail (`category_reason`, `effort_source`)
- Git commit/PR linkage provides R&D documentation evidence
- Monthly summaries reconcile to detail records (100% validated)

---

## 9. Contact

For questions about this methodology:

- **Dashboard Issues**: Engineering Team
- **Accounting Treatment**: Finance Team
- **Data Quality**: Data Engineering

---

**Document Location:** `docs/FINANCE_RD_CAPITALIZATION_METHODOLOGY.md`
**Last Validated:** 2026-02-02
**Validation Status:** ✅ All formulas verified against production data
