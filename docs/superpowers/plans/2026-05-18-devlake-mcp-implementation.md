# Devlake MCP V1 — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Devlake MCP server (Python sidecar, ECS-deployed, Okta-JWT-protected) that exposes 13 curated analytics tools over the existing Devlake MySQL — operationally identical to the existing Superset MCP, application code net-new.

**Architecture:** Standalone Python ASGI service (Starlette + Anthropic MCP SDK), reusing the `devlake` MySQL credentials with SELECT-only enforcement at the application layer, behind the existing public ALB with host-based routing on `devlake-mcp.internal.finally.com`. JWT validated in-process against Okta JWKS. Reads from 16 declarative metric catalog entries.

**Tech Stack:** Python 3.12, Anthropic MCP SDK, Starlette/Uvicorn, SQLAlchemy 2.x, PyMySQL, PyJWT (with crypto), PyYAML, sqlparse, pytest. Deploy via ECR + ECS Fargate + Terraform.

**Spec reference:** `docs/superpowers/specs/2026-05-18-devlake-mcp-design.md`

**Cross-repo:** Implementation lives in `finally-internal`:
- App code: `finally-internal/docker/devlake-mcp/`
- Terraform: `finally-internal/terraform/workloads/finally-internal/`

The plan paths below are **absolute** so the agent knows exactly where to write.

---

## File Structure

**New files in `finally-internal/docker/devlake-mcp/`:**

```
Dockerfile
requirements.txt
pyproject.toml
README.md
metrics.yaml
app/
  __init__.py
  server.py          # ASGI entrypoint; assembles middleware + MCP SDK
  config.py          # env-var loading, validation
  db.py              # SQLAlchemy engine, AST guardrail
  catalog.py         # YAML loader + Metric/Dimension/Measure dataclasses
  scoping.py         # 3-tier project filter helper
  auth.py            # JWTAuthMiddleware + JWKS cache
  metadata.py        # RFC 9728 /.well-known/oauth-protected-resource handler
  tools/
    __init__.py
    discovery.py     # list_projects, list_engineers, list_metrics, describe_metric
    team.py          # get_team_scorecard, get_reviewer_load, get_cycle_times, get_code_quality
    engineer.py      # get_engineer_week
    business.py      # get_investment_profile, get_ai_adoption, get_initiatives
    generic.py       # query_metric
tests/
  conftest.py
  test_db.py
  test_catalog.py
  test_scoping.py
  test_auth.py
  test_metadata.py
  test_tools_discovery.py
  test_tools_team.py
  test_tools_engineer.py
  test_tools_business.py
  test_tools_generic.py
```

**New files in `finally-internal/terraform/workloads/finally-internal/`:**

```
ecr-devlake-mcp.tf
ecs-devlake-mcp.tf
sql/devlake-mcp-bootstrap.sql       # documentation only; not executed by TF
```

**Modified files in `finally-internal/terraform/workloads/finally-internal/`:**

```
alb-public.tf       # +3 resources for target group, listener rule, route53
variables.tf        # +4 variables
locals.tf           # +1 internal_tools.devlake_mcp block
okta.tf             # +1 secret manager block cloning superset_okta_oidc
```

**Working directory abbreviations used below:**
- `$APP` = `/home/ubuntu/workspaces/finally/finally-internal/docker/devlake-mcp`
- `$TF` = `/home/ubuntu/workspaces/finally/finally-internal/terraform/workloads/finally-internal`

---

## Task 1 — Project skeleton

**Files:**
- Create: `$APP/Dockerfile`
- Create: `$APP/requirements.txt`
- Create: `$APP/pyproject.toml`
- Create: `$APP/README.md`
- Create: `$APP/app/__init__.py`
- Create: `$APP/tests/__init__.py`
- Create: `$APP/tests/conftest.py`

- [ ] **Step 1: Create `requirements.txt`**

```
mcp==1.6.1
starlette==0.39.2
uvicorn[standard]==0.30.6
sqlalchemy==2.0.30
pymysql==1.1.1
pyjwt[crypto]==2.9.0
pyyaml==6.0.2
sqlparse==0.5.1
httpx==0.27.2
```

- [ ] **Step 2: Create `pyproject.toml`** (pytest config only)

```toml
[tool.pytest.ini_options]
testpaths = ["tests"]
pythonpath = ["."]
addopts = "-ra -q --strict-markers"
markers = [
  "integration: requires live DB",
]
```

- [ ] **Step 3: Create `Dockerfile`**

```dockerfile
FROM python:3.12-slim

WORKDIR /app

COPY requirements.txt /app/
RUN pip install --no-cache-dir -r requirements.txt

COPY app /app/app
COPY metrics.yaml /app/metrics.yaml

ENV PYTHONUNBUFFERED=1
EXPOSE 5009

CMD ["uvicorn", "app.server:app", "--host", "0.0.0.0", "--port", "5009"]
```

- [ ] **Step 4: Create empty package markers**

`$APP/app/__init__.py`, `$APP/tests/__init__.py`: each empty file (one newline).

- [ ] **Step 5: Create `tests/conftest.py`** (shared pytest fixtures)

```python
import pytest


@pytest.fixture
def fake_env(monkeypatch):
    """Minimal env vars for unit tests; integration tests override."""
    monkeypatch.setenv("DEVLAKE_MCP_DB_URL", "sqlite+pysqlite:///:memory:")
    monkeypatch.setenv("DEVLAKE_MCP_DEFAULT_PROJECT", "test-project")
    monkeypatch.setenv("MCP_AUTH_ENABLED", "false")
    monkeypatch.setenv("MCP_JWT_ISSUER", "https://test.okta.com/oauth2/default")
    monkeypatch.setenv("MCP_JWT_AUDIENCE", "api://default")
    monkeypatch.setenv("MCP_JWKS_URI", "https://test.okta.com/oauth2/default/v1/keys")
    monkeypatch.setenv("MCP_JWT_ALGORITHM", "RS256")
    monkeypatch.setenv("MCP_RESOURCE_URL", "https://test.local/mcp")
```

- [ ] **Step 6: Create `README.md`**

```markdown
# devlake-mcp

MCP server exposing curated Devlake analytics tools to Claude over HTTPS+JWT.

## Local dev

```bash
pip install -r requirements.txt
export DEVLAKE_MCP_DB_URL='mysql+pymysql://devlake:...@host/devlake'
export MCP_AUTH_ENABLED=false
uvicorn app.server:app --reload --port 5009
```

## Tests

`pytest`. Integration tests gated behind `-m integration`.

## Deploy

ECR push + Terraform apply in `finally-internal`. See `docs/superpowers/plans/2026-05-18-devlake-mcp-implementation.md`.
```

- [ ] **Step 7: Verify `pip install` works in a fresh venv**

```bash
cd $APP && python -m venv .venv && source .venv/bin/activate && pip install -r requirements.txt
```

Expected: no errors. Exit cleanly.

- [ ] **Step 8: Commit**

```bash
cd /home/ubuntu/workspaces/finally/finally-internal
git checkout -b devlake-mcp
git add docker/devlake-mcp/
git commit -m "feat(devlake-mcp): project skeleton"
```

---

## Task 2 — `config.py` (env-var loader)

**Files:**
- Create: `$APP/app/config.py`
- Create: `$APP/tests/test_config.py`

- [ ] **Step 1: Write failing test**

`$APP/tests/test_config.py`:

```python
import os
import pytest
from app.config import Config


def test_config_loads_from_env(fake_env):
    cfg = Config.from_env()
    assert cfg.db_url == "sqlite+pysqlite:///:memory:"
    assert cfg.default_project == "test-project"
    assert cfg.auth_enabled is False
    assert cfg.jwt_issuer == "https://test.okta.com/oauth2/default"
    assert cfg.jwt_audiences == ["api://default"]


def test_config_audience_split(monkeypatch, fake_env):
    monkeypatch.setenv("MCP_JWT_AUDIENCE", "api://default,https://x.com/mcp")
    cfg = Config.from_env()
    assert cfg.jwt_audiences == ["api://default", "https://x.com/mcp"]


def test_config_requires_db_url(monkeypatch):
    monkeypatch.delenv("DEVLAKE_MCP_DB_URL", raising=False)
    with pytest.raises(RuntimeError, match="DEVLAKE_MCP_DB_URL"):
        Config.from_env()
```

- [ ] **Step 2: Run test, expect FAIL**

```bash
cd $APP && pytest tests/test_config.py -v
```

Expected: `ModuleNotFoundError: No module named 'app.config'`.

- [ ] **Step 3: Implement `config.py`**

```python
import os
from dataclasses import dataclass


@dataclass(frozen=True)
class Config:
    db_url: str
    default_project: str
    auth_enabled: bool
    jwt_issuer: str
    jwt_audiences: list[str]
    jwks_uri: str
    jwt_algorithm: str
    resource_url: str

    @classmethod
    def from_env(cls) -> "Config":
        def req(key: str) -> str:
            v = os.environ.get(key)
            if not v:
                raise RuntimeError(f"missing required env var: {key}")
            return v

        return cls(
            db_url=req("DEVLAKE_MCP_DB_URL"),
            default_project=req("DEVLAKE_MCP_DEFAULT_PROJECT"),
            auth_enabled=os.environ.get("MCP_AUTH_ENABLED", "true").lower() == "true",
            jwt_issuer=req("MCP_JWT_ISSUER"),
            jwt_audiences=[a.strip() for a in req("MCP_JWT_AUDIENCE").split(",") if a.strip()],
            jwks_uri=req("MCP_JWKS_URI"),
            jwt_algorithm=os.environ.get("MCP_JWT_ALGORITHM", "RS256"),
            resource_url=req("MCP_RESOURCE_URL"),
        )
```

- [ ] **Step 4: Verify tests pass**

`pytest tests/test_config.py -v` → 3 PASS.

- [ ] **Step 5: Commit**

```bash
git add docker/devlake-mcp/app/config.py docker/devlake-mcp/tests/test_config.py
git commit -m "feat(devlake-mcp): env-var Config loader"
```

---

## Task 3 — `db.py` (SQLAlchemy engine + SELECT-only AST guardrail)

**Files:**
- Create: `$APP/app/db.py`
- Create: `$APP/tests/test_db.py`

- [ ] **Step 1: Write failing tests**

`$APP/tests/test_db.py`:

```python
import pytest
from sqlalchemy import text
from app.db import Database, NonSelectQueryError


def test_database_runs_select(fake_env):
    db = Database.from_url("sqlite+pysqlite:///:memory:")
    rows = db.fetch_all(text("SELECT 1 AS n"))
    assert rows == [{"n": 1}]


def test_database_rejects_insert(fake_env):
    db = Database.from_url("sqlite+pysqlite:///:memory:")
    with pytest.raises(NonSelectQueryError):
        db.fetch_all(text("INSERT INTO t VALUES (1)"))


def test_database_rejects_update(fake_env):
    db = Database.from_url("sqlite+pysqlite:///:memory:")
    with pytest.raises(NonSelectQueryError):
        db.fetch_all(text("UPDATE t SET x = 1"))


def test_database_rejects_delete(fake_env):
    db = Database.from_url("sqlite+pysqlite:///:memory:")
    with pytest.raises(NonSelectQueryError):
        db.fetch_all(text("DELETE FROM t"))


def test_database_rejects_drop(fake_env):
    db = Database.from_url("sqlite+pysqlite:///:memory:")
    with pytest.raises(NonSelectQueryError):
        db.fetch_all(text("DROP TABLE t"))


def test_database_allows_show(fake_env):
    # SHOW is a SELECT-equivalent introspection; allow.
    db = Database.from_url("sqlite+pysqlite:///:memory:")
    # sqlite doesn't support SHOW so we just verify the gate doesn't reject it
    from app.db import _is_readonly
    assert _is_readonly("SHOW TABLES") is True


def test_database_rejects_multi_statement(fake_env):
    db = Database.from_url("sqlite+pysqlite:///:memory:")
    with pytest.raises(NonSelectQueryError):
        db.fetch_all(text("SELECT 1; DROP TABLE t"))
```

- [ ] **Step 2: Run, expect FAIL**

```bash
pytest tests/test_db.py -v
```

Expected: `ModuleNotFoundError`.

- [ ] **Step 3: Implement `db.py`**

