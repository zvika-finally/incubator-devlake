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

## Effort Inference Data Flow (NEW)

```
┌─────────────────────────────────────────────────────────────────┐
│                     EFFORT INFERENCE PIPELINE                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  1. collectDeveloperActivity (Swarmia Model)            │    │
│  │     Input: commits, pull_requests, issues               │    │
│  │     Output: developer_monthly_fte                       │    │
│  │     Logic: weighted activity → normalized FTE           │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              ↓                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  2. inferGitEffort (git2effort methodology)             │    │
│  │     Input: issue_commits, pull_request_issues           │    │
│  │     Output: cached git_effort per issue                 │    │
│  │     Logic: active_days × productive_hours + reviews     │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              ↓                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  3. calculateCosts (multi-source fusion)                │    │
│  │     Input: issues + git_effort + FTE                    │    │
│  │     Output: cost_allocations with effort_source         │    │
│  │     Logic: Jira time > estimate > points > git > FTE    │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Effort Source Priority

| Priority | Source | Confidence | When Used |
|----------|--------|------------|-----------|
| 1 | jira_time | HIGH | time_spent_minutes > 0 |
| 2 | jira_estimate | MEDIUM | original_estimate_minutes > 0 |
| 3 | story_points | MEDIUM | story_point > 0 |
| 4 | git_inferred | INFERRED | linked commits/PRs exist |
| 5 | fte_distributed | LOW | FTE allocation fallback |

## Developer Monthly FTE Table

| Column | Type | Source | Transformation |
|--------|------|--------|----------------|
| id | varchar(255) | Generated | `{developer_id}:{fiscal_month}` |
| developer_id | varchar(255) | commits.author_id | Aggregated |
| fiscal_month | varchar(10) | commits.authored_date | `YYYY-MM` format |
| prs_authored | int | pull_requests | COUNT per month |
| prs_reviewed | int | pull_request_comments | COUNT per month |
| commits_authored | int | commits | COUNT per month |
| raw_activity_score | decimal(10,2) | Calculated | Weighted sum of activities |
| baseline_score | decimal(10,2) | Calculated | Team median × multiplier |
| adjusted_fte | decimal(3,2) | Calculated | Normalized FTE (0-1.0) |

## Cost Allocation Effort Source Fields

| Column | Type | Source | Description |
|--------|------|--------|-------------|
| effort_source | varchar(50) | getHoursWorkedMultiSource | jira_time, jira_estimate, story_points, git_inferred, fte_distributed |
| confidence_level | varchar(20) | getHoursWorkedMultiSource | high, medium, inferred, low |
| git_coding_hours | decimal(10,2) | inferGitEffort | Hours from coding activity |
| git_review_hours | decimal(10,2) | inferGitEffort | Hours from review activity |
| git_complexity_factor | decimal(5,2) | inferGitEffort | Complexity multiplier |
| git_active_days | int | inferGitEffort | Days with commits |
| effort_validated | bool | getHoursWorkedMultiSource | Jira vs Git cross-validated |
| validation_variance_pct | decimal(8,2) | getHoursWorkedMultiSource | Variance between Jira and Git |
| linked_commit_shas | text | inferGitEffort | JSON array of commit SHAs (R&D audit) |
| linked_pr_ids | text | inferGitEffort | JSON array of PR IDs (R&D audit) |

## Plugin Task Execution Order

```
findevops plugin subtasks:
  1. collectDeveloperActivity (creates developer_monthly_fte - Swarmia FTE model)
  2. inferGitEffort           (caches git effort per issue - git2effort)
  3. calculateCosts           (creates cost_allocations with multi-source effort)
  4. categorizeCapitalization (updates cost_allocations with ASC 350-40 phase)
  5. calculateDeploymentCosts (creates deployment_costs)
```

## Dependencies

| Dependency | Required Tables | Notes |
|------------|-----------------|-------|
| Jira plugin | issues, board_issues, _tool_jira_issues | Must run before findevops |
| CI/CD plugins | cicd_deployment_commits | Required for deployment cost metrics |
| project_mapping | project_mapping | Links boards to projects |
