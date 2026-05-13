# AI-Era Engineering Signals — Design

**Status:** Draft for review
**Date:** 2026-05-13
**Owner:** CTO / Engineering Leadership
**Target plugin:** new `aimeasure` plugin in `incubator-devlake` fork

## 1. Problem

The Finally fork has built nine plugins covering AI utilization (`aidetector`, `claudecode`, `cursor`, `q_dev`), engineering cost (`findevops`), capacity (`capacityplanner`), business linkage (`businessmetrics`), quality tooling (`testmo`), and GitOps (`argocd`). Yet none of them answer the single most important question for a fintech engineering org in 2026:

> **Is AI actually making our engineering org faster, or just busier — at what cost, with what quality trade-offs?**

Industry data is split. Anthropic reports +67% PRs per engineer; METR's RCT showed experienced developers on mature codebases were 19% *slower* with AI while feeling 20% faster. CodeRabbit, Cortex, and GitClear all show concerning quality regression patterns. The DORA 2024 report identified a ~7.2% drop in delivery stability and ~1.5% drop in throughput correlated with AI adoption.

The fork measures *adoption*. It does not yet measure *outcome*. This design closes that gap.

## 2. Goals

1. Answer the strategic question — "Is AI worth it for our team?" — with evidence, not vibes
2. Cohort every engineering metric by AI assistance level so trends are comparable apples-to-apples
3. Surface invisible work (verification effort, dark matter) that the current platform misses
4. Produce a board-grade narrative quarterly that finance and exec leadership can act on
5. Stay audit-ready for SOX/SOC2 — every classification decision is traceable to versioned rules

## 3. Non-goals

- Replacing `aidetector`, `findevops`, `claudecode`, `cursor`, or any existing collector — `aimeasure` reads from them, doesn't overlap with them
- Building a survey-only DXI tool — sentiment is one signal among many, not the headline
- Predicting individual engineer performance — this platform measures patterns, never grades people
- Real-time alerting — weekly/monthly cadence is the target; alerting is future work

## 4. Architecture

A new plugin `aimeasure` sits on top of existing collectors and produces cross-source analytics. Source plugins collect raw signals; `aimeasure` does cohorting, aggregation, and joins.

```
[aidetector]    → ai_usage_signals       ┐
[claudecode]    → claude_code_metrics    │ raw signal sources
[cursor]        → cursor_metrics         │ (read-only inputs)
[slack]         → _tool_slack_messages   │
[findevops]     → cost_allocations       │
[git/github]    → pull_requests, etc.    ┘
                              │
                              ▼
                   ┌─────────────────────┐
                   │  aimeasure plugin   │
                   │ - classifyPRCohort  │  ← Phase A foundation
                   │ - quality cohort    │     (reused by B + C)
                   │ - verification eff. │
                   │ - throughput cohort │
                   │ - cost per outcome  │
                   └─────────────────────┘
                              │
              ┌───────────────┼───────────────┐
              ▼               ▼               ▼
      AIQualityCohort   InvisibleWork    AIROISummary
        (Phase A)         (Phase B)        (Phase C)
```

### 4.1 Why a separate plugin

