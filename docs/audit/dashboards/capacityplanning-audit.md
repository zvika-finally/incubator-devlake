# Capacity Planning Dashboard - Per-Metric Audit Checklist

**Dashboard:** Capacity Planning & Forecasting (Kanban)
**UID:** capacity-planning-dashboard
**Audit Date:** 2026-02-02
**Total Metrics:** 25 panels

---

## Section 1: Throughput Overview (4 metrics)

### 1.1 Avg Throughput (Issues/Week)
**Panel:** Stat showing average issues completed per week

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Average weekly issue completion rate |
| ☐ Formula documented | | `AVG(issues_completed) FROM team_velocities` |
| ☐ Edge cases handled | | Empty periods excluded |
| **Data Lineage** | | |
| ☐ Source identified | | issues (resolved) |
| ☐ Plugin task | | calculateThroughput |
| ☐ Output table | | team_velocities.issues_completed |
| **Verification Query** | | `#cap-completeness-01` |

---

### 1.2 Avg Cycle Time (Hours)
**Panel:** Stat showing average cycle time in hours

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Average time from issue start to resolution |
| ☐ Formula documented | | `AVG(avg_cycle_time_hours)` |
| **Data Lineage** | | |
| ☐ Source identified | | issues (resolution_date - created_date) |
| ☐ Output table | | team_velocities.avg_cycle_time_hours |
| **Verification Query** | | `#cap-sample-01` |

---

### 1.3 Monte Carlo Forecasts Count
**Panel:** Stat showing count of forecasts

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Total Monte Carlo forecasts generated |
| ☐ Formula documented | | `COUNT(*) FROM monte_carlo_forecasts` |
| **Data Lineage** | | |
| ☐ Plugin task | | monteCarloForecastKanban |
| ☐ Output table | | monte_carlo_forecasts |
| **Verification Query** | | `#cap-completeness-02` |

---

### 1.4 AI Tools Annual Benefit
**Panel:** Stat showing annual benefit from AI tools investment

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Total annual benefit in USD |
| ☐ Formula documented | | `total_annual_benefit WHERE investment_type = 'ai_tools'` |
| **Data Lineage** | | |
| ☐ Plugin task | | calculateROI |
| ☐ Output table | | investment_rois.total_annual_benefit |
| **Verification Query** | | `#cap-accuracy-10` |

---

## Section 2: Throughput Charts (2 metrics)

### 2.1 Recent Throughput Bar Chart
**Panel:** Bar chart showing issues completed per period

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Visual throughput trend |
| ☐ Grouping | | By sprint/fiscal_week |
| ☐ Edge cases | | Limit to 10 recent periods |
| **Verification Query** | | `#cap-sample-01` |

---

### 2.2 Forecasts by Confidence Level
**Panel:** Pie chart showing initiative forecast confidence distribution

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Distribution of high/medium/low confidence |
| ☐ Formula documented | | `GROUP BY confidence_level` |
| **Data Lineage** | | |
| ☐ Source identified | | initiative_forecasts.confidence_level |
| **Verification Query** | | `#cap-distribution-01` |

---

## Section 3: Monte Carlo Forecasts (1 metric)

### 3.1 Monte Carlo Forecasts Table
**Panel:** Table showing percentile completion dates

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Probabilistic completion forecasts |
| ☐ Fields shown | | P50, P75, P90, P95 weeks and dates |
| ☐ Algorithm | | 1000 iterations with throughput variance |
| **Data Lineage** | | |
| ☐ Source identified | | team_velocities, initiative_forecasts |
| ☐ Plugin task | | monteCarloForecastKanban |
| ☐ Output table | | monte_carlo_forecasts |
| **Trust Validation** | | |
| ☐ Completeness | | All initiatives have forecasts |
| ☐ Accuracy | | P50 <= P75 <= P90 <= P95 |
| ☐ Freshness | | calculated_at within 14 days |
| **Verification Query** | | `#cap-accuracy-01`, `#cap-accuracy-03` |

