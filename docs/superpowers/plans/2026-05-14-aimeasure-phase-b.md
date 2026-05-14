# `aimeasure` Phase B — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship Phase B of the `aimeasure` plugin — surface the verification effort that AI-generated code creates (review-to-author ratio, review comments per LOC by cohort), and the "dark matter" that doesn't show up in PR counts (Slack participation by category, after-hours patterns, a behavioral sentiment proxy).

**Architecture:** Three new subtasks layered on top of Phase A's `pr_ai_cohort` foundation. Reads source-plugin tables read-only (`pull_request_comments`, `pull_request_reviewers`, `_tool_slack_channel_messages`, `_tool_slack_channels`); writes its own tables (`engineer_verification_effort`, `engineer_slack_signals`, `engineer_dxi_proxy`, plus one helper config table `slack_channel_categories`). Subtasks run in any order — they only depend on Phase A's outputs.

**Tech Stack:** Go 1.20, GORM v2, MySQL 8 / PostgreSQL 14 (dual-driver), e2e via `helpers/e2ehelper`, Grafana 9. Dependencies: Phase A must be deployed (provides `pr_ai_cohort`); `slack` plugin must be enabled (provides `_tool_slack_channel_messages`); `github` (or another PR collector) must be enabled (provides `pull_request_comments`, `pull_request_reviewers`).

**Spec reference:** `docs/superpowers/specs/2026-05-13-ai-era-signals-design.md` § 6 (Phase B).

**Out of scope** (Phase B+ or Phase C):
- LLM-based tone/sentiment analysis of message text (Phase B uses behavioral proxies only)
- DXI survey ingestion pipeline (table has nullable survey columns; ingestion is manual until a survey plugin exists)
- Cross-tenant Slack workspaces (assumes one `ConnectionId`)
- DM-channel data (privacy-out by default; not collected by `slack` plugin anyway)

---

## File Structure

**New files:**

```
backend/plugins/aimeasure/
├── models/
│   ├── engineer_verification_effort.go              # verification effort table
│   ├── engineer_slack_signals.go                    # slack participation table
│   ├── engineer_dxi_proxy.go                        # sentiment proxy table
│   ├── slack_channel_category.go                    # channel→category mapping table
│   └── migrationscripts/
│       └── 20260514_phase_b_schema.go               # creates the 4 new tables
├── tasks/
│   ├── period.go                                    # ISO-week helpers (PeriodWeekStart, etc.)
│   ├── period_test.go
│   ├── compute_verification_effort.go               # subtask 4
│   ├── compute_verification_effort_test.go
│   ├── compute_slack_signals.go                     # subtask 5
│   ├── compute_slack_signals_test.go
│   ├── compute_sentiment_proxy.go                   # subtask 6
│   └── compute_sentiment_proxy_test.go
└── e2e/
    ├── compute_verification_effort_test.go          # e2e for subtask 4
    ├── compute_slack_signals_test.go                # e2e for subtask 5
    ├── compute_sentiment_proxy_test.go              # e2e for subtask 6
    └── fixtures/
        ├── pull_request_comments.csv
        ├── pull_request_reviewers.csv
        ├── slack_channels.csv
        ├── slack_channel_messages.csv
        ├── slack_channel_categories.csv             # seed for the helper table
        ├── pr_ai_cohort.csv                         # Phase A output, used as input
        ├── expected_engineer_verification_effort.csv
        ├── expected_engineer_slack_signals.csv
        └── expected_engineer_dxi_proxy.csv

grafana/dashboards/
└── InvisibleWork.json                               # 7-panel Phase B dashboard
```

**Modified files:**

- `backend/plugins/aimeasure/impl/impl.go` — register the 3 new subtasks in `SubTaskMetas()`, extend `MakeMetricPluginPipelinePlanV200` subtask list, extend `RequiredDataEntities` to include slack tables and PR comment tables, extend `GetTablesInfo`
- `backend/plugins/aimeasure/models/migrationscripts/register.go` — append the new migration script
- `backend/plugins/aimeasure/README.md` — Phase B section + resolved-decisions tracker
- `backend/plugins/table_info_test.go` — no change needed (aimeasure already registered; the FeedIn captures all tables returned by `GetTablesInfo`)

---

## Task 1 — Phase B data models

**Files:**
- Create: `backend/plugins/aimeasure/models/engineer_verification_effort.go`
- Create: `backend/plugins/aimeasure/models/engineer_slack_signals.go`
- Create: `backend/plugins/aimeasure/models/engineer_dxi_proxy.go`
- Create: `backend/plugins/aimeasure/models/slack_channel_category.go`
- Modify: `backend/plugins/aimeasure/models/models_test.go` (extend `TestTableNames`)

- [ ] **Step 1: Extend the failing build test**

Add to `backend/plugins/aimeasure/models/models_test.go`:

```go
func TestPhaseBTableNames(t *testing.T) {
	cases := map[string]string{
		"engineer_verification_effort": EngineerVerificationEffort{}.TableName(),
		"engineer_slack_signals":       EngineerSlackSignals{}.TableName(),
		"engineer_dxi_proxy":           EngineerDxiProxy{}.TableName(),
		"aimeasure_slack_channel_categories": SlackChannelCategory{}.TableName(),
	}
	for expected, actual := range cases {
		if expected != actual {
			t.Errorf("expected table name %q, got %q", expected, actual)
		}
	}
}
```

Run: `docker run --rm -v "/home/ubuntu/workspaces/finally/incubator-devlake:/workspace" -w /workspace/backend mericodev/lake-builder:latest sh -c 'go test ./plugins/aimeasure/models/... 2>&1 | tail -5'`. Expect compile error: `undefined: EngineerVerificationEffort` etc.

- [ ] **Step 2: Implement `EngineerVerificationEffort`**

Create `backend/plugins/aimeasure/models/engineer_verification_effort.go`:

```go
/* Apache 2.0 header */

package models

import "time"

// EngineerVerificationEffort records how much time an engineer spent reviewing
// vs authoring code per ISO week. One row per (engineer_id, period_week).
// Written by computeVerificationEffort, recomputed on every run.
type EngineerVerificationEffort struct {
	EngineerId               string    `gorm:"primaryKey;type:varchar(255)" json:"engineerId"`
	PeriodWeek               time.Time `gorm:"primaryKey;type:date" json:"periodWeek"` // ISO week start (Monday) at 00:00 UTC
	AuthorMinutes            int       `gorm:"type:int" json:"authorMinutes"`
	ReviewerMinutes          int       `gorm:"type:int" json:"reviewerMinutes"`
	ReviewToAuthorRatio      float64   `gorm:"type:decimal(8,4)" json:"reviewToAuthorRatio"`
	ReviewCommentsTotal      int       `gorm:"type:int" json:"reviewCommentsTotal"`
	ReviewCommentsPerLoc     float64   `gorm:"type:decimal(8,4)" json:"reviewCommentsPerLoc"`
	ReviewCommentsHighCohort int       `gorm:"type:int" json:"reviewCommentsHighCohort"`
	ReviewCommentsPerLocHigh float64   `gorm:"type:decimal(8,4)" json:"reviewCommentsPerLocHigh"`
	ComputedAt               time.Time `gorm:"not null" json:"computedAt"`
}

func (EngineerVerificationEffort) TableName() string {
	return "engineer_verification_effort"
}
```

- [ ] **Step 3: Implement `EngineerSlackSignals`**

Create `backend/plugins/aimeasure/models/engineer_slack_signals.go`:

```go
/* Apache 2.0 header */

package models

import "time"

// ChannelCategory enumerates the buckets aimeasure groups Slack channels into.
// Stored as varchar; mapping is kept in slack_channel_categories.
type ChannelCategory string

const (
	CategoryEngineering        ChannelCategory = "engineering"
	CategoryIncidentSupport    ChannelCategory = "incident_support"
	CategoryDesignArchitecture ChannelCategory = "design_architecture"
	CategoryGeneral            ChannelCategory = "general"
)

// EngineerSlackSignals records per-engineer per-week Slack participation
// in each channel category. One row per (engineer_id, period_week, channel_category).
type EngineerSlackSignals struct {
	EngineerId               string          `gorm:"primaryKey;type:varchar(255)" json:"engineerId"`
	PeriodWeek               time.Time       `gorm:"primaryKey;type:date" json:"periodWeek"`
	ChannelCategory          ChannelCategory `gorm:"primaryKey;type:varchar(32)" json:"channelCategory"`
	MessageCount             int             `gorm:"type:int" json:"messageCount"`
	ThreadParticipationCount int             `gorm:"type:int" json:"threadParticipationCount"`
	AfterHoursMessageCount   int             `gorm:"type:int" json:"afterHoursMessageCount"`
	AfterHoursRatio          float64         `gorm:"type:decimal(5,4)" json:"afterHoursRatio"`
	ComputedAt               time.Time       `gorm:"not null" json:"computedAt"`
}

func (EngineerSlackSignals) TableName() string {
	return "engineer_slack_signals"
}
```

- [ ] **Step 4: Implement `EngineerDxiProxy`**

Create `backend/plugins/aimeasure/models/engineer_dxi_proxy.go`:

```go
/* Apache 2.0 header */

package models

import "time"

// EngineerDxiProxy holds a behavioral sentiment proxy plus optional survey data
// for an engineer in a given ISO week. Survey fields are nullable — populated
// only if a DXI/eNPS survey ingest exists (out of scope for Phase B).
type EngineerDxiProxy struct {
	EngineerId         string     `gorm:"primaryKey;type:varchar(255)" json:"engineerId"`
	PeriodWeek         time.Time  `gorm:"primaryKey;type:date" json:"periodWeek"`
	SentimentScore     float64    `gorm:"type:decimal(5,2)" json:"sentimentScore"`     // 0–100, behavioral
	BadDeveloperDayFlag bool      `gorm:"type:bool" json:"badDeveloperDayFlag"`
	LastSurveyDate     *time.Time `gorm:"type:date" json:"lastSurveyDate,omitempty"`
	LastSurveyDxi      *float64   `gorm:"type:decimal(5,2)" json:"lastSurveyDxi,omitempty"`
	ComputedAt         time.Time  `gorm:"not null" json:"computedAt"`
}

func (EngineerDxiProxy) TableName() string {
	return "engineer_dxi_proxy"
}
```