```python
import sqlparse
from sqlalchemy import create_engine
from sqlalchemy.engine import Engine
from sqlalchemy.sql import ClauseElement
from typing import Any


class NonSelectQueryError(RuntimeError):
    """Raised when the AST guardrail detects a non-SELECT statement."""


_ALLOWED_TYPES = {"SELECT", "SHOW", "DESCRIBE", "EXPLAIN"}


def _is_readonly(sql: str) -> bool:
    """Return True iff every statement in the SQL string is read-only."""
    statements = [s for s in sqlparse.parse(sql) if str(s).strip()]
    if not statements:
        return False
    if len(statements) > 1:
        return False
    stmt = statements[0]
    return stmt.get_type().upper() in _ALLOWED_TYPES


class Database:
    def __init__(self, engine: Engine) -> None:
        self._engine = engine

    @classmethod
    def from_url(cls, url: str) -> "Database":
        engine = create_engine(
            url,
            pool_size=5,
            max_overflow=10,
            pool_recycle=300,
            future=True,
        )
        return cls(engine)

    def fetch_all(self, statement: ClauseElement | Any) -> list[dict]:
        sql_text = str(statement.compile(compile_kwargs={"literal_binds": False})) \
            if hasattr(statement, "compile") else str(statement)
        if not _is_readonly(sql_text):
            raise NonSelectQueryError(f"non-readonly statement rejected: {sql_text[:120]!r}")
        with self._engine.connect() as conn:
            result = conn.execute(statement)
            return [dict(row._mapping) for row in result.fetchall()]
```

- [ ] **Step 4: Verify tests pass**

```bash
pytest tests/test_db.py -v
```

Expected: all 7 PASS.

- [ ] **Step 5: Commit**

```bash
git add docker/devlake-mcp/app/db.py docker/devlake-mcp/tests/test_db.py
git commit -m "feat(devlake-mcp): db wrapper with SELECT-only AST guardrail"
```

---

## Task 4 — `catalog.py` (YAML loader + dataclasses)

**Files:**
- Create: `$APP/app/catalog.py`
- Create: `$APP/tests/test_catalog.py`
- Create: `$APP/tests/fixtures/sample_metrics.yaml`

- [ ] **Step 1: Create the sample YAML fixture**

`$APP/tests/fixtures/sample_metrics.yaml`:

```yaml
- name: verification_effort
  description: Per-engineer per-week effort.
  source_tables: [engineer_verification_effort]
  primary_key: [engineer_id, period_week]
  time_column: period_week
  default_grain: week
  scoping: global
  dimensions:
    - { name: engineer_id, type: account_id }
    - { name: period_week, type: date }
  measures:
    - { name: author_minutes, type: int, unit: minutes, description: "Authoring time" }
  known_gaps: []

- name: dora_pr_metrics
  description: DORA per-PR metrics.
  source_tables: [project_pr_metrics]
  primary_key: [id]
  time_column: pr_created_date
  default_grain: day
  scoping: project_column
  project_column: project_name
  dimensions: []
  measures:
    - { name: lead_time_minutes, type: int, unit: minutes, description: "Lead time" }
  known_gaps: []
```

- [ ] **Step 2: Write failing tests**

`$APP/tests/test_catalog.py`:

```python
import pytest
from pathlib import Path
from app.catalog import MetricCatalog, Scoping


FIXTURE = Path(__file__).parent / "fixtures" / "sample_metrics.yaml"


def test_catalog_loads():
    cat = MetricCatalog.from_file(FIXTURE)
    assert len(cat.metrics) == 2
    assert {m.name for m in cat.metrics} == {"verification_effort", "dora_pr_metrics"}


def test_catalog_get_by_name():
    cat = MetricCatalog.from_file(FIXTURE)
    m = cat.get("verification_effort")
    assert m.source_tables == ["engineer_verification_effort"]
    assert m.scoping == Scoping.GLOBAL
    assert m.measures[0].unit == "minutes"


def test_catalog_get_missing_raises():
    cat = MetricCatalog.from_file(FIXTURE)
    with pytest.raises(KeyError):
        cat.get("nonexistent")


def test_catalog_scoping_parsed():
    cat = MetricCatalog.from_file(FIXTURE)
    assert cat.get("dora_pr_metrics").scoping == Scoping.PROJECT_COLUMN
    assert cat.get("dora_pr_metrics").project_column == "project_name"


def test_catalog_validates_unit_present():
    bad = "- name: x\n  description: x\n  source_tables: [t]\n  primary_key: [id]\n  time_column: null\n  default_grain: none\n  scoping: global\n  dimensions: []\n  measures:\n    - { name: c, type: int }\n  known_gaps: []\n"
    import tempfile
    with tempfile.NamedTemporaryFile("w", suffix=".yaml", delete=False) as f:
        f.write(bad)
        f.flush()
        with pytest.raises(ValueError, match="unit"):
            MetricCatalog.from_file(Path(f.name))
```

- [ ] **Step 3: Run, expect FAIL**

```bash
pytest tests/test_catalog.py -v
```

- [ ] **Step 4: Implement `catalog.py`**

```python
from dataclasses import dataclass, field
from enum import Enum
from pathlib import Path
import yaml


class Scoping(str, Enum):
    PROJECT_COLUMN = "project_column"
    PROJECT_MAPPING = "project_mapping"
    GLOBAL = "global"


@dataclass(frozen=True)
class Dimension:
    name: str
    type: str
    joinable_to: str | None = None


@dataclass(frozen=True)
class Measure:
    name: str
    type: str
    unit: str
    description: str = ""


@dataclass(frozen=True)
class Metric:
    name: str
    description: str
    source_tables: list[str]
    primary_key: list[str]
    time_column: str | None
    default_grain: str
    scoping: Scoping
    dimensions: list[Dimension]
    measures: list[Measure]
    known_gaps: list[str] = field(default_factory=list)
    project_column: str | None = None
    mapping_table: str | None = None
    join_column: str | None = None


class MetricCatalog:
    def __init__(self, metrics: list[Metric]) -> None:
        self.metrics = metrics
        self._by_name = {m.name: m for m in metrics}

    @classmethod
    def from_file(cls, path: Path) -> "MetricCatalog":
        data = yaml.safe_load(path.read_text())
        metrics = [cls._parse_metric(entry) for entry in data]
        return cls(metrics)

    @staticmethod
    def _parse_metric(entry: dict) -> Metric:
        measures = []
        for m in entry["measures"]:
            if "unit" not in m:
                raise ValueError(f"measure {m.get('name')!r} missing required 'unit' field")
            measures.append(Measure(
                name=m["name"],
                type=m["type"],
                unit=m["unit"],
                description=m.get("description", ""),
            ))
        dims = [
            Dimension(name=d["name"], type=d["type"], joinable_to=d.get("joinable_to"))
            for d in entry.get("dimensions", [])
        ]
        return Metric(
            name=entry["name"],
            description=entry["description"],
            source_tables=entry["source_tables"],
            primary_key=entry["primary_key"],
            time_column=entry.get("time_column"),
            default_grain=entry["default_grain"],
            scoping=Scoping(entry["scoping"]),
            dimensions=dims,
            measures=measures,
            known_gaps=entry.get("known_gaps", []),
            project_column=entry.get("project_column"),
            mapping_table=entry.get("mapping_table"),
            join_column=entry.get("join_column"),
        )

    def get(self, name: str) -> Metric:
        if name not in self._by_name:
            raise KeyError(f"metric not in catalog: {name!r}")
        return self._by_name[name]

    def names(self) -> list[str]:
        return [m.name for m in self.metrics]
```

- [ ] **Step 5: Verify tests pass**

```bash
pytest tests/test_catalog.py -v
```

Expected: 5 PASS.

- [ ] **Step 6: Commit**

```bash
git add docker/devlake-mcp/app/catalog.py docker/devlake-mcp/tests/test_catalog.py docker/devlake-mcp/tests/fixtures/
git commit -m "feat(devlake-mcp): metric catalog YAML loader + dataclasses"
```

---

## Task 5 — Write `metrics.yaml` (16 entries)

**Files:**
- Create: `$APP/metrics.yaml`
- Create: `$APP/tests/test_metrics_yaml.py`

- [ ] **Step 1: Write the production catalog**

`$APP/metrics.yaml` — full 16 entries. Each follows the schema from Task 4. Below is the complete content:

