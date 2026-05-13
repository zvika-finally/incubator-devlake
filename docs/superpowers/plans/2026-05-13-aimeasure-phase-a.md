# `aimeasure` Phase A — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the `aimeasure` plugin with Phase A scope — classify merged PRs by AI assistance level, compute defect signals (revert/hotfix within 14 days), track batch-size and refactor-ratio drift, surface everything in a Grafana dashboard.

**Architecture:** New Go plugin at `backend/plugins/aimeasure/` mirroring the structure of `aidetector` and `findevops`. Reads source-plugin tables read-only (`ai_usage_signals`, `pull_requests`, `commits`, `commit_files`); writes its own tables (`pr_ai_cohort`, `pr_defect_signals`, `pr_change_composition`, plus two identity-resolution helper tables). Three subtasks run in order: `classifyPRCohort` → `computeChangeComposition` → `computeQualityCohort`.

**Tech Stack:** Go 1.20, GORM v2, MySQL 8 / PostgreSQL 14 (dual-driver), e2e via `helpers/e2ehelper`, Grafana 9. Dependencies: existing `aidetector`, `claudecode`, `cursor` plugins must be enabled (read-only consumption of their tables).

**Spec reference:** `docs/superpowers/specs/2026-05-13-ai-era-signals-design.md` § 5 (Phase A).

---

## File Structure

**New files:**

```
backend/plugins/aimeasure/
├── aimeasure.go                              # standalone entry point
├── impl/
│   └── impl.go                                # PluginMeta + SubTaskMetas wiring
├── models/
│   ├── pr_ai_cohort.go                        # AI cohort table model
│   ├── pr_defect_signals.go                   # defect signals table model
│   ├── pr_change_composition.go               # change composition table model
│   ├── account_override.go                    # identity override table model
│   ├── engineer_role.go                       # seniority opt-in table model
│   └── migrationscripts/
│       ├── 20260513_init_schema.go            # creates the 5 tables
│       └── register.go                         # migration registry
├── tasks/
│   ├── task_data.go                            # AIMeasureOptions, AIMeasureTaskData
│   ├── classify_pr_cohort.go                  # subtask 1
│   ├── classify_pr_cohort_test.go             # cohort-classifier unit tests
│   ├── compute_change_composition.go          # subtask 2
│   ├── compute_change_composition_test.go     # batch-bucket unit tests
│   ├── compute_quality_cohort.go              # subtask 3
│   └── compute_quality_cohort_test.go         # hotfix-detection unit tests
└── e2e/
    ├── classify_pr_cohort_test.go             # e2e for subtask 1
    ├── compute_change_composition_test.go     # e2e for subtask 2
    ├── compute_quality_cohort_test.go         # e2e for subtask 3
    └── fixtures/
        ├── ai_usage_signals.csv
        ├── pull_requests.csv
        ├── commits.csv
        ├── commit_files.csv
        ├── expected_pr_ai_cohort.csv
        ├── expected_pr_defect_signals.csv
        └── expected_pr_change_composition.csv

grafana/dashboards/
└── AIQualityCohort.json                       # 5-panel dashboard

backend/plugins/
└── table_info_test.go                          # MODIFY: add aimeasure to plugin enumeration
```

**Modified files:**

- `backend/plugins/table_info_test.go` — add `aimeasure` import and `FeedIn` call

---

## Task 1 — Scaffold the plugin directory and entry point

**Files:**
- Create: `backend/plugins/aimeasure/aimeasure.go`
- Create: `backend/plugins/aimeasure/impl/impl.go` (skeleton — subtasks filled in Task 9)

- [ ] **Step 1: Create the plugin directory**

```bash
mkdir -p /home/ubuntu/workspaces/finally/incubator-devlake/backend/plugins/aimeasure/{impl,models/migrationscripts,tasks,e2e/fixtures}
```

- [ ] **Step 2: Create the entry point file**

Create `backend/plugins/aimeasure/aimeasure.go`:

```go
/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"github.com/apache/incubator-devlake/core/runner"
	"github.com/apache/incubator-devlake/plugins/aimeasure/impl"
	"github.com/spf13/cobra"
)

// PluginEntry exports for Framework to search and load
var PluginEntry impl.AIMeasure //nolint

// standalone mode for debugging
func main() {
	cmd := &cobra.Command{Use: "aimeasure"}

	projectName := cmd.Flags().StringP("projectName", "p", "", "project name")
	timeAfter := cmd.Flags().StringP("timeAfter", "a", "", "process PRs merged after this time (RFC3339)")

	cmd.Run = func(cmd *cobra.Command, args []string) {
		runner.DirectRun(cmd, args, PluginEntry, map[string]interface{}{
			"projectName": *projectName,
		}, *timeAfter)
	}
	runner.RunCmd(cmd)
}
```

- [ ] **Step 3: Create the impl skeleton**

Create `backend/plugins/aimeasure/impl/impl.go`:

```go
/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package impl

import (
	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/aimeasure/models"
	"github.com/apache/incubator-devlake/plugins/aimeasure/models/migrationscripts"
)

// make sure interfaces are implemented
var _ interface {
	plugin.PluginMeta
	plugin.PluginInit
	plugin.PluginTask
	plugin.PluginModel
	plugin.PluginMigration
} = (*AIMeasure)(nil)

type AIMeasure struct{}

func (p AIMeasure) Init(basicRes context.BasicRes) errors.Error {
	return nil
}

func (p AIMeasure) Description() string {
	return "Analytics layer: classifies PRs by AI cohort and computes quality/verification/cost signals from upstream plugin data"
}

func (p AIMeasure) Name() string {
	return "aimeasure"
}

func (p AIMeasure) RootPkgPath() string {
	return "github.com/apache/incubator-devlake/plugins/aimeasure"
}

func (p AIMeasure) MigrationScripts() []plugin.MigrationScript {
	return migrationscripts.All()
}

func (p AIMeasure) GetTablesInfo() []dal.Tabler {
	return []dal.Tabler{
		&models.PRAICohort{},
		&models.PRDefectSignals{},
		&models.PRChangeComposition{},
		&models.AccountOverride{},
		&models.EngineerRole{},
	}
}

func (p AIMeasure) SubTaskMetas() []plugin.SubTaskMeta {
	// filled in Task 9 — empty for now so the build succeeds
	return []plugin.SubTaskMeta{}
}

func (p AIMeasure) PrepareTaskData(taskCtx plugin.TaskContext, options map[string]interface{}) (interface{}, errors.Error) {
	// filled in Task 9 — return nil for now
	return nil, nil
}
```

- [ ] **Step 4: Verify the package compiles in the lake-builder container**

```bash
docker run --rm -v "/home/ubuntu/workspaces/finally/incubator-devlake:/workspace" -w /workspace/backend mericodev/lake-builder:latest sh -c '
go build ./plugins/aimeasure/...
'
```

