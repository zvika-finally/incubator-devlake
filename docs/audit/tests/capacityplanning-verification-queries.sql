-- ============================================
-- CAPACITY PLANNING DASHBOARD VERIFICATION QUERIES
-- Run these to validate metric calculations
-- ============================================

-- ============================================
-- SECTION 1: COMPLETENESS CHECKS
-- ============================================

-- #cap-completeness-01: Team velocities exist for tracked projects
SELECT
    'cap-completeness-01' as check_id,
    'Team velocities exist for tracked projects' as check_name,
    (SELECT COUNT(DISTINCT project_name) FROM project_mapping WHERE `table` = 'boards') as tracked_projects,
    (SELECT COUNT(DISTINCT project_name) FROM team_velocities) as projects_with_velocities,
    CASE
        WHEN (SELECT COUNT(DISTINCT project_name) FROM project_mapping WHERE `table` = 'boards')
             <= (SELECT COUNT(DISTINCT project_name) FROM team_velocities)
        THEN 'PASS' ELSE 'REVIEW'
    END as status;

-- #cap-completeness-02: Monte Carlo forecasts exist for initiatives
SELECT
    'cap-completeness-02' as check_id,
    'Monte Carlo forecasts exist for initiatives' as check_name,
    (SELECT COUNT(*) FROM initiative_forecasts) as total_initiative_forecasts,
    (SELECT COUNT(*) FROM monte_carlo_forecasts) as total_mc_forecasts,
    CASE
        WHEN (SELECT COUNT(*) FROM initiative_forecasts) <= (SELECT COUNT(*) FROM monte_carlo_forecasts)
        THEN 'PASS' ELSE 'REVIEW'
    END as status;

-- #cap-completeness-03: Flow efficiency metrics exist for completed issues
SELECT
    'cap-completeness-03' as check_id,
    'Flow efficiency for completed issues' as check_name,
    (SELECT COUNT(*) FROM issues WHERE resolution_date IS NOT NULL) as completed_issues,
    (SELECT COUNT(*) FROM issue_flow_metrics) as issues_with_flow_metrics,
    CASE
        WHEN (SELECT COUNT(*) FROM issue_flow_metrics) > 0 THEN 'PASS'
        ELSE 'REVIEW'
    END as status;

-- #cap-completeness-04: Project flow summaries exist for projects with flow metrics
SELECT
    'cap-completeness-04' as check_id,
    'Project flow summaries exist' as check_name,
    (SELECT COUNT(DISTINCT project_name) FROM issue_flow_metrics) as projects_with_flow_metrics,
    (SELECT COUNT(DISTINCT project_name) FROM project_flow_summaries) as projects_with_summaries,
    CASE
        WHEN (SELECT COUNT(DISTINCT project_name) FROM issue_flow_metrics)
             <= (SELECT COUNT(DISTINCT project_name) FROM project_flow_summaries)
        THEN 'PASS' ELSE 'REVIEW'
    END as status;

-- ============================================
-- SECTION 2: ACCURACY CHECKS - MONTE CARLO
-- ============================================

-- #cap-accuracy-01: Monte Carlo percentiles are correctly ordered (P50 <= P75 <= P90 <= P95)
SELECT
    'cap-accuracy-01' as check_id,
    'Monte Carlo percentiles are ordered' as check_name,
    COUNT(*) as violations,
    CASE WHEN COUNT(*) = 0 THEN 'PASS' ELSE 'FAIL' END as status
FROM monte_carlo_forecasts
WHERE p50_sprints > p75_sprints
   OR p75_sprints > p90_sprints
   OR p90_sprints > p95_sprints;

-- #cap-accuracy-02: Monte Carlo simulation count matches setting (default 1000)
SELECT
    'cap-accuracy-02' as check_id,
    'Simulation count is correct' as check_name,
    MIN(simulation_count) as min_simulations,
    MAX(simulation_count) as max_simulations,
    CASE
        WHEN MIN(simulation_count) >= 100 THEN 'PASS'
        ELSE 'REVIEW'
    END as status
FROM monte_carlo_forecasts
WHERE simulation_count > 0;