```yaml
- name: verification_effort
  description: "Per-engineer per-ISO-week authoring vs reviewing time and AI-cohort comment density (Phase B aimeasure)."
  source_tables: [engineer_verification_effort]
  primary_key: [engineer_id, period_week]
  time_column: period_week
  default_grain: week
  scoping: global
  dimensions:
    - { name: engineer_id, type: account_id, joinable_to: "accounts.id" }
    - { name: period_week, type: date }
  measures:
    - { name: author_minutes, type: int, unit: minutes, description: "Proxy authoring time (LOC/5 clipped [10,240])" }
    - { name: reviewer_minutes, type: int, unit: minutes, description: "Proxy review time (15+2*comments clipped [10,120])" }
    - { name: review_to_author_ratio, type: float, unit: ratio, description: "reviewer/author; 0 when denominator is 0" }
    - { name: review_comments_total, type: int, unit: count }
    - { name: review_comments_per_loc, type: float, unit: ratio }
    - { name: review_comments_high_cohort, type: int, unit: count, description: "Comments on HIGH-AI-cohort PRs" }
    - { name: review_comments_per_loc_high, type: float, unit: ratio }
  known_gaps:
    - "Not project-scoped — spans all PRs regardless of project parameter (follow-up: add project_name column to Phase B aggregate tables)."

- name: slack_signals
  description: "Per-engineer per-week per-category Slack participation (Phase B aimeasure)."
  source_tables: [engineer_slack_signals]
  primary_key: [engineer_id, period_week, channel_category]
  time_column: period_week
  default_grain: week
  scoping: global
  dimensions:
    - { name: engineer_id, type: account_id }
    - { name: period_week, type: date }
    - { name: channel_category, type: string }
  measures:
    - { name: message_count, type: int, unit: count }
    - { name: thread_participation_count, type: int, unit: count }
    - { name: after_hours_message_count, type: int, unit: count }
    - { name: after_hours_ratio, type: float, unit: ratio }
  known_gaps:
    - "Not project-scoped."
    - "Currently 0 rows; depends on Slack ingestion landing volume."

- name: dxi_proxy
  description: "Behavioral 0-100 sentiment score per-engineer per-week (Phase B aimeasure). Behavioral proxy only — NOT a real DXI survey."
  source_tables: [engineer_dxi_proxy]
  primary_key: [engineer_id, period_week]
  time_column: period_week
  default_grain: week
  scoping: global
  dimensions:
    - { name: engineer_id, type: account_id }
    - { name: period_week, type: date }
  measures:
    - { name: sentiment_score, type: float, unit: score_0_100 }
    - { name: bad_developer_day_flag, type: bool, unit: bool }
  known_gaps:
    - "Behavioral proxy, not survey-based — do not present as DXI score to leadership."
    - "Not project-scoped."

- name: pr_ai_cohort
  description: "Per-PR AI cohort classification (NONE/LOW/MEDIUM/HIGH) from aimeasure Phase A."
  source_tables: [pr_ai_cohort]
  primary_key: [pr_id]
  time_column: classified_at
  default_grain: day
  scoping: project_mapping
  mapping_table: boards
  join_column: base_repo_id
  dimensions:
    - { name: pr_id, type: string, joinable_to: "pull_requests.id" }
    - { name: ai_cohort, type: enum }
  measures:
    - { name: confidence_score, type: int, unit: score_0_100 }
    - { name: has_explicit_marker, type: bool, unit: bool }
    - { name: has_commit_trailer, type: bool, unit: bool }
  known_gaps: []

- name: pr_change_composition
  description: "Per-PR change composition: refactor ratio, batch size bucket, additive lines."
  source_tables: [pr_change_composition]
  primary_key: [pr_id]
  time_column: null
  default_grain: none
  scoping: project_mapping
  mapping_table: boards
  join_column: base_repo_id
  dimensions:
    - { name: pr_id, type: string }
    - { name: batch_size_bucket, type: enum }
  measures:
    - { name: additions, type: int, unit: lines }
    - { name: deletions, type: int, unit: lines }
    - { name: refactor_ratio, type: float, unit: ratio }
  known_gaps: []

- name: pr_defect_signals
  description: "Per-PR defect proxies: revert/hotfix/incident within 14 days."
  source_tables: [pr_defect_signals]
  primary_key: [pr_id]
  time_column: window_close_date
  default_grain: day
  scoping: project_mapping
  mapping_table: boards
  join_column: base_repo_id
  dimensions:
    - { name: pr_id, type: string }
  measures:
    - { name: has_revert14d, type: bool, unit: bool }
    - { name: has_hotfix14d, type: bool, unit: bool }
    - { name: has_incident14d, type: bool, unit: bool }
    - { name: total_defect_count, type: int, unit: count }
  known_gaps:
    - "Incident signals depend on incidents table (currently sparse: 1 row in prod)."

- name: dora_pr_metrics
  description: "DORA-style per-PR delivery metrics from project_pr_metrics."
  source_tables: [project_pr_metrics]
  primary_key: [id]
  time_column: pr_created_date
  default_grain: day
  scoping: project_column
  project_column: project_name
  dimensions:
    - { name: pr_id, type: string, joinable_to: "pull_requests.id" }
  measures:
    - { name: coding_time_minutes, type: int, unit: minutes }
    - { name: pickup_time_minutes, type: int, unit: minutes }
    - { name: review_time_minutes, type: int, unit: minutes }
    - { name: deploy_time_minutes, type: int, unit: minutes }
    - { name: change_lead_time_minutes, type: int, unit: minutes }
  known_gaps: []

- name: deployments
  description: "CICD deployments (merge-to-production)."
  source_tables: [cicd_deployments, cicd_deployment_commits]
  primary_key: [id]
  time_column: started_date
  default_grain: day
  scoping: project_mapping
  mapping_table: cicd_scopes
  join_column: cicd_scope_id
  dimensions:
    - { name: environment, type: enum }
    - { name: result, type: enum }
  measures:
    - { name: duration_sec, type: int, unit: seconds }
  known_gaps: []

- name: business_initiatives
  description: "Business initiatives extracted from Jira epics with effort + forecasts."
  source_tables: [business_initiatives, initiative_forecasts]
  primary_key: [initiative_id]
  time_column: created_date
  default_grain: month
  scoping: project_mapping
  mapping_table: boards
  join_column: board_id
  dimensions:
    - { name: initiative_id, type: string }
    - { name: status, type: enum }
    - { name: stage, type: enum }
  measures:
    - { name: effort_story_points, type: float, unit: story_points }
    - { name: forecast_completion_date, type: date, unit: date }
  known_gaps: []

- name: investment_rois
  description: "ROI per initiative + cost allocations + monthly summaries (findevops + businessmetrics)."
  source_tables: [investment_rois, cost_allocations, monthly_cost_summaries, deployment_costs]
  primary_key: [initiative_id]
  time_column: period_month
  default_grain: month
  scoping: project_mapping
  mapping_table: boards
  join_column: initiative_id
  dimensions:
    - { name: initiative_id, type: string }
    - { name: category, type: enum, joinable_to: "ASC 350-40 stages" }
  measures:
    - { name: total_cost_usd, type: float, unit: usd }
    - { name: forecasted_value_usd, type: float, unit: usd }
    - { name: roi_pct, type: float, unit: percent }
  known_gaps: []

- name: ai_usage
  description: "AI tool adoption per engineer (Claude Code + Cursor + aidetector signals)."
  source_tables: [ai_churn_metrics, claude_code_user_metrics, cursor_user_metrics]
  primary_key: [account_id, period_month]
  time_column: period_month
  default_grain: month
  scoping: project_column
  project_column: project_name
  dimensions:
    - { name: account_id, type: account_id }
    - { name: tool, type: enum, joinable_to: "claude_code|cursor|other" }
  measures:
    - { name: messages_sent, type: int, unit: count }
    - { name: lines_accepted, type: int, unit: lines }
    - { name: ai_contribution_ratio, type: float, unit: ratio }
  known_gaps:
    - "claude_code_user_metrics and cursor_user_metrics are currently empty in prod."

- name: team_health
  description: "Pre-computed per-period team health score (businessmetrics)."
  source_tables: [team_health_scores]
  primary_key: [project_name, period_month]
  time_column: period_month
  default_grain: month
  scoping: project_column
  project_column: project_name
  dimensions:
    - { name: period_month, type: date }
  measures:
    - { name: health_score, type: float, unit: score_0_100 }
    - { name: health_level, type: enum, unit: enum, description: "elite|high|medium|low" }
  known_gaps: []

- name: team_velocity
  description: "Pre-computed per-period team velocity (businessmetrics)."
  source_tables: [team_velocities]
  primary_key: [project_name, period_month]
  time_column: period_month
  default_grain: month
  scoping: project_column
  project_column: project_name
  dimensions:
    - { name: period_month, type: date }
  measures:
    - { name: story_points_completed, type: float, unit: story_points }
    - { name: prs_merged, type: int, unit: count }
  known_gaps: []

- name: issue_flow
  description: "Pre-computed cycle times per issue (issue_flow_metrics)."
  source_tables: [issue_flow_metrics]
  primary_key: [issue_id]
  time_column: resolution_date
  default_grain: day
  scoping: project_column
  project_column: project_name
  dimensions:
    - { name: issue_id, type: string, joinable_to: "issues.id" }
    - { name: assignee_id, type: account_id }
  measures:
    - { name: lead_time_minutes, type: int, unit: minutes }
    - { name: cycle_time_minutes, type: int, unit: minutes }
  known_gaps: []

- name: code_quality
  description: "Code quality from SonarQube extraction (cq_*)."
  source_tables: [cq_file_metrics, cq_issues, cq_issue_impacts]
  primary_key: [file_id]
  time_column: created_date
  default_grain: month
  scoping: project_mapping
  mapping_table: cq_projects
  join_column: cq_project_id
  dimensions:
    - { name: cq_project_id, type: string }
    - { name: severity, type: enum }
  measures:
    - { name: code_smells, type: int, unit: count }
    - { name: bugs, type: int, unit: count }
    - { name: vulnerabilities, type: int, unit: count }
    - { name: coverage, type: float, unit: percent }
    - { name: complexity, type: int, unit: count }
  known_gaps: []

- name: fte_allocation
  description: "FTE allocation per engineer per month + work allocation by category."
  source_tables: [developer_monthly_fte, developer_baselines, work_allocations]
  primary_key: [account_id, period_month]
  time_column: period_month
  default_grain: month
  scoping: project_column
  project_column: project_name
  dimensions:
    - { name: account_id, type: account_id }
    - { name: category, type: enum }
  measures:
    - { name: fte_fraction, type: float, unit: ratio }
    - { name: allocation_pct, type: float, unit: percent }
  known_gaps: []
```

- [ ] **Step 2: Write loading test**

`$APP/tests/test_metrics_yaml.py`:

```python
from pathlib import Path
from app.catalog import MetricCatalog, Scoping


YAML = Path(__file__).parent.parent / "metrics.yaml"


def test_production_catalog_loads():
    cat = MetricCatalog.from_file(YAML)
    assert len(cat.metrics) == 16


def test_all_16_named():
    cat = MetricCatalog.from_file(YAML)
    expected = {
        "verification_effort", "slack_signals", "dxi_proxy",
        "pr_ai_cohort", "pr_change_composition", "pr_defect_signals",
        "dora_pr_metrics", "deployments",
        "business_initiatives", "investment_rois", "ai_usage",
        "team_health", "team_velocity", "issue_flow",
        "code_quality", "fte_allocation",
    }
    assert {m.name for m in cat.metrics} == expected


def test_every_measure_has_unit():
    cat = MetricCatalog.from_file(YAML)
    for m in cat.metrics:
        for measure in m.measures:
            assert measure.unit, f"measure {m.name}.{measure.name} missing unit"


def test_scoping_distribution():
    cat = MetricCatalog.from_file(YAML)
    counts = {Scoping.GLOBAL: 0, Scoping.PROJECT_COLUMN: 0, Scoping.PROJECT_MAPPING: 0}
    for m in cat.metrics:
        counts[m.scoping] += 1
    assert counts[Scoping.GLOBAL] == 3            # Phase B aggregates
    assert counts[Scoping.PROJECT_COLUMN] >= 6
    assert counts[Scoping.PROJECT_MAPPING] >= 5
```

- [ ] **Step 3: Run tests, verify pass**

```bash
pytest tests/test_metrics_yaml.py -v
```

Expected: 4 PASS.

- [ ] **Step 4: Commit**

```bash
git add docker/devlake-mcp/metrics.yaml docker/devlake-mcp/tests/test_metrics_yaml.py
git commit -m "feat(devlake-mcp): production metrics.yaml with 16 entries"
```

---

## Task 6 — `scoping.py` (3-tier project filter helper)

**Files:**
- Create: `$APP/app/scoping.py`
- Create: `$APP/tests/test_scoping.py`

- [ ] **Step 1: Write failing tests**

`$APP/tests/test_scoping.py`:

```python
import pytest
from app.scoping import ScopingHelper
from app.catalog import Metric, Scoping, Dimension, Measure


def m(scoping: Scoping, project_column=None, mapping_table=None, join_column=None):
    return Metric(
        name="x", description="", source_tables=["t"],
        primary_key=["id"], time_column=None, default_grain="none",
        scoping=scoping, dimensions=[], measures=[Measure("c", "int", "count")],
        project_column=project_column, mapping_table=mapping_table, join_column=join_column,
    )


def test_project_column_filter():
    helper = ScopingHelper()
    where, params = helper.where_clause(m(Scoping.PROJECT_COLUMN, project_column="project_name"), "demo")
    assert where == "project_name = :_proj"
    assert params == {"_proj": "demo"}


def test_project_mapping_filter():
    helper = ScopingHelper()
    metric = m(Scoping.PROJECT_MAPPING, mapping_table="boards", join_column="base_repo_id")
    join, where, params = helper.join_clause(metric, "demo", alias="t")
    assert "JOIN project_mapping pm" in join
    assert "pm.row_id = t.base_repo_id" in join
    assert "pm.`table` = 'boards'" in join
    assert "pm.project_name = :_proj" in where
    assert params == {"_proj": "demo"}


def test_global_no_filter():
    helper = ScopingHelper()
    where, params = helper.where_clause(m(Scoping.GLOBAL), "demo")
    assert where == ""
    assert params == {}


def test_global_emits_note():
    helper = ScopingHelper()
    note = helper.note(m(Scoping.GLOBAL), "demo")
    assert "global" in note.lower()
```

- [ ] **Step 2: Run, expect FAIL**

```bash
pytest tests/test_scoping.py -v
```

- [ ] **Step 3: Implement `scoping.py`**

```python
from app.catalog import Metric, Scoping


class ScopingHelper:
    """Builds SQL fragments to scope a query by Devlake project."""

    def where_clause(self, metric: Metric, project: str) -> tuple[str, dict]:
        """For project_column-scoped metrics. Returns (where_fragment, params)."""
        if metric.scoping == Scoping.PROJECT_COLUMN:
            return f"{metric.project_column} = :_proj", {"_proj": project}
        return "", {}

    def join_clause(self, metric: Metric, project: str, alias: str) -> tuple[str, str, dict]:
        """For project_mapping-scoped metrics. Returns (join_sql, where_sql, params)."""
        if metric.scoping != Scoping.PROJECT_MAPPING:
            return "", "", {}
        join = (
            f"JOIN project_mapping pm ON pm.row_id = {alias}.{metric.join_column} "
            f"AND pm.`table` = '{metric.mapping_table}'"
        )
        where = "pm.project_name = :_proj"
        return join, where, {"_proj": project}

    def note(self, metric: Metric, project: str) -> str | None:
        """Human-readable note for response payloads when scoping has caveats."""
        if metric.scoping == Scoping.GLOBAL:
            return (
                f"metric {metric.name!r} is global — value covers all data, "
                f"NOT filtered to project {project!r}."
            )
        return None
```

- [ ] **Step 4: Verify tests pass**

```bash
pytest tests/test_scoping.py -v
```

Expected: 4 PASS.

- [ ] **Step 5: Commit**

```bash
git add docker/devlake-mcp/app/scoping.py docker/devlake-mcp/tests/test_scoping.py
git commit -m "feat(devlake-mcp): 3-tier project scoping helper"
```

---

## Task 7 — `auth.py` (JWT middleware + JWKS cache)

**Files:**
- Create: `$APP/app/auth.py`
- Create: `$APP/tests/test_auth.py`

- [ ] **Step 1: Write failing tests**

`$APP/tests/test_auth.py`:

```python
import time
import pytest
import jwt
from cryptography.hazmat.primitives.asymmetric import rsa
from cryptography.hazmat.primitives import serialization
from unittest.mock import AsyncMock

from app.auth import JWTAuthMiddleware


@pytest.fixture
def rsa_keypair():
    private = rsa.generate_private_key(public_exponent=65537, key_size=2048)
    public_pem = private.public_key().public_bytes(
        encoding=serialization.Encoding.PEM,
        format=serialization.PublicFormat.SubjectPublicKeyInfo,
    )
    return private, public_pem


def make_token(private, *, iss, aud, sub="alice@example.com", exp_offset=300):
    return jwt.encode(
        {"iss": iss, "aud": aud, "sub": sub, "email": sub, "iat": int(time.time()), "exp": int(time.time()) + exp_offset},
        private,
        algorithm="RS256",
    )


async def fake_app(scope, receive, send):
    await send({"type": "http.response.start", "status": 200, "headers": []})
    await send({"type": "http.response.body", "body": b"ok"})


@pytest.mark.asyncio
async def test_rejects_missing_token(rsa_keypair):
    private, public = rsa_keypair
    mw = JWTAuthMiddleware(
        fake_app,
        issuer="https://test/oauth2/default",
        audiences=["api://default"],
        jwks_uri="https://test/keys",
        algorithm="RS256",
        resource_url="https://x.com/mcp",
        _public_key_override=public,  # test seam
    )
    sent = []
    async def send(msg): sent.append(msg)
    await mw({"type": "http", "path": "/mcp", "method": "POST", "headers": []}, AsyncMock(), send)
    status = next(m["status"] for m in sent if m["type"] == "http.response.start")
    assert status == 401


@pytest.mark.asyncio
async def test_accepts_valid_token(rsa_keypair):
    private, public = rsa_keypair
    token = make_token(private, iss="https://test/oauth2/default", aud="api://default")
    mw = JWTAuthMiddleware(
        fake_app,
        issuer="https://test/oauth2/default",
        audiences=["api://default"],
        jwks_uri="https://test/keys",
        algorithm="RS256",
        resource_url="https://x.com/mcp",
        _public_key_override=public,
    )
    sent = []
    async def send(msg): sent.append(msg)
    scope = {
        "type": "http", "path": "/mcp", "method": "POST",
        "headers": [(b"authorization", f"Bearer {token}".encode())],
    }
    await mw(scope, AsyncMock(), send)
    status = next(m["status"] for m in sent if m["type"] == "http.response.start")
    assert status == 200


@pytest.mark.asyncio
async def test_well_known_unauthenticated(rsa_keypair):
    _, public = rsa_keypair
    mw = JWTAuthMiddleware(
        fake_app,
        issuer="https://test/oauth2/default",
        audiences=["api://default"],
        jwks_uri="https://test/keys",
        algorithm="RS256",
        resource_url="https://x.com/mcp",
        _public_key_override=public,
    )
    sent = []
    async def send(msg): sent.append(msg)
    scope = {"type": "http", "path": "/.well-known/oauth-protected-resource", "method": "GET", "headers": []}
    await mw(scope, AsyncMock(), send)
    status = next(m["status"] for m in sent if m["type"] == "http.response.start")
    assert status == 200
```

Add `pytest-asyncio` to `requirements.txt`:

```
pytest-asyncio==0.24.0
```

- [ ] **Step 2: Implement `auth.py`**

```python
import json
import time
import jwt
from typing import Any


class JWTAuthMiddleware:
    """ASGI middleware: validates Okta-issued JWTs against JWKS.

    Public endpoints: GET /.well-known/oauth-protected-resource (RFC 9728).
    All other paths require Authorization: Bearer <jwt>.
    """

    def __init__(
        self,
        app,
        *,
        issuer: str,
        audiences: list[str],
        jwks_uri: str,
        algorithm: str,
        resource_url: str,
        _public_key_override: bytes | None = None,
    ) -> None:
        self.app = app
        self.issuer = issuer
        self.audiences = audiences
        self.jwks_uri = jwks_uri
        self.algorithm = algorithm
        self.resource_url = resource_url
        self._public_key_override = _public_key_override
        self._jwks_cache: dict[str, Any] | None = None
        self._jwks_expires_at: float = 0.0

    async def __call__(self, scope, receive, send):
        if scope.get("type") != "http":
            return await self.app(scope, receive, send)

        path = scope.get("path", "")
        if path == "/.well-known/oauth-protected-resource":
            return await self._serve_oauth_metadata(send)

        token = self._extract_bearer(scope)
        if not token:
            return await self._unauthorized(send, "missing_token")

        try:
            key = self._public_key_override or self._signing_key_for(token)
            claims = jwt.decode(
                token, key=key, algorithms=[self.algorithm],
                audience=self.audiences, issuer=self.issuer,
                options={"require": ["exp", "iat", "iss", "aud"]},
            )
        except jwt.PyJWTError as e:
            return await self._unauthorized(send, str(e))

        scope.setdefault("state", {})["claims"] = claims
        await self.app(scope, receive, send)

    def _extract_bearer(self, scope) -> str | None:
        for k, v in scope.get("headers", []):
            if k.lower() == b"authorization":
                value = v.decode()
                if value.startswith("Bearer "):
                    return value[len("Bearer "):]
        return None

    def _signing_key_for(self, token: str):
        import httpx
        if not self._jwks_cache or time.time() > self._jwks_expires_at:
            r = httpx.get(self.jwks_uri, timeout=5.0)
            r.raise_for_status()
            self._jwks_cache = r.json()
            self._jwks_expires_at = time.time() + 300
        kid = jwt.get_unverified_header(token).get("kid")
        for k in self._jwks_cache.get("keys", []):
            if k.get("kid") == kid:
                return jwt.algorithms.RSAAlgorithm.from_jwk(json.dumps(k))
        raise jwt.PyJWTError(f"no JWKS key for kid={kid!r}")

    async def _serve_oauth_metadata(self, send):
        body = json.dumps({
            "resource": self.resource_url,
            "authorization_servers": [self.issuer],
            "scopes_supported": ["devlake_api"],
            "bearer_methods_supported": ["header"],
        }).encode()
        await send({"type": "http.response.start", "status": 200, "headers": [
            (b"content-type", b"application/json"),
        ]})
        await send({"type": "http.response.body", "body": body})

    async def _unauthorized(self, send, reason: str):
        body = json.dumps({"error": "unauthorized", "reason": reason}).encode()
        await send({"type": "http.response.start", "status": 401, "headers": [
            (b"content-type", b"application/json"),
            (b"www-authenticate", f'Bearer resource_metadata="{self.resource_url}/.well-known/oauth-protected-resource"'.encode()),
        ]})
        await send({"type": "http.response.body", "body": body})
```

- [ ] **Step 3: Run tests**

```bash
pytest tests/test_auth.py -v
```

Expected: 3 PASS.

- [ ] **Step 4: Commit**

```bash
git add docker/devlake-mcp/app/auth.py docker/devlake-mcp/tests/test_auth.py docker/devlake-mcp/requirements.txt
git commit -m "feat(devlake-mcp): JWT auth middleware + JWKS cache + RFC 9728 well-known"
```

---

## Task 8 — Server harness (`server.py` with MCP SDK)

**Files:**
- Create: `$APP/app/server.py`

- [ ] **Step 1: Implement minimal `server.py` with the MCP SDK + echo tool**

```python
from mcp.server.fastmcp import FastMCP
from mcp.server.fastmcp.utilities.types import Image
from starlette.applications import Starlette
from starlette.routing import Mount

from app.auth import JWTAuthMiddleware
from app.config import Config
from app.db import Database
from app.catalog import MetricCatalog
from pathlib import Path


def build_app():
    cfg = Config.from_env()
    db = Database.from_url(cfg.db_url)
    catalog = MetricCatalog.from_file(Path(__file__).parent.parent / "metrics.yaml")

    mcp = FastMCP("devlake")

    @mcp.tool()
    def ping() -> str:
        """Returns 'pong' — wiring smoke test."""
        return "pong"

    # Stash dependencies on the app for tools to find
    mcp.devlake = {"db": db, "catalog": catalog, "cfg": cfg}

    app = mcp.streamable_http_app()
    if cfg.auth_enabled:
        app = JWTAuthMiddleware(
            app,
            issuer=cfg.jwt_issuer,
            audiences=cfg.jwt_audiences,
            jwks_uri=cfg.jwks_uri,
            algorithm=cfg.jwt_algorithm,
            resource_url=cfg.resource_url,
        )
    return app


app = build_app()
```

- [ ] **Step 2: Smoke test by hand**

```bash
cd $APP
DEVLAKE_MCP_DB_URL='sqlite+pysqlite:///:memory:' \
DEVLAKE_MCP_DEFAULT_PROJECT='test' \
MCP_AUTH_ENABLED=false \
MCP_JWT_ISSUER='-' MCP_JWT_AUDIENCE='-' MCP_JWKS_URI='-' MCP_RESOURCE_URL='-' \
  uvicorn app.server:app --port 5009 &
sleep 2
curl -s http://localhost:5009/mcp/tools/list | head -50
kill %1
```

Expected: JSON listing the `ping` tool.

- [ ] **Step 3: Commit**

```bash
git add docker/devlake-mcp/app/server.py
git commit -m "feat(devlake-mcp): MCP server harness with ping tool"
```

---

## Task 9 — Discovery tools (`tools/discovery.py`)

**Files:**
- Create: `$APP/app/tools/__init__.py`
- Create: `$APP/app/tools/discovery.py`
- Create: `$APP/tests/test_tools_discovery.py`

- [ ] **Step 1: Empty `__init__.py`**

`$APP/app/tools/__init__.py`: one newline.

- [ ] **Step 2: Write failing tests for discovery tools**

`$APP/tests/test_tools_discovery.py`:

```python
from unittest.mock import MagicMock
from app.tools.discovery import list_projects_impl, list_metrics_impl, describe_metric_impl
from app.catalog import MetricCatalog
from pathlib import Path

YAML = Path(__file__).parent.parent / "metrics.yaml"


def test_list_projects_returns_rows():
    fake_db = MagicMock()
    fake_db.fetch_all.return_value = [
        {"name": "finally-DevEx", "description": ""},
        {"name": "finally-DevEx-MCP", "description": "MCP dedicated"},
    ]
    out = list_projects_impl(fake_db)
    assert len(out["projects"]) == 2
    assert out["projects"][0]["name"] == "finally-DevEx"


def test_list_metrics_returns_16():
    cat = MetricCatalog.from_file(YAML)
    out = list_metrics_impl(cat)
    assert len(out["metrics"]) == 16
    assert all("name" in m and "description" in m for m in out["metrics"])


def test_describe_metric_returns_full_schema():
    cat = MetricCatalog.from_file(YAML)
    out = describe_metric_impl(cat, "verification_effort")
    assert out["name"] == "verification_effort"
    assert out["scoping"] == "global"
    assert any(m["name"] == "author_minutes" for m in out["measures"])
    assert "known_gaps" in out


def test_describe_metric_unknown_returns_error():
    cat = MetricCatalog.from_file(YAML)
    out = describe_metric_impl(cat, "nope")
    assert "error" in out
```

- [ ] **Step 3: Implement `discovery.py`**

```python
from sqlalchemy import text
from app.catalog import MetricCatalog
from app.db import Database


def list_projects_impl(db: Database) -> dict:
    rows = db.fetch_all(text("SELECT name, description FROM projects ORDER BY name"))
    return {"projects": rows}


def list_engineers_impl(
    db: Database,
    project: str | None = None,
    active_since: str | None = None,
    search: str | None = None,
) -> dict:
    """Lists accounts; uses pull_requests.author_id activity as 'last_seen'.

    Activity is keyed off pull_requests across the warehouse (not project-scoped today).
    """
    sql = """
        SELECT a.id AS account_id, a.full_name, a.email,
               MAX(pr.created_date) AS last_seen
          FROM accounts a
          LEFT JOIN pull_requests pr ON pr.author_id = a.id
         WHERE 1=1
    """
    params: dict = {}
    if active_since:
        sql += " AND pr.created_date >= :since"
        params["since"] = active_since
    if search:
        sql += " AND (a.full_name LIKE :q OR a.email LIKE :q)"
        params["q"] = f"%{search}%"
    sql += " GROUP BY a.id, a.full_name, a.email ORDER BY last_seen DESC NULLS LAST"
    rows = db.fetch_all(text(sql).bindparams(**params))
    return {"engineers": rows}


def list_metrics_impl(catalog: MetricCatalog) -> dict:
    return {"metrics": [{"name": m.name, "description": m.description} for m in catalog.metrics]}


def describe_metric_impl(catalog: MetricCatalog, name: str) -> dict:
    try:
        m = catalog.get(name)
    except KeyError as e:
        return {"error": str(e), "available": catalog.names()}
    return {
        "name": m.name,
        "description": m.description,
        "source_tables": m.source_tables,
        "primary_key": m.primary_key,
        "time_column": m.time_column,
        "default_grain": m.default_grain,
        "scoping": m.scoping.value,
        "project_column": m.project_column,
        "mapping_table": m.mapping_table,
        "join_column": m.join_column,
        "dimensions": [
            {"name": d.name, "type": d.type, "joinable_to": d.joinable_to}
            for d in m.dimensions
        ],
        "measures": [
            {"name": x.name, "type": x.type, "unit": x.unit, "description": x.description}
            for x in m.measures
        ],
        "known_gaps": m.known_gaps,
    }
```