Expected output: no compile errors (will fail until `models/` package exists — that's Task 2; this step verifies the structure compiles cleanly *after* Task 2; for now expect an error mentioning `package models` not found and stop here — proceed to Task 2.).

- [ ] **Step 5: Commit the scaffolding**

```bash
git add backend/plugins/aimeasure/
git commit -m "feat(aimeasure): scaffold new analytics plugin"
```

---

## Task 2 — Define data models

**Files:**
- Create: `backend/plugins/aimeasure/models/pr_ai_cohort.go`
- Create: `backend/plugins/aimeasure/models/pr_defect_signals.go`
- Create: `backend/plugins/aimeasure/models/pr_change_composition.go`
- Create: `backend/plugins/aimeasure/models/account_override.go`
- Create: `backend/plugins/aimeasure/models/engineer_role.go`

- [ ] **Step 1: Write a failing build test**

Create `backend/plugins/aimeasure/models/models_test.go`:

```go
/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package models

import "testing"

func TestTableNames(t *testing.T) {
	cases := map[string]string{
		"pr_ai_cohort":             PRAICohort{}.TableName(),
		"pr_defect_signals":        PRDefectSignals{}.TableName(),
		"pr_change_composition":    PRChangeComposition{}.TableName(),
		"aimeasure_account_overrides": AccountOverride{}.TableName(),
		"aimeasure_engineer_roles": EngineerRole{}.TableName(),
	}
	for expected, actual := range cases {
		if expected != actual {
			t.Errorf("expected table name %q, got %q", expected, actual)
		}
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
docker run --rm -v "/home/ubuntu/workspaces/finally/incubator-devlake:/workspace" -w /workspace/backend mericodev/lake-builder:latest sh -c '
go test ./plugins/aimeasure/models/... 2>&1 | tail -5
'
```

Expected: compile error — `undefined: PRAICohort` etc.

- [ ] **Step 3: Implement `PRAICohort` model**

Create `backend/plugins/aimeasure/models/pr_ai_cohort.go`:

```go
/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package models

import "time"

// AICohort enumerates the AI-assistance level of a PR.
// Values are stored as varchar strings, never numeric IDs.
type AICohort string

const (
	CohortNone   AICohort = "NONE"
	CohortLow    AICohort = "LOW"
	CohortMedium AICohort = "MEDIUM"
	CohortHigh   AICohort = "HIGH"
)

// PRAICohort is the cohort classification for a merged pull request.
// One row per PR. Rewritten by the classifyPRCohort subtask.
type PRAICohort struct {
	PRId              string    `gorm:"primaryKey;type:varchar(255)" json:"prId"`
	AICohort          AICohort  `gorm:"type:varchar(20);not null;index" json:"aiCohort"`
	ConfidenceScore   int       `gorm:"type:int" json:"confidenceScore"`
	HasExplicitMarker bool      `gorm:"type:bool" json:"hasExplicitMarker"`
	HasCommitTrailer  bool      `gorm:"type:bool" json:"hasCommitTrailer"`
	ClassifierVersion string    `gorm:"type:varchar(32);not null" json:"classifierVersion"`
	ClassifiedAt      time.Time `gorm:"not null" json:"classifiedAt"`
}

func (PRAICohort) TableName() string {
	return "pr_ai_cohort"
}
```

- [ ] **Step 4: Implement `PRDefectSignals` model**

Create `backend/plugins/aimeasure/models/pr_defect_signals.go`:

```go
/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package models

import "time"

// PRDefectSignals records whether a merged PR was followed by a defect indicator
// (revert / hotfix / incident) within a 14-day window. One row per PR; recomputed
// nightly until WindowCloseDate passes (30 days after merge).
type PRDefectSignals struct {
	PRId             string    `gorm:"primaryKey;type:varchar(255)" json:"prId"`
	HasRevert14d     bool      `gorm:"type:bool" json:"hasRevert14d"`
	HasHotfix14d     bool      `gorm:"type:bool" json:"hasHotfix14d"`
	HasIncident14d   bool      `gorm:"type:bool" json:"hasIncident14d"`
	IncidentDataAvailable bool `gorm:"type:bool" json:"incidentDataAvailable"`
	TotalDefectCount int       `gorm:"type:int" json:"totalDefectCount"`
	WindowCloseDate  time.Time `gorm:"not null;index" json:"windowCloseDate"`
	ComputedAt       time.Time `gorm:"not null" json:"computedAt"`
}

func (PRDefectSignals) TableName() string {
	return "pr_defect_signals"
}
```

- [ ] **Step 5: Implement `PRChangeComposition` model**

Create `backend/plugins/aimeasure/models/pr_change_composition.go`:

```go
/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package models

import "time"

// BatchBucket buckets PR sizes for distribution analysis.
type BatchBucket string

const (
	BucketXS BatchBucket = "XS" // < 50 LOC
	BucketS  BatchBucket = "S"  // 50 - 200 LOC
	BucketM  BatchBucket = "M"  // 200 - 500 LOC
	BucketL  BatchBucket = "L"  // 500 - 1000 LOC
	BucketXL BatchBucket = "XL" // > 1000 LOC
)

// PRChangeComposition records the size and refactor-ratio characteristics of a merged PR.
// One row per PR; written once at merge time.
type PRChangeComposition struct {
	PRId           string      `gorm:"primaryKey;type:varchar(255)" json:"prId"`
	Additions      int         `gorm:"type:int" json:"additions"`
	Deletions      int         `gorm:"type:int" json:"deletions"`
	FileCount      int         `gorm:"type:int" json:"fileCount"`
	AdditiveLines  int         `gorm:"type:int" json:"additiveLines"`
	RefactorLines  int         `gorm:"type:int" json:"refactorLines"`
	RefactorRatio  float64     `gorm:"type:decimal(5,4)" json:"refactorRatio"`
	BatchBucket    BatchBucket `gorm:"type:varchar(4);not null;index" json:"batchBucket"`
	ComputedAt     time.Time   `gorm:"not null" json:"computedAt"`
}

func (PRChangeComposition) TableName() string {
	return "pr_change_composition"
}
```

- [ ] **Step 6: Implement identity-resolution helper models**

Create `backend/plugins/aimeasure/models/account_override.go`:

```go
/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package models

// AccountOverride is a manually-maintained mapping from a source identity
// (Slack user ID, GitHub login, Jira accountId) to a DevLake account ID.
// Populated by engineering leadership when automatic resolution fails.
type AccountOverride struct {
	SourceSystem string `gorm:"primaryKey;type:varchar(50)" json:"sourceSystem"` // "slack" / "github" / "jira"
	SourceId     string `gorm:"primaryKey;type:varchar(255)" json:"sourceId"`
	AccountId    string `gorm:"type:varchar(255);not null" json:"accountId"`
	Note         string `gorm:"type:varchar(500)" json:"note"`
}

func (AccountOverride) TableName() string {
	return "aimeasure_account_overrides"
}
```

Create `backend/plugins/aimeasure/models/engineer_role.go`:

```go
/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package models

// EngineerRole is the opt-in seniority tag for an engineer.
// Manually maintained — the platform never auto-classifies.
type EngineerRole struct {
	AccountId string `gorm:"primaryKey;type:varchar(255)" json:"accountId"`
	Role      string `gorm:"type:varchar(50);not null" json:"role"` // "junior" / "mid" / "senior" / "staff" / "principal"
	UpdatedAt string `gorm:"type:varchar(32)" json:"updatedAt"`     // ISO date when last set
}

func (EngineerRole) TableName() string {
	return "aimeasure_engineer_roles"
}
```

- [ ] **Step 7: Run the test, verify it passes**

```bash
docker run --rm -v "/home/ubuntu/workspaces/finally/incubator-devlake:/workspace" -w /workspace/backend mericodev/lake-builder:latest sh -c '
go test ./plugins/aimeasure/models/... -v 2>&1 | tail -10
'
```

Expected: `--- PASS: TestTableNames` and `PASS`.

- [ ] **Step 8: Commit**

```bash
git add backend/plugins/aimeasure/models/
git commit -m "feat(aimeasure): add Phase A data models"
```

---

## Task 3 — Init schema migration

**Files:**
- Create: `backend/plugins/aimeasure/models/migrationscripts/20260513_init_schema.go`
- Create: `backend/plugins/aimeasure/models/migrationscripts/register.go`

- [ ] **Step 1: Create the registry first (will be empty briefly)**

Create `backend/plugins/aimeasure/models/migrationscripts/register.go`:

```go
/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package migrationscripts

import "github.com/apache/incubator-devlake/core/plugin"

// All returns all the migration scripts for the aimeasure plugin.
func All() []plugin.MigrationScript {
	return []plugin.MigrationScript{
		new(initSchema),
	}
}
```

- [ ] **Step 2: Write the migration script**

Create `backend/plugins/aimeasure/models/migrationscripts/20260513_init_schema.go`:

```go
/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package migrationscripts

import (
	"time"

	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/helpers/migrationhelper"
)

type initSchema struct{}

// Phase A tables. Mirrored shape of models/* — defined here so that this
// migration is self-contained and idempotent even if model definitions change later.

type prAICohort20260513 struct {
	PRId              string    `gorm:"primaryKey;type:varchar(255)"`
	AICohort          string    `gorm:"type:varchar(20);not null;index"`
	ConfidenceScore   int       `gorm:"type:int"`
	HasExplicitMarker bool      `gorm:"type:bool"`
	HasCommitTrailer  bool      `gorm:"type:bool"`
	ClassifierVersion string    `gorm:"type:varchar(32);not null"`
	ClassifiedAt      time.Time `gorm:"not null"`
}

func (prAICohort20260513) TableName() string { return "pr_ai_cohort" }

type prDefectSignals20260513 struct {
	PRId                  string    `gorm:"primaryKey;type:varchar(255)"`
	HasRevert14d          bool      `gorm:"type:bool"`
	HasHotfix14d          bool      `gorm:"type:bool"`
	HasIncident14d        bool      `gorm:"type:bool"`
	IncidentDataAvailable bool      `gorm:"type:bool"`
	TotalDefectCount      int       `gorm:"type:int"`
	WindowCloseDate       time.Time `gorm:"not null;index"`
	ComputedAt            time.Time `gorm:"not null"`
}

func (prDefectSignals20260513) TableName() string { return "pr_defect_signals" }

type prChangeComposition20260513 struct {
	PRId          string    `gorm:"primaryKey;type:varchar(255)"`
	Additions     int       `gorm:"type:int"`
	Deletions     int       `gorm:"type:int"`
	FileCount     int       `gorm:"type:int"`
	AdditiveLines int       `gorm:"type:int"`
	RefactorLines int       `gorm:"type:int"`
	RefactorRatio float64   `gorm:"type:decimal(5,4)"`
	BatchBucket   string    `gorm:"type:varchar(4);not null;index"`
	ComputedAt    time.Time `gorm:"not null"`
}

func (prChangeComposition20260513) TableName() string { return "pr_change_composition" }

type accountOverride20260513 struct {
	SourceSystem string `gorm:"primaryKey;type:varchar(50)"`
	SourceId     string `gorm:"primaryKey;type:varchar(255)"`
	AccountId    string `gorm:"type:varchar(255);not null"`
	Note         string `gorm:"type:varchar(500)"`
}

func (accountOverride20260513) TableName() string { return "aimeasure_account_overrides" }

type engineerRole20260513 struct {
	AccountId string `gorm:"primaryKey;type:varchar(255)"`
	Role      string `gorm:"type:varchar(50);not null"`
	UpdatedAt string `gorm:"type:varchar(32)"`
}

func (engineerRole20260513) TableName() string { return "aimeasure_engineer_roles" }

func (*initSchema) Up(basicRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&prAICohort20260513{},
		&prDefectSignals20260513{},
		&prChangeComposition20260513{},
		&accountOverride20260513{},
		&engineerRole20260513{},
	)
}

func (*initSchema) Version() uint64 {
	return 20260513000001
}

func (*initSchema) Name() string {
	return "aimeasure: initialize Phase A schema"
}
```

- [ ] **Step 3: Verify the migration package compiles**

```bash
docker run --rm -v "/home/ubuntu/workspaces/finally/incubator-devlake:/workspace" -w /workspace/backend mericodev/lake-builder:latest sh -c '
go build ./plugins/aimeasure/models/migrationscripts/...
'
```

Expected: no output (clean compile).

- [ ] **Step 4: Commit**

```bash
git add backend/plugins/aimeasure/models/migrationscripts/
git commit -m "feat(aimeasure): add Phase A init schema migration"
```

---

## Task 4 — Wire the plugin into `table_info_test.go`

**Files:**
- Modify: `backend/plugins/table_info_test.go`

- [ ] **Step 1: Run the existing test to verify failure mode**

```bash
docker run --rm -v "/home/ubuntu/workspaces/finally/incubator-devlake:/workspace" -w /workspace/backend mericodev/lake-builder:latest sh -c '
curl -sL https://github.com/vektra/mockery/releases/download/v2.53.5/mockery_2.53.5_Linux_x86_64.tar.gz | tar -xz -C /go/bin mockery
chmod +x /go/bin/mockery
make mock 2>&1 >/dev/null
go test -run "^Test_GetPluginTablesInfo$" -v ./plugins/ 2>&1 | tail -5
'
```

Expected: `FAIL` with `Number of actual plugins (42) and tested plugins (41) don't match` (one extra plugin now exists: `aimeasure`).

- [ ] **Step 2: Add the import and FeedIn call**

In `backend/plugins/table_info_test.go`, add the import (alphabetically between `aidetector` and `argocd`):

```go
aimeasure "github.com/apache/incubator-devlake/plugins/aimeasure/impl"
```

Add the FeedIn call inside `Test_GetPluginTablesInfo` (after `aidetector`):

```go
checker.FeedIn("aimeasure/models", aimeasure.AIMeasure{}.GetTablesInfo)
```

- [ ] **Step 3: Run the test, verify it passes**

```bash
docker run --rm -v "/home/ubuntu/workspaces/finally/incubator-devlake:/workspace" -w /workspace/backend mericodev/lake-builder:latest sh -c '
go test -run "^Test_GetPluginTablesInfo$" -v ./plugins/ 2>&1 | tail -5
'
```

Expected: `--- PASS: Test_GetPluginTablesInfo`.

- [ ] **Step 4: Commit**

```bash
git add backend/plugins/table_info_test.go
git commit -m "test(aimeasure): register in plugin table-info test"
```

---

## Task 5 — Define task data and options struct

**Files:**
- Create: `backend/plugins/aimeasure/tasks/task_data.go`

- [ ] **Step 1: Write a failing test for option decoding**

Create `backend/plugins/aimeasure/tasks/task_data_test.go`:

```go
/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tasks

import "testing"

func TestDecodeAndValidateTaskOptions_AppliesDefaults(t *testing.T) {
	opts, err := DecodeAndValidateTaskOptions(map[string]interface{}{
		"projectName": "demo",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if opts.ProjectName != "demo" {
		t.Errorf("expected projectName 'demo', got %q", opts.ProjectName)
	}
	if opts.HighCohortThreshold != 65 {
		t.Errorf("expected default threshold 65, got %d", opts.HighCohortThreshold)
	}
	if opts.LowCohortThreshold != 30 {
		t.Errorf("expected default low threshold 30, got %d", opts.LowCohortThreshold)
	}
	if opts.DefectWindowDays != 14 {
		t.Errorf("expected default window 14, got %d", opts.DefectWindowDays)
	}
}

func TestDecodeAndValidateTaskOptions_AcceptsOverrides(t *testing.T) {
	opts, err := DecodeAndValidateTaskOptions(map[string]interface{}{
		"projectName":         "demo",
		"highCohortThreshold": 70,
		"lowCohortThreshold":  40,
		"defectWindowDays":    21,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if opts.HighCohortThreshold != 70 {
		t.Errorf("expected 70, got %d", opts.HighCohortThreshold)
	}
	if opts.LowCohortThreshold != 40 {
		t.Errorf("expected 40, got %d", opts.LowCohortThreshold)
	}
	if opts.DefectWindowDays != 21 {
		t.Errorf("expected 21, got %d", opts.DefectWindowDays)
	}
}
```

- [ ] **Step 2: Run the test, verify it fails to compile**

```bash
docker run --rm -v "/home/ubuntu/workspaces/finally/incubator-devlake:/workspace" -w /workspace/backend mericodev/lake-builder:latest sh -c '
go test ./plugins/aimeasure/tasks/... 2>&1 | tail -5
'
```

Expected: `undefined: DecodeAndValidateTaskOptions` etc.

- [ ] **Step 3: Implement `task_data.go`**

Create `backend/plugins/aimeasure/tasks/task_data.go`:

```go
/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tasks

import (
	"github.com/apache/incubator-devlake/core/errors"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
)

// ClassifierVersion is the current version of the cohort classification rules.
// Increment whenever the rules change; the version is persisted on every PR row
// to give audit traceability ("this PR was classified under rules version v1").
const ClassifierVersion = "v1"

type AIMeasureOptions struct {
	ProjectName         string `json:"projectName"`
	HighCohortThreshold int    `json:"highCohortThreshold"` // score >= this → MEDIUM or HIGH
	LowCohortThreshold  int    `json:"lowCohortThreshold"`  // score >= this → LOW
	DefectWindowDays    int    `json:"defectWindowDays"`    // revert/hotfix look-back window
}

type AIMeasureTaskData struct {
	Options *AIMeasureOptions
}

// DecodeAndValidateTaskOptions parses options from a generic map and applies defaults.
func DecodeAndValidateTaskOptions(options map[string]interface{}) (*AIMeasureOptions, errors.Error) {
	var op AIMeasureOptions
	err := helper.Decode(options, &op, nil)
	if err != nil {
		return nil, errors.Default.Wrap(err, "error decoding aimeasure task options")
	}
	if op.HighCohortThreshold == 0 {
		op.HighCohortThreshold = 65
	}
	if op.LowCohortThreshold == 0 {
		op.LowCohortThreshold = 30
	}
	if op.DefectWindowDays == 0 {
		op.DefectWindowDays = 14
	}
	return &op, nil
}
```

- [ ] **Step 4: Run the test, verify it passes**

```bash
docker run --rm -v "/home/ubuntu/workspaces/finally/incubator-devlake:/workspace" -w /workspace/backend mericodev/lake-builder:latest sh -c '
go test ./plugins/aimeasure/tasks/... -v 2>&1 | tail -10
'
```

Expected: `--- PASS: TestDecodeAndValidateTaskOptions_AppliesDefaults` and `--- PASS: TestDecodeAndValidateTaskOptions_AcceptsOverrides`.

- [ ] **Step 5: Commit**

```bash
git add backend/plugins/aimeasure/tasks/
git commit -m "feat(aimeasure): task options with defaults (threshold=65, window=14d)"
```

---

## Task 6 — `classifyPRCohort` subtask (the foundation)

**Files:**
- Create: `backend/plugins/aimeasure/tasks/classify_pr_cohort.go`
- Create: `backend/plugins/aimeasure/tasks/classify_pr_cohort_test.go`

- [ ] **Step 1: Write a failing unit test for the pure classifier logic**

Create `backend/plugins/aimeasure/tasks/classify_pr_cohort_test.go`:

```go
/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tasks

import (
	"testing"

	"github.com/apache/incubator-devlake/plugins/aimeasure/models"
)

func TestClassify_ExplicitMarkerOverridesScore(t *testing.T) {
	input := ClassifyInput{
		ConfidenceScore:   10,
		HasExplicitMarker: true,
		HasCommitTrailer:  false,
		HighThreshold:     65,
		LowThreshold:      30,
	}
	if got := Classify(input); got != models.CohortHigh {
		t.Errorf("expected HIGH due to explicit marker, got %s", got)
	}
}

func TestClassify_CommitTrailerOverridesScore(t *testing.T) {
	input := ClassifyInput{
		ConfidenceScore:   10,
		HasExplicitMarker: false,
		HasCommitTrailer:  true,
		HighThreshold:     65,
		LowThreshold:      30,
	}
	if got := Classify(input); got != models.CohortHigh {
		t.Errorf("expected HIGH due to commit trailer, got %s", got)
	}
}

func TestClassify_MediumByScore(t *testing.T) {
	input := ClassifyInput{
		ConfidenceScore: 75,
		HighThreshold:   65,
		LowThreshold:    30,
	}
	if got := Classify(input); got != models.CohortMedium {
		t.Errorf("expected MEDIUM, got %s", got)
	}
}

func TestClassify_LowByScore(t *testing.T) {
	input := ClassifyInput{
		ConfidenceScore: 45,
		HighThreshold:   65,
		LowThreshold:    30,
	}
	if got := Classify(input); got != models.CohortLow {
		t.Errorf("expected LOW, got %s", got)
	}
}

func TestClassify_NoneByScore(t *testing.T) {
	input := ClassifyInput{
		ConfidenceScore: 15,
		HighThreshold:   65,
		LowThreshold:    30,
	}
	if got := Classify(input); got != models.CohortNone {
		t.Errorf("expected NONE, got %s", got)
	}
}

func TestClassify_ThresholdEdgeCases(t *testing.T) {
	hi := ClassifyInput{ConfidenceScore: 65, HighThreshold: 65, LowThreshold: 30}
	if got := Classify(hi); got != models.CohortMedium {
		t.Errorf("score == high threshold should be MEDIUM, got %s", got)
	}
	lo := ClassifyInput{ConfidenceScore: 30, HighThreshold: 65, LowThreshold: 30}
	if got := Classify(lo); got != models.CohortLow {
		t.Errorf("score == low threshold should be LOW, got %s", got)
	}
	just := ClassifyInput{ConfidenceScore: 29, HighThreshold: 65, LowThreshold: 30}
	if got := Classify(just); got != models.CohortNone {
		t.Errorf("score < low threshold should be NONE, got %s", got)
	}
}
```

- [ ] **Step 2: Run the test, verify it fails to compile**

```bash
docker run --rm -v "/home/ubuntu/workspaces/finally/incubator-devlake:/workspace" -w /workspace/backend mericodev/lake-builder:latest sh -c '
go test ./plugins/aimeasure/tasks/... 2>&1 | tail -5
'
```

Expected: `undefined: Classify` and `undefined: ClassifyInput`.

- [ ] **Step 3: Implement the subtask with the pure classifier extracted**

Create `backend/plugins/aimeasure/tasks/classify_pr_cohort.go`:

```go
/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tasks

import (
	"regexp"
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/aimeasure/models"
)

var ClassifyPRCohortMeta = plugin.SubTaskMeta{
	Name:             "classifyPRCohort",
	EntryPoint:       ClassifyPRCohort,
	EnabledByDefault: true,
	Description:      "Classify each merged PR into NONE/LOW/MEDIUM/HIGH AI-assistance cohort",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CODE},
}

// ClassifyInput is the pure-function input for cohort classification.
// Extracted so the rule is unit-testable without DB access.
type ClassifyInput struct {
	ConfidenceScore   int
	HasExplicitMarker bool
	HasCommitTrailer  bool
	HighThreshold     int
	LowThreshold      int
}

// Classify applies the cohort decision rules. Pure function.
func Classify(in ClassifyInput) models.AICohort {
	if in.HasExplicitMarker || in.HasCommitTrailer {
		return models.CohortHigh
	}
	if in.ConfidenceScore >= in.HighThreshold {
		return models.CohortMedium
	}
	if in.ConfidenceScore >= in.LowThreshold {
		return models.CohortLow
	}
	return models.CohortNone
}

// commitTrailerRE matches the `Co-authored-by: <tool>` trailer for common AI agents.
var commitTrailerRE = regexp.MustCompile(`(?im)^Co-authored-by:.*(claude|copilot|cursor|devin|github\s*copilot|anthropic)`)

// HasAITrailer returns true if any of the supplied commit messages contains an AI trailer.
func HasAITrailer(commitMessages []string) bool {
	for _, msg := range commitMessages {
		if commitTrailerRE.MatchString(msg) {
			return true
		}
	}
	return false
}

// prCohortInput is the row shape returned by the classify cursor.
type prCohortInput struct {
	PRId              string `gorm:"column:pr_id"`
	ConfidenceScore   int    `gorm:"column:confidence_score"`
	HasExplicitMarker bool   `gorm:"column:has_explicit_marker"`
}

// ClassifyPRCohort is the subtask entrypoint. Reads ai_usage_signals + commits,
// writes one pr_ai_cohort row per merged PR. Idempotent: rerunning overwrites.
func ClassifyPRCohort(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*AIMeasureTaskData)
	db := taskCtx.GetDal()
	logger := taskCtx.GetLogger()

	cursor, err := db.Cursor(
		dal.Select("pull_requests.id AS pr_id, COALESCE(ai_usage_signals.ai_confidence_score, 0) AS confidence_score, COALESCE(ai_usage_signals.explicit_tool_detected, FALSE) AS has_explicit_marker"),
		dal.From("pull_requests"),
		dal.Join("LEFT JOIN ai_usage_signals ON ai_usage_signals.pull_request_id = pull_requests.id"),
		dal.Where("pull_requests.merged_date IS NOT NULL"),
	)
	if err != nil {
		return errors.Default.Wrap(err, "failed to query PRs for classification")
	}
	defer cursor.Close()

	now := time.Now().UTC()
	count := 0
	for cursor.Next() {
		var in prCohortInput
		if err := db.Fetch(cursor, &in); err != nil {
			return errors.Default.Wrap(err, "row scan failed")
		}

		// Pull commit messages for the PR
		messages, err := loadCommitMessages(db, in.PRId)
		if err != nil {
			logger.Warn(err, "could not load commits for PR %s; treating as no-trailer", in.PRId)
		}
		hasTrailer := HasAITrailer(messages)

		cohort := Classify(ClassifyInput{
			ConfidenceScore:   in.ConfidenceScore,
			HasExplicitMarker: in.HasExplicitMarker,
			HasCommitTrailer:  hasTrailer,
			HighThreshold:     data.Options.HighCohortThreshold,
			LowThreshold:      data.Options.LowCohortThreshold,
		})

		row := &models.PRAICohort{
			PRId:              in.PRId,
			AICohort:          cohort,
			ConfidenceScore:   in.ConfidenceScore,
			HasExplicitMarker: in.HasExplicitMarker,
			HasCommitTrailer:  hasTrailer,
			ClassifierVersion: ClassifierVersion,
			ClassifiedAt:      now,
		}
		if err := db.CreateOrUpdate(row); err != nil {
			return errors.Default.Wrap(err, "failed to upsert pr_ai_cohort row")
		}
		count++
	}
	logger.Info("classifyPRCohort processed %d PRs", count)
	return nil
}

// commitMessage is a scan target for the trailer query.
type commitMessage struct {
	Message string `gorm:"column:message"`
}

func loadCommitMessages(db dal.Dal, prId string) ([]string, errors.Error) {
	var rows []commitMessage
	err := db.All(&rows,
		dal.Select("commits.message"),
		dal.From("commits"),
		dal.Join("INNER JOIN pull_request_commits prc ON prc.commit_sha = commits.sha"),
		dal.Where("prc.pull_request_id = ?", prId),
	)
	if err != nil {
		return nil, err
	}
	msgs := make([]string, 0, len(rows))
	for _, r := range rows {
		msgs = append(msgs, r.Message)
	}
	return msgs, nil
}

```

- [ ] **Step 4: Run the test, verify pure-logic tests pass**

```bash
docker run --rm -v "/home/ubuntu/workspaces/finally/incubator-devlake:/workspace" -w /workspace/backend mericodev/lake-builder:latest sh -c '
go test -run "^TestClassify" ./plugins/aimeasure/tasks/... -v 2>&1 | tail -15
'
```

Expected: all 6 `TestClassify_*` tests PASS.

- [ ] **Step 5: Add a unit test for `HasAITrailer`**

Append to `backend/plugins/aimeasure/tasks/classify_pr_cohort_test.go`:

```go
func TestHasAITrailer(t *testing.T) {
	cases := []struct {
		name  string
		input []string
		want  bool
	}{
		{"empty", nil, false},
		{"plain commit", []string{"fix: nil deref in widget"}, false},
		{"claude trailer", []string{"feat: thing\n\nCo-authored-by: Claude <claude@anthropic.com>"}, true},
		{"copilot trailer", []string{"feat: thing\n\nCo-authored-by: GitHub Copilot <copilot@github.com>"}, true},
		{"cursor trailer", []string{"feat: thing\n\nCo-authored-by: Cursor"}, true},
		{"trailer in second commit", []string{"a", "b\n\nCo-authored-by: Claude"}, true},
		{"trailer in body only is fine too", []string{"Refactor stuff\nCo-authored-by: Claude"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := HasAITrailer(c.input); got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}
```

- [ ] **Step 6: Run the trailer test, verify it passes**

```bash
docker run --rm -v "/home/ubuntu/workspaces/finally/incubator-devlake:/workspace" -w /workspace/backend mericodev/lake-builder:latest sh -c '
go test -run "^TestHasAITrailer$" ./plugins/aimeasure/tasks/... -v 2>&1 | tail -15
'
```

Expected: all 7 sub-tests PASS.

- [ ] **Step 7: Commit**

```bash
git add backend/plugins/aimeasure/tasks/classify_pr_cohort.go backend/plugins/aimeasure/tasks/classify_pr_cohort_test.go
git commit -m "feat(aimeasure): classifyPRCohort subtask with pure-function classifier"
```

---

## Task 7 — `computeChangeComposition` subtask

**Files:**
- Create: `backend/plugins/aimeasure/tasks/compute_change_composition.go`
- Create: `backend/plugins/aimeasure/tasks/compute_change_composition_test.go`

- [ ] **Step 1: Write failing unit tests for the bucketing logic**

Create `backend/plugins/aimeasure/tasks/compute_change_composition_test.go`:

```go
/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tasks

import (
	"testing"

	"github.com/apache/incubator-devlake/plugins/aimeasure/models"
)

func TestBatchBucket(t *testing.T) {
	cases := []struct {
		loc  int
		want models.BatchBucket
	}{
		{0, models.BucketXS},
		{49, models.BucketXS},
		{50, models.BucketS},
		{199, models.BucketS},
		{200, models.BucketM},
		{499, models.BucketM},
		{500, models.BucketL},
		{999, models.BucketL},
		{1000, models.BucketXL},
		{50_000, models.BucketXL},
	}
	for _, c := range cases {
		if got := BucketFor(c.loc); got != c.want {
			t.Errorf("BucketFor(%d) = %s, want %s", c.loc, got, c.want)
		}
	}
}

func TestRefactorRatio(t *testing.T) {
	cases := []struct {
		name       string
		additive   int
		refactor   int
		want       float64
		wantApprox bool
	}{
		{"all additive", 100, 0, 0.0, false},
		{"all refactor", 0, 100, 1.0, false},
		{"half and half", 50, 50, 0.5, false},
		{"empty PR (avoid div by 0)", 0, 0, 0.0, false},
		{"33% refactor", 200, 100, 0.3333, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ComputeRefactorRatio(c.additive, c.refactor)
			if c.wantApprox {
				if got < c.want-0.001 || got > c.want+0.001 {
					t.Errorf("got %v, want approx %v", got, c.want)
				}
			} else if got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run the test, verify it fails**

```bash
docker run --rm -v "/home/ubuntu/workspaces/finally/incubator-devlake:/workspace" -w /workspace/backend mericodev/lake-builder:latest sh -c '
go test ./plugins/aimeasure/tasks/... 2>&1 | tail -5
'
```

Expected: `undefined: BucketFor`, `undefined: ComputeRefactorRatio`.

- [ ] **Step 3: Implement the subtask**

Create `backend/plugins/aimeasure/tasks/compute_change_composition.go`:

```go
/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tasks

import (
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/aimeasure/models"
)

var ComputeChangeCompositionMeta = plugin.SubTaskMeta{
	Name:             "computeChangeComposition",
	EntryPoint:       ComputeChangeComposition,
	EnabledByDefault: true,
	Description:      "Compute per-PR batch size, file count, and refactor ratio for change-composition drift tracking",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CODE},
}