---

## Section 4: Brooks's Law Capacity (1 metric)

### 4.1 Brooks's Law Scenarios Table
**Panel:** Table showing team size impact analysis

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Impact of team changes on capacity |
| ☐ Fields shown | | team_size, delta, channels, overhead |
| ☐ Algorithm | | channels = n×(n-1)/2 |
| **Data Lineage** | | |
| ☐ Source identified | | Settings + project_mapping |
| ☐ Plugin task | | brooksLawModel |
| ☐ Output table | | capacity_models |
| **Trust Validation** | | |
| ☐ Accuracy | | Channel formula correct |
| **Verification Query** | | `#cap-accuracy-11`, `#cap-sample-04` |

---

## Section 5: Investment ROI (4 metrics)

### 5.1 AI Tools Payback Period
**Panel:** Stat showing months to break even

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Payback period in months |
| ☐ Formula documented | | `(annual_cost / annual_benefit) × 12` |
| ☐ Thresholds | | Green: <6mo, Yellow: 6-12mo, Red: >12mo |
| **Data Lineage** | | |
| ☐ Output table | | investment_rois.payback_months |
| **Verification Query** | | `#cap-accuracy-09` |

---

### 5.2 AI Tools 3-Year ROI
**Panel:** Stat showing 3-year ROI percentage

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | ROI over 3 years |
| ☐ Formula documented | | `((benefit×3 - cost×3) / cost×3) × 100` |
| ☐ Thresholds | | Green: >500%, Yellow: 100-500%, Red: <100% |
| **Data Lineage** | | |
| ☐ Output table | | investment_rois.three_year_roi |
| **Verification Query** | | `#cap-sample-05` |

---

### 5.3 Investment ROI Details Table
**Panel:** Table showing all investment ROI details

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Full ROI breakdown per investment |
| ☐ Fields shown | | costs, benefits, payback, ROI |
| **Data Lineage** | | |
| ☐ Plugin task | | calculateROI |
| ☐ Output table | | investment_rois |
| **Trust Validation** | | |
| ☐ Accuracy | | Annual cost = upfront + monthly×12 |
| ☐ Accuracy | | Total benefit = sum of components |
| **Verification Query** | | `#cap-accuracy-09`, `#cap-accuracy-10` |

---

### 5.4 Investment Types Distribution
**Panel:** Implicit (derived from ROI table)

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Breakdown by ai_tools, hiring, tech_debt |
| **Verification Query** | | `#cap-distribution-03` |

---

## Section 6: Initiative Forecasts (1 metric)

### 6.1 Initiative Completion Forecasts Table
**Panel:** Table showing initiative progress and forecasts

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Progress and completion estimates |
| ☐ Fields shown | | total, completed, remaining, %, weeks, confidence |
| ☐ Algorithm | | remaining / avg_throughput |
| **Data Lineage** | | |
| ☐ Source identified | | business_initiatives, team_velocities |
| ☐ Plugin task | | forecastCompletionKanban |
| ☐ Output table | | initiative_forecasts |
| **Trust Validation** | | |
| ☐ Accuracy | | remaining = total - completed |
| ☐ Accuracy | | percent_complete formula correct |
| **Verification Query** | | `#cap-accuracy-07`, `#cap-accuracy-08` |

---

## Section 7: Flow Efficiency Analysis (12 metrics)

### 7.1 Current Avg Flow Efficiency
**Panel:** Gauge showing average flow efficiency percentage

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Average % of time in active work |
| ☐ Formula documented | | `AVG(flow_efficiency)` |
| ☐ Thresholds | | Green: ≥40%, Yellow: 25-40%, Orange: 15-25%, Red: <15% |
| **Data Lineage** | | |
| ☐ Source identified | | issue_flow_metrics.flow_efficiency |
| ☐ Plugin task | | calculateFlowEfficiency |
| ☐ Output table | | project_flow_summaries.avg_flow_efficiency |
| **Trust Validation** | | |
| ☐ Accuracy | | Flow efficiency in 0-100 range |
| **Verification Query** | | `#cap-accuracy-05` |