-- #cap-accuracy-03: Monte Carlo dates are in future from calculation date
SELECT
    'cap-accuracy-03' as check_id,
    'Monte Carlo dates are future dates' as check_name,
    COUNT(*) as past_date_violations,
    CASE WHEN COUNT(*) = 0 THEN 'PASS' ELSE 'REVIEW' END as status
FROM monte_carlo_forecasts
WHERE p50_date < calculated_at
   OR p90_date < calculated_at;

-- ============================================
-- SECTION 3: ACCURACY CHECKS - FLOW EFFICIENCY
-- ============================================

-- #cap-accuracy-04: Flow efficiency formula is correct
-- Formula: flow_efficiency = (active_days / total_days) * 100
SELECT
    'cap-accuracy-04' as check_id,
    'Flow efficiency formula is correct' as check_name,
    issue_key,
    total_days,
    active_days,
    flow_efficiency as stored_efficiency,
    CASE
        WHEN total_days > 0
        THEN ROUND((active_days / total_days) * 100, 2)
        ELSE 0
    END as calculated_efficiency,
    CASE
        WHEN total_days = 0 THEN 'SKIP'
        WHEN ABS(flow_efficiency - (active_days / total_days * 100)) < 0.5
        THEN 'PASS' ELSE 'FAIL'
    END as status
FROM issue_flow_metrics
WHERE total_days > 0
ORDER BY completed_at DESC
LIMIT 5;

-- #cap-accuracy-05: Flow efficiency is within valid range (0-100)
SELECT
    'cap-accuracy-05' as check_id,
    'Flow efficiency is 0-100' as check_name,
    MIN(flow_efficiency) as min_efficiency,
    MAX(flow_efficiency) as max_efficiency,
    CASE
        WHEN MIN(flow_efficiency) >= 0 AND MAX(flow_efficiency) <= 100
        THEN 'PASS' ELSE 'FAIL'
    END as status
FROM issue_flow_metrics
WHERE flow_efficiency IS NOT NULL;

-- #cap-accuracy-06: Waiting days = total days - active days
SELECT
    'cap-accuracy-06' as check_id,
    'Waiting days formula is correct' as check_name,
    COUNT(*) as violations,
    CASE WHEN COUNT(*) = 0 THEN 'PASS' ELSE 'FAIL' END as status
FROM issue_flow_metrics
WHERE ABS(waiting_days - (total_days - active_days)) > 0.1;

-- ============================================
-- SECTION 4: ACCURACY CHECKS - INITIATIVE FORECASTS
-- ============================================

-- #cap-accuracy-07: Percent complete formula is correct
-- Formula: percent_complete = (completed_story_points / total_story_points) * 100
SELECT
    'cap-accuracy-07' as check_id,
    'Percent complete formula is correct' as check_name,
    initiative_name,
    total_story_points,
    completed_story_points,
    percent_complete as stored_percent,
    CASE
        WHEN total_story_points > 0
        THEN ROUND((completed_story_points * 100.0 / total_story_points), 2)
        ELSE 0
    END as calculated_percent,
    CASE
        WHEN total_story_points = 0 THEN 'SKIP'
        WHEN ABS(percent_complete - (completed_story_points * 100.0 / total_story_points)) < 0.5
        THEN 'PASS' ELSE 'FAIL'
    END as status
FROM initiative_forecasts
WHERE total_story_points > 0
ORDER BY calculated_at DESC
LIMIT 5;

-- #cap-accuracy-08: Remaining = Total - Completed
SELECT
    'cap-accuracy-08' as check_id,
    'Remaining story points formula correct' as check_name,
    COUNT(*) as violations,
    CASE WHEN COUNT(*) = 0 THEN 'PASS' ELSE 'FAIL' END as status
FROM initiative_forecasts
WHERE remaining_story_points != (total_story_points - completed_story_points);

-- ============================================
-- SECTION 5: ACCURACY CHECKS - ROI
-- ============================================

-- #cap-accuracy-09: Annual cost calculation (upfront + monthly*12)
SELECT
    'cap-accuracy-09' as check_id,
    'Annual cost formula is correct' as check_name,
    investment_name,
    upfront_cost,
    monthly_cost,
    annual_cost as stored_annual,
    (upfront_cost + monthly_cost * 12) as calculated_annual,
    CASE
        WHEN ABS(annual_cost - (upfront_cost + monthly_cost * 12)) < 0.01
        THEN 'PASS' ELSE 'FAIL'
    END as status