// BucketFor returns the batch-size bucket for a total LOC change count.
func BucketFor(loc int) models.BatchBucket {
	switch {
	case loc < 50:
		return models.BucketXS
	case loc < 200:
		return models.BucketS
	case loc < 500:
		return models.BucketM
	case loc < 1000:
		return models.BucketL
	default:
		return models.BucketXL
	}
}

// ComputeRefactorRatio returns refactor_lines / (additive_lines + refactor_lines).
// Returns 0.0 when both are 0 (avoids div-by-zero on empty PRs).
func ComputeRefactorRatio(additive, refactor int) float64 {
	total := additive + refactor
	if total == 0 {
		return 0.0
	}
	return float64(refactor) / float64(total)
}

// Heuristic for additive vs. refactor classification: a file is treated as
// "additive" if it had no deletions across all the PR's commits (suggesting a
// brand-new file); files with any deletions contribute their additions+deletions
// to refactor_lines. For Phase A this approximates pre-existing-file detection
// without needing a separate base-ref diff query. Refined in a later phase.
//
// PR file-level aggregate. Scanned from the cursor.
type prFileAgg struct {
	PRId      string `gorm:"column:pr_id"`
	FilePath  string `gorm:"column:file_path"`
	FileAdd   int    `gorm:"column:file_add"`
	FileDel   int    `gorm:"column:file_del"`
}

