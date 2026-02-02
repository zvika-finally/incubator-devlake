# AI Detection Dashboard - Per-Metric Audit Checklist

**Dashboard:** AI-Assisted Development
**UID:** ai-detection-dashboard
**Audit Date:** 2026-02-02
**Total Metrics:** 28 panels

---

## Section 1: AI Detection Overview (8 metrics)

### 1.1 Explicit AI Markers
**Panel:** Stat panel showing count of PRs with explicit AI markers

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Count of PRs where explicit_tool_detected = true |
| ☐ Formula documented | | `COUNT(*) FROM ai_usage_signals WHERE explicit_tool_detected = true` |
| ☐ Edge cases handled | | NULL values, empty strings |
| **Data Lineage** | | |
| ☐ Source identified | | pull_requests.body, commits.message |
| ☐ Plugin task | | detectExplicitSignals |
| ☐ Output table | | ai_usage_signals.explicit_tool_detected |
| **Trust Validation** | | |
| ☐ Completeness | | All merged PRs should be analyzed |
| ☐ Accuracy | | Explicit markers correctly parsed |
| ☐ Freshness | | detected_at within 7 days |
| **Verification Query** | | `#ai-accuracy-02` |

---

### 1.2 Avg AI Confidence
**Panel:** Stat panel showing average confidence score

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Average AI confidence score across all PRs |
| ☐ Formula documented | | `AVG(ai_confidence_score) FROM ai_usage_signals` |
| ☐ Edge cases handled | | Empty dataset, NULL scores |
| **Data Lineage** | | |
| ☐ Source identified | | Multiple signal scores combined |
| ☐ Plugin task | | scoreAIConfidence |
| ☐ Output table | | ai_usage_signals.ai_confidence_score |
| **Trust Validation** | | |
| ☐ Completeness | | All PRs have confidence scores |
| ☐ Accuracy | | Scores in 0-100 range |
| ☐ Freshness | | Recent calculation |
| **Verification Query** | | `#ai-accuracy-01` |

---

### 1.3 Total PRs Analyzed
**Panel:** Stat panel showing total PRs with AI signals

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Count of all PRs analyzed for AI signals |
| ☐ Formula documented | | `COUNT(*) FROM ai_usage_signals` |
| ☐ Edge cases handled | | Should match merged PR count |
| **Data Lineage** | | |
| ☐ Source identified | | pull_requests (merged) |
| ☐ Plugin task | | All detection subtasks |
| ☐ Output table | | ai_usage_signals |
| **Trust Validation** | | |
| ☐ Completeness | | Match against total merged PRs |
| ☐ Accuracy | | No duplicates |
| ☐ Freshness | | Recent data |
| **Verification Query** | | `#ai-completeness-01` |

---

### 1.4 High Confidence (≥70%)
**Panel:** Stat panel showing count of high-confidence AI PRs

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | PRs with strong AI indicators |
| ☐ Formula documented | | `COUNT(*) WHERE ai_confidence_score >= 70` |
| ☐ Edge cases handled | | Threshold boundary (70 inclusive) |
| **Verification Query** | | `#ai-distribution-01` |

---

### 1.5 Medium Confidence (40-69%)
**Panel:** Stat panel showing count of medium-confidence PRs

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | PRs with possible AI indicators |
| ☐ Formula documented | | `COUNT(*) WHERE ai_confidence_score BETWEEN 40 AND 69` |
| ☐ Edge cases handled | | Threshold boundaries |
| **Verification Query** | | `#ai-distribution-01` |

---

### 1.6 AI Tools Detected
**Panel:** Table showing detected AI tools and counts

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Breakdown by detected tool (Copilot, Cursor, etc.) |
| ☐ Formula documented | | `GROUP BY detected_tool` |
| ☐ Edge cases handled | | Unknown tools, multiple tools per PR |
| **Verification Query** | | `#ai-distribution-03` |

---

### 1.7 AI Detection Trend Over Time
**Panel:** Time series showing AI detection over time

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Trend of AI-assisted PRs over time |
| ☐ Formula documented | | Grouped by merged_date |
| ☐ Edge cases handled | | Gaps in data, timezone handling |
| **Time Aggregation** | | Daily/Weekly grouping |

---

## Section 2: Code Churn Analysis (5 metrics)

### 2.1 AI Code Churn (30d)
**Panel:** Stat showing average churn ratio for AI-assisted PRs

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Average code churn within 30 days for AI PRs |
| ☐ Formula documented | | `AVG(churn_ratio_30_days) WHERE is_ai_assisted = true` |
| ☐ Edge cases handled | | PRs younger than 30 days excluded |
| **Data Lineage** | | |
| ☐ Source identified | | commit_files post-merge |
| ☐ Plugin task | | analyzeCodeChurn |
| ☐ Output table | | ai_churn_metrics, project_churn_summaries |
| **Trust Validation** | | |
| ☐ Completeness | | All eligible PRs have churn data |
| ☐ Accuracy | | Churn calculation correct |
| **Verification Query** | | `#ai-accuracy-03`, `#ai-churn-01` |

---

### 2.2 Non-AI Code Churn (30d)
**Panel:** Stat showing average churn ratio for non-AI PRs

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Average code churn for non-AI PRs |
| ☐ Formula documented | | `AVG(churn_ratio_30_days) WHERE is_ai_assisted = false` |
| **Verification Query** | | `#ai-churn-01` |

---

### 2.3 AI vs Non-AI Difference %
**Panel:** Stat showing percentage difference in churn

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Relative difference in churn between AI and non-AI |
| ☐ Formula documented | | `(AI_churn - NonAI_churn) / NonAI_churn * 100` |
| ☐ Edge cases handled | | Division by zero if non-AI churn is 0 |
| **Verification Query** | | `#ai-churn-03` |