- [ ] **Step 4: Wire into `server.py`** — add to `build_app()`:

```python
from app.tools.discovery import (
    list_projects_impl, list_engineers_impl, list_metrics_impl, describe_metric_impl,
)

# inside build_app() after `mcp = FastMCP(...)`:
@mcp.tool()
def list_projects() -> dict:
    """List Devlake project entities."""
    return list_projects_impl(db)

@mcp.tool()
def list_engineers(project: str | None = None, active_since: str | None = None, search: str | None = None) -> dict:
    """List engineers (account_ids). Optional: project, active_since (ISO date), search (name/email substring)."""
    return list_engineers_impl(db, project, active_since, search)

@mcp.tool()
def list_metrics() -> dict:
    """List all queryable metrics with one-line descriptions."""
    return list_metrics_impl(catalog)

@mcp.tool()
def describe_metric(name: str) -> dict:
    """Return full schema for one metric: dimensions, measures, units, scoping, known gaps."""
    return describe_metric_impl(catalog, name)
```

- [ ] **Step 5: Run tests**

```bash
pytest tests/test_tools_discovery.py -v
```

Expected: 4 PASS.

- [ ] **Step 6: Commit**

```bash
git add docker/devlake-mcp/app/tools/ docker/devlake-mcp/tests/test_tools_discovery.py docker/devlake-mcp/app/server.py
git commit -m "feat(devlake-mcp): discovery tools (list_projects/engineers/metrics, describe_metric)"
```

---

## Task 10 — `query_metric` (generic escape hatch)

**Files:**
- Create: `$APP/app/tools/generic.py`
- Create: `$APP/tests/test_tools_generic.py`

- [ ] **Step 1: Write failing tests**

`$APP/tests/test_tools_generic.py`:

```python
import pytest
from unittest.mock import MagicMock
from app.tools.generic import query_metric_impl
from app.catalog import MetricCatalog
from app.scoping import ScopingHelper
from pathlib import Path


YAML = Path(__file__).parent.parent / "metrics.yaml"


def test_query_metric_unknown_metric_returns_error():
    cat = MetricCatalog.from_file(YAML)
    helper = ScopingHelper()
    fake_db = MagicMock()
    out = query_metric_impl(fake_db, cat, helper, metric="nope", project="x")
    assert "error" in out


def test_query_metric_rejects_unknown_measure():
    cat = MetricCatalog.from_file(YAML)
    helper = ScopingHelper()
    fake_db = MagicMock()
    out = query_metric_impl(fake_db, cat, helper, metric="dora_pr_metrics", project="x", measures=["bogus"])
    assert "error" in out
    assert "bogus" in out["error"]


def test_query_metric_project_column_filter_applied():
    cat = MetricCatalog.from_file(YAML)
    helper = ScopingHelper()
    fake_db = MagicMock()
    fake_db.fetch_all.return_value = [{"coding_time_minutes": 100}]
    out = query_metric_impl(
        fake_db, cat, helper,
        metric="dora_pr_metrics", project="demo",
        measures=["coding_time_minutes"], limit=10,
    )
    assert fake_db.fetch_all.called
    # verify the SQL string includes the project filter
    call_arg = fake_db.fetch_all.call_args[0][0]
    rendered = str(call_arg)
    assert "project_name = :_proj" in rendered or "project_name = '_proj'" in rendered


def test_query_metric_global_metric_includes_note():
    cat = MetricCatalog.from_file(YAML)
    helper = ScopingHelper()
    fake_db = MagicMock()
    fake_db.fetch_all.return_value = []
    out = query_metric_impl(fake_db, cat, helper, metric="verification_effort", project="demo")
    assert "note" in out
    assert "global" in out["note"].lower()
```

- [ ] **Step 2: Implement `query_metric_impl`**

```python
from sqlalchemy import text
from app.catalog import MetricCatalog, Scoping
from app.db import Database
from app.scoping import ScopingHelper


def query_metric_impl(
    db: Database,
    catalog: MetricCatalog,
    scoping_helper: ScopingHelper,
    *,
    metric: str,
    project: str,
    measures: list[str] | None = None,
    filters: dict | None = None,
    group_by: list[str] | None = None,
    since: str | None = None,
    until: str | None = None,
    limit: int = 100,
) -> dict:
    try:
        m = catalog.get(metric)
    except KeyError as e:
        return {"error": str(e), "available": catalog.names()}

    measure_names = {x.name for x in m.measures}
    selected = measures or [x.name for x in m.measures]
    invalid = [s for s in selected if s not in measure_names]
    if invalid:
        return {"error": f"unknown measure(s): {invalid}", "available_measures": list(measure_names)}

    dim_names = {d.name for d in m.dimensions}
    grouped = group_by or []
    invalid_g = [g for g in grouped if g not in dim_names]
    if invalid_g:
        return {"error": f"unknown group_by dimension(s): {invalid_g}", "available_dimensions": list(dim_names)}

    # Source table (V1 supports only the first source_table per metric;
    # multi-source metrics handle joins in curated tools, not query_metric.)
    table = m.source_tables[0]
    alias = "t"
    select_cols = [f"SUM({alias}.{x}) AS {x}" if any(g for g in grouped) else f"{alias}.{x} AS {x}" for x in selected]
    select_cols = [f"{alias}.{g} AS {g}" for g in grouped] + select_cols

    sql = f"SELECT {', '.join(select_cols)} FROM {table} {alias}"
    params: dict = {}

    if m.scoping == Scoping.PROJECT_COLUMN:
        where, p = scoping_helper.where_clause(m, project)
        sql += f" WHERE {where}"
        params.update(p)
    elif m.scoping == Scoping.PROJECT_MAPPING:
        join, where, p = scoping_helper.join_clause(m, project, alias=alias)
        sql += f" {join} WHERE {where}"
        params.update(p)
    else:
        sql += " WHERE 1=1"

    if since and m.time_column:
        sql += f" AND {alias}.{m.time_column} >= :_since"
        params["_since"] = since
    if until and m.time_column:
        sql += f" AND {alias}.{m.time_column} < :_until"
        params["_until"] = until

    for k, v in (filters or {}).items():
        if k in dim_names:
            sql += f" AND {alias}.{k} = :{k}"
            params[k] = v

    if grouped:
        sql += " GROUP BY " + ", ".join(f"{alias}.{g}" for g in grouped)

    sql += f" LIMIT {int(limit)}"

    stmt = text(sql).bindparams(**params)
    rows = db.fetch_all(stmt)
    out: dict = {"metric": m.name, "rows": rows, "rowcount": len(rows)}
    note = scoping_helper.note(m, project)
    if note:
        out["note"] = note
    return out
```

- [ ] **Step 3: Run tests**

```bash
pytest tests/test_tools_generic.py -v
```

Expected: 4 PASS.

- [ ] **Step 4: Wire into `server.py`** (after the discovery tools)

```python
from app.tools.generic import query_metric_impl
from app.scoping import ScopingHelper

scoping = ScopingHelper()

@mcp.tool()
def query_metric(
    metric: str,
    project: str | None = None,
    measures: list[str] | None = None,
    filters: dict | None = None,
    group_by: list[str] | None = None,
    since: str | None = None,
    until: str | None = None,
    limit: int = 100,
) -> dict:
    """Parameterised aggregate query against a catalog metric. Use describe_metric() first to learn the grammar."""
    return query_metric_impl(
        db, catalog, scoping,
        metric=metric, project=project or cfg.default_project,
        measures=measures, filters=filters, group_by=group_by,
        since=since, until=until, limit=limit,
    )
```

- [ ] **Step 5: Commit**

```bash
git add docker/devlake-mcp/app/tools/generic.py docker/devlake-mcp/tests/test_tools_generic.py docker/devlake-mcp/app/server.py
git commit -m "feat(devlake-mcp): query_metric generic tool"
```

---

## Task 11 — `get_team_scorecard`

**Files:**
- Create: `$APP/app/tools/team.py`
- Create: `$APP/tests/test_tools_team.py`

- [ ] **Step 1: Write failing test**

`$APP/tests/test_tools_team.py`:

```python
from unittest.mock import MagicMock
from app.tools.team import team_scorecard_impl


def test_team_scorecard_returns_blocks():
    fake_db = MagicMock()
    fake_db.fetch_all.side_effect = [
        # velocity: PRs merged, issues completed
        [{"prs_merged": 42, "issues_completed": 18}],
        # shipping: deploy freq, lead time p50/p95
        [{"deploy_count": 87, "lead_time_p50_min": 1200, "lead_time_p95_min": 8400}],
        # team velocity table
        [{"story_points_completed": 38.0, "prs_merged": 42}],
        # team health
        [{"health_score": 72.5, "health_level": "high"}],
        # ai mix
        [{"ai_cohort": "HIGH", "n": 5}, {"ai_cohort": "NONE", "n": 37}],
        # code quality summary
        [{"avg_coverage": 78.2, "total_bugs": 12, "total_smells": 540}],
        # incidents (will be empty in real life)
        [],
    ]
    out = team_scorecard_impl(fake_db, project="demo", since="2026-04-01", until="2026-05-01")
    assert "velocity" in out and out["velocity"]["prs_merged"] == 42
    assert "shipping" in out and out["shipping"]["deploy_count"] == 87
    assert "reliability" in out
    assert out["reliability"]["status"] == "no_data"  # incidents empty
    assert "ai_mix" in out and out["ai_mix"]["HIGH"] == 5
    assert "code_health" in out
```

- [ ] **Step 2: Implement `team.py`** (just `team_scorecard_impl` for now)

```python
from sqlalchemy import text
from app.db import Database


def team_scorecard_impl(
    db: Database, *, project: str, since: str, until: str,
) -> dict:
    # 1. Velocity from project_pr_metrics + issues
    velocity = db.fetch_all(text("""
        SELECT
          (SELECT COUNT(*) FROM project_pr_metrics WHERE project_name = :p AND pr_merged_date BETWEEN :s AND :u) AS prs_merged,
          (SELECT COUNT(*) FROM issues i WHERE i.status = 'DONE' AND i.resolution_date BETWEEN :s AND :u) AS issues_completed
    """).bindparams(p=project, s=since, u=until))

    # 2. Shipping from cicd_deployments via project_mapping
    shipping = db.fetch_all(text("""
        SELECT
          COUNT(*) AS deploy_count,
          PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY duration_sec / 60) AS lead_time_p50_min,
          PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration_sec / 60) AS lead_time_p95_min
        FROM cicd_deployments d
        JOIN project_mapping pm ON pm.row_id = d.cicd_scope_id AND pm.`table` = 'cicd_scopes'
        WHERE pm.project_name = :p AND d.started_date BETWEEN :s AND :u
    """).bindparams(p=project, s=since, u=until))

    # 3. Pre-computed team velocity
    team_vel = db.fetch_all(text("""
        SELECT story_points_completed, prs_merged
        FROM team_velocities WHERE project_name = :p AND period_month BETWEEN :s AND :u
        ORDER BY period_month DESC LIMIT 1
    """).bindparams(p=project, s=since, u=until))

    # 4. Team health
    health = db.fetch_all(text("""
        SELECT health_score, health_level FROM team_health_scores
        WHERE project_name = :p AND period_month BETWEEN :s AND :u
        ORDER BY period_month DESC LIMIT 1
    """).bindparams(p=project, s=since, u=until))

    # 5. AI cohort mix
    ai_mix = db.fetch_all(text("""
        SELECT c.ai_cohort, COUNT(*) AS n
        FROM pr_ai_cohort c
        JOIN pull_requests pr ON pr.id = c.pr_id
        JOIN project_mapping pm ON pm.row_id = pr.base_repo_id AND pm.`table` = 'boards'
        WHERE pm.project_name = :p AND c.classified_at BETWEEN :s AND :u
        GROUP BY c.ai_cohort
    """).bindparams(p=project, s=since, u=until))

    # 6. Code quality
    code_health = db.fetch_all(text("""
        SELECT AVG(coverage) AS avg_coverage,
               SUM(CASE WHEN type='BUG' THEN 1 ELSE 0 END) AS total_bugs,
               SUM(CASE WHEN type='CODE_SMELL' THEN 1 ELSE 0 END) AS total_smells
        FROM cq_issues
    """))

    # 7. Reliability (incidents)
    incidents = db.fetch_all(text("""
        SELECT COUNT(*) AS n FROM incidents WHERE created_date BETWEEN :s AND :u
    """).bindparams(s=since, u=until))

    reliability = {"status": "no_data", "note": "incidents table sparse; ingest not yet wired"}
    if incidents and incidents[0]["n"] > 0:
        reliability = {"status": "available", "incident_count": incidents[0]["n"]}

    return {
        "project": project, "since": since, "until": until,
        "velocity": velocity[0] if velocity else {},
        "shipping": shipping[0] if shipping else {},
        "team_velocity": team_vel[0] if team_vel else {},
        "team_health": health[0] if health else {},
        "ai_mix": {row["ai_cohort"]: row["n"] for row in ai_mix},
        "code_health": code_health[0] if code_health else {},
        "reliability": reliability,
    }
```

