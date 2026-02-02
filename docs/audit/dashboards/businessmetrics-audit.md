# Business Metrics Dashboard - Per-Metric Audit Checklist

**Dashboard:** Team Health & Working Agreements
**UID:** business-metrics-dashboard
**Audit Date:** 2026-02-02
**Total Metrics:** 20 panels

---

## Section 1: Team Health Overview (8 metrics)

### 1.1 Overall Health Score
**Panel:** Gauge showing team health score (0-100)

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Sum of 4 DORA metric scores (each 0-25) |
| ☐ Formula documented | | `deployment_frequency_score + lead_time_score + change_failure_rate_score + time_to_restore_score` |
| ☐ Edge cases handled | | Missing DORA data results in 0 for that metric |
| **Data Lineage** | | |
| ☐ Source identified | | project_pr_metrics (DORA plugin) |
| ☐ Plugin task | | calculateHealthScore |
| ☐ Output table | | team_health_scores |
| **Trust Validation** | | |
| ☐ Completeness | | All tracked projects have health scores |
| ☐ Accuracy | | Score = sum of 4 DORA scores |
| ☐ Freshness | | calculated_at within 7 days |
| **Verification Query** | | `#biz-accuracy-01` |

---

### 1.2 Health Level Distribution
**Panel:** Pie chart showing excellent/good/fair/poor distribution

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Distribution of projects by health level |
| ☐ Formula documented | | excellent(80+), good(60-79), fair(40-59), poor(<40) |
| ☐ Edge cases handled | | Empty projects excluded |
| **Data Lineage** | | |
| ☐ Source identified | | team_health_scores.health_level |
| ☐ Plugin task | | calculateHealthScore |
| **Verification Query** | | `#biz-distribution-01` |

---

### 1.3 Deployment Frequency Score
**Panel:** Stat showing deployment frequency component (0-25)

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Score based on deployment frequency tier |
| ☐ Formula documented | | Elite(25): Multiple/day, High(20): Daily, Medium(15): Weekly, Low(10): Monthly, Poor(5): <Monthly |
| ☐ Edge cases handled | | No deployments = 0 |
| **Data Lineage** | | |
| ☐ Source identified | | project_pr_metrics.deployment_frequency |
| ☐ Output table | | team_health_scores.deployment_frequency_score |
| **Verification Query** | | `#biz-accuracy-02` |

---

### 1.4 Lead Time Score
**Panel:** Stat showing lead time component (0-25)

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Score based on lead time tier |
| ☐ Formula documented | | Elite(25): <1hr, High(20): <1day, Medium(15): <1week, Low(10): <1month, Poor(5): >1month |
| ☐ Edge cases handled | | No PRs = 0 |
| **Data Lineage** | | |
| ☐ Source identified | | project_pr_metrics.pr_cycle_time |
| ☐ Output table | | team_health_scores.lead_time_score |
| **Verification Query** | | `#biz-accuracy-02` |

---

### 1.5 Change Failure Rate Score
**Panel:** Stat showing CFR component (0-25)

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Score based on change failure rate tier |
| ☐ Formula documented | | Elite(25): <5%, High(20): <10%, Medium(15): <15%, Low(10): <25%, Poor(5): >25% |
| ☐ Edge cases handled | | No failures = 25 (best score) |
| **Data Lineage** | | |
| ☐ Source identified | | project_pr_metrics.change_failure_rate |
| ☐ Output table | | team_health_scores.change_failure_rate_score |
| **Verification Query** | | `#biz-accuracy-02` |

---

### 1.6 Time to Restore Score
**Panel:** Stat showing TTR component (0-25)

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Score based on time to restore tier |
| ☐ Formula documented | | Elite(25): <1hr, High(20): <1day, Medium(15): <1week, Low(10): <1month, Poor(5): >1month |
| ☐ Edge cases handled | | No incidents = 0 (no data to score) |
| **Data Lineage** | | |
| ☐ Source identified | | project_pr_metrics.time_to_restore |
| ☐ Output table | | team_health_scores.time_to_restore_score |
| **Verification Query** | | `#biz-accuracy-02` |

---