func ComputeChangeComposition(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	logger := taskCtx.GetLogger()

	// Step 1: enumerate merged PRs (id + total additions/deletions)
	type prTotals struct {
		Id        string `gorm:"column:id"`
		Additions int    `gorm:"column:additions"`
		Deletions int    `gorm:"column:deletions"`
	}
	var prs []prTotals
	if err := db.All(&prs,
		dal.Select("id, additions, deletions"),
		dal.From("pull_requests"),
		dal.Where("merged_date IS NOT NULL"),
	); err != nil {
		return errors.Default.Wrap(err, "failed to enumerate merged PRs")
	}

	now := time.Now().UTC()
	count := 0
	for _, pr := range prs {
		// Step 2: per-PR per-file aggregate of additions/deletions
		var files []prFileAgg
		if err := db.All(&files,
			dal.Select("? AS pr_id, cf.file_path AS file_path, SUM(cf.additions) AS file_add, SUM(cf.deletions) AS file_del", pr.Id),
			dal.From("commit_files cf"),
			dal.Join("INNER JOIN pull_request_commits prc ON prc.commit_sha = cf.commit_sha"),
			dal.Where("prc.pull_request_id = ?", pr.Id),
			dal.Groupby("cf.file_path"),
		); err != nil {
			return errors.Default.Wrap(err, "failed to aggregate commit_files for PR "+pr.Id)
		}

		additive, refactor, fileCount := 0, 0, len(files)
		for _, f := range files {
			if f.FileDel == 0 {
				additive += f.FileAdd
			} else {
				refactor += f.FileAdd + f.FileDel
			}
		}
		totalLOC := pr.Additions + pr.Deletions

		out := &models.PRChangeComposition{
			PRId:          pr.Id,
			Additions:     pr.Additions,
			Deletions:     pr.Deletions,
			FileCount:     fileCount,
			AdditiveLines: additive,
			RefactorLines: refactor,
			RefactorRatio: ComputeRefactorRatio(additive, refactor),
			BatchBucket:   BucketFor(totalLOC),
			ComputedAt:    now,
		}
		if err := db.CreateOrUpdate(out); err != nil {
			return errors.Default.Wrap(err, "failed to upsert pr_change_composition row")
		}
		count++
	}
	logger.Info("computeChangeComposition processed %d PRs", count)
	return nil
}
```

- [ ] **Step 4: Run the test, verify pure-logic tests pass**

```bash
docker run --rm -v "/home/ubuntu/workspaces/finally/incubator-devlake:/workspace" -w /workspace/backend mericodev/lake-builder:latest sh -c '
go test -run "^TestBatchBucket$|^TestRefactorRatio$" ./plugins/aimeasure/tasks/... -v 2>&1 | tail -15
'
```

Expected: both tests PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/plugins/aimeasure/tasks/compute_change_composition.go backend/plugins/aimeasure/tasks/compute_change_composition_test.go
git commit -m "feat(aimeasure): computeChangeComposition subtask with batch bucketing"
```