- [ ] **Step 3: Wire into `server.py`**

```python
from app.tools.team import team_scorecard_impl

@mcp.tool()
def get_team_scorecard(since: str, until: str, project: str | None = None) -> dict:
    """Top-level engineering scorecard: velocity, shipping, code health, AI mix, reliability stub."""
    return team_scorecard_impl(db, project=project or cfg.default_project, since=since, until=until)
```

- [ ] **Step 4: Run tests**

```bash
pytest tests/test_tools_team.py -v
```

Expected: 1 PASS.

- [ ] **Step 5: Commit**

```bash
git add docker/devlake-mcp/app/tools/team.py docker/devlake-mcp/tests/test_tools_team.py docker/devlake-mcp/app/server.py
git commit -m "feat(devlake-mcp): get_team_scorecard tool"
```

---

## Task 12 — `get_reviewer_load`

**Files:**
- Modify: `$APP/app/tools/team.py`
- Modify: `$APP/tests/test_tools_team.py`

- [ ] **Step 1: Add failing test**

Append to `test_tools_team.py`:

```python
def test_reviewer_load_returns_per_engineer():
    fake_db = MagicMock()
    fake_db.fetch_all.return_value = [
        {"engineer_id": "alice", "author_minutes": 100, "reviewer_minutes": 50,
         "review_to_author_ratio": 0.5, "review_comments_per_loc": 0.04,
         "after_hours_ratio": 0.10},
        {"engineer_id": "bob", "author_minutes": 0, "reviewer_minutes": 300,
         "review_to_author_ratio": 0.0, "review_comments_per_loc": 0.08,
         "after_hours_ratio": 0.22},
    ]
    from app.tools.team import reviewer_load_impl
    out = reviewer_load_impl(fake_db, project="demo", since="2026-04-01", until="2026-05-01")
    assert len(out["engineers"]) == 2
    assert out["note"]  # mentions global scoping
```

- [ ] **Step 2: Implement**

Append to `team.py`:

```python
def reviewer_load_impl(
    db: Database, *, project: str, since: str, until: str, top_n: int = 20,
) -> dict:
    rows = db.fetch_all(text("""
        SELECT engineer_id,
               SUM(author_minutes) AS author_minutes,
               SUM(reviewer_minutes) AS reviewer_minutes,
               AVG(review_to_author_ratio) AS review_to_author_ratio,
               AVG(review_comments_per_loc) AS review_comments_per_loc
        FROM engineer_verification_effort
        WHERE period_week BETWEEN :s AND :u
        GROUP BY engineer_id
        ORDER BY reviewer_minutes DESC
        LIMIT :n
    """).bindparams(s=since, u=until, n=top_n))

    # Join after-hours signals (slack)
    slack = db.fetch_all(text("""
        SELECT engineer_id, AVG(after_hours_ratio) AS after_hours_ratio
        FROM engineer_slack_signals
        WHERE period_week BETWEEN :s AND :u
        GROUP BY engineer_id
    """).bindparams(s=since, u=until))
    slack_map = {r["engineer_id"]: r["after_hours_ratio"] for r in slack}
    for r in rows:
        r["after_hours_ratio"] = slack_map.get(r["engineer_id"])

    return {
        "project": project, "since": since, "until": until,
        "engineers": rows,
        "note": "engineer_verification_effort and engineer_slack_signals are global metrics; not project-scoped today.",
    }
```

Wire into `server.py`:

```python
from app.tools.team import reviewer_load_impl

@mcp.tool()
def get_reviewer_load(since: str, until: str, project: str | None = None, top_n: int = 20) -> dict:
    """Per-engineer review-vs-author ratio + after-hours proxy, sorted by reviewer minutes."""
    return reviewer_load_impl(db, project=project or cfg.default_project, since=since, until=until, top_n=top_n)
```

- [ ] **Step 3: Tests pass + commit**

```bash
pytest tests/test_tools_team.py -v
git add docker/devlake-mcp/app/tools/team.py docker/devlake-mcp/tests/test_tools_team.py docker/devlake-mcp/app/server.py
git commit -m "feat(devlake-mcp): get_reviewer_load tool"
```

---

## Task 13 — `get_cycle_times`

**Files:**
- Modify: `$APP/app/tools/team.py`
- Modify: `$APP/tests/test_tools_team.py`

- [ ] **Step 1: Add failing test**

```python
def test_cycle_times_returns_percentiles():
    fake_db = MagicMock()
    fake_db.fetch_all.return_value = [
        {"phase": "issue_to_pr_open", "p50_min": 1200, "p95_min": 8400, "p99_min": 20000},
        {"phase": "pr_open_to_merge", "p50_min": 600, "p95_min": 4800, "p99_min": 14400},
        {"phase": "merge_to_deploy", "p50_min": 30, "p95_min": 240, "p99_min": 1800},
    ]
    from app.tools.team import cycle_times_impl
    out = cycle_times_impl(fake_db, project="demo", since="2026-04-01", until="2026-05-01")
    assert len(out["phases"]) == 3
```

- [ ] **Step 2: Implement**

Append to `team.py`:

```python
def cycle_times_impl(
    db: Database, *, project: str, since: str, until: str,
) -> dict:
    rows = db.fetch_all(text("""
        SELECT 'issue_to_pr_open' AS phase,
               PERCENTILE_CONT(0.5)  WITHIN GROUP (ORDER BY lead_time_minutes) AS p50_min,
               PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY lead_time_minutes) AS p95_min,
               PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY lead_time_minutes) AS p99_min
        FROM issue_flow_metrics
        WHERE project_name = :p AND resolution_date BETWEEN :s AND :u
        UNION ALL
        SELECT 'pr_open_to_merge',
               PERCENTILE_CONT(0.5)  WITHIN GROUP (ORDER BY review_time_minutes),
               PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY review_time_minutes),
               PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY review_time_minutes)
        FROM project_pr_metrics
        WHERE project_name = :p AND pr_merged_date BETWEEN :s AND :u
        UNION ALL
        SELECT 'merge_to_deploy',
               PERCENTILE_CONT(0.5)  WITHIN GROUP (ORDER BY deploy_time_minutes),
               PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY deploy_time_minutes),
               PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY deploy_time_minutes)
        FROM project_pr_metrics
        WHERE project_name = :p AND pr_merged_date BETWEEN :s AND :u
    """).bindparams(p=project, s=since, u=until))
    return {"project": project, "since": since, "until": until, "phases": rows}
```

Wire into `server.py`:

```python
from app.tools.team import cycle_times_impl

@mcp.tool()
def get_cycle_times(since: str, until: str, project: str | None = None) -> dict:
    """Issue→PR-open, PR-open→merge, merge→deploy percentiles (p50/p95/p99)."""
    return cycle_times_impl(db, project=project or cfg.default_project, since=since, until=until)
```

- [ ] **Step 3: Test + commit**

```bash
pytest tests/test_tools_team.py -v
git add docker/devlake-mcp/app/tools/team.py docker/devlake-mcp/tests/test_tools_team.py docker/devlake-mcp/app/server.py
git commit -m "feat(devlake-mcp): get_cycle_times tool"
```

---

## Task 14 — `get_code_quality`

**Files:**
- Modify: `$APP/app/tools/team.py`
- Modify: `$APP/tests/test_tools_team.py`

- [ ] **Step 1: Test**

```python
def test_code_quality_returns_components():
    fake_db = MagicMock()
    fake_db.fetch_all.return_value = [
        {"cq_project_id": "p1", "coverage": 78.2, "bugs": 12, "smells": 540, "complexity": 1200},
    ]
    from app.tools.team import code_quality_impl
    out = code_quality_impl(fake_db, project="demo", since="2026-04-01", until="2026-05-01")
    assert len(out["projects"]) == 1
```

- [ ] **Step 2: Implement**

```python
def code_quality_impl(db: Database, *, project: str, since: str, until: str) -> dict:
    rows = db.fetch_all(text("""
        SELECT cqp.id AS cq_project_id,
               AVG(cqfm.coverage) AS coverage,
               SUM(CASE WHEN ci.type='BUG' THEN 1 ELSE 0 END) AS bugs,
               SUM(CASE WHEN ci.type='CODE_SMELL' THEN 1 ELSE 0 END) AS smells,
               SUM(cqfm.complexity) AS complexity
        FROM cq_projects cqp
        LEFT JOIN cq_file_metrics cqfm ON cqfm.cq_project_id = cqp.id
        LEFT JOIN cq_issues ci ON ci.cq_project_id = cqp.id
        JOIN project_mapping pm ON pm.row_id = cqp.id AND pm.`table` = 'cq_projects'
        WHERE pm.project_name = :p
        GROUP BY cqp.id
    """).bindparams(p=project))
    return {"project": project, "projects": rows}
```

Wire into `server.py`:

```python
from app.tools.team import code_quality_impl

@mcp.tool()
def get_code_quality(project: str | None = None, since: str | None = None, until: str | None = None) -> dict:
    """SonarQube-derived per-component coverage / bugs / smells / complexity."""
    return code_quality_impl(db, project=project or cfg.default_project, since=since or "1970-01-01", until=until or "2100-01-01")
```

- [ ] **Step 3: Test + commit**

```bash
pytest tests/test_tools_team.py -v
git add docker/devlake-mcp/app/tools/team.py docker/devlake-mcp/tests/test_tools_team.py docker/devlake-mcp/app/server.py
git commit -m "feat(devlake-mcp): get_code_quality tool"
```

---

## Task 15 — `get_engineer_week`

**Files:**
- Create: `$APP/app/tools/engineer.py`
- Create: `$APP/tests/test_tools_engineer.py`

- [ ] **Step 1: Test**

`$APP/tests/test_tools_engineer.py`:

```python
from unittest.mock import MagicMock
from app.tools.engineer import engineer_week_impl


def test_engineer_week_returns_blocks():
    fake_db = MagicMock()
    fake_db.fetch_all.side_effect = [
        [{"prs_authored": 4, "prs_reviewed": 7, "lines_added": 850, "lines_removed": 200}],
        [{"issues_touched": 3}],
        [{"commits": 12}],
        [{"ai_cohort": "HIGH", "n": 2}, {"ai_cohort": "NONE", "n": 2}],
        [{"author_minutes": 220, "reviewer_minutes": 145, "review_to_author_ratio": 0.66}],
    ]
    out = engineer_week_impl(fake_db, engineer_id="alice", period_week="2026-05-04")
    assert out["prs_authored"] == 4
    assert out["ai_mix"]["HIGH"] == 2
    assert out["verification_effort"]["author_minutes"] == 220
```

- [ ] **Step 2: Implement**

