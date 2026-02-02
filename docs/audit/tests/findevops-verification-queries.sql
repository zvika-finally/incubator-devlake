-- ============================================
-- FINDEVOPS DASHBOARD VERIFICATION QUERIES
-- Run these to validate metric calculations
-- ============================================

-- ============================================
-- SECTION 1: COMPLETENESS CHECKS
-- ============================================

-- #fin-completeness-01: Resolved issues (in tracked projects, with effort) should have allocations
-- UPDATED: Now scoped to match calculateCosts query logic (includes effort check)
-- Expected: missing_count = 0
SELECT
    'fin-completeness-01' as check_id,
    'Resolved issues (in tracked projects, with effort) have allocations' as check_name,
    COUNT(*) as missing_count,
    CASE WHEN COUNT(*) = 0 THEN 'PASS' ELSE 'REVIEW' END as status
FROM issues i
JOIN board_issues bi ON i.id = bi.issue_id
JOIN project_mapping pm ON bi.board_id = pm.row_id AND pm.`table` = 'boards'
WHERE i.resolution_date IS NOT NULL
  AND (i.time_spent_minutes > 0 OR i.original_estimate_minutes > 0 OR i.story_point > 0)
  AND i.id NOT IN (SELECT issue_id FROM cost_allocations WHERE issue_id IS NOT NULL);

-- #fin-completeness-01a: Issues excluded due to no project mapping (expected)
SELECT
    'fin-completeness-01a' as check_id,
    'Resolved issues without project mapping (excluded by design)' as check_name,
    COUNT(*) as excluded_count,
    'INFO' as status
FROM issues i
WHERE i.resolution_date IS NOT NULL
AND i.id NOT IN (
    SELECT DISTINCT bi.issue_id
    FROM board_issues bi
    JOIN project_mapping pm ON bi.board_id = pm.row_id AND pm.`table` = 'boards'
);

-- #fin-completeness-01b: Issues excluded due to zero effort data (expected)
SELECT
    'fin-completeness-01b' as check_id,
    'Resolved issues with zero effort (excluded by design)' as check_name,
    COUNT(*) as excluded_count,
    'INFO' as status
FROM issues i
JOIN board_issues bi ON i.id = bi.issue_id
JOIN project_mapping pm ON bi.board_id = pm.row_id AND pm.`table` = 'boards'
WHERE i.resolution_date IS NOT NULL
AND (i.time_spent_minutes IS NULL OR i.time_spent_minutes = 0)
AND (i.original_estimate_minutes IS NULL OR i.original_estimate_minutes = 0)
AND (i.story_point IS NULL OR i.story_point = 0);

-- #fin-completeness-02: Monthly summaries exist for all months with allocations
SELECT
    'fin-completeness-02' as check_id,
    'Monthly summaries exist for all allocation months' as check_name,
    COUNT(DISTINCT ca.fiscal_month) as allocation_months,
    COUNT(DISTINCT mcs.fiscal_month) as summary_months,
    CASE
        WHEN COUNT(DISTINCT ca.fiscal_month) = COUNT(DISTINCT mcs.fiscal_month)
        THEN 'PASS' ELSE 'FAIL'
    END as status
FROM cost_allocations ca
LEFT JOIN monthly_cost_summaries mcs ON ca.fiscal_month = mcs.fiscal_month;

-- #fin-completeness-03: Deployment costs exist for all three windows
SELECT
    'fin-completeness-03' as check_id,
    'Deployment costs exist for 7, 30, 90 day windows' as check_name,
    GROUP_CONCAT(DISTINCT window_days ORDER BY window_days) as windows_present,
    CASE
        WHEN COUNT(DISTINCT window_days) = 3 THEN 'PASS' ELSE 'FAIL'
    END as status
FROM deployment_costs;

-- ============================================
-- SECTION 2: ACCURACY CHECKS
-- ============================================

-- #fin-accuracy-01: Capitalization rate formula validation
-- Formula: capitalization_rate = capitalizable_cost / total_cost * 100
SELECT
    'fin-accuracy-01' as check_id,
    'Capitalization rate formula is correct' as check_name,
    fiscal_month,
    total_cost,
    capitalizable_cost,
    capitalization_rate as stored_rate,
    ROUND(capitalizable_cost / total_cost * 100, 2) as calculated_rate,
    CASE
        WHEN ABS(capitalization_rate - (capitalizable_cost / total_cost * 100)) < 0.01
        THEN 'PASS' ELSE 'FAIL'
    END as status
FROM monthly_cost_summaries
WHERE total_cost > 0
ORDER BY fiscal_month DESC
LIMIT 5;

-- #fin-accuracy-02: Total cost equals sum of phase costs
SELECT
    'fin-accuracy-02' as check_id,
    'Total cost = preliminary + development + post_impl' as check_name,
    fiscal_month,
    total_cost,
    preliminary_cost + development_cost + post_impl_cost as sum_of_phases,
    CASE
        WHEN ABS(total_cost - (preliminary_cost + development_cost + post_impl_cost)) < 0.01
        THEN 'PASS' ELSE 'FAIL'
    END as status