- [ ] **Step 5: Implement `SlackChannelCategory`**

Create `backend/plugins/aimeasure/models/slack_channel_category.go`:

```go
/* Apache 2.0 header */

package models

// SlackChannelCategory maps a Slack channel (by name or ID) to one of the
// aimeasure ChannelCategory values. Manually maintained — engineering leadership
// curates the mapping. Lookup falls back to "general" for unmapped channels.
type SlackChannelCategory struct {
	ChannelKey string `gorm:"primaryKey;type:varchar(255)" json:"channelKey"` // channel name OR channel ID (whichever is more stable)
	Category   string `gorm:"type:varchar(32);not null" json:"category"`
	Note       string `gorm:"type:varchar(500)" json:"note,omitempty"`
}

func (SlackChannelCategory) TableName() string {
	return "aimeasure_slack_channel_categories"
}
```

- [ ] **Step 6: Re-run the test, verify it passes**

```bash
docker run --rm -v "/home/ubuntu/workspaces/finally/incubator-devlake:/workspace" -w /workspace/backend mericodev/lake-builder:latest sh -c '
go test ./plugins/aimeasure/models/... -v 2>&1 | tail -10
'
```

Expected: `TestTableNames` AND `TestPhaseBTableNames` both PASS.

- [ ] **Step 7: Commit**

```bash
git add backend/plugins/aimeasure/models/
git commit -m "feat(aimeasure): Phase B data models (verification, slack, sentiment)"
```

---

## Task 2 — Init schema migration for Phase B tables

**Files:**
- Create: `backend/plugins/aimeasure/models/migrationscripts/20260514_phase_b_schema.go`
- Modify: `backend/plugins/aimeasure/models/migrationscripts/register.go`

- [ ] **Step 1: Write the migration script with frozen struct snapshots**

Create `backend/plugins/aimeasure/models/migrationscripts/20260514_phase_b_schema.go`:

```go
/* Apache 2.0 header */

package migrationscripts

import (
	"time"

	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/helpers/migrationhelper"
)

type phaseBSchema struct{}

// Phase B tables. Snapshot types are frozen to this migration date so future
// model changes do not retroactively alter what this migration creates.

type engineerVerificationEffort20260514 struct {
	EngineerId               string    `gorm:"primaryKey;type:varchar(255)"`
	PeriodWeek               time.Time `gorm:"primaryKey;type:date"`
	AuthorMinutes            int       `gorm:"type:int"`
	ReviewerMinutes          int       `gorm:"type:int"`
	ReviewToAuthorRatio      float64   `gorm:"type:decimal(8,4)"`
	ReviewCommentsTotal      int       `gorm:"type:int"`
	ReviewCommentsPerLoc     float64   `gorm:"type:decimal(8,4)"`
	ReviewCommentsHighCohort int       `gorm:"type:int"`
	ReviewCommentsPerLocHigh float64   `gorm:"type:decimal(8,4)"`
	ComputedAt               time.Time `gorm:"not null"`
}

func (engineerVerificationEffort20260514) TableName() string { return "engineer_verification_effort" }

type engineerSlackSignals20260514 struct {
	EngineerId               string    `gorm:"primaryKey;type:varchar(255)"`
	PeriodWeek               time.Time `gorm:"primaryKey;type:date"`
	ChannelCategory          string    `gorm:"primaryKey;type:varchar(32)"`
	MessageCount             int       `gorm:"type:int"`
	ThreadParticipationCount int       `gorm:"type:int"`
	AfterHoursMessageCount   int       `gorm:"type:int"`
	AfterHoursRatio          float64   `gorm:"type:decimal(5,4)"`
	ComputedAt               time.Time `gorm:"not null"`
}

func (engineerSlackSignals20260514) TableName() string { return "engineer_slack_signals" }

type engineerDxiProxy20260514 struct {
	EngineerId          string     `gorm:"primaryKey;type:varchar(255)"`
	PeriodWeek          time.Time  `gorm:"primaryKey;type:date"`
	SentimentScore      float64    `gorm:"type:decimal(5,2)"`
	BadDeveloperDayFlag bool       `gorm:"type:bool"`
	LastSurveyDate      *time.Time `gorm:"type:date"`
	LastSurveyDxi       *float64   `gorm:"type:decimal(5,2)"`
	ComputedAt          time.Time  `gorm:"not null"`
}

func (engineerDxiProxy20260514) TableName() string { return "engineer_dxi_proxy" }

type slackChannelCategory20260514 struct {
	ChannelKey string `gorm:"primaryKey;type:varchar(255)"`
	Category   string `gorm:"type:varchar(32);not null"`
	Note       string `gorm:"type:varchar(500)"`
}

func (slackChannelCategory20260514) TableName() string { return "aimeasure_slack_channel_categories" }

func (*phaseBSchema) Up(basicRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&engineerVerificationEffort20260514{},
		&engineerSlackSignals20260514{},
		&engineerDxiProxy20260514{},
		&slackChannelCategory20260514{},
	)
}

func (*phaseBSchema) Version() uint64 {
	return 20260514000001
}

func (*phaseBSchema) Name() string {
	return "aimeasure: Phase B schema (verification, slack, sentiment + channel mapping)"
}
```

- [ ] **Step 2: Register the migration**

Edit `backend/plugins/aimeasure/models/migrationscripts/register.go`:

```go
func All() []plugin.MigrationScript {
	return []plugin.MigrationScript{
		new(initSchema),
		new(phaseBSchema), // Phase B — added 2026-05-14
	}
}
```

- [ ] **Step 3: Verify the migration package compiles**

```bash
docker run --rm -v "/home/ubuntu/workspaces/finally/incubator-devlake:/workspace" -w /workspace/backend mericodev/lake-builder:latest sh -c '
go build ./plugins/aimeasure/...
'
```

Expected: clean compile.

- [ ] **Step 4: Commit**

```bash
git add backend/plugins/aimeasure/models/migrationscripts/
git commit -m "feat(aimeasure): Phase B schema migration"
```

---

## Task 3 — Wire `GetTablesInfo` for the new tables

**Files:**
- Modify: `backend/plugins/aimeasure/impl/impl.go`

- [ ] **Step 1: Add the new models to `GetTablesInfo`**

In `backend/plugins/aimeasure/impl/impl.go`, extend `GetTablesInfo`:

```go
func (p AIMeasure) GetTablesInfo() []dal.Tabler {
	return []dal.Tabler{
		&models.PRAICohort{},
		&models.PRDefectSignals{},
		&models.PRChangeComposition{},
		&models.AccountOverride{},
		&models.EngineerRole{},
		// Phase B
		&models.EngineerVerificationEffort{},
		&models.EngineerSlackSignals{},
		&models.EngineerDxiProxy{},
		&models.SlackChannelCategory{},
	}
}
```

- [ ] **Step 2: Run the parent table-info test, verify it passes**

```bash
docker run --rm -v "/home/ubuntu/workspaces/finally/incubator-devlake:/workspace" -w /workspace/backend mericodev/lake-builder:latest sh -c '
go install github.com/vektra/mockery/v2@v2.20.0 && make mock >/dev/null 2>&1 && go test -run "^Test_GetPluginTablesInfo$" -v ./plugins/ 2>&1 | tail -5
'
```

Expected: `--- PASS: Test_GetPluginTablesInfo`. (If `make mock` fails because of host Go version, skip — local `go build ./plugins/aimeasure/...` is sufficient and CI runs the full mock+test chain.)

- [ ] **Step 3: Commit**

```bash
git add backend/plugins/aimeasure/impl/impl.go
git commit -m "feat(aimeasure): register Phase B tables in GetTablesInfo"
```

---

## Task 4 — ISO-week and category-mapping helpers (pure functions)

**Files:**
- Create: `backend/plugins/aimeasure/tasks/period.go`
- Create: `backend/plugins/aimeasure/tasks/period_test.go`

ISO week truncation is one of the spots where MySQL and Postgres diverge most painfully. Doing it in Go and emitting a literal `DATE` parameter sidesteps the issue and matches the pattern Phase A used for `WindowCloseDate`.

- [ ] **Step 1: Write failing tests for `PeriodWeekStart` and `CategorizeChannel`**

Create `backend/plugins/aimeasure/tasks/period_test.go`:

```go
/* Apache 2.0 header */

package tasks

import (
	"testing"
	"time"

	"github.com/apache/incubator-devlake/plugins/aimeasure/models"
)

func TestPeriodWeekStart(t *testing.T) {
	cases := []struct {
		name string
		in   string // RFC3339
		want string // YYYY-MM-DD of Monday at 00:00 UTC
	}{
		{"monday stays put", "2026-05-11T00:00:00Z", "2026-05-11"},
		{"tuesday rolls back", "2026-05-12T15:30:00Z", "2026-05-11"},
		{"sunday rolls back to prior monday", "2026-05-17T23:59:59Z", "2026-05-11"},
		{"timezone is normalized to UTC", "2026-05-11T02:00:00-05:00", "2026-05-11"}, // = 07:00 UTC
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in, _ := time.Parse(time.RFC3339, c.in)
			got := PeriodWeekStart(in)
			gotStr := got.Format("2006-01-02")
			if gotStr != c.want {
				t.Errorf("PeriodWeekStart(%s) = %s, want %s", c.in, gotStr, c.want)
			}
			if got.Hour() != 0 || got.Minute() != 0 || got.Second() != 0 {
				t.Errorf("week start should be 00:00:00, got %v", got)
			}
		})
	}
}

func TestIsAfterHours(t *testing.T) {
	// After-hours = before 09:00 or after 18:00 local-business-day OR weekend.
	// For Phase B we use UTC and a fixed 09:00-18:00 window; per-engineer
	// timezone is Phase C work.
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"monday 10am UTC", "2026-05-11T10:00:00Z", false},
		{"monday 8am UTC", "2026-05-11T08:00:00Z", true},
		{"monday 8pm UTC", "2026-05-11T20:00:00Z", true},
		{"saturday noon UTC", "2026-05-16T12:00:00Z", true},
		{"sunday 2pm UTC", "2026-05-17T14:00:00Z", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in, _ := time.Parse(time.RFC3339, c.in)
			if got := IsAfterHours(in); got != c.want {
				t.Errorf("IsAfterHours(%s) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

func TestCategorizeChannel(t *testing.T) {
	mapping := map[string]models.ChannelCategory{
		"eng-platform":  models.CategoryEngineering,
		"C0123ABCDEF":   models.CategoryIncidentSupport, // mapped by ID
		"design-review": models.CategoryDesignArchitecture,
	}
	cases := []struct {
		name        string
		channelName string
		channelId   string
		want        models.ChannelCategory
	}{
		{"name match wins over id", "eng-platform", "C0099", models.CategoryEngineering},
		{"id match used if name missing", "", "C0123ABCDEF", models.CategoryIncidentSupport},
		{"unmapped falls back to general", "random-banter", "C9999", models.CategoryGeneral},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := CategorizeChannel(mapping, c.channelName, c.channelId); got != c.want {
				t.Errorf("CategorizeChannel(%q,%q) = %s, want %s", c.channelName, c.channelId, got, c.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run the test, verify it fails**

Expected: `undefined: PeriodWeekStart`, `undefined: IsAfterHours`, `undefined: CategorizeChannel`.

- [ ] **Step 3: Implement the helpers**

Create `backend/plugins/aimeasure/tasks/period.go`:

```go
/* Apache 2.0 header */

