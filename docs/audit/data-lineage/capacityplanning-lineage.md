# Capacity Planning Data Lineage

## Overview

The Capacity Planning plugin provides forecasting and planning capabilities using Kanban throughput metrics (issue counts) and optional Scrum velocity metrics (story points). It implements Monte Carlo simulations for probabilistic forecasting, Brooks's Law capacity modeling, and ROI calculations for development investments.

## Data Flow Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        SOURCE TABLES                            │
├─────────────────────────────────────────────────────────────────┤
│  Domain Layer (Core DevLake)                                    │
│  - issues               │  Issue data with status, dates        │
│  - sprints              │  Sprint definitions (optional/Scrum)  │
│  - issue_changelogs     │  Status transitions for flow metrics  │
│  - pull_requests        │  PR data for team metrics             │
│  - commits              │  Commit data                          │
│  - project_mapping      │  Project associations                 │
│                                                                  │
│  Plugin Tables (Optional)                                        │
│  - business_initiatives │  Epic-based initiative tracking       │
└─────────────────────────────────────────────────────────────────┘
                               ↓
┌─────────────────────────────────────────────────────────────────┐
│                CAPACITYPLANNER PLUGIN SUBTASKS                   │
├─────────────────────────────────────────────────────────────────┤
│  1. calculateThroughput (Kanban)                                 │
│     Input: issues (resolved, by week)                            │
│     Logic: Count completed issues per week                       │
│     Output: team_velocities (issues_completed, avg_cycle_time)   │
├─────────────────────────────────────────────────────────────────┤
│  2. calculateVelocity (Scrum - optional)                         │
│     Input: issues + sprints                                      │
│     Logic: Sum story points per sprint                           │
│     Output: team_velocities (story_points_completed)             │
├─────────────────────────────────────────────────────────────────┤
│  3. forecastCompletionKanban                                     │
│     Input: issues (by initiative), team_velocities               │
│     Logic: remaining_issues / avg_throughput                     │
│     Output: initiative_forecasts                                 │
├─────────────────────────────────────────────────────────────────┤
│  4. monteCarloForecastKanban                                     │
│     Input: team_velocities, initiative_forecasts                 │
│     Logic: 1000 iterations with throughput variance              │
│     Output: monte_carlo_forecasts (P50, P75, P90, P95)           │
├─────────────────────────────────────────────────────────────────┤
│  5. forecastCompletion (Scrum - optional)                        │
│     Input: issues (by initiative), team_velocities               │
│     Logic: remaining_story_points / avg_velocity                 │
│     Output: initiative_forecasts                                 │
├─────────────────────────────────────────────────────────────────┤
│  6. monteCarloForecast (Scrum - optional)                        │
│     Input: team_velocities, initiative_forecasts                 │
│     Logic: 1000 iterations with velocity variance                │
│     Output: monte_carlo_forecasts                                │
├─────────────────────────────────────────────────────────────────┤
│  7. brooksLawModel                                               │
│     Input: project_mapping, team configuration                   │
│     Logic: channels = n×(n-1)/2, overhead factors                │
│     Output: capacity_models                                      │
├─────────────────────────────────────────────────────────────────┤
│  8. calculateROI                                                 │
│     Input: settings (costs, hours_saved, etc.)                   │
│     Logic: Direct + Productivity + Quality benefits              │
│     Output: investment_rois                                      │
├─────────────────────────────────────────────────────────────────┤
│  9. calculateFlowEfficiency                                      │
│     Input: issues, issue_changelogs                              │
│     Logic: active_days / total_days × 100                        │
│     Output: issue_flow_metrics, project_flow_summaries           │
└─────────────────────────────────────────────────────────────────┘
                               ↓
┌─────────────────────────────────────────────────────────────────┐
│                CAPACITYPLANNER OUTPUT TABLES                     │
├─────────────────────────────────────────────────────────────────┤
│  team_velocities           │ Throughput/velocity per period      │
│  initiative_forecasts      │ Completion date predictions         │
│  monte_carlo_forecasts     │ Probabilistic forecasts (P50-P95)   │
│  capacity_scenarios        │ What-if scenario calculations       │
│  capacity_models           │ Brooks's Law team capacity          │
│  investment_rois           │ ROI calculations for investments    │
│  issue_flow_metrics        │ Per-issue flow efficiency           │
│  project_flow_summaries    │ Aggregated flow efficiency          │
└─────────────────────────────────────────────────────────────────┘
                               ↓
