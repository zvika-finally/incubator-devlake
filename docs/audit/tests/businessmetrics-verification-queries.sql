-- ============================================
-- BUSINESS METRICS DASHBOARD VERIFICATION QUERIES
-- Run these to validate metric calculations
-- ============================================

-- ============================================
-- SECTION 1: COMPLETENESS CHECKS
-- ============================================

-- #biz-completeness-01: All Epics should be extracted as initiatives
SELECT
    'biz-completeness-01' as check_id,
    'All Epics extracted as initiatives' as check_name,
    (SELECT COUNT(*) FROM issues WHERE type = 'Epic') as total_epics,
    (SELECT COUNT(*) FROM business_initiatives) as total_initiatives,
    CASE
        WHEN (SELECT COUNT(*) FROM issues WHERE type = 'Epic')
             <= (SELECT COUNT(*) FROM business_initiatives)
        THEN 'PASS' ELSE 'REVIEW'
    END as status;

-- #biz-completeness-02: Health scores exist for all tracked projects
SELECT
    'biz-completeness-02' as check_id,
    'Health scores exist for tracked projects' as check_name,
    (SELECT COUNT(DISTINCT project_name) FROM project_mapping WHERE `table` = 'boards') as tracked_projects,
    (SELECT COUNT(DISTINCT project_name) FROM team_health_scores) as projects_with_scores,
    CASE
        WHEN (SELECT COUNT(DISTINCT project_name) FROM project_mapping WHERE `table` = 'boards')
             <= (SELECT COUNT(DISTINCT project_name) FROM team_health_scores)
        THEN 'PASS' ELSE 'REVIEW'
    END as status;

-- #biz-completeness-03: Compliance summaries exist for projects with agreements
SELECT
    'biz-completeness-03' as check_id,
    'Compliance summaries exist for projects with agreements' as check_name,
    (SELECT COUNT(DISTINCT project_name) FROM working_agreements WHERE enabled = true) as projects_with_agreements,
    (SELECT COUNT(DISTINCT project_name) FROM agreement_compliance_summaries) as projects_with_summaries,
    CASE
        WHEN (SELECT COUNT(DISTINCT project_name) FROM working_agreements WHERE enabled = true)
             <= (SELECT COUNT(DISTINCT project_name) FROM agreement_compliance_summaries)
        THEN 'PASS' ELSE 'REVIEW'
    END as status;

-- ============================================
-- SECTION 2: ACCURACY CHECKS - HEALTH SCORE
-- ============================================

-- #biz-accuracy-01: Health score is sum of 4 DORA scores (max 100)
SELECT
    'biz-accuracy-01' as check_id,
    'Health score = sum of DORA scores' as check_name,
    project_name,
    health_score as stored_score,
    deployment_frequency_score + lead_time_score +
        change_failure_rate_score + time_to_restore_score as calculated_score,
    CASE
        WHEN health_score = (deployment_frequency_score + lead_time_score +
            change_failure_rate_score + time_to_restore_score)
        THEN 'PASS' ELSE 'FAIL'
    END as status
FROM team_health_scores
ORDER BY calculated_at DESC
LIMIT 5;

-- #biz-accuracy-02: Individual DORA scores are 0-25
SELECT
    'biz-accuracy-02' as check_id,
    'DORA scores are within 0-25 range' as check_name,
    COUNT(*) as violations,
    CASE WHEN COUNT(*) = 0 THEN 'PASS' ELSE 'FAIL' END as status
FROM team_health_scores
WHERE deployment_frequency_score < 0 OR deployment_frequency_score > 25
   OR lead_time_score < 0 OR lead_time_score > 25
   OR change_failure_rate_score < 0 OR change_failure_rate_score > 25
   OR time_to_restore_score < 0 OR time_to_restore_score > 25;