### 1.7 Health Level Badge
**Panel:** Text showing health_level (excellent/good/fair/poor)

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Health level classification based on score |
| ☐ Formula documented | | Derived from health_score thresholds |
| **Data Lineage** | | |
| ☐ Source identified | | team_health_scores.health_level |
| **Verification Query** | | `#biz-accuracy-03` |

---

### 1.8 Health Trend Over Time
**Panel:** Time series showing health score trend

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Historical health scores over time |
| ☐ Time aggregation | | Weekly/Monthly grouping |
| ☐ Edge cases handled | | Gaps in data (holidays, etc.) |
| **Data Lineage** | | |
| ☐ Source identified | | team_health_scores (historical records) |
| **Freshness Query** | | `#biz-freshness-01` |

---

## Section 2: Working Agreement Compliance (7 metrics)

### 2.1 Overall Compliance Rate
**Panel:** Gauge showing compliance percentage

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Percentage of checks that passed |
| ☐ Formula documented | | `(total_checks - violations_count) / total_checks * 100` |
| ☐ Edge cases handled | | 0 checks = 100% (no violations possible) |
| **Data Lineage** | | |
| ☐ Source identified | | agreement_compliance_summaries |
| ☐ Plugin task | | checkAgreements |
| ☐ Output table | | agreement_compliance_summaries.compliance_rate |
| **Trust Validation** | | |
| ☐ Completeness | | All projects with agreements have summaries |
| ☐ Accuracy | | Formula matches stored rate |
| ☐ Freshness | | calculated_at within 7 days |
| **Verification Query** | | `#biz-accuracy-04` |

---

### 2.2 Active Violations Count
**Panel:** Stat showing current unresolved violations

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Count of violations without resolved_at |
| ☐ Formula documented | | `COUNT(*) WHERE resolved_at IS NULL` |
| ☐ Edge cases handled | | Future resolved_at is invalid |
| **Data Lineage** | | |
| ☐ Source identified | | agreement_violations |
| ☐ Output table | | agreement_violations.resolved_at |
| **Verification Query** | | `#biz-consistency-02` |

---

### 2.3 Violations by Agreement Type
**Panel:** Bar chart showing violations per agreement type

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Breakdown of violations by type |
| ☐ Grouping | | pr_size, review_time, merge_time, etc. |
| **Data Lineage** | | |
| ☐ Source identified | | agreement_violations.agreement_type |
| **Verification Query** | | `#biz-distribution-03` |

---

### 2.4 Agreement Types Enabled
**Panel:** Table showing configured agreement types

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | List of active working agreements |
| ☐ Fields shown | | agreement_type, threshold_value, enabled |
| **Data Lineage** | | |
| ☐ Source identified | | working_agreements |
| **Verification Query** | | `#biz-distribution-03` |

---

### 2.5 Violation Count Consistency
**Panel:** Hidden/diagnostic - validates summary vs detail counts

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Summary violations_count matches actual violations |
| ☐ Formula documented | | `summary.violations_count = COUNT(violations) in period` |
| **Verification Query** | | `#biz-accuracy-05` |

---

### 2.6 Compliance Trend Over Time
**Panel:** Time series showing compliance rate trend

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Historical compliance rates |
| ☐ Time aggregation | | Weekly/Monthly |
| **Data Lineage** | | |
| ☐ Source identified | | agreement_compliance_summaries (historical) |
| **Freshness Query** | | `#biz-freshness-02` |

---

### 2.7 Recent Violations Table
**Panel:** Table showing recent violation details

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Detail view for investigation |
| ☐ Fields shown | | project, type, actual_value, threshold, detected_at |
| **Data Lineage** | | |
| ☐ Source identified | | agreement_violations |
| **Verification Query** | | `#biz-sample-02` |

---

## Section 3: Business Initiatives (5 metrics)

### 3.1 Total Initiatives
**Panel:** Stat showing count of business initiatives

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Count of extracted Epics as initiatives |
| ☐ Formula documented | | `COUNT(*) FROM business_initiatives` |
| **Data Lineage** | | |
| ☐ Source identified | | issues (type='Epic') |
| ☐ Plugin task | | extractBusinessGoals |
| ☐ Output table | | business_initiatives |
| **Trust Validation** | | |
| ☐ Completeness | | All Epics extracted as initiatives |
| **Verification Query** | | `#biz-completeness-01` |

