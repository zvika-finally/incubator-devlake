-- ============================================
-- AI DETECTION DASHBOARD VERIFICATION QUERIES
-- Run these to validate metric calculations
-- ============================================

-- ============================================
-- SECTION 1: COMPLETENESS CHECKS
-- ============================================

-- #ai-completeness-01: All merged PRs should have AI signals
SELECT
    'ai-completeness-01' as check_id,
    'All merged PRs have AI signals' as check_name,
    (SELECT COUNT(*) FROM pull_requests WHERE merged_date IS NOT NULL) as total_merged_prs,
    (SELECT COUNT(DISTINCT pull_request_id) FROM ai_usage_signals) as prs_with_signals,
    (SELECT COUNT(*) FROM pull_requests WHERE merged_date IS NOT NULL) -
        (SELECT COUNT(DISTINCT pull_request_id) FROM ai_usage_signals) as missing,
    CASE
        WHEN (SELECT COUNT(*) FROM pull_requests WHERE merged_date IS NOT NULL)
             = (SELECT COUNT(DISTINCT pull_request_id) FROM ai_usage_signals)
        THEN 'PASS' ELSE 'REVIEW'
    END as status;

-- #ai-completeness-02: PRs with signals should have churn metrics (if old enough)
SELECT
    'ai-completeness-02' as check_id,
    'Merged PRs (>30 days old) have churn metrics' as check_name,
    (SELECT COUNT(*) FROM pull_requests
     WHERE merged_date IS NOT NULL
     AND merged_date < DATE_SUB(NOW(), INTERVAL 30 DAY)) as eligible_prs,
    (SELECT COUNT(DISTINCT pull_request_id) FROM ai_churn_metrics) as prs_with_churn,
    'INFO' as status;

-- #ai-completeness-03: Developer baselines exist for active developers
SELECT
    'ai-completeness-03' as check_id,
    'Developers with merged PRs have baselines' as check_name,
    (SELECT COUNT(DISTINCT author_id) FROM pull_requests
     WHERE merged_date IS NOT NULL) as active_developers,
    (SELECT COUNT(DISTINCT developer_id) FROM developer_baselines) as developers_with_baselines,
    CASE
        WHEN (SELECT COUNT(DISTINCT author_id) FROM pull_requests WHERE merged_date IS NOT NULL)
             <= (SELECT COUNT(DISTINCT developer_id) FROM developer_baselines)
        THEN 'PASS' ELSE 'REVIEW'
    END as status;

-- #ai-completeness-04: Project churn summaries exist for all projects with AI signals
SELECT
    'ai-completeness-04' as check_id,
    'Projects have churn summaries' as check_name,
    (SELECT COUNT(DISTINCT project_name) FROM ai_usage_signals) as projects_with_signals,
    (SELECT COUNT(DISTINCT project_name) FROM project_churn_summaries) as projects_with_summaries,
    CASE
        WHEN (SELECT COUNT(DISTINCT project_name) FROM ai_usage_signals)
             <= (SELECT COUNT(DISTINCT project_name) FROM project_churn_summaries)
        THEN 'PASS' ELSE 'REVIEW'
    END as status;

-- ============================================
-- SECTION 2: ACCURACY CHECKS
-- ============================================

-- #ai-accuracy-01: Confidence scores are within valid range (0-100)
SELECT
    'ai-accuracy-01' as check_id,
    'AI confidence scores are 0-100' as check_name,
    MIN(ai_confidence_score) as min_score,
    MAX(ai_confidence_score) as max_score,
    CASE
        WHEN MIN(ai_confidence_score) >= 0 AND MAX(ai_confidence_score) <= 100
        THEN 'PASS' ELSE 'FAIL'
    END as status
FROM ai_usage_signals;

-- #ai-accuracy-02: Explicit detection flag consistency
-- If explicit_tool_detected = true, explicit_signal_score should be > 0
SELECT
    'ai-accuracy-02' as check_id,
    'Explicit detection flag matches score' as check_name,
    COUNT(*) as inconsistent_count,
    CASE WHEN COUNT(*) = 0 THEN 'PASS' ELSE 'FAIL' END as status
FROM ai_usage_signals
WHERE explicit_tool_detected = true AND (explicit_signal_score IS NULL OR explicit_signal_score = 0);

-- #ai-accuracy-03: Churn ratio formula validation (sample check)
-- churn_ratio = churn_within_X_days / initial_additions
SELECT
    'ai-accuracy-03' as check_id,
    'Churn ratio formula is correct' as check_name,
    pull_request_id,
    initial_additions,
    churn_within_30_days,
    churn_ratio_30_days as stored_ratio,
    CASE
        WHEN initial_additions > 0
        THEN ROUND(churn_within_30_days / initial_additions, 2)
        ELSE 0
    END as calculated_ratio,
    CASE
        WHEN initial_additions = 0 THEN 'SKIP'
        WHEN ABS(churn_ratio_30_days - (churn_within_30_days / initial_additions)) < 0.01
        THEN 'PASS' ELSE 'FAIL'
    END as status