- Each existing plugin's scope is "collect X from source Y." `aimeasure`'s scope is "cross-source analytics."
- The AI-cohort classifier (Phase A's core) feeds every later phase. Embedding it in `aidetector` would couple analytics to PR detection; in `findevops` would couple it to cost. Both are wrong layering.
- New plugin = clean migrations (no churn in audit history of `findevops`), clean Grafana namespace, clean test boundary.

### 4.2 Reuse boundary

`aimeasure` reads from source-plugin tables **read-only**. It does NOT write back. Its outputs are its own tables, consumed by Grafana and (in Phase C) optionally by `findevops` rollups.

### 4.3 Identity resolution (shared across all phases)

Several signals require resolving a person across Slack, GitHub, and Jira identities. The plugin uses DevLake's existing `accounts` cross-domain table (already populated by `org` plugin + per-source account collectors). For matches that fail (consultants, ex-employees with stale Slack identities), the plugin reads from a new manual-override table `aimeasure_account_overrides` (engineer-maintained, version-controlled mapping). PRs missing identity resolution are excluded from per-engineer rollups (counted in org totals only) and flagged in a metadata column.

Team membership comes from DevLake's `teams`/`team_users` tables. Seniority is *not* inferred from the platform; rollups that need "senior vs junior" framing read it from an optional `aimeasure_engineer_roles` table (manually populated, defaults absent → treated as "unknown" and excluded from senior-specific thresholds). The platform never auto-classifies seniority.

### 4.4 Plugin lifecycle

Mirrors existing fork patterns:
- Migrations directory for new tables
- Subtasks per phase: `classifyPRCohort`, `computeQualityCohort`, `computeChangeComposition`, `computeVerificationEffort`, `computeSlackSignals`, `computeSentimentProxy`, `computeThroughputCohort`, `computeCostPerOutcome`
- Each subtask independently runnable, idempotent (rerun = overwrite)
- Initially `EnabledByDefault: false` so the rollout is gated per environment

## 5. Phase A — Quality cohorting (foundation)

### 5.1 The AI-cohort classifier

This is the load-bearing piece. Every later phase reuses it.

**Inputs (read-only):**
- `ai_usage_signals.ai_confidence_score` and `.explicit_tool_detected` (from `aidetector`)
- Commit trailer regex `Co-authored-by: (Claude|Copilot|Cursor|Devin|GitHub Copilot)` across all commits in the PR
- Per-engineer `claudecode_metrics` / `cursor_metrics` usage intensity during the PR's authoring window

**Output table — `pr_ai_cohort`:**

| Column | Type | Notes |
|---|---|---|
| pr_id | varchar PK | references `pull_requests.id` |
| ai_cohort | enum | `NONE` / `LOW` / `MEDIUM` / `HIGH` |
| confidence_score | int | 0–100 from `aidetector` |
| has_explicit_marker | bool | from `aidetector.explicit_tool_detected` |
| has_commit_trailer | bool | regex match on commit messages |
| classifier_version | varchar | rule-set version for audit trail |
| classified_at | datetime | last computed |

**Cohort definition:**

| Level | Condition |
|---|---|
| HIGH | `has_explicit_marker = true` OR `has_commit_trailer = true` |
| MEDIUM | `confidence_score >= 65` |
| LOW | `confidence_score` between 30 and 64 |
| NONE | `confidence_score < 30` |

Four levels, not binary. METR's research specifically called out binary cohorting as too crude.

### 5.2 Quality signals

**Table `pr_defect_signals`** — for each merged PR, computed at merge and recomputed nightly until the 30-day window closes:

| Column | Type | Notes |
|---|---|---|
| pr_id | varchar PK | |
| has_revert_14d | bool | later commit matches `^Revert ` and cites this PR's merge SHA |
| has_hotfix_14d | bool | PR within 14 days with title matching `(?i)(hotfix\|urgent\|emergency)` touching ≥50% of same files |
| has_incident_14d | bool | incident in Opsgenie/PagerDuty referencing PR's commits (omitted if data unavailable) |
| total_defect_count | int | sum of true booleans |
| window_close_date | datetime | 30 days after merge — when this row freezes |
| computed_at | datetime | |

**Table `pr_change_composition`** — once per merged PR:

| Column | Type | Notes |
|---|---|---|
| pr_id | varchar PK | |
| additions | int | total lines added |
| deletions | int | total lines deleted |
| file_count | int | |
| additive_lines | int | lines added to files that didn't exist before this PR |
| refactor_lines | int | lines touched in pre-existing files |
| refactor_ratio | float | `refactor_lines / (additive_lines + refactor_lines)` |
| batch_bucket | enum | `XS`/`S`/`M`/`L`/`XL` (<50/50–200/200–500/500–1000/>1000 LOC) |

GitClear's churn finding directly drives this: AI shifts work toward `additive_lines` and away from `refactor_lines`. Tracking the ratio over time, per cohort, is the leading indicator of tech-debt accrual.

### 5.3 Subtasks

1. `classifyPRCohort` — joins source signals, writes `pr_ai_cohort`. Idempotent. Runs after `aidetector`.
2. `computeQualityCohort` — joins merged PRs to reverts/hotfix/incidents, writes `pr_defect_signals`. Runs nightly; recomputes for PRs whose 30-day window is still open.
3. `computeChangeComposition` — diffs the PR, writes `pr_change_composition`. Runs once per merged PR.

### 5.4 Dashboard — `AIQualityCohort.json`

Five Grafana panels:
- Defect rate over time, four lines (one per cohort)
- Batch-size distribution by cohort (stacked bar over time)
- Refactor ratio over time, four lines (one per cohort)
- Top engineers by 14-day-defect-rate (table)
- Recent flagged PRs (HIGH cohort + (has_revert OR XL batch))

### 5.5 Signal playbook — Phase A

#### Defect rate by AI cohort

| State | Threshold | What it means |
|---|---|---|
| 🟢 Healthy | HIGH-cohort rate ≤ 1.3× NONE-cohort; both <10% absolute | Comparable quality |
| 🟡 Warning | HIGH ≥ 1.5× NONE, or HIGH ≥ 15% absolute | Quality gap emerging |
| 🔴 Failure | HIGH ≥ 2× NONE, or HIGH cohort trending up while NONE flat | Industry baseline (CodeRabbit: 1.7×, Cortex: +23.5%) |

**Failure mode:** AI lets engineers ship code they don't fully understand; review gaps let regressions slip through.

**Mitigations:**
- Require 2 reviewers on HIGH-cohort PRs touching critical paths (define via CODEOWNERS; assumes CODEOWNERS is actively maintained — if not, that's the prerequisite)
- Mandate test coverage delta on HIGH cohort (CI: line-coverage must not drop)
- Auto-add "AI-assisted review checklist" comment on HIGH cohort PRs
- Per-engineer defect rate coaching for chronic outliers (pair programming, prompting workshops — not punishment)

#### Batch-size distribution by AI cohort

| State | Threshold | What it means |
|---|---|---|
| 🟢 Healthy | HIGH-cohort median <200 LOC; <10% in L/XL | Engineers still splitting; reviewers can keep up |
| 🟡 Warning | HIGH-cohort median 200–500 LOC, or L/XL share >20% | AI productivity outpacing review discipline |
| 🔴 Failure | HIGH-cohort median >500 LOC, or >30% in L/XL, or "PRs grow + review time per PR drops" | Reviewers giving up — silent quality death (Salesforce pattern) |

**Failure mode:** AI removes the friction that used to cap PR size. Engineers ship bigger PRs; reviewers skim or rubber-stamp; bugs aren't caught. DORA 2024 identified this as the most likely root cause of AI's stability regression.

**Mitigations:**
- CI bot auto-comments on PRs >500 LOC requesting a split rationale
- PR template: "Can this be split? Why not?"
- Engineering norm published: *"AI velocity ≠ permission to skip splitting."*
- Coach individual outliers via 1:1
- Hard block on PRs >1000 LOC without explicit override label

#### Refactor ratio by AI cohort

| State | Threshold | What it means |
|---|---|---|
| 🟢 Healthy | refactor_ratio 20–40%, similar across cohorts | Maintenance investment ongoing |
| 🟡 Warning | HIGH-cohort refactor_ratio < NONE-cohort by ≥10 percentage points | AI encouraging "add" over "fix" |
| 🔴 Failure | Org-wide refactor_ratio trending down over 3+ months | Compounding tech debt (GitClear pattern) |

**Failure mode:** AI excels at generating new code, struggles at restructuring. Engineers default to "ask AI to add X" over "ask AI to clean up Y." Codebase rots while velocity dashboard looks healthy. The most insidious signal — won't bite for 6–12 months.

**Mitigations:**
- Refactor budget per sprint (% of capacity protected for non-feature work) — visible in `capacityplanner`
- Refactor-of-the-week showcase — celebrate cleanup PRs
- Internal prompt-pattern library teaching prompts that ask AI to *simplify/delete*
- Track org refactor:add ratio as a leading-indicator OKR (target: ≥1:3)

#### Cohort distribution itself

| State | Threshold | What it means |
|---|---|---|
| 🟢 Healthy | 30–60% of PRs HIGH+MEDIUM; per-engineer variance modest | Getting value from AI investment; spread adoption |
| 🟡 Warning | <20% HIGH+MEDIUM (low adoption), or per-engineer variance >40 percentage points | License under-utilized OR equity/training gap |
| 🔴 Failure | >75% HIGH-cohort (over-dependency / deskilling) OR sudden drop (tooling outage / pushback) | Both ends of the spectrum are bad |

**Mitigations:**
- *Low adoption:* training, pair programming, ergonomics audit, cultural-friction check
- *Uneven adoption:* private per-engineer dashboards for self-correction; team-level coaching for laggard pods
- *Over-adoption:* "AI-off pairing" sessions; principal-eng-led code reviews focused on understanding; rotate engineers through non-AI tasks

### 5.6 Cross-signal red flag (Phase A)

**The productivity illusion (early form):** HIGH cohort throughput up + HIGH cohort defect rate up + refactor ratio down → net negative, but looks positive in vanity metrics. If this pattern appears, Phase C will quantify it; Phase A is the early-warning detection.

### 5.7 Open decisions — Phase A

1. **Incident data source.** Does Finally currently ingest from Opsgenie / PagerDuty / Sentry? If yes, `has_incident_14d` is included. If no, Phase A ships with reverts + hotfix only; incident join added later.
2. **AI-cohort threshold.** Reuse `aidetector`'s existing 65 to keep mental model consistent, or recalibrate against Finally's data?
3. **Hotfix detection.** Title-regex + file-overlap heuristic vs. a label-based convention (e.g., `label:hotfix`). Cleaner if your team has a label convention; regex+overlap ships otherwise.

## 6. Phase B — Verification effort + Dark matter

### 6.1 Schemas

**Table `engineer_verification_effort`** — per engineer per ISO week:

| Column | Type |
|---|---|
| engineer_id | varchar |
| period_week | date |
| author_minutes | int |
| reviewer_minutes | int |
| review_to_author_ratio | float |
| review_comments_total | int |
| review_comments_per_loc | float |
| review_comments_high_cohort | int |
| review_comments_per_loc_high | float |

**Table `engineer_slack_signals`** — per engineer per ISO week:

| Column | Type |
|---|---|
| engineer_id | varchar |
| period_week | date |
| channel_category | varchar (`engineering`/`incident_support`/`design_architecture`/`general`) |
| message_count | int |
| thread_participation_count | int |
| after_hours_message_count | int |
| after_hours_ratio | float |

**Table `engineer_dxi_proxy`** — per engineer per ISO week:

| Column | Type |
|---|---|
| engineer_id | varchar |
| period_week | date |
| sentiment_score | float (0–100, behavioral proxy) |
| bad_developer_day_flag | bool |
| last_survey_date | date (nullable) |
| last_survey_dxi | float (nullable) |

### 6.2 Subtasks

1. `computeVerificationEffort` — joins GitHub PR review events to PR cohort table, aggregates per engineer per week
2. `computeSlackSignals` — reads `_tool_slack_channel_messages`, joins to channel→category mapping table, aggregates
3. `computeSentimentProxy` — derives behavioral sentiment from after-hours patterns + (optionally) message-content tone analysis at the aggregate level only

### 6.3 Dashboard — `InvisibleWork.json`

Panels:
- Review-to-author ratio per engineer (heatmap by week)
- Top reviewers by load share (stacked area)
- Review-comments-per-LOC by AI cohort (line)
- Slack participation by category, per engineer (stacked bar)
- After-hours message ratio per engineer (alert table when >threshold)
- Sentiment score trend by team

### 6.4 Signal playbook — Phase B

#### Review-to-author ratio (per engineer per period)

| State | Threshold | What it means |
|---|---|---|
| 🟢 Healthy | Senior: 0.5–1.5; junior: 0.2–0.5; top reviewer ≤25% of total review load (senior/junior from `aimeasure_engineer_roles`; "unknown" engineers excluded from per-role bands) | Review work distributed; juniors learning |
| 🟡 Warning | 1–2 engineers ≥2.0, or top reviewer >30% of load, or junior ratio = 0 | Concentration risk |
| 🔴 Failure | Top reviewer >40%, OR org-wide ratio trending up without throughput trending up | Verification eating capacity; senior burnout incoming |

**Failure mode:** Anthropic reports 21 tool calls per agent session, engineers fully delegating only 0–20% of work. Every AI output needs human verification; that work concentrates on whoever does code review. If 2 seniors do 60% of reviews, they're not shipping, they're losing weekends, and they're a single point of knowledge failure.

**Mitigations:**
- Round-robin review assignment with load balancing (CODEOWNERS + bot)
- Mandatory junior-as-reviewer for low-risk PRs (learning + load distribution)
- Calendar-blocked "review hours" — reduce context-switch cost
- Pair-review rotations weekly
- If senior at >40% load: redistribute code-area ownership next sprint

#### Review comments per LOC (cohorted by AI)

| State | Threshold | What it means |
|---|---|---|
| 🟢 Healthy | 1 comment per 50–100 LOC; HIGH-cohort PRs get *more* comments/LOC than NONE | Reviewers applying extra scrutiny to AI work |
| 🟡 Warning | <1 per 200 LOC on HIGH, or HIGH comment density trending down | Reviewers tiring of pointing out AI errors |
| 🔴 Failure | HIGH-cohort density ≤ NONE; large PRs getting fewer comments/LOC | Rubber-stamping (Salesforce pattern) |

**Failure mode:** Salesforce found review time per PR *declining* on large PRs — reviewers gave up. Sonar: 96% of devs don't fully trust AI code but only 48% verify it.

**Mitigations:**
- Required-review-checklist comment auto-posted on HIGH cohort PRs
- Review-training workshops with AI-code failure-mode examples
- Per-reviewer comment-density coaching (not punishment)
- Hard cap on PR size for single-reviewer approval (>500 LOC → 2 reviewers)

#### Slack participation by category

| State | Threshold | What it means |
|---|---|---|
| 🟢 Healthy | Mix matches role; senior eng ≥15% in design-architecture; support load <20% per person | Right people in right conversations |
| 🟡 Warning | One engineer >40% of support, or senior <5% in design, or engagement dropping >30% MoM | Concentration OR disengagement |
| 🔴 Failure | One engineer >60% of support; senior at near-zero design participation; sudden full disengagement | Burnout cascade OR senior checked out (likely to leave) |

**Failure mode:** Most-helpful engineers carry disproportionate support load. Seniors who stop participating in design discussions have mentally checked out 6 months before they quit.

**Mitigations:**
- Formal on-call/support rotation (don't let it default to who's nice)
- Recognize design-channel participation in promotion packets — make invisible work visible
- 1:1 check-ins triggered by engagement drops >30% MoM
- Pair "nice" engineers with peers during support spikes; rotate

#### After-hours Slack activity

| State | Threshold | What it means |
|---|---|---|
| 🟢 Healthy | <5% after-hours; spikes only around real incidents | Healthy boundaries |
| 🟡 Warning | 5–15% after-hours, or concentrated in 1–2 engineers | Cultural creep OR specific-person burnout |
| 🔴 Failure | >15% sustained 4+ weeks, or trending up org-wide | "Bad Developer Days" institutionalized |

**Failure mode:** Routine (not emergency) work in nights/weekends signals understaffing, toxic always-on culture, or invisible timezone-shifted load. All three predict attrition.

**Mitigations:**
- Hard on-call policies with formal comp time
- Investigate cause specifically — cross-timezone, burnout, or culture?
- Leadership modeling: managers visibly *not* responding after hours
- Single-engineer concentration → 1:1 immediately, not next month

#### Sentiment / DXI proxy

| State | Threshold | What it means |
|---|---|---|
| 🟢 Healthy | Survey "I have what I need" ≥70%; behavioral proxy stable | Org functioning |
| 🟡 Warning | Survey 50–70%, or behavioral proxy dropping 4+ weeks | Investigate; targeted 1:1s |
| 🔴 Failure | Survey <50%, or drop concentrated in one team | Structural problem; not fixable with pizza |

**Failure mode:** Atlassian 2025: leadership empathy gap widened from 44% to 63%. When engineers distrust leadership, surveys degrade *and* attrition accelerates.

**Mitigations:**
- Targeted 1:1s with affected engineers — conversations, not surveys
- Fix structural causes (tooling friction, on-call load, unclear priorities)
- Do NOT manage to the survey number — Goodhart's law applies
- Cross-reference with signals 1+3+4 to find the *real* cause

### 6.5 Cross-signal red flags — Phase B

- **Senior burnout cascade:** high review-to-author + high after-hours + sentiment dropping → senior will leave within 2 quarters
- **Reviewer collapse:** Phase A's "batch size up + refactor down" + Phase B's "comments/LOC down on HIGH cohort" → confirms cause is review fatigue
- **Dark matter ghost:** low PR count + high design participation + high support participation → invisible-but-essential; promote them or risk losing them

### 6.6 Open decisions — Phase B

1. **Slack privacy scope.** Public engineering channels only? Or also incident channels (with redaction)? Or DMs (likely no)? Legal/security review required before ingestion.
2. **Channel-category mapping.** Hard-coded in Go config (mirroring `categorize_capitalization.go`) or DB-config-table (faster iteration)?
3. **Sentiment proxy method.** Behavioral-only (after-hours frequency, response-time drift) OR add tone analysis via LLM (more accurate, requires sending message text to a model, additional privacy review)?
4. **DXI survey integration.** Optional quarterly survey, or behavioral-proxy-only for now?

## 7. Phase C — Throughput cohort + Cost-per-outcome

### 7.1 Schemas

**Table `team_throughput_cohort`** — per team per ISO week per cohort:

| Column | Type |
|---|---|
| team_id | varchar |
| period_week | date |
| ai_cohort | enum |
| pr_count | int |
| feature_count | int |
| cycle_time_p50_hours | float |
| cycle_time_p90_hours | float |

**Table `team_cost_per_outcome`** — per team per ISO month:

| Column | Type |
|---|---|
| team_id | varchar |
| period_month | date |
| ai_token_spend_usd | decimal |
| ai_license_spend_usd | decimal |
| ai_infra_spend_usd | decimal |
| engineer_comp_share_usd | decimal |
| total_cost_usd | decimal |
| merged_prs | int |
| features_shipped | int |
| cost_per_merged_pr | decimal |
| cost_per_feature | decimal |

**Table `ai_roi_summary`** — per team per quarter:

| Column | Type |
|---|---|
| team_id | varchar |
| period_quarter | varchar (`YYYY-Q#`) |
| baseline_cost_per_pr | decimal (pre-AI reference) |
| actual_cost_per_pr | decimal |
| throughput_lift_pct | float |
| defect_cost_estimate | decimal |
| roi_pct | float |

### 7.2 Subtasks

1. `computeThroughputCohort` — joins PRs+cohort+features+team mapping; rolls up
2. `computeCostPerOutcome` — pulls from `findevops` cost tables, adds AI-vendor billing imports, joins to throughput
3. `computeAIROI` — computes integrated ROI per team per quarter

### 7.3 Dashboard — `AIROISummary.json`

Board-grade panels:
- AI ROI by team, current quarter (number panel, color-coded)
- Throughput cohort lift vs. defect cohort gap (scatter, one dot per team-quarter)
- Cost-per-feature trend over time (line, with R&D vs. operational split)
- Cost-per-merged-PR with AI cost as % share (stacked area)
- Quarterly narrative summary panel (text, hand-written each quarter)

### 7.4 Signal playbook — Phase C

#### PRs per engineer, cohorted by AI assistance

| State | Threshold | What it means |
|---|---|---|
| 🟢 Healthy | HIGH-cohort throughput 1.2–1.5× NONE; per-engineer variance modest | AI delivering real, distributed lift |
| 🟡 Warning | HIGH-cohort barely above NONE (≤1.1×), OR HIGH up but Phase A defect rate up | Productivity illusion forming |
| 🔴 Failure | HIGH-cohort *lower* than NONE (METR pattern), OR throughput-defect divergence >20% | AI is net negative for this team |

**Failure mode:** METR RCT result is real — experienced developers on mature codebases were 19% slower while feeling 20% faster. Your data shows you which case you're in.

**Mitigations:**
- Pair high-throughput with low-throughput engineers — transfer prompt patterns
- Internal prompt-pattern library for your codebase
- New-hire onboarding includes codebase-specific AI-tool training
- If HIGH < NONE: stop scaling investment; do root-cause retro

#### Feature shipping rate

| State | Threshold | What it means |
|---|---|---|
| 🟢 Healthy | Features/quarter trending up; cycle time trending down; both cohorts contributing | Real velocity hitting customers |
| 🟡 Warning | PRs up but features flat | Busy-work; engineers shipping non-aggregating code |
| 🔴 Failure | Features up + defect rate up + refactor down | Shipping more, worse, with growing debt |

**Mitigations:**
- Require issue-link discipline on PRs (CI enforced)
- Quarterly feature retro: shipped vs. not-shipped, why
- Pair sales/CS feedback with feature ship rate

#### Cost-per-merged-PR (all-in)

| State | Threshold | What it means |
|---|---|---|
| 🟢 Healthy | Cost-per-PR flat or declining; AI direct costs <5% of total eng cost | AI paying for itself |
| 🟡 Warning | Cost-per-PR rising without throughput gain | Tool spend not converting |
| 🔴 Failure | "Token maxing" — high tokens-per-PR + low throughput delta + flat features | Pragmatic Engineer's documented anti-pattern |

**Mitigations:**
- Switch per-seat to per-token billing if utilization low
- Token budget per team, surfaced on dashboard
- Investigate high-token-per-PR engineers — sometimes learning, sometimes gaming
- Never use raw token-spend as productivity proxy

#### Cost-per-feature (rolled up via `businessmetrics`)

| State | Threshold | What it means |
|---|---|---|
| 🟢 Healthy | Cost-per-feature declining QoQ; R&D clearly more expensive than maintenance | Genuine efficiency gain; finance can claim R&D credits |
| 🟡 Warning | Cost-per-feature flat or slightly up | AI investment hasn't translated to outcomes yet |
| 🔴 Failure | R&D and maintenance features cost the same | Capitalization opportunity missed → tax credit left on the table |

**Mitigations:**
- Quarterly feature retro: which were expensive, why
- Right-size feature scoping (most over-budget features under-scoped at start)
- Annual capitalization audit with finance

#### AI ROI — the integrated answer

`ROI = (additional_value_delivered − cost_of_AI) / cost_of_AI`

| State | Threshold | What it means |
|---|---|---|
| 🟢 Healthy | ROI > 100% sustained over 2+ quarters | AI investment justified |
| 🟡 Warning | ROI 30–100% | Marginal; investigate where gain is going |
| 🔴 Failure | ROI < 30% sustained, or negative | Stop scaling; root-cause retro |

**Failure mode:** Industry split — Anthropic claims +67% PRs; METR shows -19%. Your team is somewhere on the spectrum; you don't know where without measuring. This is the single most important question this platform exists to answer.

**Mitigations:**
- If ROI low: check Phase B verification effort — gain may be eaten by review load
- If ROI negative: pause investment, root-cause (skill / tooling fit / codebase maturity / process)
- If ROI high: scale carefully — team-level variance often hides

### 7.5 Cross-signal red flag — Phase C

**The productivity illusion (full pattern):** throughput up + cycle time down + Phase A defect rate up + Phase A refactor down + AI cost up → looks like a quarterly win, destroys long-term value. The most important pattern this entire platform exists to detect.

### 7.6 Open decisions — Phase C

1. **Baseline-cost-per-PR.** Use pre-AI historical baseline (pre-Q1 2025) or rolling 12-month average?
2. **Value attribution.** Throughput-based proxy (PRs × baseline cost) or feature-based (more accurate but requires `businessmetrics` linkage to be complete)?
3. **Defect cost estimate.** Industry-standard placeholder ($5k/incident) or organization-specific from past data?
4. **Cost ingestion.** Does Finally already pull Anthropic/OpenAI billing exports into DevLake, or does that need a new collector (separate scope)?

## 8. Cross-cutting concerns

### 8.1 Operating cadence

| Cadence | Audience | Focus | Duration |
|---|---|---|---|
| Weekly | Eng managers | Phase A defect signals, Phase B review-load, after-hours flags | 15 min |
| Monthly | CTO + eng managers | Cohort distribution, refactor ratio, sentiment trends, dark-matter contributors | 30 min |
| Quarterly | Board / exec | Phase C AI ROI summary, capitalization rollup | 60 min |
| Annually | With finance | R&D credit claim documentation | N/A |

### 8.2 What NOT to do with this data

- Don't make cohort-level metrics individual OKRs — engineers will game cohort assignment
- Don't publish per-engineer defect rates publicly — kills psychological safety
- Don't ban high cohorts — the goal is *quality at velocity*, not *less AI*
- Don't surveil message tone per-message — only aggregate roll-ups
- Don't blame after-hours engineers — fix the system, not the person
- Don't use dark-matter data punitively — for promoting *up*, never for managing *out*
- Don't reduce engineering productivity to a single ROI number for performance reviews
- Don't optimize for short-term cost-per-PR — incentivizes batching/gaming
- Don't measure ROI in a single quarter — needs 3–6 months minimum
- Don't share board-grade dashboards with the full engineering org — context will be lost

### 8.3 Privacy and audit

- All classification decisions are traceable to a versioned rule-set (`classifier_version` column on every output table)
- Slack message text is aggregated, never stored per-message at the user-facing layer; raw rows live in `_tool_slack_channel_messages` (existing) and are accessed read-only
- Per-engineer dashboards (Phase B sentiment, defect rate) are private — visible to the engineer and their manager only
- Quarterly classifier-version snapshots committed to git for SOX audit trail

## 9. Sequencing and timeline

| Phase | Effort | What ships | Dependencies |
|---|---|---|---|
| A — Quality cohorting | ~2 weeks | `pr_ai_cohort`, `pr_defect_signals`, `pr_change_composition` tables + AIQualityCohort dashboard | `aidetector` data (already in fork); commit-trailer parsing (new); optional incident data |
| B — Verification + dark matter | ~2 weeks | `engineer_verification_effort`, `engineer_slack_signals`, `engineer_dxi_proxy` tables + InvisibleWork dashboard | Phase A cohort table; `slack` plugin operational with mapped channels |
| C — Throughput + cost | ~1 week | `team_throughput_cohort`, `team_cost_per_outcome`, `ai_roi_summary` tables + AIROISummary dashboard | Phase A + B tables; `findevops` cost tables; AI vendor billing ingestion |

Each phase ships an independently useful dashboard. Stopping after Phase A still answers the industry's most-asked risk question (quality degradation). Stopping after Phase B answers the staffing/burnout question. Phase C is the integrated executive answer.

## 10. Out of scope / future work

- Real-time alerting (Phase D candidate)
- Cross-org benchmarking against industry data
- AI-generated narrative reports for board decks (would require LLM integration; defer)
- Integration with external HRIS for promotion-data export
- Multi-tenant: this design assumes one Finally engineering org
- Survey infrastructure for DXI — assume external tool for now (Culture Amp, Officevibe, or similar)

## 11. Open decisions summary

To resolve before Phase A implementation kicks off:

1. **Incident data source** — do we have Opsgenie/PagerDuty/Sentry in DevLake today?
2. **AI cohort threshold** — reuse `aidetector`'s 65 or recalibrate?
3. **Hotfix detection** — title regex + file overlap, or label convention?

Before Phase B:

4. **Engineer identity resolution coverage** — what % of Slack messages, PR reviews, and commits successfully map to a DevLake account today? If <90%, identity-resolution gaps need filling before per-engineer rollups are trustworthy.
5. **Slack privacy scope** — public eng channels only? Plus incident? Plus DMs (probably never)?
6. **Channel-category mapping** — Go config or DB config table?
7. **Sentiment proxy method** — behavioral-only or behavioral + LLM tone analysis?
8. **DXI survey integration** — quarterly survey now or later?
9. **Seniority labelling** — who maintains `aimeasure_engineer_roles`? (CTO assistant? HR feed? Annual snapshot?)

Before Phase C:

10. **Baseline-cost-per-PR** — pre-AI historical or rolling 12-month?
11. **Value attribution** — throughput proxy or feature-based?
12. **Defect cost estimate** — industry placeholder or org-specific?
13. **AI vendor billing ingestion** — exists or new collector needed?

---

## Appendix A — References

- DORA 2024 State of DevOps Report — https://dora.dev/research/2024/dora-report/
- DX AI Measurement Framework — https://getdx.com/whitepaper/ai-measurement-framework/
- DX Q4 2025 AI-assisted engineering impact report — https://getdx.com/blog/ai-assisted-engineering-q4-impact-report-2025/
- Atlassian State of Developer Experience 2025 — https://www.atlassian.com/teams/software-development/state-of-developer-experience-2025
- METR — Measuring early-2025 AI impact on experienced OS developers — https://metr.org/blog/2025-07-10-early-2025-ai-experienced-os-dev-study/
- Anthropic — How AI is transforming work at Anthropic — https://www.anthropic.com/research/how-ai-is-transforming-work-at-anthropic
- CodeRabbit — State of AI vs. Human Code Generation Report — https://www.coderabbit.ai/blog/state-of-ai-vs-human-code-generation-report
- GitClear — Coding on Copilot: Downward Pressure on Code Quality — https://www.gitclear.com/coding_on_copilot_data_shows_ais_downward_pressure_on_code_quality
- Salesforce Engineering — Scaling Code Reviews Amid AI-Generated Code — https://engineering.salesforce.com/scaling-code-reviews-adapting-to-a-surge-in-ai-generated-code/
- Forsgren — How to measure AI developer productivity in 2025 — https://www.lennysnewsletter.com/p/how-to-measure-ai-developer-productivity
- Cognition — Devin 2025 Performance Review — https://cognition.ai/blog/devin-annual-performance-review-2025
- Pragmatic Engineer — Token-maxing pattern at Meta/MS/Salesforce — https://tldrecap.tech/posts/2026/aie-europe/token-maximizing-big-tech-metrics/