┌─────────────────────────────────────────────────────────────────┐
│                   GRAFANA DASHBOARD                              │
│            Capacity Planning & Forecasting (Kanban)              │
│              (capacity-planning-dashboard)                       │
└─────────────────────────────────────────────────────────────────┘
```

## Table Schemas

### team_velocities

| Column | Type | Source | Transformation |
|--------|------|--------|----------------|
| id | varchar(255) | Generated | `{project}:{sprint_id or fiscal_week}` |
| project_name | varchar(255) | project_mapping | Aggregation key |
| sprint_id | varchar(255) | sprints | Optional (Scrum) |
| sprint_name | varchar(255) | sprints | Optional (Scrum) |
| fiscal_week | varchar(10) | Calculated | "2026-W05" format |
| story_points_completed | int | issues | SUM(story_point) per sprint/week |
| issues_completed | int | issues | COUNT of resolved issues |
| prs_merged | int | pull_requests | COUNT of merged PRs |
| commit_count | int | commits | COUNT of commits |
| avg_cycle_time_hours | decimal(10,2) | issues | AVG(resolution_date - created_date) |
| avg_lead_time_hours | decimal(10,2) | issues | AVG(resolution_date - first_commit) |
| team_size | int | Settings/Inferred | Developer count |
| available_hours | decimal(10,2) | Settings | Configured hours |
| sprint_start_date | timestamp | sprints | Sprint start (optional) |
| sprint_end_date | timestamp | sprints/calculated | Sprint/week end |
| calculated_at | timestamp | Generated | Calculation timestamp |

### initiative_forecasts

| Column | Type | Source | Transformation |
|--------|------|--------|----------------|
| id | varchar(255) | Generated | `{initiative_id}:{timestamp}` |
| initiative_id | varchar(255) | business_initiatives | Link to initiative |
| initiative_name | varchar(500) | business_initiatives | Display name |
| total_story_points | int | issues | Total issues/points in initiative |
| completed_story_points | int | issues | Resolved issues/points |
| remaining_story_points | int | Calculated | total - completed |
| percent_complete | decimal(5,2) | Calculated | (completed/total) × 100 |
| avg_velocity | decimal(10,2) | team_velocities | AVG throughput/velocity |
| estimated_sprints | int | Calculated | remaining / avg_velocity |
| estimated_completion_date | timestamp | Calculated | NOW + (sprints × sprint_length) |
| confidence_level | varchar(20) | Calculated | high/medium/low |
| velocity_std_dev | decimal(10,2) | team_velocities | Standard deviation |
| scenario_data | text | Calculated | JSON with best/worst cases |
| calculated_at | timestamp | Generated | Calculation timestamp |

### monte_carlo_forecasts

| Column | Type | Source | Transformation |
|--------|------|--------|----------------|
| id | varchar(255) | Generated | `{initiative_id}:{timestamp}` |
| initiative_id | varchar(255) | initiative_forecasts | Link to initiative |
| simulation_count | int | Settings | Default: 1000 |
| velocity_variance | decimal(5,2) | Settings | Default: 0.25 (25%) |
| p50_sprints | int | Simulation | 50th percentile (median) |
| p75_sprints | int | Simulation | 75th percentile |
| p90_sprints | int | Simulation | 90th percentile |
| p95_sprints | int | Simulation | 95th percentile (conservative) |
| p50_date | timestamp | Calculated | NOW + (p50_sprints × week_length) |
| p75_date | timestamp | Calculated | NOW + (p75_sprints × week_length) |
| p90_date | timestamp | Calculated | NOW + (p90_sprints × week_length) |
| p95_date | timestamp | Calculated | NOW + (p95_sprints × week_length) |
| earliest_days | int | Simulation | Best case |
| latest_days | int | Simulation | Worst case |
| calculated_at | timestamp | Generated | Calculation timestamp |

### capacity_models

| Column | Type | Source | Transformation |
|--------|------|--------|----------------|
| id | varchar(255) | Generated | `{project}:{scenario}` |
| project_name | varchar(255) | project_mapping | Project key |
| scenario_name | varchar(255) | Settings | Scenario description |
| current_team_size | int | Settings/Inferred | Current team count |
| team_size_delta | int | Settings | Change (+N or -N) |
| ramp_up_weeks | int | Settings | Default: 8 weeks |
| current_channels | int | Calculated | n×(n-1)/2 |
| new_channels | int | Calculated | (n+delta)×(n+delta-1)/2 |
| overhead_factor | decimal(5,2) | Calculated | Communication overhead |
| productivity_factor | decimal(5,2) | Calculated | Effective capacity multiplier |
| projected_deploy_delta | decimal(5,2) | Calculated | % change in deploy frequency |
| projected_lead_delta | decimal(5,2) | Calculated | % change in lead time |
| calculated_at | timestamp | Generated | Calculation timestamp |

### investment_rois

| Column | Type | Source | Transformation |
|--------|------|--------|----------------|
| id | varchar(255) | Generated | `{investment_name}:{timestamp}` |
| investment_name | varchar(255) | Settings | Investment identifier |
| investment_type | varchar(50) | Settings | ai_tools, hiring, tech_debt |
| upfront_cost | decimal(12,2) | Settings | One-time cost |
| monthly_cost | decimal(12,2) | Settings | Recurring monthly cost |
| annual_cost | decimal(12,2) | Calculated | upfront + (monthly × 12) |
| direct_benefit | decimal(12,2) | Calculated | hours_saved × 52 × hourly_cost |
| productivity_benefit | decimal(12,2) | Calculated | team_hours × gain% × hourly_cost |
| quality_benefit | decimal(12,2) | Calculated | bug_hours × improvement% × hourly_cost |
| total_annual_benefit | decimal(12,2) | Calculated | Sum of benefits |
| payback_months | decimal(10,2) | Calculated | (annual_cost / annual_benefit) × 12 |
| three_year_roi | decimal(10,2) | Calculated | ((benefit×3 - cost×3) / cost×3) × 100 |
| parameters | text | Settings | JSON with input parameters |
| calculated_at | timestamp | Generated | Calculation timestamp |

### issue_flow_metrics

| Column | Type | Source | Transformation |
|--------|------|--------|----------------|
| id | varchar(255) | Generated | `{issue_id}` |
| project_name | varchar(255) | project_mapping | Project key |
| issue_id | varchar(255) | issues | Issue identifier |
| issue_key | varchar(100) | issues | Human-readable key |
| issue_type | varchar(50) | issues | Bug, Story, Task, etc. |
| total_days | decimal(10,2) | issue_changelogs | Calendar days from start to done |
| active_days | decimal(10,2) | issue_changelogs | Days in active statuses |
| waiting_days | decimal(10,2) | Calculated | total_days - active_days |
| flow_efficiency | decimal(5,2) | Calculated | (active_days / total_days) × 100 |
| started_at | timestamp | issue_changelogs | First transition from open |
| completed_at | timestamp | issue_changelogs | Transition to done |
| status_breakdown | text | issue_changelogs | JSON: {"In Progress": 5.5, ...} |
| transition_count | int | issue_changelogs | Total status changes |
| calculated_at | timestamp | Generated | Calculation timestamp |

### project_flow_summaries

| Column | Type | Source | Transformation |
|--------|------|--------|----------------|
| id | varchar(255) | Generated | `{project}:{period}` |
| project_name | varchar(255) | Aggregation | Project key |
| sprint_id | varchar(255) | sprints | Optional (Scrum) |
| sprint_name | varchar(255) | sprints | Optional (Scrum) |
| period_start | timestamp | Calculated | Period start date |
| period_end | timestamp | Calculated | Period end date |
| issue_count | int | issue_flow_metrics | COUNT of completed issues |
| avg_flow_efficiency | decimal(5,2) | issue_flow_metrics | AVG(flow_efficiency) |
| median_flow_efficiency | decimal(5,2) | issue_flow_metrics | MEDIAN(flow_efficiency) |
| p90_flow_efficiency | decimal(5,2) | issue_flow_metrics | P90(flow_efficiency) |
| avg_total_days | decimal(10,2) | issue_flow_metrics | AVG(total_days) |
| avg_active_days | decimal(10,2) | issue_flow_metrics | AVG(active_days) |
| avg_waiting_days | decimal(10,2) | issue_flow_metrics | AVG(waiting_days) |
| excellent_count | int | issue_flow_metrics | COUNT(flow_efficiency >= 40) |
| good_count | int | issue_flow_metrics | COUNT(25 <= flow_efficiency < 40) |
| average_count | int | issue_flow_metrics | COUNT(15 <= flow_efficiency < 25) |
| poor_count | int | issue_flow_metrics | COUNT(flow_efficiency < 15) |
| calculated_at | timestamp | Generated | Calculation timestamp |

## Key Algorithms

### Monte Carlo Simulation (Kanban)

```
for iteration in 1..1000:
    remaining = total_issues - completed_issues
    weeks = 0
    while remaining > 0:
        weekly_throughput = gaussian_random(avg_throughput, stddev × variance)
        remaining -= max(1, weekly_throughput)
        weeks += 1
    store(completion_weeks)