FROM investment_rois
ORDER BY calculated_at DESC
LIMIT 5;

-- #cap-accuracy-10: Total annual benefit sum is correct
SELECT
    'cap-accuracy-10' as check_id,
    'Total benefit = sum of components' as check_name,
    investment_name,
    direct_benefit,
    productivity_benefit,
    quality_benefit,
    total_annual_benefit as stored_total,
    (direct_benefit + productivity_benefit + quality_benefit) as calculated_total,
    CASE
        WHEN ABS(total_annual_benefit - (direct_benefit + productivity_benefit + quality_benefit)) < 0.01
        THEN 'PASS' ELSE 'FAIL'
    END as status
FROM investment_rois
ORDER BY calculated_at DESC
LIMIT 5;

-- ============================================
-- SECTION 6: ACCURACY CHECKS - BROOKS'S LAW
-- ============================================

-- #cap-accuracy-11: Communication channels formula (n*(n-1)/2)
SELECT
    'cap-accuracy-11' as check_id,
    'Communication channels formula correct' as check_name,
    project_name,
    scenario_name,
    current_team_size,
    current_channels as stored_channels,
    (current_team_size * (current_team_size - 1) / 2) as calculated_channels,
    CASE
        WHEN current_channels = (current_team_size * (current_team_size - 1) / 2)
        THEN 'PASS' ELSE 'FAIL'
    END as status
FROM capacity_models
ORDER BY calculated_at DESC
LIMIT 5;

-- ============================================
-- SECTION 7: CONSISTENCY CHECKS
-- ============================================

-- #cap-consistency-01: Project flow summary categories sum to issue count
SELECT
    'cap-consistency-01' as check_id,
    'Flow category counts sum to total' as check_name,
    project_name,
    sprint_name,
    issue_count,
    (excellent_count + good_count + average_count + poor_count) as sum_categories,
    CASE
        WHEN issue_count = (excellent_count + good_count + average_count + poor_count)
        THEN 'PASS' ELSE 'FAIL'
    END as status
FROM project_flow_summaries
ORDER BY period_end DESC
LIMIT 5;

-- #cap-consistency-02: Throughput data matches recent periods
SELECT
    'cap-consistency-02' as check_id,
    'Throughput data has recent periods' as check_name,
    MAX(sprint_end_date) as most_recent_period,
    DATEDIFF(NOW(), MAX(sprint_end_date)) as days_since_last,
    CASE
        WHEN DATEDIFF(NOW(), MAX(sprint_end_date)) <= 14 THEN 'PASS'
        ELSE 'REVIEW'
    END as status
FROM team_velocities;

-- ============================================
-- SECTION 8: FRESHNESS CHECKS
-- ============================================

-- #cap-freshness-01: Team velocities are recent
SELECT
    'cap-freshness-01' as check_id,
    'Team velocities are fresh (within 14 days)' as check_name,
    MAX(calculated_at) as most_recent,
    DATEDIFF(NOW(), MAX(calculated_at)) as days_old,
    CASE
        WHEN DATEDIFF(NOW(), MAX(calculated_at)) <= 14 THEN 'PASS'
        ELSE 'FAIL'
    END as status
FROM team_velocities;

-- #cap-freshness-02: Monte Carlo forecasts are recent
SELECT
    'cap-freshness-02' as check_id,
    'Monte Carlo forecasts are fresh (within 14 days)' as check_name,
    MAX(calculated_at) as most_recent,
    DATEDIFF(NOW(), MAX(calculated_at)) as days_old,
    CASE
        WHEN DATEDIFF(NOW(), MAX(calculated_at)) <= 14 THEN 'PASS'
        ELSE 'FAIL'
    END as status
FROM monte_carlo_forecasts;

-- #cap-freshness-03: Flow metrics are recent
SELECT
    'cap-freshness-03' as check_id,
    'Flow metrics are fresh (within 14 days)' as check_name,
    MAX(calculated_at) as most_recent,
    DATEDIFF(NOW(), MAX(calculated_at)) as days_old,
    CASE
        WHEN DATEDIFF(NOW(), MAX(calculated_at)) <= 14 THEN 'PASS'
        ELSE 'FAIL'
    END as status