package tasks

import (
	"time"

	"github.com/apache/incubator-devlake/plugins/aimeasure/models"
)

// PeriodWeekStart returns the Monday-00:00 UTC of the ISO week containing t.
// All Phase B per-week aggregations bucket by this value.
func PeriodWeekStart(t time.Time) time.Time {
	u := t.UTC()
	// Go's Weekday: Sunday=0, Monday=1 ... Saturday=6. Convert to Mon=0..Sun=6.
	offset := int(u.Weekday()) - 1
	if offset < 0 {
		offset = 6 // Sunday → 6 days back to last Monday
	}
	monday := u.AddDate(0, 0, -offset)
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.UTC)
}

// IsAfterHours returns true when t falls outside Mon–Fri 09:00–18:00 UTC.
// Phase B is workspace-uniform; per-engineer timezone is Phase C work and
// will replace this with a lookup against an engineer_timezone table.
func IsAfterHours(t time.Time) bool {
	u := t.UTC()
	switch u.Weekday() {
	case time.Saturday, time.Sunday:
		return true
	}
	h := u.Hour()
	return h < 9 || h >= 18
}

// CategorizeChannel resolves a channel to its aimeasure category. It tries
// the channel name first (more stable for human-curated mappings), then the
// channel ID. Unmapped channels fall back to CategoryGeneral.
func CategorizeChannel(mapping map[string]models.ChannelCategory, channelName, channelId string) models.ChannelCategory {
	if c, ok := mapping[channelName]; ok && channelName != "" {
		return c
	}
	if c, ok := mapping[channelId]; ok && channelId != "" {
		return c
	}
	return models.CategoryGeneral
}
```

- [ ] **Step 4: Run the test, verify it passes**

```bash
docker run --rm -v "/home/ubuntu/workspaces/finally/incubator-devlake:/workspace" -w /workspace/backend mericodev/lake-builder:latest sh -c '
go test -run "^TestPeriod|^TestIsAfterHours$|^TestCategorizeChannel$" ./plugins/aimeasure/tasks/... -v 2>&1 | tail -20
'
```

Expected: all sub-tests PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/plugins/aimeasure/tasks/period.go backend/plugins/aimeasure/tasks/period_test.go
git commit -m "feat(aimeasure): ISO-week, after-hours, and channel-category helpers"
```

---

## Task 5 — `computeVerificationEffort` subtask

**Files:**
- Create: `backend/plugins/aimeasure/tasks/compute_verification_effort.go`
- Create: `backend/plugins/aimeasure/tasks/compute_verification_effort_test.go`

The pure-function core: convert raw author/reviewer activity into the per-week aggregates. The DB code on top reads from `pull_requests`, `pull_request_comments`, `pull_request_reviewers`, and `pr_ai_cohort`.

**Time model (Phase B simplifying assumption):** the spec calls for `author_minutes` and `reviewer_minutes` but DevLake doesn't track wall-clock authoring time. We use a proxy:
- **author_minutes:** estimated as `PR LOC / 5` (5 LOC/min sustained writing — industry rough) clipped to [10, 240] per PR, summed.
- **reviewer_minutes:** estimated as `15 + 2 × num_review_comments` per reviewed PR, clipped to [10, 120].

Both proxies are documented in the README as Phase B heuristics; Phase C may replace with timer-based or git-blame-based measurements.

- [ ] **Step 1: Write failing tests for the pure aggregator**

Create `backend/plugins/aimeasure/tasks/compute_verification_effort_test.go`:

```go
/* Apache 2.0 header */

package tasks

import "testing"

func TestEstimateAuthorMinutes(t *testing.T) {
	cases := []struct {
		loc  int
		want int
	}{
		{0, 10},      // floor
		{50, 10},     // 10 min == floor
		{100, 20},
		{500, 100},
		{2000, 240},  // ceiling
		{10000, 240}, // ceiling holds
	}
	for _, c := range cases {
		if got := EstimateAuthorMinutes(c.loc); got != c.want {
			t.Errorf("EstimateAuthorMinutes(%d) = %d, want %d", c.loc, got, c.want)
		}
	}
}

func TestEstimateReviewerMinutes(t *testing.T) {
	cases := []struct {
		numComments int
		want        int
	}{
		{0, 15},  // baseline
		{5, 25},  // 15 + 2*5
		{30, 75},
		{60, 120}, // ceiling
		{100, 120},
	}
	for _, c := range cases {
		if got := EstimateReviewerMinutes(c.numComments); got != c.want {
			t.Errorf("EstimateReviewerMinutes(%d) = %d, want %d", c.numComments, got, c.want)
		}
	}
}

func TestSafeRatio(t *testing.T) {
	cases := []struct {
		num, denom int
		want       float64
	}{
		{0, 0, 0.0},
		{100, 0, 0.0},     // div by zero → 0
		{50, 100, 0.5},
		{300, 100, 3.0},
	}
	for _, c := range cases {
		got := SafeRatio(c.num, c.denom)
		if got != c.want {
			t.Errorf("SafeRatio(%d,%d) = %v, want %v", c.num, c.denom, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run the test, verify it fails (compile error on the three helpers)**

- [ ] **Step 3: Implement the subtask**

Create `backend/plugins/aimeasure/tasks/compute_verification_effort.go`:

```go
/* Apache 2.0 header */

package tasks

import (
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/aimeasure/models"
)

var ComputeVerificationEffortMeta = plugin.SubTaskMeta{
	Name:             "computeVerificationEffort",
	EntryPoint:       ComputeVerificationEffort,
	EnabledByDefault: true,
	Description:      "Aggregate per-engineer per-week authoring vs reviewing effort and AI-cohort comment density",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CODE},
}

// EstimateAuthorMinutes returns the proxy authoring minutes for a PR of the given LOC.
// Proxy: 5 LOC/min, clipped to [10, 240]. Documented in the README.
func EstimateAuthorMinutes(loc int) int {
	m := loc / 5
	if m < 10 {
		return 10
	}
	if m > 240 {
		return 240
	}
	return m
}

// EstimateReviewerMinutes returns the proxy review minutes given the number of
// review comments on the PR. Proxy: 15 + 2*comments, clipped to [10, 120].
func EstimateReviewerMinutes(numComments int) int {
	m := 15 + 2*numComments
	if m < 10 {
		return 10
	}
	if m > 120 {
		return 120
	}
	return m
}

// SafeRatio returns num/denom or 0.0 when denom is 0.
func SafeRatio(num, denom int) float64 {
	if denom == 0 {
		return 0.0
	}
	return float64(num) / float64(denom)
}

// authoredPR is one row from the authored-PR scan.
type authoredPR struct {
	PRId       string    `gorm:"column:pr_id"`
	AuthorId   string    `gorm:"column:author_id"`
	MergedDate time.Time `gorm:"column:merged_date"`
	Additions  int       `gorm:"column:additions"`
	Deletions  int       `gorm:"column:deletions"`
	AICohort   string    `gorm:"column:ai_cohort"`
}

// reviewerActivity is one row from the reviewer-comment scan.
type reviewerActivity struct {
	PRId        string    `gorm:"column:pr_id"`
	ReviewerId  string    `gorm:"column:reviewer_id"`
	CommentedAt time.Time `gorm:"column:created_date"`
	AICohort    string    `gorm:"column:ai_cohort"`
}

// effortBucket is the aggregator key (engineer, week).
type effortBucket struct {
	EngineerId string
	WeekStart  time.Time
}

// effortAgg is the running aggregate for a (engineer, week) bucket.
type effortAgg struct {
	AuthorMinutes            int
	ReviewerMinutes          int
	AuthorLOC                int
	ReviewedLOC              int
	ReviewCommentsTotal      int
	ReviewCommentsHighCohort int
	ReviewedLOCHigh          int
}