FROM ai_churn_metrics
WHERE initial_additions > 0
ORDER BY merged_at DESC
LIMIT 5;

-- #ai-accuracy-04: Component scores sum should not exceed 100
SELECT
    'ai-accuracy-04' as check_id,
    'Component scores capped at 100' as check_name,
    COUNT(*) as violations,
    CASE WHEN COUNT(*) = 0 THEN 'PASS' ELSE 'FAIL' END as status
FROM ai_usage_signals
WHERE ai_confidence_score > 100;

-- #ai-accuracy-05: Velocity multiplier is reasonable (0.1 to 10x range)
SELECT
    'ai-accuracy-05' as check_id,
    'Velocity multiplier in reasonable range' as check_name,
    MIN(velocity_multiplier) as min_velocity,
    MAX(velocity_multiplier) as max_velocity,
    CASE
        WHEN MIN(velocity_multiplier) >= 0.1 AND MAX(velocity_multiplier) <= 10
        THEN 'PASS' ELSE 'REVIEW'
    END as status
FROM ai_usage_signals
WHERE velocity_multiplier IS NOT NULL AND velocity_multiplier > 0;

-- ============================================
-- SECTION 3: CONSISTENCY CHECKS
-- ============================================

-- #ai-consistency-01: High confidence PRs should have is_ai_assisted = true in churn
SELECT
    'ai-consistency-01' as check_id,
    'High confidence PRs marked as AI-assisted in churn' as check_name,
    COUNT(*) as mismatch_count,
    CASE WHEN COUNT(*) = 0 THEN 'PASS' ELSE 'REVIEW' END as status
FROM ai_usage_signals s
JOIN ai_churn_metrics c ON s.pull_request_id = c.pull_request_id
WHERE s.ai_confidence_score >= 70 AND c.is_ai_assisted = false;

-- #ai-consistency-02: Low confidence PRs should have is_ai_assisted = false in churn
SELECT
    'ai-consistency-02' as check_id,
    'Low confidence PRs marked as non-AI in churn' as check_name,
    COUNT(*) as mismatch_count,
    CASE WHEN COUNT(*) = 0 THEN 'PASS' ELSE 'REVIEW' END as status
FROM ai_usage_signals s
JOIN ai_churn_metrics c ON s.pull_request_id = c.pull_request_id
WHERE s.ai_confidence_score < 70 AND c.is_ai_assisted = true;

-- #ai-consistency-03: Project names match between tables
SELECT
    'ai-consistency-03' as check_id,
    'Project names consistent across tables' as check_name,
    (SELECT COUNT(DISTINCT project_name) FROM ai_usage_signals) as signal_projects,
    (SELECT COUNT(DISTINCT project_name) FROM ai_churn_metrics) as churn_projects,
    (SELECT COUNT(DISTINCT project_name) FROM project_churn_summaries) as summary_projects,
    'INFO' as status;

-- ============================================
-- SECTION 4: CHURN ANALYSIS CHECKS
-- ============================================

-- #ai-churn-01: Project churn summaries have valid counts
SELECT
    'ai-churn-01' as check_id,
    'Project churn summaries have valid data' as check_name,
    project_name,
    ai_pr_count,
    non_ai_pr_count,
    ai_avg_churn_ratio_30,
    non_ai_avg_churn_ratio_30,
    CASE
        WHEN ai_pr_count >= 0 AND non_ai_pr_count >= 0 THEN 'PASS'
        ELSE 'FAIL'
    END as status
FROM project_churn_summaries
LIMIT 10;

-- #ai-churn-02: Churn ratios are non-negative
SELECT
    'ai-churn-02' as check_id,
    'Churn ratios are non-negative' as check_name,
    COUNT(*) as negative_count,
    CASE WHEN COUNT(*) = 0 THEN 'PASS' ELSE 'FAIL' END as status
FROM ai_churn_metrics
WHERE churn_ratio_7_days < 0 OR churn_ratio_30_days < 0;

-- #ai-churn-03: Churn difference calculation is correct
SELECT
    'ai-churn-03' as check_id,
    'Churn difference percent calculated correctly' as check_name,
    project_name,
    ai_avg_churn_ratio_30,
    non_ai_avg_churn_ratio_30,
    churn_difference_percent as stored,
    CASE
        WHEN non_ai_avg_churn_ratio_30 > 0
        THEN ROUND((ai_avg_churn_ratio_30 - non_ai_avg_churn_ratio_30) / non_ai_avg_churn_ratio_30 * 100, 2)
        ELSE 0
    END as calculated,
    CASE
        WHEN non_ai_avg_churn_ratio_30 = 0 THEN 'SKIP'
        WHEN ABS(churn_difference_percent - ((ai_avg_churn_ratio_30 - non_ai_avg_churn_ratio_30) / non_ai_avg_churn_ratio_30 * 100)) < 1
        THEN 'PASS' ELSE 'REVIEW'
    END as status