```python
from sqlalchemy import text
from app.db import Database


def engineer_week_impl(db: Database, *, engineer_id: str, period_week: str) -> dict:
    pr_activity = db.fetch_all(text("""
        SELECT
          (SELECT COUNT(*) FROM pull_requests WHERE author_id = :e AND merged_date BETWEEN :w AND DATE_ADD(:w, INTERVAL 7 DAY)) AS prs_authored,
          (SELECT COUNT(DISTINCT prc.pull_request_id) FROM pull_request_comments prc
           JOIN pull_requests pr ON pr.id = prc.pull_request_id
           WHERE prc.account_id = :e AND prc.account_id != pr.author_id
                 AND prc.created_date BETWEEN :w AND DATE_ADD(:w, INTERVAL 7 DAY)) AS prs_reviewed,
          (SELECT COALESCE(SUM(additions), 0) FROM pull_requests WHERE author_id = :e AND merged_date BETWEEN :w AND DATE_ADD(:w, INTERVAL 7 DAY)) AS lines_added,
          (SELECT COALESCE(SUM(deletions), 0) FROM pull_requests WHERE author_id = :e AND merged_date BETWEEN :w AND DATE_ADD(:w, INTERVAL 7 DAY)) AS lines_removed
    """).bindparams(e=engineer_id, w=period_week))

    issues = db.fetch_all(text("""
        SELECT COUNT(DISTINCT issue_id) AS issues_touched FROM issue_assignees
        WHERE assignee_id = :e
    """).bindparams(e=engineer_id))

    commits = db.fetch_all(text("""
        SELECT COUNT(*) AS commits FROM commits
        WHERE author_id = :e AND authored_date BETWEEN :w AND DATE_ADD(:w, INTERVAL 7 DAY)
    """).bindparams(e=engineer_id, w=period_week))

    ai_mix = db.fetch_all(text("""
        SELECT c.ai_cohort, COUNT(*) AS n
        FROM pr_ai_cohort c
        JOIN pull_requests pr ON pr.id = c.pr_id
        WHERE pr.author_id = :e AND pr.merged_date BETWEEN :w AND DATE_ADD(:w, INTERVAL 7 DAY)
        GROUP BY c.ai_cohort
    """).bindparams(e=engineer_id, w=period_week))

    verification = db.fetch_all(text("""
        SELECT author_minutes, reviewer_minutes, review_to_author_ratio
        FROM engineer_verification_effort WHERE engineer_id = :e AND period_week = :w
    """).bindparams(e=engineer_id, w=period_week))

    return {
        "engineer_id": engineer_id, "period_week": period_week,
        **(pr_activity[0] if pr_activity else {}),
        "issues_touched": issues[0]["issues_touched"] if issues else 0,
        "commits": commits[0]["commits"] if commits else 0,
        "ai_mix": {row["ai_cohort"]: row["n"] for row in ai_mix},
        "verification_effort": verification[0] if verification else {},
    }
```

Wire into `server.py`:

```python
from app.tools.engineer import engineer_week_impl

@mcp.tool()
def get_engineer_week(engineer_id: str, period_week: str, project: str | None = None) -> dict:
    """Per-engineer week summary: PRs authored/reviewed, issues, commits, AI mix, Phase B aggregates."""
    return engineer_week_impl(db, engineer_id=engineer_id, period_week=period_week)
```

- [ ] **Step 3: Test + commit**

```bash
pytest tests/test_tools_engineer.py -v
git add docker/devlake-mcp/app/tools/engineer.py docker/devlake-mcp/tests/test_tools_engineer.py docker/devlake-mcp/app/server.py
git commit -m "feat(devlake-mcp): get_engineer_week tool"
```

---

## Task 16 — `get_investment_profile`, `get_ai_adoption`, `get_initiatives`

(Grouped because each is small and shares the `business.py` module.)

**Files:**
- Create: `$APP/app/tools/business.py`
- Create: `$APP/tests/test_tools_business.py`

- [ ] **Step 1: Tests**

`$APP/tests/test_tools_business.py`:

```python
from unittest.mock import MagicMock
from app.tools.business import (
    investment_profile_impl, ai_adoption_impl, initiatives_impl,
)


def test_investment_profile_splits_categories():
    fake_db = MagicMock()
    fake_db.fetch_all.return_value = [
        {"category": "new_features", "pct": 60.0, "total_usd": 120000.0},
        {"category": "ktlo", "pct": 25.0, "total_usd": 50000.0},
        {"category": "maintenance", "pct": 15.0, "total_usd": 30000.0},
    ]
    out = investment_profile_impl(fake_db, project="demo", since="2026-01-01", until="2026-05-01")
    assert sum(c["pct"] for c in out["categories"]) == 100.0


def test_ai_adoption_returns_per_tool():
    fake_db = MagicMock()
    fake_db.fetch_all.side_effect = [
        [{"tool": "claude_code", "active_engineers": 20}],
        [{"tool": "cursor", "active_engineers": 8}],
        [{"ai_contribution_pct": 32.5}],
        [{"rework_pct": 4.2}],
    ]
    out = ai_adoption_impl(fake_db, project="demo", since="2026-04-01", until="2026-05-01")
    assert out["adoption"]["claude_code"] == 20
    assert out["ai_contribution_pct"] == 32.5


def test_initiatives_returns_list():
    fake_db = MagicMock()
    fake_db.fetch_all.return_value = [
        {"initiative_id": "INIT-1", "status": "in_progress",
         "effort_story_points": 50.0, "forecast_completion_date": "2026-06-15",
         "roi_pct": 12.3},
    ]
    out = initiatives_impl(fake_db, project="demo", since="2026-01-01", until="2026-12-31")
    assert len(out["initiatives"]) == 1
```

- [ ] **Step 2: Implement `business.py`**

```python
from sqlalchemy import text
from app.db import Database


def investment_profile_impl(db: Database, *, project: str, since: str, until: str) -> dict:
    rows = db.fetch_all(text("""
        SELECT wa.category,
               (SUM(wa.fte_fraction) / NULLIF((SELECT SUM(fte_fraction) FROM work_allocations
                                                WHERE project_name = :p AND period_month BETWEEN :s AND :u), 0)) * 100 AS pct,
               SUM(ca.allocated_cost_usd) AS total_usd
        FROM work_allocations wa
        LEFT JOIN cost_allocations ca ON ca.initiative_id = wa.initiative_id
        WHERE wa.project_name = :p AND wa.period_month BETWEEN :s AND :u
        GROUP BY wa.category
        ORDER BY pct DESC
    """).bindparams(p=project, s=since, u=until))
    return {"project": project, "since": since, "until": until, "categories": rows}


def ai_adoption_impl(db: Database, *, project: str, since: str, until: str) -> dict:
    claude = db.fetch_all(text("""
        SELECT 'claude_code' AS tool, COUNT(DISTINCT account_id) AS active_engineers
        FROM claude_code_user_metrics
        WHERE period_month BETWEEN :s AND :u
    """).bindparams(s=since, u=until))
    cursor = db.fetch_all(text("""
        SELECT 'cursor' AS tool, COUNT(DISTINCT account_id) AS active_engineers
        FROM cursor_user_metrics
        WHERE period_month BETWEEN :s AND :u
    """).bindparams(s=since, u=until))
    contribution = db.fetch_all(text("""
        SELECT (SUM(CASE WHEN c.ai_cohort IN ('HIGH', 'MEDIUM') THEN 1 ELSE 0 END) * 100.0 / NULLIF(COUNT(*), 0)) AS ai_contribution_pct
        FROM pr_ai_cohort c
        JOIN pull_requests pr ON pr.id = c.pr_id
        JOIN project_mapping pm ON pm.row_id = pr.base_repo_id AND pm.`table` = 'boards'
        WHERE pm.project_name = :p AND c.classified_at BETWEEN :s AND :u
    """).bindparams(p=project, s=since, u=until))
    rework = db.fetch_all(text("""
        SELECT (SUM(CASE WHEN d.total_defect_count > 0 THEN 1 ELSE 0 END) * 100.0 / NULLIF(COUNT(*), 0)) AS rework_pct
        FROM pr_defect_signals d
        JOIN pr_ai_cohort c ON c.pr_id = d.pr_id
        WHERE c.ai_cohort = 'HIGH' AND d.window_close_date BETWEEN :s AND :u
    """).bindparams(s=since, u=until))

    return {
        "project": project, "since": since, "until": until,
        "adoption": {r["tool"]: r["active_engineers"] for r in (claude + cursor)},
        "ai_contribution_pct": contribution[0]["ai_contribution_pct"] if contribution else None,
        "ai_rework_pct": rework[0]["rework_pct"] if rework else None,
    }


def initiatives_impl(
    db: Database, *, project: str, since: str, until: str, status: str | None = None,
) -> dict:
    sql = """
        SELECT bi.id AS initiative_id, bi.status, bi.effort_story_points,
               f.forecast_completion_date,
               r.roi_pct
        FROM business_initiatives bi
        LEFT JOIN initiative_forecasts f ON f.initiative_id = bi.id
        LEFT JOIN investment_rois r ON r.initiative_id = bi.id
        WHERE bi.created_date BETWEEN :s AND :u
    """
    params = {"s": since, "u": until}
    if status:
        sql += " AND bi.status = :st"
        params["st"] = status
    sql += " ORDER BY bi.created_date DESC"
    rows = db.fetch_all(text(sql).bindparams(**params))
    return {"project": project, "since": since, "until": until, "initiatives": rows}
```

Wire into `server.py`:

```python
from app.tools.business import investment_profile_impl, ai_adoption_impl, initiatives_impl

@mcp.tool()
def get_investment_profile(since: str, until: str, project: str | None = None) -> dict:
    """% effort split by work category (new features / KTLO / maintenance), with $-totals."""
    return investment_profile_impl(db, project=project or cfg.default_project, since=since, until=until)

@mcp.tool()
def get_ai_adoption(since: str, until: str, project: str | None = None) -> dict:
    """AI tool adoption: active engineers per tool, AI contribution ratio, AI rework rate."""
    return ai_adoption_impl(db, project=project or cfg.default_project, since=since, until=until)

@mcp.tool()
def get_initiatives(since: str, until: str, project: str | None = None, status: str | None = None) -> dict:
    """Business initiatives with effort, forecast, ROI."""
    return initiatives_impl(db, project=project or cfg.default_project, since=since, until=until, status=status)
```

- [ ] **Step 3: Test + commit**

```bash
pytest tests/test_tools_business.py -v
git add docker/devlake-mcp/app/tools/business.py docker/devlake-mcp/tests/test_tools_business.py docker/devlake-mcp/app/server.py
git commit -m "feat(devlake-mcp): business tools (investment_profile, ai_adoption, initiatives)"
```

---

## Task 17 — Build first image + push to ECR

**Files:**
- Modify: `finally-internal/docker/devlake-mcp/README.md` (build doc)
- Create: `finally-internal/terraform/workloads/finally-internal/ecr-devlake-mcp.tf`

- [ ] **Step 1: Add ECR repo Terraform**

`$TF/ecr-devlake-mcp.tf`:

```hcl
resource "aws_ecr_repository" "devlake_mcp" {
  count                = var.enable_devlake_mcp ? 1 : 0
  name                 = "finally-internal/devlake-mcp"
  image_tag_mutability = "MUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }

  encryption_configuration {
    encryption_type = "AES256"
  }

  tags = merge(local.common_tags, {
    Name        = "finally-internal-devlake-mcp"
    Application = "devlake"
    Component   = "mcp-server"
  })
}

resource "aws_ecr_lifecycle_policy" "devlake_mcp" {
  count      = var.enable_devlake_mcp ? 1 : 0
  repository = aws_ecr_repository.devlake_mcp[0].name

  policy = jsonencode({
    rules = [{
      rulePriority = 1
      description  = "retain last 10 images"
      selection    = { tagStatus = "any", countType = "imageCountMoreThan", countNumber = 10 }
      action       = { type = "expire" }
    }]
  })
}
```

- [ ] **Step 2: Add variables** to `$TF/variables.tf`:

```hcl
variable "enable_devlake_mcp"          { type = bool;   default = false }
variable "devlake_mcp_cpu"             { type = number; default = 512 }
variable "devlake_mcp_memory"          { type = number; default = 1024 }
variable "devlake_mcp_default_project" { type = string; default = "finally-DevEx-MCP" }
```

- [ ] **Step 3: Apply just the ECR repo**

```bash
cd $TF
terraform apply -target=aws_ecr_repository.devlake_mcp -target=aws_ecr_lifecycle_policy.devlake_mcp
```

Verify the repo URL printed by terraform.

- [ ] **Step 4: Build and push the first image**

```bash
cd /home/ubuntu/workspaces/finally/finally-internal/docker/devlake-mcp
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin <ACCOUNT>.dkr.ecr.us-east-1.amazonaws.com
docker build -t devlake-mcp:latest .
docker tag devlake-mcp:latest <ACCOUNT>.dkr.ecr.us-east-1.amazonaws.com/finally-internal/devlake-mcp:latest
docker push <ACCOUNT>.dkr.ecr.us-east-1.amazonaws.com/finally-internal/devlake-mcp:latest
```

- [ ] **Step 5: Commit Terraform**

```bash
cd /home/ubuntu/workspaces/finally/finally-internal
git add terraform/workloads/finally-internal/ecr-devlake-mcp.tf terraform/workloads/finally-internal/variables.tf
git commit -m "feat(terraform): ECR repo + variables for devlake-mcp"
```

---

## Task 18 — Okta secret + locals

**Files:**
- Modify: `$TF/okta.tf`
- Modify: `$TF/locals.tf`

- [ ] **Step 1: Add Okta OIDC secret** to `$TF/okta.tf` (clone of `superset_okta_oidc`):