---

### 2.4 PRs with Churn Data
**Panel:** Stat showing count of PRs with churn metrics

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | PRs old enough to have 30-day churn data |
| ☐ Formula documented | | `COUNT(*) FROM ai_churn_metrics` |
| **Verification Query** | | `#ai-completeness-02` |

---

### 2.5 Code Churn Over Time
**Panel:** Time series comparing AI vs non-AI churn

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Trend of churn ratios over time |
| ☐ Formula documented | | Grouped by merged_at month |
| **Time Aggregation** | | Monthly |

---

## Section 3: AI Tool Usage - Cursor & Claude Code (10 metrics)

### 3.1-3.3 Cursor Metrics
**Panels:** Suggestions (30d), Accepted (30d), Accept Rate

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Direct metrics from Cursor Business API |
| ☐ Data source | | External: cursor_usage_metrics table (optional) |
| ☐ Edge cases | | API not configured, missing data |
| **Note** | | Optional - requires Cursor Business API integration |

---

### 3.4-3.6 Claude Code Metrics
**Panels:** Tool Uses (30d), Sessions (30d), Lines Added (30d)

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Direct metrics from Claude Code Admin API |
| ☐ Data source | | External: claude_code_usage_metrics table (optional) |
| ☐ Edge cases | | API not configured, missing data |
| **Note** | | Optional - requires Claude Code Admin API integration |

---

### 3.7-3.10 Usage Trends and Top Users
**Panels:** Cursor Usage Over Time, Claude Code Usage Over Time, Top Cursor Users, Top Claude Code Users

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Time series and leaderboards |
| ☐ Data source | | cursor_usage_metrics, claude_code_usage_metrics |
| **Note** | | Optional - external API data |

---

## Section 4: AI Impact Analysis (5 metrics)

### 4.1 PR Throughput Change
**Panel:** Stat showing change in PR throughput after AI adoption

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | % change in PRs/developer after AI tools |
| ☐ Formula documented | | `(after_count - before_count) / before_count * 100` |
| ☐ Edge cases handled | | Division by zero, no before period |
| **Data Lineage** | | |
| ☐ Plugin task | | calculateAIImpact |
| ☐ Output table | | ai_impact_metrics |
| **Verification Query** | | `#ai-freshness-02` |

---

### 4.2 Review Time Change
**Panel:** Stat showing change in review time (faster = positive)

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | % reduction in time from PR open to merge |
| ☐ Formula documented | | Based on avg_pr_cycle_time comparison |

---

### 4.3 Lead Time Change
**Panel:** Stat showing change in lead time (faster = positive)

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | % reduction in lead time for changes |
| ☐ Dependency | | Requires DORA plugin data |

---

## Section 5: Detailed Analysis (3 metrics)

### 5.1 PRs with Explicit AI Tool Markers
**Panel:** Table listing PRs with explicit AI markers

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Detailed list for manual review |
| ☐ Fields shown | | PR title, tool, confidence, date |
| **Verification Query** | | `#ai-sample-01` |

---

### 5.2 Distribution by Detection Confidence
**Panel:** Pie/bar chart showing confidence distribution

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Visual breakdown of High/Medium/Low |
| **Verification Query** | | `#ai-distribution-01` |

---

### 5.3 All PRs by AI Confidence Score
**Panel:** Table with all PRs and signal breakdown

| Dimension | Status | Notes |
|-----------|--------|-------|
| **Logic** | | |
| ☐ Outcome defined | | Full details for investigation |
| ☐ Fields shown | | PR, confidence, explicit_score, rapid_commit_score, etc. |
| **Verification Query** | | `#ai-sample-02` |

---

## Summary Checklist

| Section | Metrics | Logic | Data Trust | Testing |
|---------|---------|-------|------------|---------|
| AI Detection Overview | 8 | ☐/8 | ☐/8 | ☐/8 |
| Code Churn Analysis | 5 | ☐/5 | ☐/5 | ☐/5 |
| AI Tool Usage | 10 | ☐/10 | ☐/10 | ☐/10 |
| AI Impact Analysis | 5 | ☐/5 | ☐/5 | ☐/5 |
| Detailed Analysis | 3 | ☐/3 | ☐/3 | ☐/3 |
| **TOTAL** | **31** | **☐/31** | **☐/31** | **☐/31** |

---

## Verification Query Mapping

| Check ID | Section | Validates |
|----------|---------|-----------|
| ai-completeness-01 | Overview | Total PRs analyzed |
| ai-completeness-02 | Churn | PRs with churn data |
| ai-accuracy-01 | Overview | Confidence score range |
| ai-accuracy-02 | Overview | Explicit detection consistency |
| ai-accuracy-03 | Churn | Churn ratio formula |
| ai-consistency-01 | Churn | High confidence → is_ai_assisted |
| ai-distribution-01 | Overview | Confidence distribution |
| ai-distribution-03 | Overview | Tools breakdown |
| ai-churn-01 | Churn | Summary data validity |
| ai-churn-03 | Churn | Difference calculation |
| ai-freshness-01 | All | Signal freshness |
| ai-freshness-02 | Impact | Impact metric freshness |
| ai-sample-01 | Detailed | Explicit detection samples |
| ai-sample-02 | Detailed | Behavioral detection samples |

---

## Notes

- **External API metrics** (Cursor, Claude Code): Marked as optional. These require separate API integrations and may not be configured in all deployments.
- **Churn analysis**: Requires PRs to be 30+ days old for complete data.
- **Impact analysis**: Requires historical data from before AI tool adoption for comparison.
