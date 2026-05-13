-- ============================================
-- PRODUCTION VERIFICATION QUERIES
-- Run against RDS to validate dashboard metrics
-- ============================================

-- ===== 1. FINDEVOPS DASHBOARD =====
SELECT '=== FINDEVOPS VERIFICATION ===' as section;

-- FIN-01: Cost allocations coverage
SELECT 'FIN-01: Coverage' as check_id,
    COUNT(DISTINCT ca.issue_id) as allocated_issues,
    (SELECT COUNT(*) FROM issues WHERE resolution_date IS NOT NULL) as resolved_issues,
    ROUND(COUNT(DISTINCT ca.issue_id) * 100.0 /
        NULLIF((SELECT COUNT(*) FROM issues WHERE resolution_date IS NOT NULL), 0), 2) as coverage_pct
FROM cost_allocations ca;

-- FIN-02: ASC 350-40 categorization populated
SELECT 'FIN-02: ASC 350-40' as check_id,
    capitalization_category,
    COUNT(*) as count
FROM cost_allocations
WHERE capitalization_category IS NOT NULL AND capitalization_category != ''
GROUP BY capitalization_category;

-- FIN-03: Phase breakdown populated
SELECT 'FIN-03: Phase costs' as check_id,
    SUM(development_cost) as development,
    SUM(post_impl_cost) as post_impl,
    SUM(preliminary_cost) as preliminary
FROM monthly_cost_summaries;

-- FIN-04: Deployment costs exist
SELECT 'FIN-04: Deployment costs' as check_id,
    window_days,
    COUNT(*) as records,
    AVG(cost_per_deployment) as avg_cpd
FROM deployment_costs
GROUP BY window_days;

-- FIN-05: Capitalization rate formula
SELECT 'FIN-05: Cap rate formula' as check_id,
    fiscal_month,
    total_cost,
    capitalizable_cost,
    capitalization_rate as stored,
    ROUND(capitalizable_cost / NULLIF(total_cost, 0) * 100, 2) as calculated,
    CASE WHEN ABS(capitalization_rate - ROUND(capitalizable_cost / NULLIF(total_cost, 0) * 100, 2)) < 0.1 THEN 'PASS' ELSE 'FAIL' END as status
FROM monthly_cost_summaries
WHERE total_cost > 0
ORDER BY fiscal_month DESC LIMIT 3;

-- ===== 2. AI DETECTION DASHBOARD =====
SELECT '=== AI DETECTION VERIFICATION ===' as section;

-- AI-01: PRs with signals
SELECT 'AI-01: Coverage' as check_id,
    (SELECT COUNT(*) FROM pull_requests WHERE merged_date IS NOT NULL) as merged_prs,
    (SELECT COUNT(DISTINCT pull_request_id) FROM ai_usage_signals) as with_signals;

-- AI-02: Confidence distribution
SELECT 'AI-02: Confidence dist' as check_id,
    CASE WHEN ai_confidence_score >= 70 THEN 'High'
         WHEN ai_confidence_score >= 40 THEN 'Medium'
         ELSE 'Low' END as level,
    COUNT(*) as count
FROM ai_usage_signals
GROUP BY level;

-- AI-03: Detected tools
SELECT 'AI-03: Tools' as check_id,
    COALESCE(NULLIF(detected_tool, ''), 'unknown') as tool,
    COUNT(*) as count
FROM ai_usage_signals
WHERE ai_confidence_score >= 40
GROUP BY tool
ORDER BY count DESC LIMIT 5;

-- AI-04: Churn comparison
SELECT 'AI-04: Churn' as check_id,
    ROUND(AVG(CASE WHEN is_ai_assisted THEN churn_ratio_30_days END), 4) as ai_churn,
    ROUND(AVG(CASE WHEN NOT is_ai_assisted THEN churn_ratio_30_days END), 4) as non_ai_churn
FROM ai_churn_metrics;

-- ===== 3. BUSINESS METRICS DASHBOARD =====
SELECT '=== BUSINESS METRICS VERIFICATION ===' as section;

-- BIZ-01: Health scores exist
SELECT 'BIZ-01: Health scores' as check_id,
    project_name,
    total_score,
    health_level,
    deploy_freq_score,
    lead_time_score,
    cfr_score,
    mttr_score
FROM team_health_scores
ORDER BY calculated_at DESC LIMIT 5;

-- BIZ-02: Score formula (sum = total)
SELECT 'BIZ-02: Score formula' as check_id,
    project_name,
    total_score as stored,
    (deploy_freq_score + lead_time_score + cfr_score + mttr_score) as calculated,
    CASE WHEN total_score = (deploy_freq_score + lead_time_score + cfr_score + mttr_score) THEN 'PASS' ELSE 'FAIL' END as status
FROM team_health_scores
ORDER BY calculated_at DESC LIMIT 5;