func ComputeVerificationEffort(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	logger := taskCtx.GetLogger()
	now := time.Now().UTC()

	buckets := make(map[effortBucket]*effortAgg)
	getOrCreate := func(eng string, week time.Time) *effortAgg {
		key := effortBucket{EngineerId: eng, WeekStart: week}
		if a, ok := buckets[key]; ok {
			return a
		}
		a := &effortAgg{}
		buckets[key] = a
		return a
	}

	// 1. Author-side: walk all merged PRs with author, join to pr_ai_cohort.
	var authored []authoredPR
	if err := db.All(&authored,
		dal.Select("pr.id AS pr_id, pr.author_id AS author_id, pr.merged_date AS merged_date, pr.additions AS additions, pr.deletions AS deletions, COALESCE(c.ai_cohort,'NONE') AS ai_cohort"),
		dal.From("pull_requests pr"),
		dal.Join("LEFT JOIN pr_ai_cohort c ON c.pr_id = pr.id"),
		dal.Where("pr.merged_date IS NOT NULL AND pr.author_id IS NOT NULL"),
	); err != nil {
		return errors.Default.Wrap(err, "failed to load authored PRs")
	}
	for _, a := range authored {
		loc := a.Additions + a.Deletions
		bucket := getOrCreate(a.AuthorId, PeriodWeekStart(a.MergedDate))
		bucket.AuthorMinutes += EstimateAuthorMinutes(loc)
		bucket.AuthorLOC += loc
	}

	// 2. Reviewer-side: walk pull_request_comments with type=REVIEW or DIFF,
	//    grouped per (reviewer, pr) so reviewer effort gets the per-PR comment-count proxy.
	type prc struct {
		PRId        string    `gorm:"column:pr_id"`
		AccountId   string    `gorm:"column:account_id"`
		CreatedDate time.Time `gorm:"column:created_date"`
		AICohort    string    `gorm:"column:ai_cohort"`
		Additions   int       `gorm:"column:additions"`
		Deletions   int       `gorm:"column:deletions"`
	}
	var comments []prc
	if err := db.All(&comments,
		dal.Select("c.pull_request_id AS pr_id, c.account_id AS account_id, c.created_date AS created_date, COALESCE(coh.ai_cohort,'NONE') AS ai_cohort, pr.additions AS additions, pr.deletions AS deletions"),
		dal.From("pull_request_comments c"),
		dal.Join("INNER JOIN pull_requests pr ON pr.id = c.pull_request_id"),
		dal.Join("LEFT JOIN pr_ai_cohort coh ON coh.pr_id = c.pull_request_id"),
		dal.Where("c.account_id IS NOT NULL AND c.account_id != pr.author_id"),
	); err != nil {
		return errors.Default.Wrap(err, "failed to load PR comments")
	}

	// Per (reviewer, pr) aggregation first, so review-minutes uses the per-PR count.
	type rpKey struct {
		Reviewer string
		PR       string
	}
	type rpAgg struct {
		Comments  int
		FirstSeen time.Time
		AICohort  string
		LOC       int
	}
	rp := map[rpKey]*rpAgg{}
	for _, c := range comments {
		k := rpKey{Reviewer: c.AccountId, PR: c.PRId}
		a, ok := rp[k]
		if !ok {
			a = &rpAgg{FirstSeen: c.CreatedDate, AICohort: c.AICohort, LOC: c.Additions + c.Deletions}
			rp[k] = a
		}
		if c.CreatedDate.Before(a.FirstSeen) {
			a.FirstSeen = c.CreatedDate
		}
		a.Comments++
	}
	for k, a := range rp {
		bucket := getOrCreate(k.Reviewer, PeriodWeekStart(a.FirstSeen))
		bucket.ReviewerMinutes += EstimateReviewerMinutes(a.Comments)
		bucket.ReviewedLOC += a.LOC
		bucket.ReviewCommentsTotal += a.Comments
		if a.AICohort == "HIGH" {
			bucket.ReviewCommentsHighCohort += a.Comments
			bucket.ReviewedLOCHigh += a.LOC
		}
	}

	// 3. Write rows.
	count := 0
	for bk, agg := range buckets {
		row := &models.EngineerVerificationEffort{
			EngineerId:               bk.EngineerId,
			PeriodWeek:               bk.WeekStart,
			AuthorMinutes:            agg.AuthorMinutes,
			ReviewerMinutes:          agg.ReviewerMinutes,
			ReviewToAuthorRatio:      SafeRatio(agg.ReviewerMinutes, agg.AuthorMinutes),
			ReviewCommentsTotal:      agg.ReviewCommentsTotal,
			ReviewCommentsPerLoc:     SafeRatio(agg.ReviewCommentsTotal, agg.ReviewedLOC),
			ReviewCommentsHighCohort: agg.ReviewCommentsHighCohort,
			ReviewCommentsPerLocHigh: SafeRatio(agg.ReviewCommentsHighCohort, agg.ReviewedLOCHigh),
			ComputedAt:               now,
		}
		if err := db.CreateOrUpdate(row); err != nil {
			return errors.Default.Wrap(err, "failed to upsert engineer_verification_effort row")
		}
		count++
	}
	logger.Info("computeVerificationEffort wrote %d (engineer, week) buckets", count)
	return nil
}
```

- [ ] **Step 4: Run unit tests**

```bash
docker run --rm -v "/home/ubuntu/workspaces/finally/incubator-devlake:/workspace" -w /workspace/backend mericodev/lake-builder:latest sh -c '
go test -run "^TestEstimate|^TestSafeRatio$" ./plugins/aimeasure/tasks/... -v 2>&1 | tail -10
'
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/plugins/aimeasure/tasks/compute_verification_effort.go backend/plugins/aimeasure/tasks/compute_verification_effort_test.go
git commit -m "feat(aimeasure): computeVerificationEffort subtask"
```

---

## Task 6 — `computeSlackSignals` subtask

**Files:**
- Create: `backend/plugins/aimeasure/tasks/compute_slack_signals.go`
- Create: `backend/plugins/aimeasure/tasks/compute_slack_signals_test.go`

Reads from `_tool_slack_channel_messages` joined to `_tool_slack_channels` (for channel name) plus the new `aimeasure_slack_channel_categories` mapping table. Aggregates per (engineer, week, category).

**Engineer ID resolution:** Slack stores `user` as the Slack user ID. We resolve to DevLake `account_id` via the existing identity-resolution chain (`accounts` domain table → `aimeasure_account_overrides` from Phase A). Unmapped Slack users get a synthetic `engineer_id = "slack:<user>"` so the dashboard can show "unmapped" rows rather than silently dropping data.

**Thread participation:** a Slack message is counted as "thread participation" when `thread_ts != "" AND thread_ts != ts` (i.e., it's a reply, not a top-level message).

- [ ] **Step 1: Write failing tests for the resolver and the per-message classification**

Create `backend/plugins/aimeasure/tasks/compute_slack_signals_test.go`:

```go
/* Apache 2.0 header */

package tasks

import "testing"

func TestResolveSlackEngineer(t *testing.T) {
	overrides := map[string]string{
		"slack:U001": "account-alice",
	}
	if got := ResolveSlackEngineer(overrides, "U001"); got != "account-alice" {
		t.Errorf("expected mapped account, got %q", got)
	}
	if got := ResolveSlackEngineer(overrides, "U999"); got != "slack:U999" {
		t.Errorf("expected synthetic id, got %q", got)
	}
	if got := ResolveSlackEngineer(overrides, ""); got != "" {
		t.Errorf("expected empty for empty user, got %q", got)
	}
}

func TestIsThreadParticipation(t *testing.T) {
	cases := []struct {
		name           string
		ts, threadTs   string
		wantParticipation bool
	}{
		{"top-level message", "1700000000.000100", "", false},
		{"thread root", "1700000000.000100", "1700000000.000100", false},
		{"thread reply", "1700000001.000200", "1700000000.000100", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsThreadParticipation(c.ts, c.threadTs); got != c.wantParticipation {
				t.Errorf("IsThreadParticipation(%q,%q) = %v, want %v", c.ts, c.threadTs, got, c.wantParticipation)
			}
		})
	}
}

func TestSlackTsToTime(t *testing.T) {
	// Slack ts format: "<unix seconds>.<microseconds>". Parsing must round-trip.
	got, err := SlackTsToTime("1700000000.000123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Unix() != 1700000000 {
		t.Errorf("expected unix=1700000000, got %d", got.Unix())
	}
}
```

- [ ] **Step 2: Run, verify it fails**

- [ ] **Step 3: Implement the subtask**

Create `backend/plugins/aimeasure/tasks/compute_slack_signals.go`:

```go
/* Apache 2.0 header */

package tasks

import (
	"strconv"
	"strings"
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/aimeasure/models"
)

var ComputeSlackSignalsMeta = plugin.SubTaskMeta{
	Name:             "computeSlackSignals",
	EntryPoint:       ComputeSlackSignals,
	EnabledByDefault: true,
	Description:      "Aggregate per-engineer per-week Slack participation by channel category",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET}, // Slack is communication-layer; ticket is the closest existing domain
}

// ResolveSlackEngineer returns the DevLake account_id for a Slack user, falling
// back to a synthetic "slack:<user>" id when unmapped. Empty user → empty id.
func ResolveSlackEngineer(overrides map[string]string, slackUserId string) string {
	if slackUserId == "" {
		return ""
	}
	if a, ok := overrides["slack:"+slackUserId]; ok {
		return a
	}
	return "slack:" + slackUserId
}

// IsThreadParticipation returns true when a message is a reply within a thread
// (not the top-level message and not the thread root).
func IsThreadParticipation(ts, threadTs string) bool {
	return threadTs != "" && threadTs != ts
}

