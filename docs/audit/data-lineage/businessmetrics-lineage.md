# Business Metrics Data Lineage

## Overview

The Business Metrics dashboard provides visibility into business alignment, team health, and working agreement compliance. It extracts business initiatives from Jira Epics, calculates DORA-based health scores, and monitors working agreement adherence.

## Data Flow Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        SOURCE SYSTEMS                            │
├─────────────────────────────────────────────────────────────────┤
│  Jira                         │  DORA Plugin                     │
│  - Epics (type='Epic')        │  - project_pr_metrics            │
│  - Issues (linked to Epics)   │  - Deployment frequency          │
│  - Labels, custom fields      │  - Lead time for changes         │
│  - Story points               │  - Change failure rate           │
│                               │  - Time to restore service       │
├─────────────────────────────────────────────────────────────────┤
│  Working Agreements API       │                                  │
│  - User-defined thresholds    │                                  │
│  - Configurable rules         │                                  │
└─────────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────────┐
│                     DEVLAKE DOMAIN TABLES                        │
├─────────────────────────────────────────────────────────────────┤
│  issues                 │ id, type, title, epic_key, story_point │
│  project_pr_metrics     │ DORA metrics per project (from dora)   │
│  project_mapping        │ project_name, table, row_id            │
└─────────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────────┐
│                BUSINESSMETRICS PLUGIN TASKS                      │
├─────────────────────────────────────────────────────────────────┤
│  1. extractBusinessGoals                                         │
│     Input: issues (type='Epic')                                  │
│     Logic: Extract Epics as business initiatives                 │
│     Output: business_initiatives                                 │
├─────────────────────────────────────────────────────────────────┤
│  2. calculateAlignment                                           │
│     Input: issues, business_initiatives                          │
│     Logic: Link issues to initiatives, calculate allocation      │
│     Output: work_allocations                                     │
├─────────────────────────────────────────────────────────────────┤
│  3. calculateHealthScore                                         │
│     Input: project_pr_metrics (DORA data)                        │
│     Logic: Score each DORA metric 0-25, sum to 0-100             │
│     Output: team_health_scores                                   │
├─────────────────────────────────────────────────────────────────┤
│  4. calculateBusinessValue                                       │
│     Input: business_initiatives, work_allocations                │
│     Logic: Calculate ROI, value scores per initiative            │
│     Output: Updated business_initiatives.value_score             │
├─────────────────────────────────────────────────────────────────┤
│  5. checkAgreements                                              │
│     Input: working_agreements, PRs/issues                        │
│     Logic: Check violations against configured thresholds        │
│     Output: agreement_violations, agreement_compliance_summaries │
└─────────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────────┐
│                BUSINESSMETRICS OUTPUT TABLES                     │
├─────────────────────────────────────────────────────────────────┤
│  business_initiatives          │ Extracted from Epics            │
│  work_allocations              │ Issue-to-initiative mapping     │
│  team_health_scores            │ DORA-based health scores        │
│  working_agreements            │ User-configured thresholds      │
│  agreement_violations          │ Detected threshold violations   │
│  agreement_compliance_summaries│ Compliance rates by period      │
└─────────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────────┐
│                   GRAFANA DASHBOARD                              │
│          Business Alignment & Team Health                        │
│              (business-metrics-dashboard)                        │
└─────────────────────────────────────────────────────────────────┘
```

## Table Schemas

### business_initiatives

| Column | Type | Source | Transformation |
|--------|------|--------|----------------|
| id | varchar(255) | Generated | `{project}:{epic_key}` |
| project_name | varchar(255) | project_mapping | Via issue → board → project |
| epic_key | varchar(100) | issues.issue_key | Direct (for Epics) |
| title | varchar(500) | issues.title | Direct |
| description | text | issues.description | Direct |
| status | varchar(50) | issues.status | Direct |
| investment_category | varchar(100) | Labels/custom field | grow/run/transform |
| business_capability | varchar(100) | Labels/custom field | Extracted from labels |
| revenue_impact | varchar(50) | Labels/custom field | high/medium/low |
| total_story_points | int | Aggregated | SUM of linked issues |
| completed_story_points | int | Aggregated | SUM of resolved issues |
| value_score | decimal(5,2) | calculateBusinessValue | Calculated 0-100 |
| calculated_at | timestamp | Generated | Calculation timestamp |

### work_allocations

| Column | Type | Source | Transformation |
|--------|------|--------|----------------|
| id | varchar(255) | Generated | `{issue_id}:{initiative_id}` |
| issue_id | varchar(255) | issues.id | Direct |
| initiative_id | varchar(255) | business_initiatives.id | Linked via epic_key |
| project_name | varchar(255) | project_mapping | Via issue |
| story_points | int | issues.story_point | Direct |
| allocation_percent | decimal(5,2) | Calculated | % of initiative total |
| created_at | timestamp | Generated | Record creation time |

### team_health_scores

| Column | Type | Source | Transformation |
|--------|------|--------|----------------|
| id | varchar(255) | Generated | `{project}:{period}` |
| project_name | varchar(255) | project_mapping | Aggregation key |
| period_start | timestamp | Calculated | Period start date |
| period_end | timestamp | Calculated | Period end date |
| health_score | int | Calculated | Sum of 4 DORA scores (0-100) |
| health_level | varchar(20) | Calculated | excellent/good/fair/poor |
| deployment_frequency_score | int | project_pr_metrics | 0-25 points |
| lead_time_score | int | project_pr_metrics | 0-25 points |
| change_failure_rate_score | int | project_pr_metrics | 0-25 points |
| time_to_restore_score | int | project_pr_metrics | 0-25 points |
| calculated_at | timestamp | Generated | Calculation timestamp |

### working_agreements

| Column | Type | Source | Transformation |
|--------|------|--------|----------------|
| id | varchar(255) | Generated | `{project}:{agreement_type}` |
| project_name | varchar(255) | API input | User-configured |
| agreement_type | varchar(100) | API input | pr_size, review_time, etc. |
| threshold_value | decimal(10,2) | API input | Configured threshold |
| threshold_unit | varchar(50) | API input | lines, hours, count, etc. |
| enabled | bool | API input | Active/inactive |
| created_at | timestamp | Generated | Creation timestamp |
| updated_at | timestamp | Generated | Last update |

### agreement_violations

| Column | Type | Source | Transformation |
|--------|------|--------|----------------|
| id | varchar(255) | Generated | UUID |
| project_name | varchar(255) | working_agreements | From agreement |
| agreement_type | varchar(100) | working_agreements | From agreement |
| entity_type | varchar(50) | Detected | 'pull_request' or 'issue' |
| entity_id | varchar(255) | Detected | PR or issue ID |
| actual_value | decimal(10,2) | Measured | Actual measurement |
| threshold_value | decimal(10,2) | working_agreements | Configured threshold |
| violation_percent | decimal(5,2) | Calculated | (actual-threshold)/threshold*100 |
| detected_at | timestamp | Generated | Detection timestamp |
| resolved_at | timestamp | Updated | Resolution timestamp (nullable) |

### agreement_compliance_summaries

| Column | Type | Source | Transformation |
|--------|------|--------|----------------|
| id | varchar(255) | Generated | `{project}:{period}` |
| project_name | varchar(255) | Aggregation | Aggregation key |
| period_start | timestamp | Calculated | Period start |
| period_end | timestamp | Calculated | Period end |
| total_checks | int | Counted | Total items checked |
| violations_count | int | Counted | Items with violations |
| compliance_rate | decimal(5,2) | Calculated | (total-violations)/total*100 |
| calculated_at | timestamp | Generated | Calculation timestamp |

## Health Score Calculation

The team health score is based on DORA metrics, with each metric contributing up to 25 points:

```
health_score = deployment_frequency_score    (0-25)
             + lead_time_score               (0-25)
             + change_failure_rate_score     (0-25)
             + time_to_restore_score         (0-25)

