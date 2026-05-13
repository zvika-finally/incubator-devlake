# aimeasure plugin

Analytics layer that classifies merged PRs by AI assistance level and computes quality, verification, and cost signals on top of existing collector plugins (`aidetector`, `claudecode`, `cursor`, `slack`, `findevops`).

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

All three subtasks are idempotent: rerunning overwrites rows. `computeQualityCohort` recomputes rows whose `window_close_date` has not yet passed; once the 30-day window closes, the row is treated as frozen.

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

## Design reference

`docs/superpowers/specs/2026-05-13-ai-era-signals-design.md` § 5 (Phase A).