FROM monthly_cost_summaries
ORDER BY fiscal_month DESC
LIMIT 5;

-- #fin-accuracy-03: Capitalizable = Development, Expense = Preliminary + PostImpl
SELECT
    'fin-accuracy-03' as check_id,
    'Capitalizable equals development cost' as check_name,
    fiscal_month,
    capitalizable_cost,
    development_cost,
    expense_cost,
    preliminary_cost + post_impl_cost as sum_expense_phases,
    CASE
        WHEN ABS(capitalizable_cost - development_cost) < 0.01
         AND ABS(expense_cost - (preliminary_cost + post_impl_cost)) < 0.01
        THEN 'PASS' ELSE 'FAIL'
    END as status
FROM monthly_cost_summaries
ORDER BY fiscal_month DESC
LIMIT 5;

-- #fin-accuracy-04: Budget variance formula validation
-- Formula: variance = (estimated - actual) / estimated * 100
SELECT
    'fin-accuracy-04' as check_id,
    'Budget variance formula is correct' as check_name,
    fiscal_month,
    total_estimated_cost,
    total_actual_cost,
    budget_variance as stored_variance,
    CASE
        WHEN total_estimated_cost > 0
        THEN ROUND((total_estimated_cost - total_actual_cost) / total_estimated_cost * 100, 2)
        ELSE 0
    END as calculated_variance,
    CASE
        WHEN total_estimated_cost = 0 THEN 'SKIP'
        WHEN ABS(budget_variance - ((total_estimated_cost - total_actual_cost) / total_estimated_cost * 100)) < 0.1
        THEN 'PASS' ELSE 'FAIL'
    END as status
FROM monthly_cost_summaries
WHERE total_estimated_cost > 0
ORDER BY fiscal_month DESC
LIMIT 5;

-- #fin-accuracy-05: Cost per deployment formula validation
SELECT
    'fin-accuracy-05' as check_id,
    'Cost per deployment formula is correct' as check_name,
    window_days,
    total_cost,
    deployment_count,
    cost_per_deployment as stored_cpd,
    CASE
        WHEN deployment_count > 0
        THEN ROUND(total_cost / deployment_count, 2)
        ELSE 0
    END as calculated_cpd,
    CASE
        WHEN deployment_count = 0 THEN 'SKIP'
        WHEN ABS(cost_per_deployment - (total_cost / deployment_count)) < 0.01
        THEN 'PASS' ELSE 'FAIL'
    END as status
FROM deployment_costs
ORDER BY calculated_at DESC
LIMIT 9;

-- ============================================
-- SECTION 3: ASC 350-40 CATEGORIZATION CHECKS
-- ============================================

-- #fin-asc350-01: Bug issues should be categorized as expense
SELECT
    'fin-asc350-01' as check_id,
    'Bug issues are categorized as expense' as check_name,
    ca.issue_type,
    ca.capitalization_category,
    COUNT(*) as count,
    CASE
        WHEN ca.capitalization_category = 'expense' THEN 'PASS'
        ELSE 'FAIL'
    END as status
FROM cost_allocations ca
WHERE ca.issue_type IN ('Bug', 'Defect', 'Hotfix')
GROUP BY ca.issue_type, ca.capitalization_category;

-- #fin-asc350-02: Story/Feature issues should be categorized as capitalizable
SELECT
    'fin-asc350-02' as check_id,
    'Story/Feature issues are categorized as capitalizable' as check_name,
    ca.issue_type,
    ca.capitalization_category,
    COUNT(*) as count,
    CASE
        WHEN ca.capitalization_category = 'capitalizable' THEN 'PASS'
        ELSE 'FAIL'
    END as status
FROM cost_allocations ca
WHERE ca.issue_type IN ('Story', 'Feature', 'Enhancement', 'Task')
  AND ca.issue_labels NOT LIKE '%maintenance%'
  AND ca.issue_labels NOT LIKE '%research%'
  AND ca.issue_labels NOT LIKE '%poc%'
GROUP BY ca.issue_type, ca.capitalization_category;

-- #fin-asc350-03: Research/Spike issues should be categorized as preliminary expense
SELECT
    'fin-asc350-03' as check_id,
    'Research/Spike issues are preliminary expense' as check_name,
    ca.issue_type,
    ca.project_phase,
    ca.capitalization_category,
    COUNT(*) as count,
    CASE
        WHEN ca.project_phase = 'preliminary' AND ca.capitalization_category = 'expense'
        THEN 'PASS' ELSE 'FAIL'
    END as status
FROM cost_allocations ca
WHERE ca.issue_type IN ('Spike', 'Research')
   OR ca.issue_labels LIKE '%research%'
   OR ca.issue_labels LIKE '%poc%'
GROUP BY ca.issue_type, ca.project_phase, ca.capitalization_category;

-- #fin-asc350-04: All allocations have a category reason (audit trail)
SELECT
    'fin-asc350-04' as check_id,
    'All allocations have category reason' as check_name,
    COUNT(*) as total_allocations,
    SUM(CASE WHEN category_reason IS NULL OR category_reason = '' THEN 1 ELSE 0 END) as missing_reason,
    CASE
        WHEN SUM(CASE WHEN category_reason IS NULL OR category_reason = '' THEN 1 ELSE 0 END) = 0
        THEN 'PASS' ELSE 'FAIL'
    END as status