// SlackTsToTime parses Slack's "<unix>.<usec>" timestamp format to a UTC time.
func SlackTsToTime(ts string) (time.Time, error) {
	dot := strings.Index(ts, ".")
	secs := ts
	if dot > 0 {
		secs = ts[:dot]
	}
	n, err := strconv.ParseInt(secs, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(n, 0).UTC(), nil
}

func ComputeSlackSignals(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	logger := taskCtx.GetLogger()
	now := time.Now().UTC()

	// Load channel→category mapping (one query).
	type chanCat struct {
		ChannelKey string `gorm:"column:channel_key"`
		Category   string `gorm:"column:category"`
	}
	var rawMapping []chanCat
	if err := db.All(&rawMapping, dal.From("aimeasure_slack_channel_categories")); err != nil {
		return errors.Default.Wrap(err, "failed to load slack channel categories")
	}
	mapping := map[string]models.ChannelCategory{}
	for _, m := range rawMapping {
		mapping[m.ChannelKey] = models.ChannelCategory(m.Category)
	}

	// Load identity overrides for Slack (one query).
	type override struct {
		SourceId  string `gorm:"column:source_id"`
		AccountId string `gorm:"column:account_id"`
	}
	var rawOverrides []override
	if err := db.All(&rawOverrides,
		dal.From("aimeasure_account_overrides"),
		dal.Where("source_system = ?", "slack"),
	); err != nil {
		return errors.Default.Wrap(err, "failed to load identity overrides")
	}
	overrides := map[string]string{}
	for _, o := range rawOverrides {
		overrides["slack:"+o.SourceId] = o.AccountId
	}

	// Stream the messages joined to channel names. Use a cursor to bound memory
	// on large workspaces.
	type slackMsg struct {
		User        string `gorm:"column:user"`
		Ts          string `gorm:"column:ts"`
		ThreadTs    string `gorm:"column:thread_ts"`
		ChannelId   string `gorm:"column:channel_id"`
		ChannelName string `gorm:"column:channel_name"`
	}
	cursor, err := db.Cursor(
		dal.Select("m.user AS user, m.ts AS ts, m.thread_ts AS thread_ts, m.channel_id AS channel_id, c.name AS channel_name"),
		dal.From("_tool_slack_channel_messages m"),
		dal.Join("LEFT JOIN _tool_slack_channels c ON c.channel_id = m.channel_id AND c.connection_id = m.connection_id"),
		dal.Where("m.user IS NOT NULL AND m.user != '' AND (m.subtype IS NULL OR m.subtype = '' OR m.subtype = 'thread_broadcast')"),
	)
	if err != nil {
		return errors.Default.Wrap(err, "failed to query slack messages")
	}
	defer cursor.Close()

	type slackKey struct {
		Engineer string
		Week     time.Time
		Category models.ChannelCategory
	}
	type slackAgg struct {
		Messages   int
		Threaded   int
		AfterHours int
	}
	buckets := map[slackKey]*slackAgg{}

	for cursor.Next() {
		var m slackMsg
		if err := db.Fetch(cursor, &m); err != nil {
			return errors.Default.Wrap(err, "row scan failed on slack message")
		}
		engineer := ResolveSlackEngineer(overrides, m.User)
		if engineer == "" {
			continue
		}
		when, terr := SlackTsToTime(m.Ts)
		if terr != nil {
			logger.Warn(terr, "failed to parse slack ts %q; skipping", m.Ts)
			continue
		}
		cat := CategorizeChannel(mapping, m.ChannelName, m.ChannelId)
		key := slackKey{Engineer: engineer, Week: PeriodWeekStart(when), Category: cat}
		a, ok := buckets[key]
		if !ok {
			a = &slackAgg{}
			buckets[key] = a
		}
		a.Messages++
		if IsThreadParticipation(m.Ts, m.ThreadTs) {
			a.Threaded++
		}
		if IsAfterHours(when) {
			a.AfterHours++
		}
	}

	count := 0
	for k, agg := range buckets {
		row := &models.EngineerSlackSignals{
			EngineerId:               k.Engineer,
			PeriodWeek:               k.Week,
			ChannelCategory:          k.Category,
			MessageCount:             agg.Messages,
			ThreadParticipationCount: agg.Threaded,
			AfterHoursMessageCount:   agg.AfterHours,
			AfterHoursRatio:          SafeRatio(agg.AfterHours, agg.Messages),
			ComputedAt:               now,
		}
		if err := db.CreateOrUpdate(row); err != nil {
			return errors.Default.Wrap(err, "failed to upsert engineer_slack_signals row")
		}
		count++
	}
	logger.Info("computeSlackSignals wrote %d (engineer, week, category) buckets", count)
	return nil
}
```

- [ ] **Step 4: Run unit tests, verify they pass**

```bash
docker run --rm -v "/home/ubuntu/workspaces/finally/incubator-devlake:/workspace" -w /workspace/backend mericodev/lake-builder:latest sh -c '
go test -run "^TestResolveSlackEngineer$|^TestIsThreadParticipation$|^TestSlackTsToTime$" ./plugins/aimeasure/tasks/... -v 2>&1 | tail -10
'
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/plugins/aimeasure/tasks/compute_slack_signals.go backend/plugins/aimeasure/tasks/compute_slack_signals_test.go
git commit -m "feat(aimeasure): computeSlackSignals subtask (per-engineer per-week per-category)"
```

---

## Task 7 — `computeSentimentProxy` subtask

**Files:**
- Create: `backend/plugins/aimeasure/tasks/compute_sentiment_proxy.go`
- Create: `backend/plugins/aimeasure/tasks/compute_sentiment_proxy_test.go`

Behavioral-only sentiment proxy for Phase B. **No message-content analysis** (privacy + LLM dependency are out of scope). The proxy reads from `engineer_slack_signals` (already aggregated by Task 6) and `engineer_verification_effort` (Task 5), so it must run after both.

**Score model (0–100, higher = healthier):**

```
score = 100
  - 40 * after_hours_ratio                            # heaviest penalty
  - 30 * clamp((review_to_author_ratio - 1.5) / 1.5)  # senior burnout signal
  - 10 if engineer_slack_signals dropped >30% WoW     # disengagement signal
```

Each penalty is clipped to its weight. `bad_developer_day_flag` is `True` when score < 50 OR (after_hours_ratio > 0.15 AND total messages dropped >50% WoW).

These are heuristics, marked clearly in the README as a Phase B proxy. The DXI survey columns stay nullable for an external ingest to populate.

- [ ] **Step 1: Failing tests for the score model**

Create `backend/plugins/aimeasure/tasks/compute_sentiment_proxy_test.go`:

```go
/* Apache 2.0 header */

package tasks

import "testing"

func TestSentimentScore(t *testing.T) {
	cases := []struct {
		name              string
		afterHoursRatio   float64
		reviewToAuthor    float64
		messageDropPct    float64 // 0..1 fraction of WoW drop
		want              float64 // approx
	}{
		{"perfect week", 0.0, 0.8, 0.0, 100},
		{"all night work", 1.0, 0.8, 0.0, 60},          // -40
		{"heavy reviewer", 0.0, 3.0, 0.0, 70},           // (3-1.5)/1.5=1.0 → -30
		{"disengaged", 0.0, 0.8, 0.5, 90},               // -10
		{"compound burnout", 1.0, 3.0, 0.6, 20},         // -40 -30 -10
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := SentimentScore(c.afterHoursRatio, c.reviewToAuthor, c.messageDropPct)
			if got < c.want-0.5 || got > c.want+0.5 {
				t.Errorf("got %v, want approx %v", got, c.want)
			}
		})
	}
}