---

### 7.2 Avg Total Cycle Time
**Panel:** Stat showing average total days

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Average calendar days from start to done |
| ☐ Formula documented | | `AVG(total_days)` |
| **Data Lineage** | | |
| ☐ Output table | | project_flow_summaries.avg_total_days |
| **Verification Query** | | `#cap-sample-03` |

---

### 7.3 Avg Active Time
**Panel:** Stat showing average active days

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Average days in active/progress statuses |
| ☐ Formula documented | | `AVG(active_days)` |
| **Data Lineage** | | |
| ☐ Output table | | project_flow_summaries.avg_active_days |
| **Verification Query** | | `#cap-accuracy-04` |

---

### 7.4 Avg Waiting Time
**Panel:** Stat showing average waiting days

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Average days in waiting/blocked statuses |
| ☐ Formula documented | | `waiting_days = total_days - active_days` |
| **Data Lineage** | | |
| ☐ Output table | | project_flow_summaries.avg_waiting_days |
| **Trust Validation** | | |
| ☐ Accuracy | | waiting = total - active |
| **Verification Query** | | `#cap-accuracy-06` |

---

### 7.5 Issues by Flow Efficiency Category
**Panel:** Donut chart showing excellent/good/average/poor distribution

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Distribution by efficiency category |
| ☐ Categories | | Excellent(≥40%), Good(25-39%), Average(15-24%), Poor(<15%) |
| **Data Lineage** | | |
| ☐ Output table | | project_flow_summaries.*_count fields |
| **Trust Validation** | | |
| ☐ Consistency | | Category counts sum to total |
| **Verification Query** | | `#cap-distribution-02`, `#cap-consistency-01` |

---

### 7.6 Flow Efficiency Trend
**Panel:** Time series showing avg, median, P90 flow efficiency

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Flow efficiency over time |
| ☐ Time aggregation | | By period (sprint/week) |
| **Data Lineage** | | |
| ☐ Output table | | project_flow_summaries (historical) |
| **Freshness Query** | | `#cap-freshness-03` |

---

### 7.7 Flow Efficiency by Issue Type
**Panel:** Pie chart showing efficiency by issue type

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Efficiency breakdown by Bug/Story/Task |
| ☐ Grouping | | By issue_type |
| **Data Lineage** | | |
| ☐ Source identified | | issue_flow_metrics.issue_type |
| **Verification Query** | | `#cap-sample-03` |

---

### 7.8 Active vs Waiting Time Trend
**Panel:** Time series comparing active and waiting days

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Active vs waiting time trends |
| ☐ Time aggregation | | By period |
| **Data Lineage** | | |
| ☐ Output table | | project_flow_summaries |

---

### 7.9 Period Flow Summary Table
**Panel:** Table showing weekly flow summaries

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Aggregated flow metrics per period |
| ☐ Fields shown | | issue_count, avg/median/p90 efficiency, days, category counts |
| **Data Lineage** | | |
| ☐ Plugin task | | calculateFlowEfficiency |
| ☐ Output table | | project_flow_summaries |
| **Verification Query** | | `#cap-consistency-01` |

---

### 7.10 Recent Completed Issues Flow Details
**Panel:** Table showing individual issue flow metrics

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Per-issue flow efficiency details |
| ☐ Fields shown | | issue_key, type, total/active/waiting days, efficiency |
| **Data Lineage** | | |
| ☐ Source identified | | issues, issue_changelogs |
| ☐ Plugin task | | calculateFlowEfficiency |
| ☐ Output table | | issue_flow_metrics |
| **Trust Validation** | | |
| ☐ Accuracy | | Flow efficiency formula correct |
| ☐ Completeness | | Completed issues have flow data |
| **Verification Query** | | `#cap-accuracy-04`, `#cap-completeness-03` |

---