FROM project_churn_summaries
LIMIT 5;

-- ============================================
-- SECTION 5: FRESHNESS CHECKS
-- ============================================

-- #ai-freshness-01: AI signals are recent
SELECT
    'ai-freshness-01' as check_id,
    'AI signals are fresh (within 7 days)' as check_name,
    MAX(detected_at) as most_recent,
    DATEDIFF(NOW(), MAX(detected_at)) as days_old,
    CASE
        WHEN DATEDIFF(NOW(), MAX(detected_at)) <= 7 THEN 'PASS'
        ELSE 'FAIL'
    END as status
FROM ai_usage_signals;

-- #ai-freshness-02: AI impact metrics are recent
SELECT
    'ai-freshness-02' as check_id,
    'AI impact metrics are fresh (within 7 days)' as check_name,
    MAX(calculated_at) as most_recent,
    DATEDIFF(NOW(), MAX(calculated_at)) as days_old,
    CASE
        WHEN DATEDIFF(NOW(), MAX(calculated_at)) <= 7 THEN 'PASS'
        ELSE 'FAIL'
    END as status
FROM ai_impact_metrics;

-- ============================================
-- SECTION 6: DISTRIBUTION CHECKS
-- ============================================

-- #ai-distribution-01: Confidence score distribution
SELECT
    'ai-distribution-01' as check_id,
    'Confidence score distribution' as check_name,
    CASE
        WHEN ai_confidence_score >= 70 THEN 'High (>=70)'
        WHEN ai_confidence_score >= 40 THEN 'Medium (40-69)'
        ELSE 'Low (<40)'
    END as confidence_level,
    COUNT(*) as count,
    ROUND(COUNT(*) * 100.0 / (SELECT COUNT(*) FROM ai_usage_signals), 2) as percent,
    'INFO' as status
FROM ai_usage_signals
GROUP BY CASE
    WHEN ai_confidence_score >= 70 THEN 'High (>=70)'
    WHEN ai_confidence_score >= 40 THEN 'Medium (40-69)'
    ELSE 'Low (<40)'
END;

-- #ai-distribution-02: Explicit vs behavioral detection
SELECT
    'ai-distribution-02' as check_id,
    'Explicit vs behavioral detection ratio' as check_name,
    SUM(CASE WHEN explicit_tool_detected = true THEN 1 ELSE 0 END) as explicit_detected,
    SUM(CASE WHEN explicit_tool_detected = false AND ai_confidence_score >= 70 THEN 1 ELSE 0 END) as behavioral_high,
    SUM(CASE WHEN explicit_tool_detected = false AND ai_confidence_score < 70 THEN 1 ELSE 0 END) as behavioral_low,
    COUNT(*) as total,
    'INFO' as status
FROM ai_usage_signals;

-- #ai-distribution-03: Detected tools breakdown
SELECT
    'ai-distribution-03' as check_id,
    'Detected AI tools breakdown' as check_name,
    COALESCE(NULLIF(detected_tool, ''), 'None/Unknown') as tool,
    COUNT(*) as count,
    'INFO' as status
FROM ai_usage_signals
GROUP BY detected_tool
ORDER BY count DESC
LIMIT 10;

-- ============================================
-- SECTION 7: SAMPLE DATA REVIEW
-- ============================================

-- #ai-sample-01: Sample explicit detections for manual review
SELECT
    'ai-sample-01' as check_id,
    'Sample explicit detections for manual review' as check_name,
    s.pull_request_id,
    p.title,
    s.detected_tool,
    s.explicit_tools,
    s.ai_confidence_score,
    'MANUAL' as status
FROM ai_usage_signals s
JOIN pull_requests p ON s.pull_request_id = p.id
WHERE s.explicit_tool_detected = true
ORDER BY s.detected_at DESC
LIMIT 5;

-- #ai-sample-02: Sample high-confidence behavioral detections
SELECT
    'ai-sample-02' as check_id,
    'Sample high-confidence behavioral (no explicit marker)' as check_name,
    s.pull_request_id,
    p.title,
    s.ai_confidence_score,
    s.rapid_commit_score,
    s.pr_size_score,
    s.lines_per_minute_score,
    'MANUAL' as status
FROM ai_usage_signals s
JOIN pull_requests p ON s.pull_request_id = p.id
WHERE s.explicit_tool_detected = false
AND s.ai_confidence_score >= 70
ORDER BY s.ai_confidence_score DESC
LIMIT 5;