---

### 3.2 Investment Category Distribution
**Panel:** Pie chart showing grow/run/transform breakdown

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Distribution by investment_category |
| ☐ Categories | | grow, run, transform, uncategorized |
| **Data Lineage** | | |
| ☐ Source identified | | business_initiatives.investment_category |
| **Verification Query** | | `#biz-distribution-02` |

---

### 3.3 Value Score Distribution
**Panel:** Histogram or bar chart of value scores

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Distribution of initiative value scores |
| ☐ Range | | 0-100 |
| **Data Lineage** | | |
| ☐ Source identified | | business_initiatives.value_score |
| ☐ Plugin task | | calculateBusinessValue |
| **Verification Query** | | `#biz-value-01` |

---

### 3.4 Story Points Aggregation
**Panel:** Stat showing total story points per initiative

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Sum of story points from linked issues |
| ☐ Formula documented | | `SUM(work_allocations.story_points)` per initiative |
| **Data Lineage** | | |
| ☐ Source identified | | issues.story_point → work_allocations |
| ☐ Plugin task | | calculateAlignment |
| **Verification Query** | | `#biz-value-02` |

---

### 3.5 Work Allocation Completeness
**Panel:** Hidden/diagnostic - validates allocation percentages

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Allocations per initiative sum to ~100% |
| ☐ Formula documented | | `SUM(allocation_percent) ≈ 100` per initiative |
| **Verification Query** | | `#biz-consistency-01` |

---

## Summary Checklist

| Section | Metrics | Logic | Data Trust | Testing | Status |
|---------|---------|-------|------------|---------|--------|
| Team Health Overview | 8 | ☐ 8/8 | ☐ 8/8 | ☐ 8/8 | ☐ Pending |
| Working Agreement Compliance | 7 | ☐ 7/7 | ☐ 7/7 | ☐ 7/7 | ☐ Pending |
| Business Initiatives | 5 | ☐ 5/5 | ☐ 5/5 | ☐ 5/5 | ☐ Pending |
| **TOTAL** | **20** | **☐ 20/20** | **☐ 20/20** | **☐ 20/20** | **☐ Pending** |

**Verification Date:** 2026-02-02
**Overall Status:** ⏳ AWAITING VERIFICATION

---

## Verification Query Mapping

| Check ID | Section | Validates |
|----------|---------|-----------|
| biz-completeness-01 | Initiatives | All Epics extracted |
| biz-completeness-02 | Health | Health scores exist |
| biz-completeness-03 | Compliance | Compliance summaries exist |
| biz-accuracy-01 | Health | Health score = sum of DORA scores |
| biz-accuracy-02 | Health | DORA scores in 0-25 range |
| biz-accuracy-03 | Health | Health level matches score |
| biz-accuracy-04 | Compliance | Compliance rate formula |
| biz-accuracy-05 | Compliance | Violation count consistency |
| biz-consistency-01 | Initiatives | Work allocations sum to 100% |
| biz-consistency-02 | Compliance | Active violations have no resolved_at |
| biz-freshness-01 | Health | Health scores are recent |
| biz-freshness-02 | Compliance | Compliance summaries are recent |
| biz-value-01 | Initiatives | Value scores in 0-100 range |
| biz-value-02 | Initiatives | Story points aggregation correct |
| biz-distribution-01 | Health | Health level distribution |
| biz-distribution-02 | Initiatives | Investment category distribution |
| biz-distribution-03 | Compliance | Agreement types distribution |
| biz-sample-01 | Health | Sample health scores |
| biz-sample-02 | Compliance | Sample violations |
| biz-sample-03 | Initiatives | Sample initiatives |

---

## Dependencies

| Dependency | Required For | Notes |
|------------|--------------|-------|
| Jira Plugin | business_initiatives | Requires Epics (type='Epic') |
| DORA Plugin | team_health_scores | Provides DORA metrics |
| project_mapping | All sections | Links issues to projects |

---

## Notes

- **DORA Dependency**: Health scores require DORA plugin to run first with valid data
- **Epic Requirement**: Business initiatives are extracted from Jira Epics only
- **Working Agreements**: Must be configured via API before violations can be detected
- **Label Consistency**: Investment categories extracted from labels (requires consistent labeling convention)