-- #biz-accuracy-03: Health level matches score thresholds
SELECT
    'biz-accuracy-03' as check_id,
    'Health level matches score' as check_name,
    project_name,
    health_score,
    health_level,
    CASE
        WHEN health_score >= 80 THEN 'excellent'
        WHEN health_score >= 60 THEN 'good'
        WHEN health_score >= 40 THEN 'fair'
        ELSE 'poor'
    END as expected_level,
    CASE
        WHEN health_level = CASE
            WHEN health_score >= 80 THEN 'excellent'
            WHEN health_score >= 60 THEN 'good'
            WHEN health_score >= 40 THEN 'fair'
            ELSE 'poor'
        END THEN 'PASS' ELSE 'FAIL'
    END as status
FROM team_health_scores
ORDER BY calculated_at DESC
LIMIT 5;

-- ============================================
-- SECTION 3: ACCURACY CHECKS - COMPLIANCE
-- ============================================

-- #biz-accuracy-04: Compliance rate formula validation
-- Formula: compliance_rate = (total_checks - violations_count) / total_checks * 100
SELECT
    'biz-accuracy-04' as check_id,
    'Compliance rate formula is correct' as check_name,
    project_name,
    total_checks,
    violations_count,
    compliance_rate as stored_rate,
    CASE
        WHEN total_checks > 0
        THEN ROUND((total_checks - violations_count) / total_checks * 100, 2)
        ELSE 100
    END as calculated_rate,
    CASE
        WHEN total_checks = 0 THEN 'SKIP'
        WHEN ABS(compliance_rate - ((total_checks - violations_count) / total_checks * 100)) < 0.1
        THEN 'PASS' ELSE 'FAIL'
    END as status
FROM agreement_compliance_summaries
ORDER BY calculated_at DESC
LIMIT 5;

-- #biz-accuracy-05: Violation count consistency
SELECT
    'biz-accuracy-05' as check_id,
    'Violation count matches actual violations' as check_name,
    s.project_name,
    s.period_start,
    s.violations_count as summary_count,
    COUNT(v.id) as actual_violations,
    CASE
        WHEN s.violations_count = COUNT(v.id) THEN 'PASS'
        ELSE 'REVIEW'
    END as status
FROM agreement_compliance_summaries s
LEFT JOIN agreement_violations v ON v.project_name = s.project_name
    AND v.detected_at >= s.period_start AND v.detected_at < s.period_end
GROUP BY s.project_name, s.period_start, s.period_end, s.violations_count
ORDER BY s.calculated_at DESC
LIMIT 5;

-- ============================================
-- SECTION 4: CONSISTENCY CHECKS
-- ============================================

-- #biz-consistency-01: Work allocations sum to 100% per initiative
SELECT
    'biz-consistency-01' as check_id,
    'Work allocations sum to ~100% per initiative' as check_name,
    initiative_id,
    SUM(allocation_percent) as total_percent,
    CASE
        WHEN ABS(SUM(allocation_percent) - 100) < 1 THEN 'PASS'
        ELSE 'REVIEW'
    END as status
FROM work_allocations
GROUP BY initiative_id
HAVING SUM(allocation_percent) > 0
ORDER BY total_percent DESC
LIMIT 5;

-- #biz-consistency-02: Active violations have no resolved_at
SELECT
    'biz-consistency-02' as check_id,
    'Active violations have no resolved_at' as check_name,
    COUNT(*) as inconsistent_count,
    CASE WHEN COUNT(*) = 0 THEN 'PASS' ELSE 'FAIL' END as status
FROM agreement_violations
WHERE resolved_at IS NOT NULL
  AND resolved_at > NOW();  -- Future resolved_at is invalid

-- ============================================
-- SECTION 5: FRESHNESS CHECKS
-- ============================================

-- #biz-freshness-01: Health scores are recent
SELECT
    'biz-freshness-01' as check_id,
    'Health scores are fresh (within 7 days)' as check_name,
    MAX(calculated_at) as most_recent,
    DATEDIFF(NOW(), MAX(calculated_at)) as days_old,
    CASE
        WHEN DATEDIFF(NOW(), MAX(calculated_at)) <= 7 THEN 'PASS'
        ELSE 'FAIL'
    END as status
FROM team_health_scores;

