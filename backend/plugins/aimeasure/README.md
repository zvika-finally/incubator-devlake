# aimeasure plugin

Analytics layer that classifies merged PRs by AI assistance level and computes quality, verification, and cost signals on top of existing collector plugins (`aidetector`, `cursor`, `slack`, `findevops`).

Phase A scope (this release):

- **classifyPRCohort** — writes one row per merged PR to `pr_ai_cohort` with one of NONE / LOW / MEDIUM / HIGH based on `aidetector` confidence + commit trailers
- **computeChangeComposition** — writes one row per merged PR to `pr_change_composition` with file count, additive vs refactor line breakdown, batch bucket (XS/S/M/L/XL)
- **computeQualityCohort** — writes one row per merged PR to `pr_defect_signals` indicating revert / hotfix / incident within 14 days

Dashboard: `grafana/dashboards/AIQualityCohort.json`.

## Configuration

Options accepted by the blueprint plan:

| Field | Default | Notes |
|---|---|---|
| `projectName` | required | DevLake project name |
| `highCohortThreshold` | 65 | Score ≥ this → MEDIUM (or HIGH via explicit signals) |
| `lowCohortThreshold` | 30 | Score ≥ this → LOW |
| `defectWindowDays` | 14 | Window for revert/hotfix/incident detection |

## Idempotency

All three subtasks are idempotent: rerunning overwrites rows via `CreateOrUpdate`. `computeQualityCohort` recomputes every merged PR on every run — defect signals can only become more accurate as the window matures, so reruns are safe. A future optimization is to skip rows whose `window_close_date` has passed (Phase B).

## Identity & seniority

The plugin never auto-classifies people. Seniority for per-role thresholds reads from `aimeasure_engineer_roles` (manual). Identity overrides for missing source→account mappings live in `aimeasure_account_overrides` (manual).

## Open decisions (Phase A)

These are not configurable yet — Phase A ships with the defaults. Revisit before Phase B:

1. **Incident data source.** If `issues` table is missing, `has_incident_14d` is always false and `incident_data_available` flag is set false. Phase B adds proper PagerDuty/Opsgenie joins.
2. **Hotfix detection.** Title regex `(?i)\b(hotfix|urgent|emergency|emergency-rollback)\b` + ≥50% file overlap. A label-based convention is the ideal upgrade path.
3. **Classifier version.** Currently `v1`. Increment in `tasks/task_data.go` when rules change.

## Tests

```
# unit
go test ./plugins/aimeasure/...

# e2e (requires DB; see docs/superpowers/plans/2026-05-13-aimeasure-phase-a.md Task 10 Step 3-4)
go test -run TestClassifyPRCohortDataFlow -v ./plugins/aimeasure/e2e/...
go test -run TestComputeChangeCompositionDataFlow -v ./plugins/aimeasure/e2e/...
go test -run TestComputeQualityCohortDataFlow -v ./plugins/aimeasure/e2e/...
```

## Phase B (verification effort + dark matter)

Three additional subtasks layered on top of the Phase A cohort:

- **computeVerificationEffort** — writes `engineer_verification_effort`. Per-engineer per-ISO-week aggregates of authoring vs reviewing time. Read-cost proxies are heuristics (see "Proxies" below).
- **computeSlackSignals** — writes `engineer_slack_signals`. Per-engineer per-week per-category Slack participation. Categories come from the manually-curated `aimeasure_slack_channel_categories` table; unmapped channels fall back to "general".
- **computeSentimentProxy** — writes `engineer_dxi_proxy`. Behavioral 0–100 sentiment score derived from after-hours ratio + review-to-author ratio + WoW message drop. **Phase B is behavioral-only — there is no message-content analysis.** Survey fields (`last_survey_date`, `last_survey_dxi`) stay nullable; ingest is Phase B+.

Dashboard: `grafana/dashboards/InvisibleWork.json` (7 panels).

### Proxies (Phase B heuristics, marked for replacement in Phase C)