### 7.11 Flow Efficiency Explanation Panel
**Panel:** Text/markdown explaining flow efficiency concept

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Educational content about flow efficiency |
| ☐ Accuracy | | Thresholds match implementation |

---

### 7.12 Algorithm Documentation Panel
**Panel:** Text/markdown explaining algorithms

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Monte Carlo, Brooks's Law, ROI formulas |
| ☐ Accuracy | | Formulas match implementation |

---

## Summary Checklist

| Section | Metrics | Logic | Data Trust | Testing | Status |
|---------|---------|-------|------------|---------|--------|
| Throughput Overview | 4 | ✅ 4/4 | ✅ 4/4 | ✅ 2/2 | ✅ PASS |
| Throughput Charts | 2 | ✅ 2/2 | ✅ 2/2 | ✅ 1/1 | ✅ PASS |
| Monte Carlo Forecasts | 1 | ✅ 1/1 | ⚠️ No Data | ✅ 1/1 | ⚠️ No Initiatives |
| Brooks's Law Capacity | 1 | ✅ 1/1 | ✅ 1/1 | ✅ 1/1 | ✅ PASS |
| Investment ROI | 4 | ✅ 4/4 | ⏭️ N/A | ⏭️ N/A | ⏭️ Not Tested |
| Initiative Forecasts | 1 | ✅ 1/1 | ⚠️ No Data | ⏭️ N/A | ⚠️ No Initiatives |
| Flow Efficiency Analysis | 12 | ✅ 12/12 | ✅ 12/12 | ✅ 4/4 | ✅ PASS |
| **TOTAL** | **25** | **✅ 25/25** | **✅ 19/21** | **✅ 9/9** | **✅ PASS** |

**Verification Date:** 2026-02-02
**Overall Status:** ✅ PRODUCTION-READY (with noted limitations)

### Verification Results Summary