-- #biz-freshness-02: Compliance summaries are recent
SELECT
    'biz-freshness-02' as check_id,
    'Compliance summaries are fresh (within 7 days)' as check_name,
    MAX(calculated_at) as most_recent,
    DATEDIFF(NOW(), MAX(calculated_at)) as days_old,
    CASE
        WHEN DATEDIFF(NOW(), MAX(calculated_at)) <= 7 THEN 'PASS'
        ELSE 'FAIL'
    END as status
FROM agreement_compliance_summaries;

-- ============================================
-- SECTION 6: BUSINESS VALUE CHECKS
-- ============================================

-- #biz-value-01: Value scores are within valid range (0-100)
SELECT
    'biz-value-01' as check_id,
    'Value scores are 0-100' as check_name,
    MIN(value_score) as min_score,
    MAX(value_score) as max_score,
    CASE
        WHEN MIN(value_score) >= 0 AND MAX(value_score) <= 100
        THEN 'PASS' ELSE 'FAIL'
    END as status
FROM business_initiatives
WHERE value_score IS NOT NULL;

-- #biz-value-02: Story points aggregation is correct
SELECT
    'biz-value-02' as check_id,
    'Story points match work allocations' as check_name,
    bi.id as initiative_id,
    bi.total_story_points as stored_total,
    COALESCE(SUM(wa.story_points), 0) as calculated_total,
    CASE
        WHEN bi.total_story_points = COALESCE(SUM(wa.story_points), 0)
        THEN 'PASS' ELSE 'REVIEW'
    END as status
FROM business_initiatives bi
LEFT JOIN work_allocations wa ON wa.initiative_id = bi.id
GROUP BY bi.id, bi.total_story_points
LIMIT 5;

-- ============================================
-- SECTION 7: DISTRIBUTION CHECKS
-- ============================================

-- #biz-distribution-01: Health level distribution
SELECT
    'biz-distribution-01' as check_id,
    'Health level distribution' as check_name,
    health_level,
    COUNT(*) as count,
    ROUND(COUNT(*) * 100.0 / (SELECT COUNT(*) FROM team_health_scores), 2) as percent,
    'INFO' as status
FROM team_health_scores
GROUP BY health_level;

-- #biz-distribution-02: Investment category distribution
SELECT
    'biz-distribution-02' as check_id,
    'Investment category distribution' as check_name,
    COALESCE(investment_category, 'Uncategorized') as category,
    COUNT(*) as count,
    SUM(total_story_points) as total_points,
    'INFO' as status
FROM business_initiatives
GROUP BY investment_category;

-- #biz-distribution-03: Agreement types distribution
SELECT
    'biz-distribution-03' as check_id,
    'Working agreement types' as check_name,
    agreement_type,
    COUNT(*) as count,
    SUM(CASE WHEN enabled THEN 1 ELSE 0 END) as enabled_count,
    'INFO' as status
FROM working_agreements
GROUP BY agreement_type;

-- ============================================
-- SECTION 8: SAMPLE DATA REVIEW
-- ============================================

-- #biz-sample-01: Sample health scores
SELECT
    'biz-sample-01' as check_id,
    'Sample health scores' as check_name,
    project_name,
    health_score,
    health_level,
    deployment_frequency_score as df_score,
    lead_time_score as lt_score,
    change_failure_rate_score as cfr_score,
    time_to_restore_score as ttr_score,
    'MANUAL' as status
FROM team_health_scores
ORDER BY calculated_at DESC
LIMIT 5;

-- #biz-sample-02: Sample violations
SELECT
    'biz-sample-02' as check_id,
    'Sample recent violations' as check_name,
    project_name,
    agreement_type,
    actual_value,
    threshold_value,
    violation_percent,
    detected_at,
    'MANUAL' as status
FROM agreement_violations
ORDER BY detected_at DESC
LIMIT 5;

-- #biz-sample-03: Sample initiatives
SELECT
    'biz-sample-03' as check_id,
    'Sample initiatives' as check_name,
    project_name,
    LEFT(title, 40) as title,
    investment_category,
    revenue_impact,
    total_story_points,
    value_score,
    'MANUAL' as status
FROM business_initiatives
ORDER BY value_score DESC
LIMIT 5;