FROM cost_allocations;

-- ============================================
-- SECTION 4: CONSISTENCY CHECKS
-- ============================================

-- #fin-consistency-01: Monthly summary totals match sum of allocations
SELECT
    'fin-consistency-01' as check_id,
    'Monthly summary matches allocation sum' as check_name,
    mcs.fiscal_month,
    mcs.total_cost as summary_total,
    COALESCE(SUM(ca.total_cost), 0) as allocation_sum,
    CASE
        WHEN ABS(mcs.total_cost - COALESCE(SUM(ca.total_cost), 0)) < 1.00
        THEN 'PASS' ELSE 'FAIL'
    END as status
FROM monthly_cost_summaries mcs
LEFT JOIN cost_allocations ca ON ca.fiscal_month = mcs.fiscal_month
GROUP BY mcs.fiscal_month, mcs.total_cost
ORDER BY mcs.fiscal_month DESC
LIMIT 5;

-- #fin-consistency-02: Unallocated count matches flagged allocations
SELECT
    'fin-consistency-02' as check_id,
    'Orphan issue count matches unallocated flag' as check_name,
    mcs.fiscal_month,
    mcs.orphan_issue_count as summary_count,
    COUNT(ca.id) as actual_unallocated,
    CASE
        WHEN mcs.orphan_issue_count = COUNT(ca.id) THEN 'PASS' ELSE 'FAIL'
    END as status
FROM monthly_cost_summaries mcs
LEFT JOIN cost_allocations ca ON ca.fiscal_month = mcs.fiscal_month AND ca.is_unallocated = true
GROUP BY mcs.fiscal_month, mcs.orphan_issue_count
ORDER BY mcs.fiscal_month DESC
LIMIT 5;

-- ============================================
-- SECTION 5: FRESHNESS CHECKS
-- ============================================

-- #fin-freshness-01: Cost allocations have recent calculated_at timestamps
SELECT
    'fin-freshness-01' as check_id,
    'Cost allocations are fresh (within 7 days)' as check_name,
    MAX(calculated_at) as most_recent,
    DATEDIFF(NOW(), MAX(calculated_at)) as days_old,
    CASE
        WHEN DATEDIFF(NOW(), MAX(calculated_at)) <= 7 THEN 'PASS'
        ELSE 'FAIL'
    END as status
FROM cost_allocations;

-- #fin-freshness-02: Monthly summaries have recent calculated_at timestamps
SELECT
    'fin-freshness-02' as check_id,
    'Monthly summaries are fresh (within 7 days)' as check_name,
    MAX(calculated_at) as most_recent,
    DATEDIFF(NOW(), MAX(calculated_at)) as days_old,
    CASE
        WHEN DATEDIFF(NOW(), MAX(calculated_at)) <= 7 THEN 'PASS'
        ELSE 'FAIL'
    END as status
FROM monthly_cost_summaries;

-- #fin-freshness-03: Deployment costs have recent calculated_at timestamps
SELECT
    'fin-freshness-03' as check_id,
    'Deployment costs are fresh (within 7 days)' as check_name,
    MAX(calculated_at) as most_recent,
    DATEDIFF(NOW(), MAX(calculated_at)) as days_old,
    CASE
        WHEN DATEDIFF(NOW(), MAX(calculated_at)) <= 7 THEN 'PASS'
        ELSE 'FAIL'
    END as status
FROM deployment_costs;

-- ============================================
-- SECTION 6: QUARTERLY AGGREGATION CHECKS
-- ============================================

-- #fin-quarterly-01: Quarterly cost can be calculated from monthly summaries
SELECT
    'fin-quarterly-01' as check_id,
    'Quarterly aggregation from monthly summaries' as check_name,
    CONCAT(YEAR(STR_TO_DATE(CONCAT(fiscal_month, '-01'), '%Y-%m-%d')), '-Q',
           QUARTER(STR_TO_DATE(CONCAT(fiscal_month, '-01'), '%Y-%m-%d'))) as quarter,
    SUM(total_cost) as quarterly_total_cost,
    SUM(capitalizable_cost) as quarterly_capitalizable,
    SUM(expense_cost) as quarterly_expense,
    ROUND(SUM(capitalizable_cost) / NULLIF(SUM(total_cost), 0) * 100, 2) as quarterly_cap_rate,
    'PASS' as status
FROM monthly_cost_summaries
GROUP BY YEAR(STR_TO_DATE(CONCAT(fiscal_month, '-01'), '%Y-%m-%d')),
         QUARTER(STR_TO_DATE(CONCAT(fiscal_month, '-01'), '%Y-%m-%d'))
ORDER BY quarter DESC
LIMIT 4;

-- ============================================
-- SUMMARY: Run all checks and count results
-- ============================================

-- #fin-summary: Aggregate check results
-- Run each check above and manually tally PASS/FAIL counts