---

## Task 8 — `computeQualityCohort` subtask

**Files:**
- Create: `backend/plugins/aimeasure/tasks/compute_quality_cohort.go`
- Create: `backend/plugins/aimeasure/tasks/compute_quality_cohort_test.go`

- [ ] **Step 1: Write failing unit tests for hotfix detection and incident-availability flag**

Create `backend/plugins/aimeasure/tasks/compute_quality_cohort_test.go`:

```go
/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tasks

import "testing"

func TestIsHotfixTitle(t *testing.T) {
	cases := []struct {
		title string
		want  bool
	}{
		{"feat: add widget", false},
		{"hotfix: prod down", true},
		{"HOTFIX(api): null deref", true},
		{"Urgent: revert deploy", true},
		{"emergency-rollback", true},
		{"fixup! 1a2b3c4 hotfix part 2", true},
		{"chore: clean up", false},
	}
	for _, c := range cases {
		if got := IsHotfixTitle(c.title); got != c.want {
			t.Errorf("IsHotfixTitle(%q) = %v, want %v", c.title, got, c.want)
		}
	}
}

func TestFileOverlapRatio(t *testing.T) {
	cases := []struct {
		name string
		a, b []string
		want float64
	}{
		{"identical", []string{"x.go", "y.go"}, []string{"x.go", "y.go"}, 1.0},
		{"disjoint", []string{"a"}, []string{"b"}, 0.0},
		{"half overlap", []string{"x.go", "y.go"}, []string{"x.go"}, 0.5},
		{"empty a", nil, []string{"x.go"}, 0.0},
		{"empty b", []string{"x.go"}, nil, 0.0},
		{"both empty", nil, nil, 0.0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := FileOverlapRatio(c.a, c.b)
			if got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run the test, verify it fails**

```bash
docker run --rm -v "/home/ubuntu/workspaces/finally/incubator-devlake:/workspace" -w /workspace/backend mericodev/lake-builder:latest sh -c '
go test ./plugins/aimeasure/tasks/... 2>&1 | tail -5
'
```

Expected: `undefined: IsHotfixTitle`, `undefined: FileOverlapRatio`.

- [ ] **Step 3: Implement the subtask**

Create `backend/plugins/aimeasure/tasks/compute_quality_cohort.go`:

```go
/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tasks

import (
	"regexp"
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/aimeasure/models"
)

var ComputeQualityCohortMeta = plugin.SubTaskMeta{
	Name:             "computeQualityCohort",
	EntryPoint:       ComputeQualityCohort,
	EnabledByDefault: true,
	Description:      "Compute defect signals (revert/hotfix/incident) within the configured window per merged PR",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CODE},
}

// hotfixTitleRE matches PR titles that indicate emergency fixes.
var hotfixTitleRE = regexp.MustCompile(`(?i)\b(hotfix|urgent|emergency|emergency-rollback)\b`)

// IsHotfixTitle returns true if a PR title matches the hotfix pattern.
func IsHotfixTitle(title string) bool {
	return hotfixTitleRE.MatchString(title)
}

// FileOverlapRatio returns |a ∩ b| / max(|a|, 1). Used to decide whether a candidate
// hotfix PR touches "enough" of the original PR's files (Phase A threshold: ≥ 0.5).
func FileOverlapRatio(a, b []string) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}
	set := make(map[string]struct{}, len(b))
	for _, f := range b {
		set[f] = struct{}{}
	}
	overlap := 0
	for _, f := range a {
		if _, ok := set[f]; ok {
			overlap++
		}
	}
	return float64(overlap) / float64(len(a))
}