FROM issue_flow_metrics;

-- ============================================
-- SECTION 9: DISTRIBUTION CHECKS
-- ============================================

-- #cap-distribution-01: Confidence level distribution
SELECT
    'cap-distribution-01' as check_id,
    'Confidence level distribution' as check_name,
    confidence_level,
    COUNT(*) as count,
    ROUND(COUNT(*) * 100.0 / (SELECT COUNT(*) FROM initiative_forecasts), 2) as percent,
    'INFO' as status
FROM initiative_forecasts
GROUP BY confidence_level;

-- #cap-distribution-02: Flow efficiency category distribution
SELECT
    'cap-distribution-02' as check_id,
    'Flow efficiency category distribution' as check_name,
    CASE
        WHEN flow_efficiency >= 40 THEN 'Excellent (>=40%)'
        WHEN flow_efficiency >= 25 THEN 'Good (25-39%)'
        WHEN flow_efficiency >= 15 THEN 'Average (15-24%)'
        ELSE 'Poor (<15%)'
    END as category,
    COUNT(*) as count,
    ROUND(COUNT(*) * 100.0 / (SELECT COUNT(*) FROM issue_flow_metrics), 2) as percent,
    'INFO' as status
FROM issue_flow_metrics
GROUP BY CASE
    WHEN flow_efficiency >= 40 THEN 'Excellent (>=40%)'
    WHEN flow_efficiency >= 25 THEN 'Good (25-39%)'
    WHEN flow_efficiency >= 15 THEN 'Average (15-24%)'
    ELSE 'Poor (<15%)'
END;

-- #cap-distribution-03: Investment types distribution
SELECT
    'cap-distribution-03' as check_id,
    'Investment types' as check_name,
    investment_type,
    COUNT(*) as count,
    SUM(total_annual_benefit) as total_benefit,
    'INFO' as status
FROM investment_rois
GROUP BY investment_type;

-- ============================================
-- SECTION 10: SAMPLE DATA REVIEW
-- ============================================

-- #cap-sample-01: Sample throughput data
SELECT
    'cap-sample-01' as check_id,
    'Sample throughput data' as check_name,
    project_name,
    fiscal_week,
    issues_completed,
    story_points_completed,
    avg_cycle_time_hours,
    team_size,
    'MANUAL' as status
FROM team_velocities
ORDER BY sprint_end_date DESC
LIMIT 5;

-- #cap-sample-02: Sample Monte Carlo forecasts
SELECT
    'cap-sample-02' as check_id,
    'Sample Monte Carlo forecasts' as check_name,
    initiative_id,
    p50_sprints as p50_weeks,
    p90_sprints as p90_weeks,
    DATE(p50_date) as p50_date,
    DATE(p90_date) as p90_date,
    earliest_days,
    latest_days,
    'MANUAL' as status
FROM monte_carlo_forecasts
ORDER BY calculated_at DESC
LIMIT 5;

-- #cap-sample-03: Sample flow efficiency data
SELECT
    'cap-sample-03' as check_id,
    'Sample flow efficiency' as check_name,
    project_name,
    issue_key,
    issue_type,
    total_days,
    active_days,
    waiting_days,
    flow_efficiency,
    'MANUAL' as status
FROM issue_flow_metrics
ORDER BY completed_at DESC
LIMIT 5;

-- #cap-sample-04: Sample Brooks's Law scenarios
SELECT
    'cap-sample-04' as check_id,
    'Sample Brooks Law scenarios' as check_name,
    project_name,
    scenario_name,
    current_team_size,
    team_size_delta,
    current_channels,
    new_channels,
    overhead_factor,
    productivity_factor,
    'MANUAL' as status
FROM capacity_models
ORDER BY calculated_at DESC
LIMIT 5;

-- #cap-sample-05: Sample ROI calculations
SELECT
    'cap-sample-05' as check_id,
    'Sample ROI calculations' as check_name,
    investment_name,
    investment_type,
    annual_cost,
    total_annual_benefit,
    payback_months,
    three_year_roi,
    'MANUAL' as status
FROM investment_rois
ORDER BY calculated_at DESC
LIMIT 5;
