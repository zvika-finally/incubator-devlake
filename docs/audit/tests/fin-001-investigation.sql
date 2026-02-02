-- ============================================
-- FIN-001 ROOT CAUSE INVESTIGATION QUERIES
-- Investigate 1,329 resolved issues missing cost allocations
-- ============================================

-- ============================================
-- SECTION 1: SCOPE MISMATCH ANALYSIS
-- Understand if issues are in tracked projects
-- ============================================

-- #fin-001-scope-01: Breakdown of missing allocations by project
-- Shows which projects have unallocated issues
SELECT
    COALESCE(pm.project_name, 'NO_PROJECT_MAPPING') as project_name,
    COUNT(DISTINCT i.id) as total_resolved,
    COUNT(DISTINCT ca.id) as with_allocations,
    COUNT(DISTINCT i.id) - COUNT(DISTINCT ca.id) as missing_allocations
FROM issues i
LEFT JOIN board_issues bi ON bi.issue_id = i.id
LEFT JOIN project_mapping pm ON pm.`table` = 'boards' AND pm.row_id = bi.board_id
LEFT JOIN cost_allocations ca ON ca.issue_id = i.id
WHERE i.resolution_date IS NOT NULL
GROUP BY pm.project_name
ORDER BY missing_allocations DESC;

-- #fin-001-scope-02: Issues without any board assignment
SELECT
    'Issues without board assignment' as category,
    COUNT(DISTINCT i.id) as count
FROM issues i
WHERE i.resolution_date IS NOT NULL
AND i.id NOT IN (SELECT DISTINCT issue_id FROM board_issues WHERE issue_id IS NOT NULL);

-- #fin-001-scope-03: Issues in boards without project_mapping
SELECT
    'Issues in unmapped boards' as category,
    COUNT(DISTINCT i.id) as count
FROM issues i
JOIN board_issues bi ON bi.issue_id = i.id
WHERE i.resolution_date IS NOT NULL
AND bi.board_id NOT IN (SELECT row_id FROM project_mapping WHERE `table` = 'boards');

-- ============================================
-- SECTION 2: EFFORT DATA ANALYSIS
-- Understand which issues have zero effort
-- ============================================

-- #fin-001-effort-01: Issues with zero effort from all Jira sources
SELECT
    'Zero Jira effort data' as category,
    COUNT(*) as count
FROM issues i
WHERE i.resolution_date IS NOT NULL
AND (i.time_spent_minutes IS NULL OR i.time_spent_minutes = 0)
AND (i.original_estimate_minutes IS NULL OR i.original_estimate_minutes = 0)
AND (i.story_point IS NULL OR i.story_point = 0);

-- #fin-001-effort-02: Effort data distribution for missing allocations
SELECT
    CASE
        WHEN i.time_spent_minutes > 0 THEN 'Has time_spent'
        WHEN i.original_estimate_minutes > 0 THEN 'Has estimate only'
        WHEN i.story_point > 0 THEN 'Has story points only'
        ELSE 'No effort data'
    END as effort_status,
    COUNT(*) as count
FROM issues i
WHERE i.resolution_date IS NOT NULL
AND i.id NOT IN (SELECT issue_id FROM cost_allocations WHERE issue_id IS NOT NULL)
GROUP BY
    CASE
        WHEN i.time_spent_minutes > 0 THEN 'Has time_spent'
        WHEN i.original_estimate_minutes > 0 THEN 'Has estimate only'
        WHEN i.story_point > 0 THEN 'Has story points only'
        ELSE 'No effort data'
    END
ORDER BY count DESC;

-- ============================================
-- SECTION 3: TEMPORAL ANALYSIS
-- Understand when missing issues were resolved
-- ============================================

-- #fin-001-temporal-01: Resolution date distribution for missing allocations
SELECT
    DATE_FORMAT(i.resolution_date, '%Y-%m') as resolution_month,
    COUNT(*) as missing_count
FROM issues i
WHERE i.resolution_date IS NOT NULL
AND i.id NOT IN (SELECT issue_id FROM cost_allocations WHERE issue_id IS NOT NULL)
GROUP BY DATE_FORMAT(i.resolution_date, '%Y-%m')
ORDER BY resolution_month DESC
LIMIT 24;

-- #fin-001-temporal-02: Compare resolution dates vs allocation dates
SELECT
    'Earliest resolution (missing)' as metric,
    MIN(i.resolution_date) as value
FROM issues i
WHERE i.resolution_date IS NOT NULL
AND i.id NOT IN (SELECT issue_id FROM cost_allocations WHERE issue_id IS NOT NULL)
UNION ALL
SELECT
    'Latest resolution (missing)' as metric,
    MAX(i.resolution_date) as value
FROM issues i
WHERE i.resolution_date IS NOT NULL
AND i.id NOT IN (SELECT issue_id FROM cost_allocations WHERE issue_id IS NOT NULL)
UNION ALL
SELECT
    'Earliest cost allocation' as metric,
    MIN(calculated_at) as value
FROM cost_allocations
UNION ALL
SELECT
    'Latest cost allocation' as metric,
    MAX(calculated_at) as value
FROM cost_allocations;

-- ============================================
-- SECTION 4: ISSUE TYPE ANALYSIS
-- Understand which types are missing
-- ============================================

-- #fin-001-type-01: Issue types for missing allocations
SELECT
    i.type as issue_type,
    COUNT(*) as missing_count
FROM issues i
WHERE i.resolution_date IS NOT NULL
AND i.id NOT IN (SELECT issue_id FROM cost_allocations WHERE issue_id IS NOT NULL)
GROUP BY i.type
ORDER BY missing_count DESC;

-- ============================================
-- SECTION 5: SUMMARY COUNTS
-- ============================================

-- #fin-001-summary: Overall breakdown
SELECT
    (SELECT COUNT(*) FROM issues WHERE resolution_date IS NOT NULL) as total_resolved_issues,
    (SELECT COUNT(DISTINCT issue_id) FROM cost_allocations WHERE issue_id IS NOT NULL) as issues_with_allocations,
    (SELECT COUNT(*) FROM issues WHERE resolution_date IS NOT NULL) -
        (SELECT COUNT(DISTINCT issue_id) FROM cost_allocations WHERE issue_id IS NOT NULL) as missing_allocations,
    (SELECT COUNT(DISTINCT project_name) FROM project_mapping WHERE `table` = 'boards') as tracked_projects;