func ComputeQualityCohort(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*AIMeasureTaskData)
	db := taskCtx.GetDal()
	logger := taskCtx.GetLogger()

	// Detect whether an incident table is present. If not, mark IncidentDataAvailable=false.
	incidentDataAvailable, err := tableExists(db, "issues") // 'issues' carries incidents when type='INCIDENT'
	if err != nil {
		logger.Warn(err, "could not detect incident table; assuming unavailable")
	}

	windowDays := data.Options.DefectWindowDays

	type prRow struct {
		Id             string    `gorm:"column:id"`
		MergedDate     time.Time `gorm:"column:merged_date"`
		Title          string    `gorm:"column:title"`
		MergeCommitSha string    `gorm:"column:merge_commit_sha"`
	}
	var prs []prRow
	if err := db.All(&prs,
		dal.Select("id, merged_date, title, merge_commit_sha"),
		dal.From("pull_requests"),
		dal.Where("merged_date IS NOT NULL"),
	); err != nil {
		return errors.Default.Wrap(err, "failed to query merged PRs")
	}

	now := time.Now().UTC()
	count := 0
	for _, pr := range prs {
		windowEnd := pr.MergedDate.Add(time.Duration(windowDays) * 24 * time.Hour)

		hasRevert, err := detectRevert(db, pr.MergeCommitSha, pr.MergedDate, windowEnd)
		if err != nil {
			logger.Warn(err, "revert detection failed for PR %s", pr.Id)
		}
		hasHotfix, err := detectHotfix(db, pr.Id, pr.MergedDate, windowEnd)
		if err != nil {
			logger.Warn(err, "hotfix detection failed for PR %s", pr.Id)
		}
		hasIncident := false
		if incidentDataAvailable {
			hasIncident, err = detectIncident(db, pr.Id, pr.MergedDate, windowEnd)
			if err != nil {
				logger.Warn(err, "incident detection failed for PR %s", pr.Id)
			}
		}

		out := &models.PRDefectSignals{
			PRId:                  pr.Id,
			HasRevert14d:          hasRevert,
			HasHotfix14d:          hasHotfix,
			HasIncident14d:        hasIncident,
			IncidentDataAvailable: incidentDataAvailable,
			TotalDefectCount:      boolToInt(hasRevert) + boolToInt(hasHotfix) + boolToInt(hasIncident),
			WindowCloseDate:       windowEnd,
			ComputedAt:            now,
		}
		if err := db.CreateOrUpdate(out); err != nil {
			return errors.Default.Wrap(err, "failed to upsert pr_defect_signals row")
		}
		count++
	}
	logger.Info("computeQualityCohort processed %d PRs", count)
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func tableExists(db dal.Dal, tableName string) (bool, errors.Error) {
	count, err := db.Count(
		dal.From("information_schema.tables"),
		dal.Where("table_name = ?", tableName),
	)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func detectRevert(db dal.Dal, mergeSha string, after, before time.Time) (bool, errors.Error) {
	if mergeSha == "" {
		return false, nil
	}
	count, err := db.Count(
		dal.From("commits"),
		dal.Where("authored_date >= ? AND authored_date < ? AND message LIKE ?", after, before, "%"+mergeSha+"%"),
	)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func detectHotfix(db dal.Dal, prId string, after, before time.Time) (bool, errors.Error) {
	// Get the original PR's file list
	type filePath struct{ FilePath string }
	var originalFiles []filePath
	err := db.All(&originalFiles,
		dal.Select("DISTINCT cf.file_path AS file_path"),
		dal.From("commit_files cf"),
		dal.Join("INNER JOIN pull_request_commits prc ON prc.commit_sha = cf.commit_sha"),
		dal.Where("prc.pull_request_id = ?", prId),
	)
	if err != nil {
		return false, err
	}
	if len(originalFiles) == 0 {
		return false, nil
	}
	originalSet := make([]string, len(originalFiles))
	for i, f := range originalFiles {
		originalSet[i] = f.FilePath
	}

	// Candidate hotfix PRs: hotfix-titled, merged within window
	type candidate struct {
		Id    string
		Title string
	}
	var candidates []candidate
	err = db.All(&candidates,
		dal.Select("id, title"),
		dal.From("pull_requests"),
		dal.Where("merged_date >= ? AND merged_date < ? AND id != ?", after, before, prId),
	)
	if err != nil {
		return false, err
	}

	for _, c := range candidates {
		if !IsHotfixTitle(c.Title) {
			continue
		}
		var hotfixFiles []filePath
		err = db.All(&hotfixFiles,
			dal.Select("DISTINCT cf.file_path AS file_path"),
			dal.From("commit_files cf"),
			dal.Join("INNER JOIN pull_request_commits prc ON prc.commit_sha = cf.commit_sha"),
			dal.Where("prc.pull_request_id = ?", c.Id),
		)
		if err != nil {
			continue
		}
		hotfixSet := make([]string, len(hotfixFiles))
		for i, f := range hotfixFiles {
			hotfixSet[i] = f.FilePath
		}
		if FileOverlapRatio(originalSet, hotfixSet) >= 0.5 {
			return true, nil
		}
	}
	return false, nil
}

func detectIncident(db dal.Dal, prId string, after, before time.Time) (bool, errors.Error) {
	// Phase A heuristic: any issue of type INCIDENT created within the window is
	// treated as a related incident. Proper PR↔incident linkage (via commit
	// references or branch metadata) is future work in Phase B.
	count, err := db.Count(
		dal.From("issues"),
		dal.Where("type = ? AND created_date >= ? AND created_date < ?", "INCIDENT", after, before),
	)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
```

- [ ] **Step 4: Run the unit tests, verify they pass**

```bash
docker run --rm -v "/home/ubuntu/workspaces/finally/incubator-devlake:/workspace" -w /workspace/backend mericodev/lake-builder:latest sh -c '
go test -run "^TestIsHotfixTitle$|^TestFileOverlapRatio$" ./plugins/aimeasure/tasks/... -v 2>&1 | tail -15
'
```

Expected: both tests PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/plugins/aimeasure/tasks/compute_quality_cohort.go backend/plugins/aimeasure/tasks/compute_quality_cohort_test.go
git commit -m "feat(aimeasure): computeQualityCohort subtask with revert/hotfix/incident detection"
```

---

## Task 9 — Wire subtasks into `impl.go` and `PrepareTaskData`

**Files:**
- Modify: `backend/plugins/aimeasure/impl/impl.go`

- [ ] **Step 1: Replace the stub `SubTaskMetas` and `PrepareTaskData`**

In `backend/plugins/aimeasure/impl/impl.go`, replace the existing stub implementations of `SubTaskMetas` and `PrepareTaskData` with:

```go
func (p AIMeasure) SubTaskMetas() []plugin.SubTaskMeta {
	return []plugin.SubTaskMeta{
		tasks.ClassifyPRCohortMeta,       // run first — produces pr_ai_cohort
		tasks.ComputeChangeCompositionMeta, // independent — produces pr_change_composition
		tasks.ComputeQualityCohortMeta,   // independent — produces pr_defect_signals
	}
}

func (p AIMeasure) PrepareTaskData(taskCtx plugin.TaskContext, options map[string]interface{}) (interface{}, errors.Error) {
	opts, err := tasks.DecodeAndValidateTaskOptions(options)
	if err != nil {
		return nil, err
	}
	return &tasks.AIMeasureTaskData{Options: opts}, nil
}
```

And add to the import block at the top:

```go
"github.com/apache/incubator-devlake/plugins/aimeasure/tasks"
```

- [ ] **Step 2: Verify the plugin compiles**

```bash
docker run --rm -v "/home/ubuntu/workspaces/finally/incubator-devlake:/workspace" -w /workspace/backend mericodev/lake-builder:latest sh -c '
go build ./plugins/aimeasure/...
'
```

Expected: no compile errors.

- [ ] **Step 3: Run all aimeasure unit tests**

```bash
docker run --rm -v "/home/ubuntu/workspaces/finally/incubator-devlake:/workspace" -w /workspace/backend mericodev/lake-builder:latest sh -c '
go test ./plugins/aimeasure/... -v 2>&1 | tail -40
'
```

Expected: all unit tests PASS (TestClassify_*, TestHasAITrailer, TestBatchBucket, TestRefactorRatio, TestIsHotfixTitle, TestFileOverlapRatio, TestTableNames, TestDecodeAndValidateTaskOptions_*).

- [ ] **Step 4: Commit**

```bash
git add backend/plugins/aimeasure/impl/impl.go
git commit -m "feat(aimeasure): wire Phase A subtasks into PluginMeta"
```

---

## Task 10 — E2E test for `classifyPRCohort`

**Files:**
- Create: `backend/plugins/aimeasure/e2e/classify_pr_cohort_test.go`
- Create: `backend/plugins/aimeasure/e2e/fixtures/pull_requests.csv`
- Create: `backend/plugins/aimeasure/e2e/fixtures/commits.csv`
- Create: `backend/plugins/aimeasure/e2e/fixtures/pull_request_commits.csv`
- Create: `backend/plugins/aimeasure/e2e/fixtures/ai_usage_signals.csv`
- Create: `backend/plugins/aimeasure/e2e/fixtures/expected_pr_ai_cohort.csv`

- [ ] **Step 1: Create the input fixtures**

Create `backend/plugins/aimeasure/e2e/fixtures/pull_requests.csv`:

```csv
id,base_repo_id,merged_date,title,merge_commit_sha,additions,deletions,status
pr1,repo1,2026-05-01 10:00:00,feat: add widget,sha-merge-1,100,10,MERGED
pr2,repo1,2026-05-02 10:00:00,feat: AI-assisted refactor,sha-merge-2,200,50,MERGED
pr3,repo1,2026-05-03 10:00:00,fix: nil deref,sha-merge-3,5,2,MERGED
pr4,repo1,2026-05-04 10:00:00,chore: cleanup,sha-merge-4,50,100,MERGED
```

Create `backend/plugins/aimeasure/e2e/fixtures/commits.csv`:

```csv
sha,authored_date,message
sha-commit-1,2026-04-30 09:00:00,feat: add widget
sha-commit-2,2026-05-01 09:00:00,feat: refactor with Claude\n\nCo-authored-by: Claude <claude@anthropic.com>
sha-commit-3,2026-05-02 09:00:00,fix: nil deref in handler
sha-commit-4,2026-05-03 09:00:00,chore: cleanup unused vars
```

Create `backend/plugins/aimeasure/e2e/fixtures/pull_request_commits.csv`:

```csv
pull_request_id,commit_sha
pr1,sha-commit-1
pr2,sha-commit-2
pr3,sha-commit-3
pr4,sha-commit-4
```

Create `backend/plugins/aimeasure/e2e/fixtures/ai_usage_signals.csv`:

```csv
id,pull_request_id,ai_confidence_score,explicit_tool_detected,detected_at
s1,pr1,15,0,2026-05-01 10:00:00
s2,pr2,90,1,2026-05-02 10:00:00
s3,pr3,45,0,2026-05-03 10:00:00
s4,pr4,5,0,2026-05-04 10:00:00
```

Create `backend/plugins/aimeasure/e2e/fixtures/expected_pr_ai_cohort.csv`:

```csv
pr_id,ai_cohort,confidence_score,has_explicit_marker,has_commit_trailer,classifier_version
pr1,NONE,15,0,0,v1
pr2,HIGH,90,1,1,v1
pr3,LOW,45,0,0,v1
pr4,NONE,5,0,0,v1
```

- [ ] **Step 2: Write the failing e2e test**

Create `backend/plugins/aimeasure/e2e/classify_pr_cohort_test.go`:

```go
/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"testing"

	"github.com/apache/incubator-devlake/core/models/common"
	"github.com/apache/incubator-devlake/core/models/domainlayer/code"
	"github.com/apache/incubator-devlake/helpers/e2ehelper"
	aidetectorModels "github.com/apache/incubator-devlake/plugins/aidetector/models"
	"github.com/apache/incubator-devlake/plugins/aimeasure/impl"
	"github.com/apache/incubator-devlake/plugins/aimeasure/models"
	"github.com/apache/incubator-devlake/plugins/aimeasure/tasks"
)

func TestClassifyPRCohortDataFlow(t *testing.T) {
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
	dataflowTester.ImportCsvIntoTabler("./fixtures/commits.csv", &code.Commit{})
	dataflowTester.ImportCsvIntoTabler("./fixtures/pull_request_commits.csv", &code.PullRequestCommit{})
	dataflowTester.ImportCsvIntoTabler("./fixtures/ai_usage_signals.csv", &aidetectorModels.AIUsageSignal{})

	dataflowTester.FlushTabler(&models.PRAICohort{})
	dataflowTester.Subtask(tasks.ClassifyPRCohortMeta, taskData)
	dataflowTester.VerifyTableWithOptions(&models.PRAICohort{}, e2ehelper.TableOptions{
		CSVRelPath:   "./fixtures/expected_pr_ai_cohort.csv",
		IgnoreTypes:  []interface{}{common.NoPKModel{}},
		IgnoreFields: []string{"classified_at"},
	})
}
```

- [ ] **Step 3: Validate fixtures by running e2e against Postgres in Docker**

```bash
docker network create aimeasure-e2e-net 2>/dev/null || true
docker run -d --rm --name pg-aimeasure --network aimeasure-e2e-net -e POSTGRES_USER=merico -e POSTGRES_PASSWORD=merico -e POSTGRES_DB=lake -p 5432:5432 postgres:14
sleep 5  # let postgres init

docker run --rm --network aimeasure-e2e-net \
  -v "/home/ubuntu/workspaces/finally/incubator-devlake:/workspace" \
  -w /workspace/backend \
  -e DB_URL='postgres://merico:merico@pg-aimeasure:5432/lake?sslmode=disable' \
  mericodev/lake-builder:latest sh -c '
curl -sL https://github.com/vektra/mockery/releases/download/v2.53.5/mockery_2.53.5_Linux_x86_64.tar.gz | tar -xz -C /go/bin mockery
chmod +x /go/bin/mockery
make mock 2>&1 >/dev/null
cp env.example .env
go test -run TestClassifyPRCohortDataFlow -v ./plugins/aimeasure/e2e/... 2>&1 | tail -20
'

docker stop pg-aimeasure
docker network rm aimeasure-e2e-net
```

Expected: `--- PASS: TestClassifyPRCohortDataFlow`.

- [ ] **Step 4: Repeat against MySQL**

```bash
docker network create aimeasure-e2e-net 2>/dev/null || true
docker run -d --rm --name mysql-aimeasure --network aimeasure-e2e-net -e MYSQL_DATABASE=lake -e MYSQL_USER=merico -e MYSQL_PASSWORD=merico -e MYSQL_ROOT_PASSWORD=root mysql:8
sleep 25  # mysql is slow to init

docker run --rm --network aimeasure-e2e-net \
  -v "/home/ubuntu/workspaces/finally/incubator-devlake:/workspace" \
  -w /workspace/backend \
  -e DB_URL='mysql://merico:merico@tcp(mysql-aimeasure:3306)/lake?charset=utf8mb4&parseTime=True' \
  mericodev/lake-builder:latest sh -c '
curl -sL https://github.com/vektra/mockery/releases/download/v2.53.5/mockery_2.53.5_Linux_x86_64.tar.gz | tar -xz -C /go/bin mockery
chmod +x /go/bin/mockery
make mock 2>&1 >/dev/null
cp env.example .env
go test -run TestClassifyPRCohortDataFlow -v ./plugins/aimeasure/e2e/... 2>&1 | tail -20
'

docker stop mysql-aimeasure
docker network rm aimeasure-e2e-net
```

Expected: `--- PASS: TestClassifyPRCohortDataFlow`.

- [ ] **Step 5: Commit**

```bash
git add backend/plugins/aimeasure/e2e/
git commit -m "test(aimeasure): e2e for classifyPRCohort across MySQL+Postgres"
```

---

## Task 11 — E2E test for `computeChangeComposition`

**Files:**
- Create: `backend/plugins/aimeasure/e2e/fixtures/commit_files.csv`
- Create: `backend/plugins/aimeasure/e2e/fixtures/expected_pr_change_composition.csv`
- Create: `backend/plugins/aimeasure/e2e/compute_change_composition_test.go`

- [ ] **Step 1: Create the additional fixture for commit files**

Create `backend/plugins/aimeasure/e2e/fixtures/commit_files.csv`:

```csv
id,commit_sha,file_path,additions,deletions
cf1,sha-commit-1,apps/api/widget.go,100,0
cf2,sha-commit-2,apps/api/handler.go,150,40
cf3,sha-commit-2,apps/api/handler_test.go,50,10
cf4,sha-commit-3,apps/api/handler.go,5,2
cf5,sha-commit-4,apps/api/cleanup.go,0,100
cf6,sha-commit-4,apps/api/util.go,50,0
```

- [ ] **Step 2: Compute and write the expected output**

Reasoning per PR:
- pr1: additions=100, deletions=10, file_count=1, additive_lines=100 (file_del=0 so all add count as additive), refactor_lines=0, ratio=0.0, totalLOC=110 → S
- pr2: additions=200, deletions=50, file_count=2, two files with mixed deletes → both refactor; refactor_lines = (150+40) + (50+10) = 250; additive=0; ratio=1.0; totalLOC=250 → M
- pr3: additions=5, deletions=2, file_count=1, mixed → refactor=7; additive=0; ratio=1.0; totalLOC=7 → XS
- pr4: additions=50, deletions=100, file_count=2; cf5 has deletions only (file_add=0, file_del=100) → since file_add==0 contributes nothing to either bucket; cf6 has additions only (file_del=0) → additive=50; refactor=0; ratio=0.0; totalLOC=150 → S

Create `backend/plugins/aimeasure/e2e/fixtures/expected_pr_change_composition.csv`:

```csv
pr_id,additions,deletions,file_count,additive_lines,refactor_lines,refactor_ratio,batch_bucket
pr1,100,10,1,100,0,0.0000,S
pr2,200,50,2,0,250,1.0000,M
pr3,5,2,1,0,7,1.0000,XS
pr4,50,100,2,50,0,0.0000,S
```

- [ ] **Step 3: Write the failing e2e test**

Create `backend/plugins/aimeasure/e2e/compute_change_composition_test.go`:

```go
/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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

func TestComputeChangeCompositionDataFlow(t *testing.T) {
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
	dataflowTester.ImportCsvIntoTabler("./fixtures/commits.csv", &code.Commit{})
	dataflowTester.ImportCsvIntoTabler("./fixtures/pull_request_commits.csv", &code.PullRequestCommit{})
	dataflowTester.ImportCsvIntoTabler("./fixtures/commit_files.csv", &code.CommitFile{})

	dataflowTester.FlushTabler(&models.PRChangeComposition{})
	dataflowTester.Subtask(tasks.ComputeChangeCompositionMeta, taskData)
	dataflowTester.VerifyTableWithOptions(&models.PRChangeComposition{}, e2ehelper.TableOptions{
		CSVRelPath:   "./fixtures/expected_pr_change_composition.csv",
		IgnoreTypes:  []interface{}{common.NoPKModel{}},
		IgnoreFields: []string{"computed_at"},
		NumericEpsilon: map[string]float64{
			"refactor_ratio": 0.0001,
		},
	})
}
```

- [ ] **Step 4: Run against both drivers (same pattern as Task 10 Steps 3-4)**

Run the Docker-based Postgres validation, then MySQL.

Expected: `--- PASS: TestComputeChangeCompositionDataFlow` on both.

- [ ] **Step 5: Commit**

```bash
git add backend/plugins/aimeasure/e2e/fixtures/commit_files.csv backend/plugins/aimeasure/e2e/fixtures/expected_pr_change_composition.csv backend/plugins/aimeasure/e2e/compute_change_composition_test.go
git commit -m "test(aimeasure): e2e for computeChangeComposition across MySQL+Postgres"
```

---

## Task 12 — E2E test for `computeQualityCohort`

**Files:**
- Create: `backend/plugins/aimeasure/e2e/fixtures/expected_pr_defect_signals.csv`
- Create: `backend/plugins/aimeasure/e2e/compute_quality_cohort_test.go`
- Modify: `backend/plugins/aimeasure/e2e/fixtures/commits.csv` (add revert and hotfix entries)
- Modify: `backend/plugins/aimeasure/e2e/fixtures/pull_requests.csv` (add hotfix PR)

- [ ] **Step 1: Extend `pull_requests.csv` to include a hotfix PR**

Replace `backend/plugins/aimeasure/e2e/fixtures/pull_requests.csv` with:

```csv
id,base_repo_id,merged_date,title,merge_commit_sha,additions,deletions,status
pr1,repo1,2026-05-01 10:00:00,feat: add widget,sha-merge-1,100,10,MERGED
pr2,repo1,2026-05-02 10:00:00,feat: AI-assisted refactor,sha-merge-2,200,50,MERGED
pr3,repo1,2026-05-03 10:00:00,fix: nil deref,sha-merge-3,5,2,MERGED
pr4,repo1,2026-05-04 10:00:00,chore: cleanup,sha-merge-4,50,100,MERGED
pr5,repo1,2026-05-05 10:00:00,hotfix: emergency revert of widget,sha-merge-5,30,10,MERGED
```

- [ ] **Step 2: Extend `commits.csv` to include a revert commit**

Replace `backend/plugins/aimeasure/e2e/fixtures/commits.csv` with:

```csv
sha,authored_date,message
sha-commit-1,2026-04-30 09:00:00,feat: add widget
sha-commit-2,2026-05-01 09:00:00,feat: refactor with Claude\n\nCo-authored-by: Claude <claude@anthropic.com>
sha-commit-3,2026-05-02 09:00:00,fix: nil deref in handler
sha-commit-4,2026-05-03 09:00:00,chore: cleanup unused vars
sha-commit-5,2026-05-04 12:00:00,Revert "feat: add widget" (sha-merge-1)
sha-commit-6,2026-05-05 09:00:00,hotfix: revert widget changes
```

- [ ] **Step 3: Extend `pull_request_commits.csv` and `commit_files.csv`**

Replace `backend/plugins/aimeasure/e2e/fixtures/pull_request_commits.csv` with:

```csv
pull_request_id,commit_sha
pr1,sha-commit-1
pr2,sha-commit-2
pr3,sha-commit-3
pr4,sha-commit-4
pr5,sha-commit-6
```

Append to `backend/plugins/aimeasure/e2e/fixtures/commit_files.csv`:

```csv
cf7,sha-commit-6,apps/api/widget.go,5,20
```

- [ ] **Step 4: Compute expected output**

Reasoning:
- pr1 (widget, merged 2026-05-01): there's a commit at 2026-05-04 referencing "sha-merge-1" → revert detected. Also pr5 ("hotfix: emergency revert of widget") merged 2026-05-05 within 14d touches `apps/api/widget.go` (cf7) which is 100% of pr1's files (just widget.go) → hotfix detected. No incident data (no `issues` table seeded). totalDefectCount=2.
- pr2 (refactor, merged 2026-05-02): no revert citing sha-merge-2; no hotfix in 14d window with ≥50% file overlap (pr5 touches widget.go but pr2 touches handler.go, handler_test.go — no overlap). totalDefectCount=0.
- pr3 (fix, merged 2026-05-03): no revert; no hotfix overlap. totalDefectCount=0.
- pr4 (chore, merged 2026-05-04): no revert; no hotfix overlap (touches cleanup.go, util.go). totalDefectCount=0.
- pr5 (hotfix, merged 2026-05-05): not itself a defect target since no later revert or hotfix overlaps. totalDefectCount=0.

Create `backend/plugins/aimeasure/e2e/fixtures/expected_pr_defect_signals.csv`:

```csv
pr_id,has_revert_14d,has_hotfix_14d,has_incident_14d,incident_data_available,total_defect_count
pr1,1,1,0,0,2
pr2,0,0,0,0,0
pr3,0,0,0,0,0
pr4,0,0,0,0,0
pr5,0,0,0,0,0
```

- [ ] **Step 5: Write the failing e2e test**

Create `backend/plugins/aimeasure/e2e/compute_quality_cohort_test.go`:

```go
/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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

func TestComputeQualityCohortDataFlow(t *testing.T) {
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
	dataflowTester.ImportCsvIntoTabler("./fixtures/commits.csv", &code.Commit{})
	dataflowTester.ImportCsvIntoTabler("./fixtures/pull_request_commits.csv", &code.PullRequestCommit{})
	dataflowTester.ImportCsvIntoTabler("./fixtures/commit_files.csv", &code.CommitFile{})

	dataflowTester.FlushTabler(&models.PRDefectSignals{})
	dataflowTester.Subtask(tasks.ComputeQualityCohortMeta, taskData)
	dataflowTester.VerifyTableWithOptions(&models.PRDefectSignals{}, e2ehelper.TableOptions{
		CSVRelPath:   "./fixtures/expected_pr_defect_signals.csv",
		IgnoreTypes:  []interface{}{common.NoPKModel{}},
		IgnoreFields: []string{"window_close_date", "computed_at"},
	})
}
```

- [ ] **Step 6: Run against both drivers**

Run the Postgres and MySQL Docker validations (same pattern as Task 10 Steps 3-4).

Expected: `--- PASS: TestComputeQualityCohortDataFlow` on both.

- [ ] **Step 7: Commit**

```bash
git add backend/plugins/aimeasure/e2e/
git commit -m "test(aimeasure): e2e for computeQualityCohort across MySQL+Postgres"
```

---

## Task 13 — Grafana dashboard `AIQualityCohort.json`

**Files:**
- Create: `grafana/dashboards/AIQualityCohort.json`

- [ ] **Step 1: Copy an existing dashboard as a starting template**

```bash
cp /home/ubuntu/workspaces/finally/incubator-devlake/grafana/dashboards/DORA.json /home/ubuntu/workspaces/finally/incubator-devlake/grafana/dashboards/AIQualityCohort.json
```

- [ ] **Step 2: Edit the new file**

Replace the contents of `grafana/dashboards/AIQualityCohort.json` with a dashboard that contains the five panels specified in the design. Use the same datasource UID and templating conventions as existing dashboards. The JSON should include these five panel definitions:

1. **Panel 1 — Defect rate over time, by cohort** (timeseries)

SQL (templated for both MySQL and Postgres via the existing `${__from}` / `${__to}` Grafana variables):
```sql
SELECT
  DATE_TRUNC('week', pr.merged_date) AS time,
  c.ai_cohort,
  AVG(CASE WHEN d.total_defect_count > 0 THEN 1.0 ELSE 0.0 END) * 100 AS defect_rate_pct
FROM pull_requests pr
JOIN pr_ai_cohort c ON c.pr_id = pr.id
LEFT JOIN pr_defect_signals d ON d.pr_id = pr.id
WHERE pr.merged_date >= $__timeFrom() AND pr.merged_date < $__timeTo()
GROUP BY DATE_TRUNC('week', pr.merged_date), c.ai_cohort
ORDER BY time
```

2. **Panel 2 — Batch-size distribution by cohort** (stacked bar)

```sql
SELECT
  c.ai_cohort,
  comp.batch_bucket,
  COUNT(*) AS pr_count
FROM pr_change_composition comp
JOIN pr_ai_cohort c ON c.pr_id = comp.pr_id
JOIN pull_requests pr ON pr.id = comp.pr_id
WHERE pr.merged_date >= $__timeFrom() AND pr.merged_date < $__timeTo()
GROUP BY c.ai_cohort, comp.batch_bucket
```

3. **Panel 3 — Refactor ratio over time, by cohort** (timeseries)

```sql
SELECT
  DATE_TRUNC('week', pr.merged_date) AS time,
  c.ai_cohort,
  AVG(comp.refactor_ratio) AS avg_refactor_ratio
FROM pull_requests pr
JOIN pr_ai_cohort c ON c.pr_id = pr.id
JOIN pr_change_composition comp ON comp.pr_id = pr.id
WHERE pr.merged_date >= $__timeFrom() AND pr.merged_date < $__timeTo()
GROUP BY DATE_TRUNC('week', pr.merged_date), c.ai_cohort
ORDER BY time
```

4. **Panel 4 — Top engineers by 14-day defect rate** (table)

```sql
SELECT
  pr.author_id AS engineer,
  COUNT(*) AS total_prs,
  SUM(CASE WHEN d.total_defect_count > 0 THEN 1 ELSE 0 END) AS prs_with_defect,
  ROUND(SUM(CASE WHEN d.total_defect_count > 0 THEN 1 ELSE 0 END) * 100.0 / COUNT(*), 2) AS defect_rate_pct
FROM pull_requests pr
JOIN pr_defect_signals d ON d.pr_id = pr.id
WHERE pr.merged_date >= $__timeFrom() AND pr.merged_date < $__timeTo()
GROUP BY pr.author_id
HAVING COUNT(*) >= 5
ORDER BY defect_rate_pct DESC
LIMIT 20
```

5. **Panel 5 — Recent flagged PRs** (table)

```sql
SELECT
  pr.id, pr.title, pr.author_id, pr.merged_date,
  c.ai_cohort, comp.batch_bucket, d.total_defect_count
FROM pull_requests pr
JOIN pr_ai_cohort c ON c.pr_id = pr.id
JOIN pr_change_composition comp ON comp.pr_id = pr.id
LEFT JOIN pr_defect_signals d ON d.pr_id = pr.id
WHERE pr.merged_date >= $__timeFrom() AND pr.merged_date < $__timeTo()
  AND (c.ai_cohort = 'HIGH' AND (d.total_defect_count > 0 OR comp.batch_bucket = 'XL'))
ORDER BY pr.merged_date DESC
LIMIT 25
```

Use the same panel JSON structure as `DORA.json` for each panel. Set `datasource` to the project's default and templating variables to `$__timeFrom()` / `$__timeTo()` exactly as shown.

- [ ] **Step 3: Run the dashboard JSON lint workflow check locally**

```bash
docker run --rm -v "/home/ubuntu/workspaces/finally/incubator-devlake:/workspace" -w /workspace mericodev/lake-builder:latest sh -c '
node -e "JSON.parse(require(\"fs\").readFileSync(\"grafana/dashboards/AIQualityCohort.json\"))" && echo "JSON valid"
'
```

Expected: `JSON valid`.

- [ ] **Step 4: Commit**

```bash
git add grafana/dashboards/AIQualityCohort.json
git commit -m "feat(aimeasure): add AIQualityCohort Grafana dashboard"
```

---

## Task 14 — Plugin README + open-decisions tracker

**Files:**
- Create: `backend/plugins/aimeasure/README.md`

- [ ] **Step 1: Write the README**

Create `backend/plugins/aimeasure/README.md`:

````markdown
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
````

- [ ] **Step 2: Commit**

```bash
git add backend/plugins/aimeasure/README.md
git commit -m "docs(aimeasure): plugin README"
```

---

## Final integration check

- [ ] **Step 1: Full unit test sweep**

```bash
docker run --rm -v "/home/ubuntu/workspaces/finally/incubator-devlake:/workspace" -w /workspace/backend mericodev/lake-builder:latest sh -c '
curl -sL https://github.com/vektra/mockery/releases/download/v2.53.5/mockery_2.53.5_Linux_x86_64.tar.gz | tar -xz -C /go/bin mockery
chmod +x /go/bin/mockery
make mock 2>&1 >/dev/null
go test ./plugins/aimeasure/... -v 2>&1 | tail -30
'
```

Expected: all unit tests pass; the Test_GetPluginTablesInfo from the parent plugins/ package can be re-run separately to confirm registration.

- [ ] **Step 2: Full lint sweep**

```bash
docker run --rm -v "/home/ubuntu/workspaces/finally/incubator-devlake:/workspace" -w /workspace/backend mericodev/lake-builder:latest sh -c '
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.53.3 2>&1 | tail -2
golangci-lint run ./plugins/aimeasure/... 2>&1
'
```

Expected: empty output (no lint findings).

- [ ] **Step 3: Branch + PR**

```bash
git checkout -b feat/aimeasure-phase-a
git push -u origin feat/aimeasure-phase-a
gh pr create -R zvika-finally/incubator-devlake -B main -H feat/aimeasure-phase-a \
  --title "feat(aimeasure): Phase A — AI-cohort classifier + quality cohorting" \
  --body "Implements Phase A of docs/superpowers/specs/2026-05-13-ai-era-signals-design.md. See docs/superpowers/plans/2026-05-13-aimeasure-phase-a.md for the task-by-task plan."
```

Expected: PR created, CI runs.

---

## Out of scope for Phase A

Defer to subsequent plans:

- **Phase B plan** (verification effort, Slack signals, sentiment proxy) — write when Phase A ships
- **Phase C plan** (throughput cohort, cost-per-outcome, ROI rollup) — write when Phase B ships
- Real-time alerting / Slack incident bot
- Survey ingestion for DXI
- Multi-tenant separation

## Spec coverage check

Verifying every Phase A requirement from `docs/superpowers/specs/2026-05-13-ai-era-signals-design.md` § 5 is covered:

| Spec § | Requirement | Task |
|---|---|---|
| 5.1 | AI-cohort classifier (4-level) | Task 6 (classifyPRCohort + pure Classify function) |
| 5.1 | `pr_ai_cohort` table | Task 2 + Task 3 |
| 5.1 | Commit trailer detection | Task 6 (HasAITrailer) |
| 5.2 | `pr_defect_signals` table | Task 2 + Task 3 |
| 5.2 | `pr_change_composition` table | Task 2 + Task 3 |
| 5.2 | Revert detection | Task 8 (detectRevert) |
| 5.2 | Hotfix detection (regex + file overlap) | Task 8 (detectHotfix, FileOverlapRatio) |
| 5.2 | Incident detection (conditional on data availability) | Task 8 (detectIncident + tableExists guard) |
| 5.2 | Batch buckets XS/S/M/L/XL | Task 7 (BucketFor) |
| 5.2 | Refactor ratio | Task 7 (ComputeRefactorRatio) |
| 5.3 | Three subtasks registered | Task 9 |
| 5.4 | `AIQualityCohort.json` dashboard with 5 panels | Task 13 |
| 4.3 | `aimeasure_account_overrides` table | Task 2 + Task 3 |
| 4.3 | `aimeasure_engineer_roles` table | Task 2 + Task 3 |
| 5.7 | Open decisions documented | Task 14 (README) |