-- BIZ-03: Initiatives
SELECT 'BIZ-03: Initiatives' as check_id,
    COUNT(*) as total,
    COUNT(CASE WHEN status = 'active' THEN 1 END) as active
FROM business_initiatives;

-- ===== 4. CAPACITY PLANNING DASHBOARD =====
SELECT '=== CAPACITY PLANNING VERIFICATION ===' as section;

-- CAP-01: Velocities exist
SELECT 'CAP-01: Velocities' as check_id,
    project_name,
    COUNT(*) as periods,
    AVG(issues_completed) as avg_throughput,
    AVG(avg_cycle_time_hours) as avg_cycle
FROM team_velocities
GROUP BY project_name;

-- CAP-02: Flow efficiency
SELECT 'CAP-02: Flow efficiency' as check_id,
    COUNT(*) as total_issues,
    ROUND(AVG(flow_efficiency), 2) as avg_efficiency,
    SUM(CASE WHEN flow_efficiency >= 40 THEN 1 ELSE 0 END) as excellent,
    SUM(CASE WHEN flow_efficiency >= 25 AND flow_efficiency < 40 THEN 1 ELSE 0 END) as good,
    SUM(CASE WHEN flow_efficiency < 15 THEN 1 ELSE 0 END) as poor
FROM issue_flow_metrics;

-- CAP-03: Flow formula check (efficiency = active/total * 100)
SELECT 'CAP-03: Flow formula' as check_id,
    issue_key,
    total_days,
    active_days,
    flow_efficiency as stored,
    ROUND(active_days / NULLIF(total_days, 0) * 100, 2) as calculated,
    CASE WHEN ABS(flow_efficiency - ROUND(active_days / NULLIF(total_days, 0) * 100, 2)) < 1 THEN 'PASS' ELSE 'FAIL' END as status
FROM issue_flow_metrics
WHERE total_days > 0
ORDER BY completed_at DESC LIMIT 5;

-- CAP-04: Monte Carlo forecasts
SELECT 'CAP-04: Monte Carlo' as check_id,
    COUNT(*) as forecast_count,
    AVG(simulation_count) as avg_simulations
FROM monte_carlo_forecasts;

-- CAP-05: Brooks Law (channels = n*(n-1)/2)
SELECT 'CAP-05: Brooks Law' as check_id,
    project_name,
    current_team_size,
    current_channels as stored,
    (current_team_size * (current_team_size - 1) / 2) as calculated,
    CASE WHEN current_channels = (current_team_size * (current_team_size - 1) / 2) THEN 'PASS' ELSE 'FAIL' END as status
FROM capacity_models
LIMIT 5;

-- ===== 5. DATA FRESHNESS =====
SELECT '=== FRESHNESS CHECK ===' as section;

SELECT 'Freshness' as check_id,
    'cost_allocations' as tbl, MAX(calculated_at) as latest, DATEDIFF(NOW(), MAX(calculated_at)) as days_old FROM cost_allocations
UNION ALL
SELECT 'Freshness', 'ai_usage_signals', MAX(detected_at), DATEDIFF(NOW(), MAX(detected_at)) FROM ai_usage_signals
UNION ALL
SELECT 'Freshness', 'team_health_scores', MAX(calculated_at), DATEDIFF(NOW(), MAX(calculated_at)) FROM team_health_scores
UNION ALL
SELECT 'Freshness', 'issue_flow_metrics', MAX(calculated_at), DATEDIFF(NOW(), MAX(calculated_at)) FROM issue_flow_metrics;

-- ===== 6. DATA COUNTS SUMMARY =====
SELECT '=== DATA COUNTS ===' as section;

SELECT 'cost_allocations' as tbl, COUNT(*) as cnt FROM cost_allocations
UNION ALL SELECT 'monthly_cost_summaries', COUNT(*) FROM monthly_cost_summaries
UNION ALL SELECT 'deployment_costs', COUNT(*) FROM deployment_costs
UNION ALL SELECT 'ai_usage_signals', COUNT(*) FROM ai_usage_signals
UNION ALL SELECT 'ai_churn_metrics', COUNT(*) FROM ai_churn_metrics
UNION ALL SELECT 'team_health_scores', COUNT(*) FROM team_health_scores
UNION ALL SELECT 'business_initiatives', COUNT(*) FROM business_initiatives
UNION ALL SELECT 'team_velocities', COUNT(*) FROM team_velocities
UNION ALL SELECT 'issue_flow_metrics', COUNT(*) FROM issue_flow_metrics
UNION ALL SELECT 'monte_carlo_forecasts', COUNT(*) FROM monte_carlo_forecasts
UNION ALL SELECT 'capacity_models', COUNT(*) FROM capacity_models
UNION ALL SELECT 'investment_rois', COUNT(*) FROM investment_rois;