Maximum: 100 points
```

### DORA Metric Scoring Thresholds

| Metric | Elite (25) | High (20) | Medium (15) | Low (10) | Poor (5) |
|--------|-----------|-----------|-------------|----------|----------|
| Deployment Frequency | Multiple/day | Daily | Weekly | Monthly | <Monthly |
| Lead Time | <1 hour | <1 day | <1 week | <1 month | >1 month |
| Change Failure Rate | <5% | <10% | <15% | <25% | >25% |
| Time to Restore | <1 hour | <1 day | <1 week | <1 month | >1 month |

### Health Level Classification

| Level | Score Range | Color |
|-------|-------------|-------|
| Excellent | 80-100 | Green |
| Good | 60-79 | Blue |
| Fair | 40-59 | Yellow |
| Poor | 0-39 | Red |

## Working Agreement Types

Supported agreement types (Swarmia-style):

| Agreement Type | Description | Default Threshold |
|----------------|-------------|-------------------|
| pr_size | Maximum lines changed per PR | 400 lines |
| review_time | Maximum time to first review | 24 hours |
| merge_time | Maximum time from open to merge | 72 hours |
| code_review_coverage | Minimum % of PRs with reviews | 80% |
| pr_open_limit | Maximum concurrent open PRs | 5 per developer |
| stale_pr_threshold | Days before PR considered stale | 7 days |

## Plugin Task Execution Order

```
businessmetrics plugin subtasks (from impl.go):
  1. extractBusinessGoals    → Extract Epics as initiatives
  2. calculateAlignment      → Link issues to initiatives
  3. calculateHealthScore    → Calculate DORA-based health
  4. calculateBusinessValue  → Calculate initiative value scores
  5. checkAgreements         → Check working agreement violations
```

## Dependencies

| Dependency | Required Tables | Notes |
|------------|-----------------|-------|
| Jira plugin | issues (type='Epic') | Must have Epics |
| DORA plugin | project_pr_metrics | Required for health scores |
| project_mapping | project_mapping | Links issues to projects |

## Data Freshness

- **business_initiatives**: Updated on each pipeline run
- **team_health_scores**: Calculated per period (weekly/monthly)
- **agreement_violations**: Real-time detection on pipeline run
- **compliance_summaries**: Aggregated by period

## Known Limitations

1. **Epic dependency**: Requires Jira Epics to exist for initiative tracking
2. **DORA dependency**: Health scores require DORA plugin to run first
3. **Manual agreements**: Working agreements must be configured via API
4. **Label parsing**: Investment categories extracted from labels (requires consistent labeling)