```hcl
resource "aws_secretsmanager_secret" "devlake_mcp_okta_oidc" {
  count       = var.enable_devlake_mcp ? 1 : 0
  name        = "finally-internal/devlake-mcp-okta-oidc-credentials"
  description = "Okta OIDC URLs for Devlake MCP JWT validation"

  tags = merge(local.common_tags, {
    Name         = "devlake-mcp-okta-oidc-credentials"
    Application  = "devlake"
    Component    = "mcp-server"
    AuthProvider = "okta"
  })
}

resource "aws_secretsmanager_secret_version" "devlake_mcp_okta_oidc" {
  count     = var.enable_devlake_mcp ? 1 : 0
  secret_id = aws_secretsmanager_secret.devlake_mcp_okta_oidc[0].id
  secret_string = jsonencode({
    issuer_url    = local.okta_issuer
    domain        = local.okta_domain_url
    authorize_url = "${local.okta_oauth_base}/v1/authorize"
    token_url     = "${local.okta_oauth_base}/v1/token"
    userinfo_url  = "${local.okta_oauth_base}/v1/userinfo"
    jwks_url      = "${local.okta_oauth_base}/v1/keys"
  })
}
```

- [ ] **Step 2: Add `internal_tools.devlake_mcp`** to `$TF/locals.tf` (inside the existing `internal_tools` block):

```hcl
devlake_mcp = {
  enabled = var.enable_devlake_mcp
  port    = 5009
  cpu     = var.devlake_mcp_cpu
  memory  = var.devlake_mcp_memory
  image   = var.enable_devlake_mcp ? "${data.aws_caller_identity.current.account_id}.dkr.ecr.${data.aws_region.current.name}.amazonaws.com/finally-internal/devlake-mcp:latest" : ""
}
```

- [ ] **Step 3: Commit**

```bash
git add terraform/workloads/finally-internal/okta.tf terraform/workloads/finally-internal/locals.tf
git commit -m "feat(terraform): devlake-mcp okta secret + locals entry"
```

---

## Task 19 — ALB target group + listener rule + DNS

**Files:**
- Modify: `$TF/alb-public.tf`

- [ ] **Step 1: Append at end of `alb-public.tf`**

```hcl
# ===========================================
# DEVLAKE MCP SERVICE (AI Agent Access)
# ===========================================

resource "aws_lb_target_group" "devlake_mcp_public" {
  count = var.enable_public_access && var.enable_devlake_mcp ? 1 : 0

  name        = "finally-devlake-mcp-tg"
  port        = 5009
  protocol    = "HTTP"
  vpc_id      = aws_vpc.finally_internal.id
  target_type = "ip"

  deregistration_delay = 30

  health_check {
    enabled             = true
    healthy_threshold   = 2
    interval            = 30
    matcher             = "200,401,405"
    path                = "/mcp"
    port                = "traffic-port"
    protocol            = "HTTP"
    timeout             = 10
    unhealthy_threshold = 3
  }

  tags = merge(local.common_tags, {
    Name        = "finally-devlake-mcp-tg"
    Application = "devlake"
    Component   = "mcp-server"
    Purpose     = "public-target-group"
  })

  lifecycle { create_before_destroy = true }
}

resource "aws_lb_listener_rule" "devlake_mcp" {
  count = var.enable_public_access && var.enable_devlake_mcp ? 1 : 0

  listener_arn = aws_lb_listener.https[0].arn
  priority     = 175

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.devlake_mcp_public[0].arn
  }

  condition {
    host_header { values = ["devlake-mcp.${var.domain_name}"] }
  }

  tags = merge(local.common_tags, {
    Name        = "devlake-mcp-listener-rule"
    Application = "devlake"
    Component   = "mcp-server"
    Purpose     = "host-based-routing-jwt-auth"
  })
}

resource "aws_route53_record" "devlake_mcp" {
  count = var.enable_public_access && var.enable_devlake_mcp ? 1 : 0

  zone_id = aws_route53_zone.public_domain[0].zone_id
  name    = "devlake-mcp.${var.domain_name}"
  type    = "A"

  alias {
    name                   = aws_lb.public[0].dns_name
    zone_id                = aws_lb.public[0].zone_id
    evaluate_target_health = true
  }
}
```

- [ ] **Step 2: Commit**

```bash
git add terraform/workloads/finally-internal/alb-public.tf
git commit -m "feat(terraform): devlake-mcp ALB target group + listener rule + Route53"
```

---

## Task 20 — ECS task definition + service

**Files:**
- Create: `$TF/ecs-devlake-mcp.tf`

- [ ] **Step 1: Create the file**

```hcl
resource "aws_cloudwatch_log_group" "devlake_mcp" {
  count             = var.enable_devlake_mcp ? 1 : 0
  name              = "/ecs/finally-internal/devlake-mcp"
  retention_in_days = var.log_retention_days

  tags = merge(local.common_tags, {
    Name        = "finally-internal-devlake-mcp-logs"
    Application = "devlake"
    Component   = "mcp-server"
  })
}

resource "aws_ecs_task_definition" "devlake_mcp" {
  count = var.enable_devlake_mcp ? 1 : 0

  family                   = "finally-internal-devlake-mcp"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = var.devlake_mcp_cpu
  memory                   = var.devlake_mcp_memory
  execution_role_arn       = aws_iam_role.ecs_execution_role.arn
  task_role_arn            = aws_iam_role.ecs_task_role.arn

  container_definitions = jsonencode([{
    name  = "devlake-mcp"
    image = local.internal_tools.devlake_mcp.image

    portMappings = [{ containerPort = 5009, protocol = "tcp" }]

    environment = [
      { name = "MCP_AUTH_ENABLED",          value = "true" },
      { name = "MCP_JWT_ISSUER",            value = local.okta_issuer },
      { name = "MCP_JWT_AUDIENCE",          value = "api://default,https://devlake-mcp.${var.domain_name}/mcp" },
      { name = "MCP_JWKS_URI",              value = "${local.okta_oauth_base}/v1/keys" },
      { name = "MCP_JWT_ALGORITHM",         value = "RS256" },
      { name = "MCP_RESOURCE_URL",          value = "https://devlake-mcp.${var.domain_name}/mcp" },
      { name = "DEVLAKE_MCP_DEFAULT_PROJECT", value = var.devlake_mcp_default_project },
    ]

    secrets = [
      { name = "DEVLAKE_MCP_DB_URL", valueFrom = "${aws_secretsmanager_secret.devlake_mysql_password.arn}:db_url::" },
    ]

    logConfiguration = {
      logDriver = "awslogs"
      options = {
        awslogs-group         = aws_cloudwatch_log_group.devlake_mcp[0].name
        awslogs-region        = var.region
        awslogs-stream-prefix = "mcp"
      }
    }

    healthCheck = {
      command     = ["CMD-SHELL", "nc -z localhost 5009"]
      interval    = 30
      timeout     = 10
      retries     = 5
      startPeriod = 120
    }
  }])

  tags = merge(local.common_tags, {
    Name        = "finally-internal-devlake-mcp-task"
    Application = "devlake"
    Component   = "mcp-server"
  })
}

resource "aws_ecs_service" "devlake_mcp" {
  count = var.enable_devlake_mcp ? 1 : 0

  name            = "finally-internal-devlake-mcp-service"
  cluster         = aws_ecs_cluster.finally_internal.id
  task_definition = aws_ecs_task_definition.devlake_mcp[0].arn
  desired_count   = 1
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = aws_subnet.private[*].id
    security_groups  = [aws_security_group.ecs.id]
    assign_public_ip = false
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.devlake_mcp_public[0].arn
    container_name   = "devlake-mcp"
    container_port   = 5009
  }

  deployment_circuit_breaker {
    enable   = true
    rollback = true
  }

  depends_on = [aws_lb_listener.https]

  tags = merge(local.common_tags, {
    Name        = "finally-internal-devlake-mcp-service"
    Application = "devlake"
    Component   = "mcp-server"
  })
}
```

**Pre-req check**: the value `${aws_secretsmanager_secret.devlake_mysql_password.arn}` requires the `devlake-mysql-password` secret to be JSON-structured with a `db_url` field. If it currently stores just the password (not a JSON blob), update the secret to include a structured `db_url` value pointing at the existing devlake user — one-time SQL-free change in Secrets Manager. Document this in the README.

- [ ] **Step 2: Commit**

```bash
git add terraform/workloads/finally-internal/ecs-devlake-mcp.tf
git commit -m "feat(terraform): devlake-mcp ECS task + service"
```

---

## Task 21 — Manual Okta + apply + smoke test

This task is **manual** — no code changes. The agent should verify each step is completed.

- [ ] **Step 1: Register Okta app** (manual, 5 min)

In Okta admin: create API Services app `devlake-mcp` (or web app if the OAuth flow needs it for claude.ai). Capture the `client_id`. No client secret needed for the public-client PKCE flow Claude uses.

- [ ] **Step 2: Add `devlake_api` scope** (manual, 2 min)

In Okta admin → Authorization Servers → `default` → Scopes → Add `devlake_api`.

- [ ] **Step 3: Add audience** (manual, 2 min)

In Okta admin → Authorization Servers → `default` → Access Policies → ensure tokens issued for the new app have `aud` claim `api://default` (already the default) and accept `https://devlake-mcp.internal.finally.com/mcp` as a valid resource URI.

- [ ] **Step 4: Assign group** (manual, 2 min)

Assign Okta group `devlake-users` (or your standard internal-tools group) to the new app.

- [ ] **Step 5: Apply terraform**

```bash
cd $TF
terraform plan -var enable_devlake_mcp=true
# review plan
terraform apply -var enable_devlake_mcp=true
```

Expected output: ECR repo, ECS log group, ECS task definition, ECS service, target group, listener rule, Route53 A record, Okta secret all created.

- [ ] **Step 6: Wait for ECS service to stabilize**

```bash
aws ecs describe-services \
  --cluster finally-internal \
  --services finally-internal-devlake-mcp-service \
  --query 'services[0].{desired:desiredCount,running:runningCount,deployments:deployments[*].status}' \
  --output json
```

Expected: `desired=1`, `running=1`, deployment `PRIMARY` with status `STEADY`.

- [ ] **Step 7: Smoke test the well-known endpoint** (no auth)

```bash
curl -s https://devlake-mcp.internal.finally.com/.well-known/oauth-protected-resource | jq .
```

Expected JSON:
```json
{
  "resource": "https://devlake-mcp.internal.finally.com/mcp",
  "authorization_servers": ["https://finally.okta.com/oauth2/default"],
  "scopes_supported": ["devlake_api"],
  "bearer_methods_supported": ["header"]
}
```

- [ ] **Step 8: Add `.mcp.json` entry locally**

In `~/.mcp.json` or project-level `.mcp.json`:

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

- [ ] **Step 9: Smoke test from Claude Code**

Start a new session, watch the OAuth dance complete, then call:

```
list_projects()
```

Expected: list including `finally-DevEx` (and `finally-DevEx-MCP` once that project is created).

```
get_team_scorecard(since="2026-04-01", until="2026-05-01")
```

Expected: shape from Task 11's test — velocity, shipping, ai_mix, code_health, reliability with `status: no_data`.

- [ ] **Step 10: Done. Commit nothing** (manual task).

---

## Acceptance Checklist (mirrors the spec)

- [ ] All 13 tools callable from claude.ai and Claude Code.
- [ ] `list_metrics()` returns exactly 16 entries.
- [ ] `describe_metric()` returns scoping rule + known_gaps for each.
- [ ] `get_team_scorecard()` returns sensible numbers for the dedicated project; reliability labeled `no_data` until incidents ingest.
- [ ] Unauthenticated request to `/mcp` returns 401 + `WWW-Authenticate` header.
- [ ] `/.well-known/oauth-protected-resource` returns RFC 9728 JSON unauthenticated.
- [ ] JWT with wrong audience rejected; valid JWT accepted.
- [ ] Audit log captures `email`, `tool`, `params`, `duration_ms` per call (CloudWatch).
- [ ] AST guardrail rejects `INSERT/UPDATE/DELETE/DROP/ALTER` (unit-tested).
- [ ] ECS task health check stays green for 24h post-deploy.

---

## Notes for the executing agent

- Use the `superpowers:subagent-driven-development` skill — one fresh subagent per task with the two-stage review pattern that produced PR #16 cleanly.
- Tasks 11–16 are independent (each is one tool added to its module) — they can run in parallel sibling worktrees if you set it up the same way as Phase B's tasks 5/6/7.
- The cross-repo aspect (spec/plan in `incubator-devlake`, code in `finally-internal`) means each subagent should be told explicitly which working directory to use; the `$APP` and `$TF` shorthands above resolve those paths.