// Nearest-rank percentile method
P(n) = sorted_data[min(len × n/100, len-1)]
```

### Brooks's Law Communication Overhead

```
// Communication channels grow quadratically
channels = n × (n-1) / 2

// Example: Team of 5 → 10 channels, Team of 10 → 45 channels

// Overhead factor calculation
channel_delta = new_channels - current_channels
overhead_factor = 1 - (channel_delta / (current_channels + 1)) × 0.1

// New hire productivity during ramp-up (default 8 weeks)
new_hire_productivity = 0.5  // 50% during ramp-up
```

### ROI Calculation

```
// Annual benefits
Direct_Benefit = hours_saved_per_week × team_size × 52 × hourly_cost
Productivity_Benefit = team_hours_per_week × (gain_percent / 100) × hourly_cost
Quality_Benefit = team_hours_per_week × 0.20 × (improvement_percent / 100) × hourly_cost
Total_Annual_Benefit = Direct + Productivity + Quality

// ROI metrics
Payback_Months = (annual_cost / annual_benefit) × 12
Three_Year_ROI = ((benefit × 3 - cost × 3) / (cost × 3)) × 100
```

### Flow Efficiency

```
// Per issue
flow_efficiency = (active_days / total_days) × 100

// Category thresholds
Excellent: >= 40%  (World-class, rare)
Good:      25-39%  (Above average)
Average:   15-24%  (Typical for most teams)
Poor:      < 15%   (Improvement opportunity)
```

## Plugin Task Execution Order

```
capacityplanner plugin subtasks (from impl.go):
  1. calculateThroughput         → Kanban: weekly issue counts
  2. forecastCompletionKanban    → Kanban: initiative forecasting
  3. monteCarloForecastKanban    → Kanban: probabilistic forecasts
  4. calculateVelocity           → Scrum: sprint-based velocity (optional)
  5. forecastCompletion          → Scrum: story-point forecasting
  6. monteCarloForecast          → Scrum: Monte Carlo with velocity
  7. brooksLawModel              → Team capacity modeling
  8. calculateROI                → Investment ROI analysis
  9. calculateFlowEfficiency     → Flow efficiency metrics