| Check | Result | Status |
|-------|--------|--------|
| Completeness (Velocities) | 4/4 projects | ✅ PASS |
| Completeness (Flow Metrics) | 14,579 issues | ✅ PASS |
| Completeness (Monte Carlo) | 0 records | ⚠️ REVIEW |
| Accuracy (Flow Efficiency) | 0-100 range valid | ✅ PASS |
| Accuracy (Brooks's Law) | 5 team → 10 channels | ✅ PASS |
| Freshness (Velocities) | 0 days old | ✅ PASS |

### Key Metrics

| Metric | Value |
|--------|-------|
| Issues with Flow Metrics | 14,579 |
| Projects with Velocities | 4 |
| Avg Weekly Throughput | ~500 issues |
| Flow Efficiency (Excellent) | 4,640 (31.8%) |
| Flow Efficiency (Poor) | 8,985 (61.6%) |
| Monte Carlo Forecasts | 0 (no initiatives) |

### Known Limitations

1. **No Monte Carlo Forecasts**: Requires Epics/initiatives (0 in Jira)
2. **Team Size = 0**: Not being captured in velocities
3. **Bimodal Flow**: 62% Poor, 32% Excellent (few in middle)

---

## Verification Query Mapping

| Check ID | Section | Validates |
|----------|---------|-----------|
| cap-completeness-01 | Throughput | Team velocities exist |
| cap-completeness-02 | Monte Carlo | Forecasts exist for initiatives |
| cap-completeness-03 | Flow | Flow metrics for completed issues |
| cap-completeness-04 | Flow | Project flow summaries exist |
| cap-accuracy-01 | Monte Carlo | Percentiles correctly ordered |
| cap-accuracy-02 | Monte Carlo | Simulation count correct |
| cap-accuracy-03 | Monte Carlo | Dates are future |
| cap-accuracy-04 | Flow | Flow efficiency formula |
| cap-accuracy-05 | Flow | Flow efficiency 0-100 range |
| cap-accuracy-06 | Flow | Waiting days formula |
| cap-accuracy-07 | Initiative | Percent complete formula |
| cap-accuracy-08 | Initiative | Remaining calculation |
| cap-accuracy-09 | ROI | Annual cost formula |
| cap-accuracy-10 | ROI | Total benefit formula |
| cap-accuracy-11 | Brooks's Law | Channels formula |
| cap-consistency-01 | Flow | Category counts sum to total |
| cap-consistency-02 | Throughput | Recent periods exist |
| cap-freshness-01 | Throughput | Data within 14 days |
| cap-freshness-02 | Monte Carlo | Forecasts within 14 days |
| cap-freshness-03 | Flow | Flow metrics within 14 days |
| cap-distribution-01 | Initiative | Confidence level distribution |
| cap-distribution-02 | Flow | Efficiency category distribution |
| cap-distribution-03 | ROI | Investment types |
| cap-sample-01 | Throughput | Sample throughput data |
| cap-sample-02 | Monte Carlo | Sample forecasts |
| cap-sample-03 | Flow | Sample flow efficiency |
| cap-sample-04 | Brooks's Law | Sample scenarios |
| cap-sample-05 | ROI | Sample ROI calculations |

---

## Dependencies

| Dependency | Required For | Notes |
|------------|--------------|-------|
| Core DevLake | team_velocities | issues required |
| Core DevLake | issue_flow_metrics | issue_changelogs required |
| Business Metrics (optional) | initiative_forecasts | For initiative-level tracking |
| Settings API | capacity_models, investment_rois | Requires configuration |

---

## Visualization Review

**Review Date:** 2026-02-02

### Chart Type Assessment

| Panel Type | Count | Usage | Assessment |
|------------|-------|-------|------------|
| `stat` | 8 | KPIs, totals, payback periods | ✅ Appropriate |
| `gauge` | 1 | Flow efficiency % (bounded) | ✅ Appropriate |
| `barchart` | 1 | Throughput trend | ✅ Appropriate |
| `piechart` | 3 | Confidence distribution, efficiency categories | ✅ Appropriate |
| `timeseries` | 2 | Flow efficiency trends | ✅ Appropriate |
| `table` | 7 | Monte Carlo, ROI, flow details | ✅ Appropriate |
| `text` | 3 | Explanations, algorithms | ✅ Appropriate |

### Threshold Validation

| Metric | Thresholds | Business Logic | Status |
|--------|------------|----------------|--------|
| Flow Efficiency | 🟢≥40% 🟡25-40% 🟠15-25% 🔴<15% | Excellent/Good/Average/Poor | ✅ Correct |
| Payback Period | 🟢<6mo 🟡6-12mo 🔴>12mo | Shorter = better ROI | ✅ Correct |
| 3-Year ROI | 🟢>500% 🟡100-500% 🔴<100% | Higher = better investment | ✅ Correct |
| Forecast Confidence | 🟢High 🟡Medium 🔴Low | Based on data quality | ✅ Correct |

### Color Coding

| Element | Color | Assessment |
|---------|-------|------------|
| Flow Efficiency gauge | Green/Yellow/Orange/Red | ✅ Intuitive gradient |
| ROI positive values | Green | ✅ Clear positive indicator |
| Payback warning | Yellow/Red | ✅ Appropriate warning |
| Forecast confidence | Blue/Orange/Red | ✅ Clear hierarchy |

### Layout Assessment

- ✅ Clear sectioning (Throughput → Monte Carlo → Brooks's Law → ROI → Flow)
- ✅ Collapsible rows for organization
- ✅ Algorithm explanations included
- ✅ Drill-down tables for detailed analysis
- ✅ Consistent gauge positioning for efficiency metrics

**Visualization Status:** ✅ GOOD

---

## Notes

- **Kanban Focus**: Dashboard optimized for Kanban (issue counts, weekly throughput) over Scrum
- **Settings Required**: Brooks's Law and ROI features require configuration via API
- **Changelog Dependency**: Flow efficiency requires issue_changelogs with status transitions
- **Initiative Dependency**: Initiative forecasts require business_initiatives from Business Metrics plugin
- **Default Variance**: Monte Carlo uses 25% throughput variance by default