| Quantity | Proxy |
|---|---|
| author_minutes(PR) | `clamp(loc/5, 10, 240)` |
| reviewer_minutes(PR, reviewer) | `clamp(15 + 2*num_review_comments, 10, 120)` |
| after_hours(t) | `weekday t outside 09:00–18:00 UTC, or weekend` |
| sentiment_score | `100 − 40·after_hours_ratio − 30·clamp((rta−1.5)/1.5,0,1) − 10·(wow_drop > 0.30)` |
| bad_developer_day_flag | `score < 50 OR (after_hours > 0.15 AND wow_drop > 0.50)` |

These are documented heuristics — they will be replaced by per-engineer timezone in Phase C and (optionally) tone-based sentiment in Phase B+.

### Slack channel categories

Edit `aimeasure_slack_channel_categories` directly (no admin UI yet). Rows look like:

```
channel_key,category,note
eng-platform,engineering,
inc-alerts,incident_support,
design-rfcs,design_architecture,
```

The `channel_key` is matched first against the channel name, then against the channel ID, then falls back to "general".

## Resolved decisions (Phase B)

All seven open decisions from the spec § 6.6 and the original plan draft are resolved. Rationale captured here so future readers know *why*.

1. **Slack scope: public engineering channels + private incident channels + private design channels.** No DMs. Aimeasure reads what the `slack` plugin already collects — operators are responsible for inviting the bot to private channels they want included. Aimeasure **does not** look at message text, only metadata (sender, timestamp, channel, thread relationship). Legal review covered Phase A; Phase B does not change the collection surface, only the analytics.
2. **Sentiment proxy: behavioral only.** Score = `f(after_hours_ratio, review_to_author_ratio, WoW message drop)`. **No LLM tone analysis.** Survey columns (`last_survey_date`, `last_survey_dxi`) stay nullable for a future ingest pipeline tracked in [issue #15](https://github.com/zvika-finally/incubator-devlake/issues/15).
3. **DXI survey ingest: deferred.** Columns ship empty in Phase B. The dedicated tracking issue (#15) lays out the work: source (Forms/Lattice/CSV), respondent→account_id mapping, k-anonymity, dashboard fallback when survey is fresh.
4. **Timezone: UTC for everyone in Phase B.** `IsAfterHours` uses UTC 09:00–18:00 Mon–Fri. Phase C adds an `aimeasure_engineer_timezones` table or directory-integration to fix non-UTC engineers; until then, engineers in non-UTC zones will appear to work "after hours" abnormally often. The dashboard documents this caveat in a panel description.
5. **Slack identity fallback: synthetic `slack:<userid>` for unmapped users.** Their rows appear in the dashboard as "unmapped" so operators can backfill mappings into `aimeasure_account_overrides` rather than silently losing data. Active unmapped users surface visibly on the per-engineer panels — that's a feature, not a bug.
6. **Channel→category mapping: DB config table (`aimeasure_slack_channel_categories`).** Operators edit via SQL; rerunning the subtask picks up changes immediately. No admin UI in Phase B — direct DB access is the workflow. A regex-based auto-classifier is a Phase C polish if curation becomes painful.
7. **Review-to-author ratio for pure reviewers: 0 (via `SafeRatio`).** A pure reviewer (`reviewer_minutes > 0, author_minutes = 0`) gets ratio=0 just like a pure author. **Caveat:** this collapses the senior-burnout signal on the ratio heatmap alone, so the dashboard adds a separate **"Reviewer minutes (raw)"** panel ranking engineers by `reviewer_minutes` regardless of authoring. The combination of "low ratio + high raw reviewer minutes" identifies pure reviewers; "high ratio + non-zero author minutes" identifies the spec's senior-burnout pattern. NULL was rejected because it breaks `AVG` aggregations across multiple downstream queries; a sentinel was rejected because it skews aggregations even worse.

## Design reference

`docs/superpowers/specs/2026-05-13-ai-era-signals-design.md` § 5 (Phase A).