```

## Dependencies

| Dependency | Required Tables | Notes |
|------------|-----------------|-------|
| Core DevLake | issues | Required for all metrics |
| Core DevLake | issue_changelogs | Required for flow efficiency |
| Sprints (optional) | sprints | Only needed for Scrum velocity |
| Business Metrics (optional) | business_initiatives | For initiative-level forecasting |

## Data Freshness

- **team_velocities**: Updated per period (weekly/sprint)
- **initiative_forecasts**: Updated on each pipeline run
- **monte_carlo_forecasts**: Updated on each pipeline run
- **capacity_models**: Updated when settings change
- **investment_rois**: Updated when settings change
- **issue_flow_metrics**: Updated per completed issue
- **project_flow_summaries**: Aggregated per period

## Known Limitations

1. **Kanban focus**: Dashboard optimized for Kanban (issue counts) over Scrum (story points)
2. **Initiative dependency**: Forecasting requires business_initiatives for initiative-level tracking
3. **Changelog dependency**: Flow efficiency requires issue_changelogs with status transitions
4. **Settings dependency**: Brooks's Law and ROI require manual configuration via API
5. **Variance assumption**: Monte Carlo uses Gaussian distribution with configurable variance (default 25%)

## Settings Configuration

The plugin supports per-project settings via the `/plugins/capacityplanner/settings/:projectName` API:

| Setting | Default | Description |
|---------|---------|-------------|
| simulation_count | 1000 | Monte Carlo iterations |
| velocity_variance | 0.25 | Throughput variance (25%) |
| ramp_up_weeks | 8 | New hire ramp-up period |
| hourly_cost | 75.00 | USD hourly cost for ROI |
| ai_hours_saved | 2.0 | Hours saved per week per user |
| ai_quality_improvement | 5.0 | % quality improvement |
| sprint_length_days | 7 | Days per sprint/week |