func TestBadDeveloperDayFlag(t *testing.T) {
	cases := []struct {
		name            string
		score           float64
		afterHoursRatio float64
		messageDropPct  float64
		want            bool
	}{
		{"healthy week", 90, 0.0, 0.0, false},
		{"score below 50", 40, 0.0, 0.0, true},
		{"after-hours + disengaged", 70, 0.20, 0.60, true},
		{"after-hours alone", 70, 0.20, 0.10, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := BadDeveloperDayFlag(c.score, c.afterHoursRatio, c.messageDropPct); got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run, verify it fails**

- [ ] **Step 3: Implement the subtask**

Create `backend/plugins/aimeasure/tasks/compute_sentiment_proxy.go`:

```go
/* Apache 2.0 header */

package tasks

import (
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/aimeasure/models"
)

var ComputeSentimentProxyMeta = plugin.SubTaskMeta{
	Name:             "computeSentimentProxy",
	EntryPoint:       ComputeSentimentProxy,
	EnabledByDefault: true,
	Description:      "Derive behavioral sentiment proxy from per-engineer Slack and verification signals",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

// SentimentScore returns a 0-100 score (higher = healthier) from the three
// behavioral inputs. Each penalty is bounded to its weight; output is clamped
// to [0, 100].
func SentimentScore(afterHoursRatio, reviewToAuthorRatio, messageDropPct float64) float64 {
	score := 100.0
	score -= 40.0 * clamp01(afterHoursRatio)
	score -= 30.0 * clamp01((reviewToAuthorRatio-1.5)/1.5)
	if messageDropPct > 0.30 {
		score -= 10.0
	}
	if score < 0 {
		score = 0
	}
	return score
}

// BadDeveloperDayFlag returns true when score < 50 OR (afterHoursRatio > 0.15 AND messageDropPct > 0.5).
func BadDeveloperDayFlag(score, afterHoursRatio, messageDropPct float64) bool {
	if score < 50 {
		return true
	}
	if afterHoursRatio > 0.15 && messageDropPct > 0.5 {
		return true
	}
	return false
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

// sentimentInput is the per-engineer per-week aggregate joining the two upstream tables.
type sentimentInput struct {
	EngineerId       string    `gorm:"column:engineer_id"`
	PeriodWeek       time.Time `gorm:"column:period_week"`
	AfterHoursRatio  float64   `gorm:"column:after_hours_ratio"`  // taken from the GENERAL category bucket as a baseline
	ReviewToAuthor   float64   `gorm:"column:review_to_author_ratio"`
	MessageCount     int       `gorm:"column:message_count"`
}

func ComputeSentimentProxy(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	logger := taskCtx.GetLogger()
	now := time.Now().UTC()

	// Pull per-engineer per-week aggregates. Join effort + slack (general category)
	// so we have all three inputs in one cursor.
	var rows []sentimentInput
	if err := db.All(&rows,
		dal.Select("e.engineer_id AS engineer_id, e.period_week AS period_week, COALESCE(s.after_hours_ratio, 0) AS after_hours_ratio, e.review_to_author_ratio AS review_to_author_ratio, COALESCE(s.message_count, 0) AS message_count"),
		dal.From("engineer_verification_effort e"),
		dal.Join("LEFT JOIN engineer_slack_signals s ON s.engineer_id = e.engineer_id AND s.period_week = e.period_week AND s.channel_category = ?", "general"),
	); err != nil {
		return errors.Default.Wrap(err, "failed to join verification+slack")
	}

	// Group by engineer so we can compute WoW message drop.
	type engWeek struct {
		Week  time.Time
		Input sentimentInput
	}
	byEng := map[string][]engWeek{}
	for _, r := range rows {
		byEng[r.EngineerId] = append(byEng[r.EngineerId], engWeek{Week: r.PeriodWeek, Input: r})
	}

	count := 0
	for eng, weeks := range byEng {
		// Sort ascending by week to compute WoW.
		sortWeeksAsc(weeks)
		var prev *engWeek
		for i := range weeks {
			w := weeks[i]
			drop := 0.0
			if prev != nil {
				drop = wowDrop(prev.Input.MessageCount, w.Input.MessageCount)
			}
			score := SentimentScore(w.Input.AfterHoursRatio, w.Input.ReviewToAuthor, drop)
			flag := BadDeveloperDayFlag(score, w.Input.AfterHoursRatio, drop)
			row := &models.EngineerDxiProxy{
				EngineerId:          eng,
				PeriodWeek:          w.Week,
				SentimentScore:      score,
				BadDeveloperDayFlag: flag,
				ComputedAt:          now,
			}
			if err := db.CreateOrUpdate(row); err != nil {
				return errors.Default.Wrap(err, "failed to upsert engineer_dxi_proxy row")
			}
			count++
			prev = &weeks[i]
		}
	}
	logger.Info("computeSentimentProxy wrote %d (engineer, week) rows", count)
	return nil
}

func sortWeeksAsc(weeks []engWeek) {
	// simple insertion sort; the slice is small per-engineer
	for i := 1; i < len(weeks); i++ {
		for j := i; j > 0 && weeks[j-1].Week.After(weeks[j].Week); j-- {
			weeks[j-1], weeks[j] = weeks[j], weeks[j-1]
		}
	}
}

func wowDrop(prev, curr int) float64 {
	if prev <= 0 {
		return 0
	}
	if curr >= prev {
		return 0
	}
	return float64(prev-curr) / float64(prev)
}
```

- [ ] **Step 4: Run unit tests**

```bash
docker run --rm -v "/home/ubuntu/workspaces/finally/incubator-devlake:/workspace" -w /workspace/backend mericodev/lake-builder:latest sh -c '
go test -run "^TestSentimentScore$|^TestBadDeveloperDayFlag$" ./plugins/aimeasure/tasks/... -v 2>&1 | tail -15
'
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/plugins/aimeasure/tasks/compute_sentiment_proxy.go backend/plugins/aimeasure/tasks/compute_sentiment_proxy_test.go
git commit -m "feat(aimeasure): computeSentimentProxy behavioral subtask"
```

---

## Task 8 — Wire Phase B subtasks into `impl.go`

**Files:**
- Modify: `backend/plugins/aimeasure/impl/impl.go`

- [ ] **Step 1: Extend `SubTaskMetas`**

```go
func (p AIMeasure) SubTaskMetas() []plugin.SubTaskMeta {
	return []plugin.SubTaskMeta{
		// Phase A
		tasks.ClassifyPRCohortMeta,
		tasks.ComputeChangeCompositionMeta,
		tasks.ComputeQualityCohortMeta,
		// Phase B — depend on Phase A's outputs being present
		tasks.ComputeVerificationEffortMeta, // reads pr_ai_cohort, writes engineer_verification_effort
		tasks.ComputeSlackSignalsMeta,       // reads slack tool tables, writes engineer_slack_signals
		tasks.ComputeSentimentProxyMeta,     // reads both above, writes engineer_dxi_proxy (run last)
	}
}
```

- [ ] **Step 2: Extend `RequiredDataEntities`**

```go
func (p AIMeasure) RequiredDataEntities() (data []map[string]interface{}, err errors.Error) {
	return []map[string]interface{}{
		{"model": "pull_requests"},
		{"model": "commits"},
		{"model": "commit_files"},
		{"model": "pull_request_commits"},
		{"model": "pull_request_comments"},
		{"model": "pull_request_reviewers"},
	}, nil
}
```

Note: the `_tool_slack_*` tables are not in the domain-layer model list — RequiredDataEntities is for domain-layer dependencies only. Slack tables are checked at runtime by the subtask itself; if absent, the subtask logs a warning and writes 0 rows. Add a docstring on `ComputeSlackSignalsMeta.Description` noting "requires slack plugin enabled".

- [ ] **Step 3: Extend `MakeMetricPluginPipelinePlanV200` subtasks list**

```go
Subtasks: []string{
    "classifyPRCohort",
    "computeChangeComposition",
    "computeQualityCohort",
    "computeVerificationEffort",
    "computeSlackSignals",
    "computeSentimentProxy",
},
```

- [ ] **Step 4: Verify build + unit tests**

```bash
docker run --rm -v "/home/ubuntu/workspaces/finally/incubator-devlake:/workspace" -w /workspace/backend mericodev/lake-builder:latest sh -c '
go build ./plugins/aimeasure/... && go test ./plugins/aimeasure/... 2>&1 | tail -10
'
```

- [ ] **Step 5: Commit**

```bash
git add backend/plugins/aimeasure/impl/impl.go
git commit -m "feat(aimeasure): wire Phase B subtasks into PluginMeta + blueprint plan"
```

---

## Task 9 — E2E test for `computeVerificationEffort`

**Files:**
- Create: `backend/plugins/aimeasure/e2e/compute_verification_effort_test.go`
- Create: `backend/plugins/aimeasure/e2e/fixtures/pull_request_comments.csv`
- Create: `backend/plugins/aimeasure/e2e/fixtures/pr_ai_cohort.csv` (Phase A output, used as input)
- Create: `backend/plugins/aimeasure/e2e/fixtures/expected_engineer_verification_effort.csv`

- [ ] **Step 1: Create the input fixtures**

`pull_request_comments.csv` (one author per PR; reviewers different from authors):

```csv
id,pull_request_id,body,account_id,created_date,commit_sha,type,review_id,status
prc1,pr1,Looks good,bob,2026-05-04 10:00:00,,REVIEW,r1,APPROVED
prc2,pr2,Why this change?,carol,2026-05-05 11:00:00,,DIFF,r2,COMMENTED
prc3,pr2,Refactor suggested,carol,2026-05-05 11:05:00,,DIFF,r2,COMMENTED
prc4,pr2,nit: rename,carol,2026-05-05 11:10:00,,DIFF,r2,COMMENTED
prc5,pr3,LGTM,bob,2026-05-06 09:00:00,,REVIEW,r3,APPROVED
```

`pr_ai_cohort.csv` (reuse Phase A's known PR set):

```csv
pr_id,ai_cohort,confidence_score,has_explicit_marker,has_commit_trailer,classifier_version,classified_at
pr1,NONE,15,0,0,v1,2026-05-13 00:00:00
pr2,HIGH,90,1,1,v1,2026-05-13 00:00:00
pr3,LOW,45,0,0,v1,2026-05-13 00:00:00
pr4,NONE,5,0,0,v1,2026-05-13 00:00:00
pr5,NONE,0,0,0,v1,2026-05-13 00:00:00
```

The `pull_requests.csv` fixture from Phase A (5 PRs, authors `alice/alice/dave/eve/frank` — extend the existing Phase A fixture to include an `author_id` column populated for each PR).

**IMPORTANT:** The Phase A `pull_requests.csv` does not have an `author_id` column today (it was not needed for Phase A's queries). Either extend the existing fixture or write a Phase-B-specific copy. **Pick: extend the shared fixture**, since later Phase B tests also need authors. Update the column header to include `author_id` and assign:
- pr1 → alice
- pr2 → alice
- pr3 → dave
- pr4 → eve
- pr5 → frank

Verify the Phase A e2e tests still pass after this change (re-run them — the extra column should not affect their assertions because they ignore `author_id` via `IgnoreFields` when verifying).

- [ ] **Step 2: Compute the expected output**

Per-engineer per-week (all PRs merged in the week starting `2026-04-27` Monday → `2026-04-27`):

- alice: authored pr1 (100+10=110 LOC, 22 min) + pr2 (200+50=250 LOC, 50 min) = author_minutes=72, reviewer_minutes=0
- dave: authored pr3 (5+2=7 LOC, 10 min) + reviewed 0 PRs as the comment fixture above shows dave didn't comment = author_minutes=10
- eve: authored pr4 (50+100=150 LOC, 30 min) = author_minutes=30
- frank: authored pr5 (30+10=40 LOC, 10 min) = author_minutes=10
- bob: reviewed pr1 (1 comment → 17 min) + pr3 (1 comment → 17 min) = reviewer_minutes=34, total_comments=2, reviewed_loc=110+7=117, comments_per_loc=0.0171
- carol: reviewed pr2 (3 comments → 21 min) = reviewer_minutes=21, total_comments=3, reviewed_loc=250 (all HIGH cohort), comments_high=3, comments_per_loc_high=0.012

Create `expected_engineer_verification_effort.csv`:

```csv
engineer_id,period_week,author_minutes,reviewer_minutes,review_to_author_ratio,review_comments_total,review_comments_per_loc,review_comments_high_cohort,review_comments_per_loc_high
alice,2026-04-27,72,0,0.0000,0,0.0000,0,0.0000
dave,2026-04-27,10,0,0.0000,0,0.0000,0,0.0000
eve,2026-04-27,30,0,0.0000,0,0.0000,0,0.0000
frank,2026-04-27,10,0,0.0000,0,0.0000,0,0.0000
bob,2026-04-27,0,34,inf-or-large,2,0.0171,0,0.0000
carol,2026-04-27,0,21,inf-or-large,3,0.0120,3,0.0120
```

**Adjustment needed before commit:** the spec calls `review_to_author_ratio` an unbounded float. For pure-reviewer rows (author_minutes=0) the ratio is theoretically infinite. The plan's `SafeRatio` returns 0 in that case — that's a documented choice. Update the expected CSV to `0.0000` for bob and carol's ratio. Note the alternative is to store NULL, which loses the "huge ratio = reviewer" signal; document the choice in the README.

Final expected CSV:

```csv
engineer_id,period_week,author_minutes,reviewer_minutes,review_to_author_ratio,review_comments_total,review_comments_per_loc,review_comments_high_cohort,review_comments_per_loc_high
alice,2026-04-27,72,0,0.0000,0,0.0000,0,0.0000
dave,2026-04-27,10,0,0.0000,0,0.0000,0,0.0000
eve,2026-04-27,30,0,0.0000,0,0.0000,0,0.0000
frank,2026-04-27,10,0,0.0000,0,0.0000,0,0.0000
bob,2026-04-27,0,34,0.0000,2,0.0171,0,0.0000
carol,2026-04-27,0,21,0.0000,3,0.0120,3,0.0120
```

- [ ] **Step 3: Write the e2e test**

```go
/* Apache 2.0 header */

package e2e

import (
	"testing"

	"github.com/apache/incubator-devlake/core/models/common"
	"github.com/apache/incubator-devlake/core/models/domainlayer/code"
	"github.com/apache/incubator-devlake/helpers/e2ehelper"
	"github.com/apache/incubator-devlake/plugins/aimeasure/impl"
	"github.com/apache/incubator-devlake/plugins/aimeasure/models"
	"github.com/apache/incubator-devlake/plugins/aimeasure/tasks"
)

func TestComputeVerificationEffortDataFlow(t *testing.T) {
	var plugin impl.AIMeasure
	dataflowTester := e2ehelper.NewDataFlowTester(t, "aimeasure", plugin)

	taskData := &tasks.AIMeasureTaskData{
		Options: &tasks.AIMeasureOptions{
			ProjectName:         "demo",
			HighCohortThreshold: 65,
			LowCohortThreshold:  30,
			DefectWindowDays:    14,
		},
	}

	dataflowTester.ImportCsvIntoTabler("./fixtures/pull_requests.csv", &code.PullRequest{})
	dataflowTester.ImportCsvIntoTabler("./fixtures/pull_request_comments.csv", &code.PullRequestComment{})
	dataflowTester.ImportCsvIntoTabler("./fixtures/pr_ai_cohort.csv", &models.PRAICohort{})

	dataflowTester.FlushTabler(&models.EngineerVerificationEffort{})
	dataflowTester.Subtask(tasks.ComputeVerificationEffortMeta, taskData)
	dataflowTester.VerifyTableWithOptions(&models.EngineerVerificationEffort{}, e2ehelper.TableOptions{
		CSVRelPath:  "./fixtures/expected_engineer_verification_effort.csv",
		IgnoreTypes: []interface{}{common.NoPKModel{}},
		IgnoreFields: []string{"computed_at"},
		NumericEpsilon: map[string]float64{
			"review_to_author_ratio":         0.0001,
			"review_comments_per_loc":        0.0001,
			"review_comments_per_loc_high":   0.0001,
		},
	})
}
```

- [ ] **Step 4: Run against Postgres + MySQL** (same pattern as Phase A Tasks 10–12 Steps 3–4)

Expected: `--- PASS: TestComputeVerificationEffortDataFlow` on both drivers.

- [ ] **Step 5: Commit**

```bash
git add backend/plugins/aimeasure/e2e/compute_verification_effort_test.go backend/plugins/aimeasure/e2e/fixtures/
git commit -m "test(aimeasure): e2e for computeVerificationEffort across MySQL+Postgres"
```

---

## Task 10 — E2E test for `computeSlackSignals`

**Files:**
- Create: `backend/plugins/aimeasure/e2e/compute_slack_signals_test.go`
- Create: `backend/plugins/aimeasure/e2e/fixtures/slack_channels.csv`
- Create: `backend/plugins/aimeasure/e2e/fixtures/slack_channel_messages.csv`
- Create: `backend/plugins/aimeasure/e2e/fixtures/slack_channel_categories.csv`
- Create: `backend/plugins/aimeasure/e2e/fixtures/account_overrides.csv`
- Create: `backend/plugins/aimeasure/e2e/fixtures/expected_engineer_slack_signals.csv`

- [ ] **Step 1: Create the input fixtures**

`slack_channels.csv` (one connection_id, three channels):

```csv
connection_id,channel_id,name,team
1,C001,eng-platform,T1
1,C002,inc-alerts,T1
1,C003,design-review,T1
```

`slack_channel_categories.csv` (the new helper table):

```csv
channel_key,category,note
eng-platform,engineering,Maps name to engineering bucket
inc-alerts,incident_support,Production incident channel
design-review,design_architecture,RFC discussions
```

`account_overrides.csv` (extend Phase A's overrides; map alice and bob's Slack ids):

```csv
source_system,source_id,account_id,note
slack,U_ALICE,alice,
slack,U_BOB,bob,
```

`slack_channel_messages.csv` (timestamp in Unix seconds; chosen for known weeks):

- 2026-05-11 10:00 UTC (Mon, business hours) → ts="1747044000"
- 2026-05-11 20:00 UTC (Mon, after hours)    → ts="1747080000"
- 2026-05-16 12:00 UTC (Sat, weekend)        → ts="1747483200"

```csv
connection_id,channel_id,ts,thread_ts,user,subtype,type,text,team,reply_count,reply_users_count,latest_reply,is_locked,subscribed,parent_user_id,client_msg_id
1,C001,1747044000.000100,,U_ALICE,,message,deploy ok,T1,0,0,,false,false,,m1
1,C002,1747080000.000200,,U_ALICE,,message,paging,T1,0,0,,false,false,,m2
1,C003,1747483200.000300,1747400000.000999,U_BOB,,message,thread reply,T1,0,0,,false,false,U_ALICE,m3
1,C001,1747044000.000400,,U_UNMAPPED,,message,from someone unmapped,T1,0,0,,false,false,,m4
```

- [ ] **Step 2: Compute expected output**

- alice: 2 top-level messages in week 2026-05-11. C001 (engineering, business hours) + C002 (incident_support, after-hours) → two rows:
  - alice, 2026-05-11, engineering, 1 msg, 0 threaded, 0 after-hours, 0.0000
  - alice, 2026-05-11, incident_support, 1 msg, 0 threaded, 1 after-hours, 1.0000
- bob: 1 thread reply in week 2026-05-11. C003 (design_architecture, Saturday so after-hours) → one row:
  - bob, 2026-05-11, design_architecture, 1 msg, 1 threaded, 1 after-hours, 1.0000
- slack:U_UNMAPPED: 1 top-level in C001 → engineering, business hours:
  - slack:U_UNMAPPED, 2026-05-11, engineering, 1 msg, 0 threaded, 0 after-hours, 0.0000

Total 4 rows.

`expected_engineer_slack_signals.csv`:

```csv
engineer_id,period_week,channel_category,message_count,thread_participation_count,after_hours_message_count,after_hours_ratio
alice,2026-05-11,engineering,1,0,0,0.0000
alice,2026-05-11,incident_support,1,0,1,1.0000
bob,2026-05-11,design_architecture,1,1,1,1.0000
slack:U_UNMAPPED,2026-05-11,engineering,1,0,0,0.0000
```

- [ ] **Step 3: Write the e2e test**

Mirror Task 9's pattern. Import the 4 input fixtures, flush `EngineerSlackSignals`, run `ComputeSlackSignalsMeta`, verify with `NumericEpsilon` on `after_hours_ratio`.

- [ ] **Step 4: Run against both drivers**

- [ ] **Step 5: Commit**

```bash
git add backend/plugins/aimeasure/e2e/compute_slack_signals_test.go backend/plugins/aimeasure/e2e/fixtures/
git commit -m "test(aimeasure): e2e for computeSlackSignals across MySQL+Postgres"
```

---

## Task 11 — E2E test for `computeSentimentProxy`

**Files:**
- Create: `backend/plugins/aimeasure/e2e/compute_sentiment_proxy_test.go`
- Create: `backend/plugins/aimeasure/e2e/fixtures/expected_engineer_dxi_proxy.csv`

Sentiment proxy reads ONLY from the two upstream Phase B tables, which the previous two e2e tests already populate. To keep this test independent (e2e tests in DevLake run in alphabetical order within a package; sequential file dependencies are fragile), this test imports the upstream tables directly as CSV fixtures.

- [ ] **Step 1: Create input fixtures**

`expected_engineer_verification_effort_for_sentiment.csv` (input to this test):

```csv
engineer_id,period_week,author_minutes,reviewer_minutes,review_to_author_ratio,review_comments_total,review_comments_per_loc,review_comments_high_cohort,review_comments_per_loc_high
alice,2026-05-04,60,0,0.0000,0,0.0000,0,0.0000
alice,2026-05-11,60,0,0.0000,0,0.0000,0,0.0000
carol,2026-05-04,30,90,3.0000,5,0.0500,5,0.0500
carol,2026-05-11,30,90,3.0000,5,0.0500,5,0.0500
```

`expected_engineer_slack_signals_for_sentiment.csv` (input):

```csv
engineer_id,period_week,channel_category,message_count,thread_participation_count,after_hours_message_count,after_hours_ratio
alice,2026-05-04,general,40,0,0,0.0000
alice,2026-05-11,general,10,0,0,0.0000
carol,2026-05-04,general,50,0,5,0.1000
carol,2026-05-11,general,50,0,40,0.8000
```

- [ ] **Step 2: Compute expected output**

Score formula recap: `100 - 40*afterHours - 30*clamp((rta-1.5)/1.5,0,1) - 10*(if wow drop > 0.30)`.

- alice, 2026-05-04: afterHours=0, rta=0, drop=0 → 100. Flag: false.
- alice, 2026-05-11: afterHours=0, rta=0, drop=(40-10)/40=0.75 → 100 - 10 = 90. Flag: false (score≥50, no after-hours bump).
- carol, 2026-05-04: afterHours=0.10, rta=3.0, drop=0 → 100 - 4 - 30 = 66. Flag: false.
- carol, 2026-05-11: afterHours=0.80, rta=3.0, drop=0 (msg count stayed same) → 100 - 32 - 30 = 38. Flag: true (score < 50).

`expected_engineer_dxi_proxy.csv`:

```csv
engineer_id,period_week,sentiment_score,bad_developer_day_flag
alice,2026-05-04,100.00,0
alice,2026-05-11,90.00,0
carol,2026-05-04,66.00,0
carol,2026-05-11,38.00,1
```

- [ ] **Step 3: Write the e2e test** (mirror Tasks 9-10 pattern; `NumericEpsilon` 0.01 on `sentiment_score`)

- [ ] **Step 4: Run against both drivers**

- [ ] **Step 5: Commit**

```bash
git add backend/plugins/aimeasure/e2e/compute_sentiment_proxy_test.go backend/plugins/aimeasure/e2e/fixtures/
git commit -m "test(aimeasure): e2e for computeSentimentProxy across MySQL+Postgres"
```

---

## Task 12 — Grafana dashboard `InvisibleWork.json`

**Files:**
- Create: `grafana/dashboards/InvisibleWork.json`

Seven panels — six per spec § 6.3 plus one disambiguation panel (1b) added because the resolved decision on review-to-author ratio (`SafeRatio → 0` for pure reviewers) collapses the senior-burnout signal on the heatmap alone. All SQL written for MySQL (the default DevLake datasource; Postgres deployments can swap `DATE_ADD/WEEKDAY` for `DATE_TRUNC('week', …)` later).

- [ ] **Step 1: Build the dashboard**

Use `grafana/dashboards/AIQualityCohort.json` from Phase A as the structural template. Top-level metadata:

- `title`: "Invisible Work — Verification Effort, Slack Signals, Sentiment Proxy"
- `uid`: "aimeasure-invisible-work"
- `tags`: ["aimeasure", "phase-b"]
- `time`: `{from: "now-90d", to: "now"}`

Panel SQL:

**Panel 1 — Review-to-author ratio heatmap (one row per engineer per week):**
```sql
SELECT
  period_week AS time,
  engineer_id,
  review_to_author_ratio AS value
FROM engineer_verification_effort
WHERE period_week >= $__timeFrom() AND period_week < $__timeTo()
ORDER BY time
```
Type: heatmap, x=time, y=engineer_id, color=value. **Panel description must note:** "Pure reviewers (no authored PRs) display as 0 due to SafeRatio; use Panel 1b to identify them."

**Panel 1b — Reviewer minutes (raw), top 20 (table) — pure-reviewer disambiguation:**
```sql
SELECT
  engineer_id,
  SUM(reviewer_minutes) AS total_reviewer_minutes,
  SUM(author_minutes)   AS total_author_minutes,
  ROUND(SUM(reviewer_minutes) * 1.0 / NULLIF(SUM(author_minutes), 0), 2) AS combined_ratio
FROM engineer_verification_effort
WHERE period_week >= $__timeFrom() AND period_week < $__timeTo()
GROUP BY engineer_id
HAVING SUM(reviewer_minutes) > 0
ORDER BY total_reviewer_minutes DESC
LIMIT 20
```
Type: table; columns: `engineer_id | total_reviewer_minutes | total_author_minutes | combined_ratio`. **Why:** Panel 1 alone collapses the senior-burnout signal because `SafeRatio` returns 0 for pure reviewers. This raw view exposes engineers who do significant review work whether or not they author — exactly the senior-burnout shape the spec describes. The `combined_ratio` column uses `NULLIF` to keep pure reviewers as NULL in the ratio column while still showing them at the top of the table.

**Panel 2 — Top reviewers by load share (stacked area):**
```sql
SELECT
  period_week AS time,
  engineer_id,
  reviewer_minutes AS value
FROM engineer_verification_effort
WHERE period_week >= $__timeFrom() AND period_week < $__timeTo()
  AND reviewer_minutes > 0
ORDER BY time
```
Type: timeseries (stacked).

**Panel 3 — Review comments per LOC by AI cohort (line):**
```sql
SELECT
  period_week AS time,
  AVG(review_comments_per_loc) AS all_cohorts,
  AVG(review_comments_per_loc_high) AS high_cohort
FROM engineer_verification_effort
WHERE period_week >= $__timeFrom() AND period_week < $__timeTo()
GROUP BY period_week
ORDER BY time
```
Type: timeseries.

**Panel 4 — Slack participation by category, per engineer (stacked bar):**
```sql
SELECT
  engineer_id,
  channel_category,
  SUM(message_count) AS total
FROM engineer_slack_signals
WHERE period_week >= $__timeFrom() AND period_week < $__timeTo()
GROUP BY engineer_id, channel_category
ORDER BY engineer_id, channel_category
```
Type: barchart (stacked).

**Panel 5 — After-hours ratio per engineer (table; sort desc, flag >15%):**
```sql
SELECT
  engineer_id,
  SUM(after_hours_message_count) AS after_hours,
  SUM(message_count) AS total,
  ROUND(SUM(after_hours_message_count) * 100.0 / NULLIF(SUM(message_count), 0), 2) AS after_hours_pct
FROM engineer_slack_signals
WHERE period_week >= $__timeFrom() AND period_week < $__timeTo()
GROUP BY engineer_id
HAVING SUM(message_count) >= 20
ORDER BY after_hours_pct DESC
LIMIT 20
```
Type: table; field config: cell color = red when after_hours_pct > 15.

**Panel 6 — Sentiment score trend by team (line; team grouping is Phase C; for Phase B just show top-20 engineers by total message count):**
```sql
SELECT
  period_week AS time,
  engineer_id,
  sentiment_score AS value
FROM engineer_dxi_proxy
WHERE period_week >= $__timeFrom() AND period_week < $__timeTo()
ORDER BY time
```
Type: timeseries.

- [ ] **Step 2: Validate JSON parses**

```bash
docker run --rm -v "/home/ubuntu/workspaces/finally/incubator-devlake:/workspace" -w /workspace mericodev/lake-builder:latest sh -c '
python3 -c "import json; json.load(open(\"grafana/dashboards/InvisibleWork.json\"))" && echo "JSON valid"
'
```

- [ ] **Step 3: Commit**

```bash
git add grafana/dashboards/InvisibleWork.json
git commit -m "feat(aimeasure): add InvisibleWork Phase B Grafana dashboard"
```

---

## Task 13 — Update plugin README + resolved-decisions tracker

**Files:**
- Modify: `backend/plugins/aimeasure/README.md`

- [ ] **Step 1: Add a Phase B section**

After the Phase A "Tests" block, insert:

```markdown
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
```

- [ ] **Step 2: Commit**

```bash
git add backend/plugins/aimeasure/README.md
git commit -m "docs(aimeasure): Phase B README + resolved-decisions tracker"
```

---

## Final integration check

- [ ] **Step 1: Full unit test sweep**

```bash
docker run --rm -v "/home/ubuntu/workspaces/finally/incubator-devlake:/workspace" -w /workspace/backend mericodev/lake-builder:latest sh -c '
go test ./plugins/aimeasure/... -v 2>&1 | tail -60
'
```

Expected: all Phase A tests plus all Phase B tests pass.

- [ ] **Step 2: Lint sweep (best effort — host Go version may not match)**

```bash
docker run --rm -v "/home/ubuntu/workspaces/finally/incubator-devlake:/workspace" -w /workspace/backend mericodev/lake-builder:latest sh -c '
go vet ./plugins/aimeasure/... 2>&1
'
```

Expected: clean.

- [ ] **Step 3: E2E sweep against Postgres + MySQL**

Spin up both containers (per Phase A Tasks 10–12 Step 3-4 instructions), set `DB_URL` + `E2E_DB_URL` per driver, run:

```bash
go test -timeout 300s -v ./plugins/aimeasure/e2e/...
```

Expected: 6 tests pass on each driver (3 Phase A + 3 Phase B).

- [ ] **Step 4: Final code review**

Dispatch a code reviewer with the full diff (`git diff <Phase-A-merge-commit>..HEAD`).

- [ ] **Step 5: PR**

```bash
git push -u origin <feature-branch>
gh pr create --base main --title "feat(aimeasure): Phase B — verification effort + dark matter" --body-file <body.md>
```

PR body should cover: summary, 4 new tables, 3 new subtasks, 7-panel dashboard, resolved-decisions reference, test plan checklist.

---

## Spec-Coverage Table

| Phase B Requirement (spec § 6) | Implementing Task |
|---|---|
| `engineer_verification_effort` schema | Task 1 (model) + Task 2 (migration) |
| `engineer_slack_signals` schema | Task 1 + Task 2 |
| `engineer_dxi_proxy` schema | Task 1 + Task 2 |
| `computeVerificationEffort` subtask | Task 5 (impl) + Task 9 (e2e) |
| `computeSlackSignals` subtask | Task 6 (impl) + Task 10 (e2e) |
| `computeSentimentProxy` subtask | Task 7 (impl) + Task 11 (e2e) |
| `InvisibleWork.json` dashboard (6 panels per spec + 1 disambiguation panel) | Task 12 |
| Identity resolution (Slack → account) | Task 6 (`ResolveSlackEngineer`) + reuses Phase A's `aimeasure_account_overrides` |
| Channel-category mapping | Task 1 (`SlackChannelCategory` model) + Task 4 (`CategorizeChannel` pure fn) + Task 6 (DB join) |
| Subtask scheduling (after Phase A) | Task 8 (extends `SubTaskMetas` + `MakeMetricPluginPipelinePlanV200`) |
| Survey integration | Out of scope; columns are nullable, populated externally |
| Cross-signal red flags (senior burnout cascade, reviewer collapse, dark-matter ghost) | Surfaced via the dashboard panels — no separate alerting in Phase B |
| Resolved-decisions tracker (all 7 from spec § 6.6 + plan draft) | Task 13 (README section) |

---

## Out of scope (deferred to Phase B+ or Phase C)

- LLM-based tone analysis of message text
- Per-engineer timezone modeling (everyone gets UTC business-hours bucket)
- DXI / eNPS survey ingest pipeline
- Slack DM-channel data
- Cross-tenant Slack workspaces (multiple `connection_id`s)
- Real timer-based author/reviewer minutes (Phase B uses LOC and comment-count proxies)
- Admin UI for `aimeasure_slack_channel_categories` (edit directly via SQL for now)
- Alerting / paging when red-flag thresholds trip (Phase C; for now humans read the dashboard)

---

## Approximate effort

- Tasks 1–4 (schema + helpers): ~half day
- Tasks 5–7 (subtask implementations): ~1.5 days (the verification-effort math is the trickiest)
- Tasks 8 (wiring): ~1 hour
- Tasks 9–11 (e2e tests): ~1 day (fixtures + cross-driver iteration)
- Task 12 (dashboard): ~half day
- Task 13 (README): ~1 hour
- Integration + review + PR: ~half day

Total: roughly **4 working days** for one engineer, or **2 days** if Phase A subagent-driven-development parallelism patterns are reused for Tasks 5/6/7 and 9/10/11.
