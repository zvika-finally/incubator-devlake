# Devlake MCP — Design Spec

**Status:** Approved design, ready for implementation plan
**Author:** Zvika Badalov (with Claude)
**Date:** 2026-05-18
**Tracks alongside:** [Superset MCP](https://github.com/zvika-finally/finally-internal/blob/main/docs/Tech%20Specs/superser_mcp_tech_spec.md) (operational pattern)

---

## Goal

Build a Devlake MCP server that gives Claude (in claude.ai and Claude Code) curated, board-credible engineering analytics over the existing Devlake warehouse. Operational pattern matches the existing Superset MCP one-for-one; the application code is net-new because Devlake — unlike Superset 6.1 — does not have an MCP server built in.

V1 is **read-only analytics**. No write/control-plane tools, no per-user data isolation, no survey ingest, no incident-data assumption.

## Non-goals (V1)

- Write/control-plane tools (enable plugin, trigger pipeline, edit blueprint). Defer to V2.
- Developer survey / DXI ingestion. No data source exists; the Phase B sentiment proxy is a behavioral approximation only.
- Reliability metrics (CFR, MTTR). Incident data is not extracted to the domain layer (`incidents = 1 row`). Tools surface the gap explicitly; computation lands when ingestion does.
- Auto-rollup by team. The org is single-team for V1; "team" = active engineers across the project. No `teams` / `team_users` seeding required.
- Project-scoped Phase B aggregates. The three aimeasure aggregate tables (`engineer_verification_effort`, `engineer_slack_signals`, `engineer_dxi_proxy`) lack a `project_name` column today. V1 surfaces this honestly; a follow-up project (`aimeasure-phase-b-plus-project-scoping`) adds it.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ Claude.ai / Claude Code (browser or CLI)                    │
│   OAuth dance against Okta → JWT with claims                │
│   aud ∈ {api://default, https://devlake-mcp.…/mcp}          │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼ HTTPS POST /mcp + Authorization: Bearer <jwt>
┌─────────────────────────────────────────────────────────────┐
│ Public ALB (internal.finally.com)                           │
│   Host: devlake-mcp.internal.finally.com → forward          │
│   No OIDC / no auth-proxy hop                               │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼ HTTP :5009
┌─────────────────────────────────────────────────────────────┐
│ ECS Fargate task: devlake-mcp                               │
│ ─────────────────────────────────────────────────────────── │
│ Python service (Anthropic MCP SDK + Starlette/Uvicorn)      │
│                                                             │
│   1. JWTAuthMiddleware — validates iss/aud/exp via JWKS     │
│      from MCP_JWKS_URI; same env-var contract as Superset   │
│   2. ToolRegistry — exposes the 13 V1 tools                 │
│   3. MetricCatalog — declarative YAML registry of the 16    │
│      V1 metrics, dimensions, measures, and scoping rules    │
│   4. MySQL client (SQLAlchemy) — reuses devlake credentials │
│      from existing Secrets Manager entry; SELECT-only via   │
│      AST guardrail + parameterized queries + catalog-driven │
│      construction                                           │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼ MySQL :3306
┌─────────────────────────────────────────────────────────────┐
│ Devlake RDS (finally-internal-devlake-mysql…rds.aws…)       │
│ Same DB read today via mysql-mcp; uses existing `devlake`   │
│ MySQL user. Code is the read-only boundary, not the role.   │
└─────────────────────────────────────────────────────────────┘
```

**What matches Superset MCP** (operational pattern):
- Direct ALB forward, JWT validated in-process (no auth-proxy hop).
- Identical env-var contract for JWT (`MCP_AUTH_ENABLED`, `MCP_JWT_ISSUER`, `MCP_JWT_AUDIENCE`, `MCP_JWKS_URI`, `MCP_JWT_ALGORITHM`, `MCP_RESOURCE_URL`).
- Same Okta issuer (`https://finally.okta.com/oauth2/default`).
- Same `.mcp.json` shape from the client side (with `callbackPort: 18089`, scope `devlake_api`).

**What differs by necessity**:
- New ECR repo `finally-internal/devlake-mcp` (different image; Devlake doesn't ship an MCP).
- MySQL access via reused `devlake` credentials (Section 4 trade-off).
- The 16-entry metric catalog is hand-written (Superset's came from Superset's codebase).

---

## V1 Tool Surface (13 tools)

### Discovery (4)

| # | Tool | Purpose |
|---|---|---|
| 1 | `list_projects()` | enumerate Devlake project entities, returns name+description |
| 2 | `list_engineers(project?, active_since?, search?)` | discover account_ids in scope, with last-seen timestamps |
| 3 | `list_metrics()` | catalog of queryable metrics + one-line descriptions |
| 4 | `describe_metric(name)` | full schema for a metric: dimensions, measures, units, scoping rule, known gaps |

### Curated (8)

| # | Tool | Returns | Reading from |
|---|---|---|---|
| 5 | `get_team_scorecard(project?, since, until, group_by?="week")` | velocity (PRs/issues), shipping (deploy freq, lead time p50/p95), reliability stub (until incidents land), code health (cq), AI contribution ratio | `project_pr_metrics`, `cicd_deployments`, `team_velocities`, `team_health_scores`, `cq_file_metrics`, `pr_ai_cohort`, `dora_benchmarks` |
| 6 | `get_engineer_week(engineer_id, period_week, project?)` | per-engineer week: PRs authored/reviewed, issues touched, commits, AI mix, Phase B aggregates if present | `pull_requests`, `pull_request_comments`, `issues`, `commits`, `pr_ai_cohort`, `engineer_verification_effort` |
| 7 | `get_reviewer_load(project?, since, until, top_n?=20)` | per-engineer review:author ratio, comments per LOC, after-hours proxy | `engineer_verification_effort`, `engineer_slack_signals` |
| 8 | `get_cycle_times(project?, since, until)` | issue→PR-open, PR-open→merge, merge→deploy; p50/p95/p99 + per-engineer | `issue_flow_metrics`, `project_pr_metrics`, `cicd_deployment_commits` |
| 9 | `get_investment_profile(project?, since, until)` | %-effort split by new features / KTLO / maintenance / tooling, $-weighted via FTE allocation | `business_initiatives`, `work_allocations`, `developer_monthly_fte`, `cost_allocations` |
| 10 | `get_ai_adoption(project?, since, until)` | % active engineers using AI tools, AI contribution ratio, AI rework rate | `claude_code_user_metrics`, `cursor_user_metrics`, `ai_churn_metrics`, `pr_ai_cohort`, `pr_defect_signals` |
| 11 | `get_initiatives(project?, since, until, status?)` | initiatives with effort, forecasted completion, ROI, AI cohort mix | `business_initiatives`, `initiative_forecasts`, `monte_carlo_forecasts`, `investment_rois`, `pull_request_issues` |
| 12 | `get_code_quality(project?, since, until)` | per-component coverage trends, code smells, complexity hotspots | `cq_file_metrics`, `cq_issues`, `cq_issue_impacts` |

### Generic (1)

| # | Tool | Purpose |
|---|---|---|
| 13 | `query_metric(metric, filters?, group_by?, since?, until?, project?, limit?=100)` | parameterised aggregate against any catalog entry; the escape hatch when no curated tool fits |

**Default project handling:** every tool takes `project` as an optional parameter, defaulting to `DEVLAKE_MCP_DEFAULT_PROJECT` env var (set to the new dedicated project at deploy time). Explicit override always allowed.

**Out of V1 (intentional):**
- Write/control-plane tools — separate design.
- Sentiment / "bad developer day" tool — depends on Slack data not yet ingested at sufficient volume.
- Raw `execute_sql` — `mysql-mcp` already exists for that; keeping devlake-mcp purely semantic.

---

## Metric Catalog (16 entries)

Single declarative YAML at `metrics.yaml`, loaded at server start. Schema:

```yaml
- name: <metric_id>
  description: <human-readable one-liner>
  source_tables: [<table>, …]
  primary_key: [<col>, …]
  time_column: <col_or_null>
  default_grain: <day|week|month|none>
  scoping: <project_column | project_mapping | global>
  project_column: <col>                # when scoping=project_column
  mapping_table: <board|repo>          # when scoping=project_mapping
  join_column: <col>                   # when scoping=project_mapping
  dimensions:
    - { name, type, joinable_to? }
  measures:
    - { name, type, unit, description }
  known_gaps: [<string>, …]
```

### V1 catalog entries

| # | Metric | Source table(s) | Scoping | Notes |
|---|---|---|---|---|
| 1 | `verification_effort` | `engineer_verification_effort` | global | Phase B aggregate; not project-scoped (known gap) |
| 2 | `slack_signals` | `engineer_slack_signals` | global | Phase B aggregate; depends on Slack ingest |
| 3 | `dxi_proxy` | `engineer_dxi_proxy` | global | behavioral proxy only — not a real survey |
| 4 | `pr_ai_cohort` | `pr_ai_cohort` | project_mapping (via PR→repo) | Phase A output |
| 5 | `pr_change_composition` | `pr_change_composition` | project_mapping | Phase A output |
| 6 | `pr_defect_signals` | `pr_defect_signals` | project_mapping | Phase A output; revert/hotfix proxies |
| 7 | `dora_pr_metrics` | `project_pr_metrics` | project_column | lead time + per-PR DORA fields |
| 8 | `deployments` | `cicd_deployments`, `cicd_deployment_commits` | project_mapping | 25k rows; well-populated |
| 9 | `business_initiatives` | `business_initiatives`, `initiative_forecasts` | project_mapping | 198 + 16,475 rows |
| 10 | `investment_rois` | `investment_rois`, `cost_allocations`, `monthly_cost_summaries`, `deployment_costs` | project_mapping (via initiative→project) | 95 + 699 + 34 + 273 rows |
| 11 | `ai_usage` | `ai_churn_metrics`, `claude_code_user_metrics`, `cursor_user_metrics` | project_column on `ai_churn_metrics`; global on others | aidetector + tool-usage |
| 12 | `team_health` | `team_health_scores` | project_column | 91 rows; pre-computed |
| 13 | `team_velocity` | `team_velocities` | project_column | 84 rows; pre-computed |
| 14 | `issue_flow` | `issue_flow_metrics` | project_column | 5,472 rows; pre-computed cycle times |
| 15 | `code_quality` | `cq_file_metrics`, `cq_issues`, `cq_issue_impacts` | project_mapping (via cq_project→repo) | SonarQube outputs |
| 16 | `fte_allocation` | `developer_monthly_fte`, `developer_baselines`, `work_allocations` | project_column on FTE; via initiative on `work_allocations` | underpins `get_investment_profile` |

**Deferred (V1.1):**
- `working_agreements` (working_agreements + compliance summaries) — niche DevEx metric.

**Design rules baked in:**
- Every measure declares its `unit` (minutes, count, ratio, usd, date) — Claude doesn't have to infer.
- Every entry declares its `scoping` rule — tools enforce project filtering correctly.
- `known_gaps` is first-class — when a metric returns nulls or "no data", the explanation comes from here, not from code.

---

## Auth + Identity (Section 5 of design)

### Okta-side (one-time, manual)

| Step | Why |
|---|---|
| Register API Services app `devlake-mcp` in Okta admin | Issues `clientId` for Claude clients |
| Add scope `devlake_api` to the existing `oauth2/default` authorization server | Token carries intent "for Devlake MCP" |
| Add `https://devlake-mcp.internal.finally.com/mcp` as a valid `aud` for the scope | Satisfies RFC 8707 Resource Indicators (claude.ai) |
| Assign Okta group `devlake-users` (or equivalent) to the app | Gates token issuance |

### Okta-side (Terraform)

- `aws_secretsmanager_secret.devlake_mcp_okta_oidc` storing discovery URLs (issuer, jwks, authorize, token). Mirrors `superset_okta_oidc` / `devlake_okta_oidc` blocks exactly.

### MCP service env vars

```
MCP_AUTH_ENABLED=true
MCP_JWT_ISSUER=https://finally.okta.com/oauth2/default
MCP_JWT_AUDIENCE=api://default,https://devlake-mcp.internal.finally.com/mcp
MCP_JWKS_URI=https://finally.okta.com/oauth2/default/v1/keys
MCP_JWT_ALGORITHM=RS256
MCP_RESOURCE_URL=https://devlake-mcp.internal.finally.com/mcp
DEVLAKE_MCP_DEFAULT_PROJECT=finally-DevEx-MCP        # the new dedicated project (name TBD)
```

Plus one secret reference (pulled into the task via `valueFrom`, never as plaintext):

```
DEVLAKE_MCP_DB_URL=<from finally-internal/devlake-mysql-password Secrets Manager entry>
```

### MCP service implementation

- `JWTAuthMiddleware` (ASGI middleware): validates `iss`/`aud`/`exp`/`iat`; multi-audience via comma-split; uses `pyjwt[crypto]` with a 5-minute JWKS cache.
- `/.well-known/oauth-protected-resource` endpoint served unauthenticated — RFC 9728 metadata so MCP clients can auto-discover.
- `WWW-Authenticate: Bearer resource_metadata="…"` header on 401s.
- Audit log: every tool dispatch records `claims.email`, `claims.sub`, `claims.iat`, tool name, parameters → CloudWatch.

### Client-side `.mcp.json` entry

```json
{
  "devlake": {
    "type": "http",
    "url": "https://devlake-mcp.internal.finally.com/mcp",
    "oauth": {
      "clientId": "<CLIENT_ID_FROM_OKTA>",
      "authServerMetadataUrl": "https://finally.okta.com/oauth2/default/.well-known/openid-configuration",
      "callbackPort": 18089,
      "scope": "openid devlake_api offline_access"
    }
  }
}
```

`callbackPort: 18089` is `+1` from Superset's `18088` to avoid collisions when a user runs both OAuth dances simultaneously.

---

## MySQL Access (Section 4 of design)

V1 reuses the existing `devlake` MySQL credentials. The MCP layer is the read-only boundary, not the DB role. Matches the existing `mysql-mcp` model.

**Code-level enforcement (3 layers):**
1. All queries built via SQLAlchemy **parameterized** statements — no string concatenation.
2. Catalog-driven query construction — tools can only emit query shapes the YAML allows.
3. SELECT-only **AST guardrail** on every emitted statement (sqlparse-based check; rejects non-SELECT).

**Query budget:** every query has a hard server-side cap `SET SESSION max_execution_time = 30000` (30s). Prevents a stuck query from holding a worker.

**Connection pool:** `create_engine(..., pool_size=5, max_overflow=10, pool_recycle=300)`.

**Why no separate read-only user (yet):** Defense-in-depth. The MCP is gated by Okta JWT; no anonymous access. Existing `mysql-mcp` uses the same model. A dedicated `devlake_mcp_ro` MySQL user can be added in V1.1 as a one-line SQL + Secrets Manager swap without code changes.

---

## Terraform Plan (Section 6 of design)

### New files

**`terraform/workloads/finally-internal/ecs-devlake-mcp.tf`** (~140 lines, structural clone of `ecs-superset-mcp.tf`):
- `aws_cloudwatch_log_group.devlake_mcp` → `/ecs/finally-internal/devlake-mcp`
- `aws_ecs_task_definition.devlake_mcp` — image from `finally-internal/devlake-mcp`, port 5009, env from Section 5
- `aws_ecs_service.devlake_mcp` — desired_count=1, FARGATE, attached to existing `aws_ecs_cluster.finally_internal`

**`terraform/workloads/finally-internal/ecr-devlake-mcp.tf`** (~30 lines):
- `aws_ecr_repository.devlake_mcp`
- `aws_ecr_lifecycle_policy.devlake_mcp` — retain last 10 images (match existing convention)

### Modified files

**`terraform/workloads/finally-internal/alb-public.tf`** (3 resources, ~90 lines added):
- `aws_lb_target_group.devlake_mcp_public` — port 5009, health check `/mcp`, matcher `200,401,405`
- `aws_lb_listener_rule.devlake_mcp` — priority 175, host header `devlake-mcp.${var.domain_name}`
- `aws_route53_record.devlake_mcp` — A alias to the public ALB

**`terraform/workloads/finally-internal/variables.tf`** (4 new variables):

```hcl
variable "enable_devlake_mcp"          { type = bool;   default = false }
variable "devlake_mcp_cpu"             { type = number; default = 512 }
variable "devlake_mcp_memory"          { type = number; default = 1024 }
variable "devlake_mcp_default_project" { type = string; default = "finally-DevEx-MCP" }
```

**`terraform/workloads/finally-internal/locals.tf`**: add `internal_tools.devlake_mcp` block with `image`, `port`, `cpu`, `memory` to mirror `internal_tools.superset`.

**`terraform/workloads/finally-internal/okta.tf`**: clone the `superset_okta_oidc` Secrets Manager block, three mechanical renames → `devlake_mcp_okta_oidc`.

### No changes needed

`ecs.tf`, `rds-devlake.tf`, `efs-devlake.tf`, `service-discovery-devlake.tf`, IAM roles (reuses `aws_iam_role.ecs_task_role` + `aws_iam_role.ecs_execution_role`).

---

## Application Layout

```
docker/devlake-mcp/
├── Dockerfile                # python:3.12-slim base
├── requirements.txt          # mcp, pyjwt[crypto], sqlalchemy, pymysql, pyyaml, starlette, uvicorn, sqlparse
├── README.md                 # build/run instructions
├── metrics.yaml              # the 16-metric catalog (Section 3)
└── app/
    ├── __init__.py
    ├── server.py             # main MCP server entry point (uvicorn + Starlette + MCP SDK)
    ├── auth.py               # JWTAuthMiddleware (Section 5)
    ├── db.py                 # SQLAlchemy engine, AST guardrail, connection pool
    ├── catalog.py            # loads metrics.yaml, validates filters
    ├── scoping.py            # 3-tier project filter helper (Section 3)
    └── tools/
        ├── discovery.py      # list_projects, list_engineers, list_metrics, describe_metric
        ├── team.py           # get_team_scorecard, get_reviewer_load, get_cycle_times, get_code_quality
        ├── engineer.py       # get_engineer_week
        ├── business.py       # get_investment_profile, get_ai_adoption, get_initiatives
        └── generic.py        # query_metric
```

---

## Order of Operations

1. **Manual Okta setup** — register app, scope, audience. ~15 min.
2. **Application code** — Dockerfile + Python server + YAML catalog + 13 tools + tests. ~1–2 weeks. Iterate locally against prod RDS via bastion/VPN with feature flag.
3. **ECR + Terraform** — push first image; apply TF. ~1–2 hours.
4. **Smoke test from claude.ai** — add `.mcp.json`, OAuth dance, call `list_projects()`, then `get_team_scorecard(project=…)`.
5. **Cutover** — set `DEVLAKE_MCP_DEFAULT_PROJECT` to the new dedicated project; validate end-to-end.

## Acceptance

- [ ] All 13 tools return valid JSON for at least one non-trivial input.
- [ ] `list_metrics()` returns all 16 catalog entries.
- [ ] `describe_metric(name)` correctly returns scoping rule and known_gaps for each entry.
- [ ] `get_team_scorecard()` returns sensible numbers for `project="finally-DevEx-MCP"` and labels reliability as "no incident data".
- [ ] Unauthenticated request to `/mcp` returns 401 + `WWW-Authenticate` header with `resource_metadata` hint.
- [ ] `/.well-known/oauth-protected-resource` returns RFC 9728 JSON unauthenticated.
- [ ] JWT with wrong audience is rejected; JWT with valid audience succeeds.
- [ ] Audit log captures `email`, `tool`, `params`, `duration_ms` for every call.
- [ ] No write to MySQL is possible — verified via AST guardrail test on `INSERT/UPDATE/DELETE/DROP/ALTER`.
- [ ] ECS task health check stays green for 24h.

## Known Gaps Carried into V1

| Gap | Impact | Resolution |
|---|---|---|
| Phase B aggregate tables not project-scoped | `verification_effort`, `slack_signals`, `dxi_proxy` are org-wide regardless of `project` parameter | Separate follow-up project: add `project_name` to the three tables + re-scope the aimeasure subtasks |
| `incidents = 1 row` | `get_team_scorecard` reliability fields return "no incident data" | Stand up PagerDuty/Opsgenie extractor; no MCP code change required |
| No developer survey | "DevEx" only available as Phase B behavioral proxy (`dxi_proxy`) | Separate ingestion project; out of MCP V1 scope |
| `claude_code_user_metrics` + `cursor_user_metrics` empty | `get_ai_adoption` falls back to aidetector signals | Track separately under aidetector roadmap |
| `pull_request_issues` covers ~40% of PRs | `get_initiatives` undercounts; some PRs not linked to issues | Tune Devlake's `linker` plugin regex / increase Jira key coverage |

## Open Questions

One non-blocking item:

- **Name of the new dedicated Devlake project.** The spec uses `finally-DevEx-MCP` as the default for `DEVLAKE_MCP_DEFAULT_PROJECT`. The user will create this project before MCP cutover; if a different name is chosen, only that one env-var default needs updating.

Everything else contentious was resolved during brainstorming.

## References

- Operational pattern: [Superset MCP Tech Spec](https://github.com/zvika-finally/finally-internal/blob/main/docs/Tech%20Specs/superser_mcp_tech_spec.md)
- Existing Terraform: `terraform/workloads/finally-internal/ecs-superset-mcp.tf` (the file to clone)
- MCP protocol: [Anthropic MCP spec (2025-06-18)](https://modelcontextprotocol.io/specification)
- RFC 8707 Resource Indicators, RFC 9728 OAuth Protected Resource Metadata
- 2026 metric frameworks: [Swarmia](https://www.swarmia.com/blog/engineering-metrics-for-leaders/), [LinearB CTO Board Template](https://linearb.io/resources/cto-board-slides), [Jellyfish KPIs](https://jellyfish.co/blog/engineering-kpis/), [DX Core 4](https://getdx.com/blog/engineering-kpis/)
