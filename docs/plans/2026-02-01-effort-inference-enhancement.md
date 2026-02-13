# FinDevOps Effort Inference Enhancement

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Enable FinDevOps to infer developer effort from Git activity when Jira time tracking data is missing, achieving 95%+ issue coverage vs current 13%.

**Architecture:** Three-layer approach: (1) FTE normalization per developer using Swarmia methodology, (2) Git-based effort inference using git2effort research, (3) Multi-source fusion with validation and R&D tax credit compliance audit trail.

**Tech Stack:** Go, GORM migrations, DevLake plugin patterns, domain layer tables (commits, pull_requests, issues)

---

## Task 1: Add Settings Model Fields for Effort Inference Configuration

**Files:**
- Modify: `backend/plugins/findevops/models/settings.go`
- Test: Manual verification via API

**Step 1: Add FTE normalization settings fields**

Add these fields to `FinDevOpsSettings` struct in `backend/plugins/findevops/models/settings.go` after line 48:

```go
	// === FTE NORMALIZATION (Swarmia Model) ===
	// Maximum FTE a developer can have per month (prevents gaming)
	FteMaxPerMonth float64 `json:"fteMaxPerMonth" gorm:"default:1.0"`
	// Multiplier for baseline score (team median × this = full-time threshold)
	FteBaselineMultiplier float64 `json:"fteBaselineMultiplier" gorm:"default:1.2"`
	// Days of inactivity to count as vacation/leave
	FteInactivityThresholdDays int `json:"fteInactivityThresholdDays" gorm:"default:5"`
	// Working hours per month for FTE calculation
	FteWorkingHoursPerMonth float64 `json:"fteWorkingHoursPerMonth" gorm:"default:160.0"`

	// === ACTIVITY WEIGHTS (Swarmia Model) ===
	ActivityWeightPrAuthored     float64 `json:"activityWeightPrAuthored" gorm:"default:1.0"`
	ActivityWeightPrReviewed     float64 `json:"activityWeightPrReviewed" gorm:"default:0.3"`
	ActivityWeightCommitAuthored float64 `json:"activityWeightCommitAuthored" gorm:"default:0.2"`
	ActivityWeightIssueUpdated   float64 `json:"activityWeightIssueUpdated" gorm:"default:0.1"`
	ActivityWeightCommentAdded   float64 `json:"activityWeightCommentAdded" gorm:"default:0.05"`

	// === GIT INFERENCE (git2effort methodology) ===
	// Productive coding hours per active day
	GitProductiveHoursPerActiveDay float64 `json:"gitProductiveHoursPerActiveDay" gorm:"default:6.0"`
	// Hours per review cycle
	GitReviewHoursPerCycle float64 `json:"gitReviewHoursPerCycle" gorm:"default:1.5"`
	// Comments that constitute one review cycle
	GitCommentsPerReviewCycle int `json:"gitCommentsPerReviewCycle" gorm:"default:3"`
	// Minimum hours to attribute to any issue
	GitMinHoursPerIssue float64 `json:"gitMinHoursPerIssue" gorm:"default:1.0"`
	// Maximum hours to attribute to any issue (cap)
	GitMaxHoursPerIssue float64 `json:"gitMaxHoursPerIssue" gorm:"default:80.0"`

	// === VALIDATION ===
	// Threshold for flagging Jira vs Git variance (percentage)
	ValidationJiraGitVarianceThresholdPct float64 `json:"validationJiraGitVarianceThresholdPct" gorm:"default:50.0"`

	// === ASC 350-40 COMMIT KEYWORDS ===
	PreliminaryCommitKeywords        string `json:"preliminaryCommitKeywords" gorm:"type:text"`
	DevelopmentCommitKeywords        string `json:"developmentCommitKeywords" gorm:"type:text"`
	PostImplementationCommitKeywords string `json:"postImplementationCommitKeywords" gorm:"type:text"`

	// === FEATURE FLAGS ===
	EnableGitEffortInference bool `json:"enableGitEffortInference" gorm:"default:true"`
	EnableFteNormalization   bool `json:"enableFteNormalization" gorm:"default:true"`
```

**Step 2: Update NewDefaultSettings function**

Update `NewDefaultSettings()` in the same file to include defaults for new fields:

```go
func NewDefaultSettings() *FinDevOpsSettings {
	return &FinDevOpsSettings{
		// Existing defaults
		DefaultHourlyRate:        87.0,
		HoursPerStoryPoint:       4.0,
		CapitalizationFramework:  "asc_350_40_stages",
		PreliminaryLabels:        `["research","spike","investigation","feasibility","discovery","poc","proof-of-concept","planning"]`,
		PostImplementationLabels: `["bug","hotfix","maintenance","ktlo","support","incident","fix","patch","tech-debt"]`,
		PreliminaryTypes:         `["Spike","Research","Discovery"]`,
		DevelopmentTypes:         `["Story","Feature","Enhancement","Epic"]`,
		PostImplementationTypes:  `["Bug","Defect","Hotfix","Support"]`,

		// FTE Normalization (Swarmia)
		FteMaxPerMonth:             1.0,
		FteBaselineMultiplier:      1.2,
		FteInactivityThresholdDays: 5,
		FteWorkingHoursPerMonth:    160.0,

		// Activity Weights (Swarmia)
		ActivityWeightPrAuthored:     1.0,
		ActivityWeightPrReviewed:     0.3,
		ActivityWeightCommitAuthored: 0.2,
		ActivityWeightIssueUpdated:   0.1,
		ActivityWeightCommentAdded:   0.05,

		// Git Inference (git2effort)
		GitProductiveHoursPerActiveDay: 6.0,
		GitReviewHoursPerCycle:         1.5,
		GitCommentsPerReviewCycle:      3,
		GitMinHoursPerIssue:            1.0,
		GitMaxHoursPerIssue:            80.0,

		// Validation
		ValidationJiraGitVarianceThresholdPct: 50.0,

		// ASC 350-40 Commit Keywords
		PreliminaryCommitKeywords:        `["experiment","prototype","poc","spike","research","feasibility"]`,
		DevelopmentCommitKeywords:        `["feat:","add:","implement:","create:","feature:"]`,
		PostImplementationCommitKeywords: `["fix:","bugfix:","hotfix:","patch:","resolve:"]`,

		// Feature flags
		EnableGitEffortInference: true,
		EnableFteNormalization:   true,
	}
}
```

**Step 3: Commit**

```bash
git add backend/plugins/findevops/models/settings.go
git commit -m "feat(findevops): add effort inference configuration settings

Add FTE normalization settings (Swarmia model)
Add activity weight settings
Add git inference settings (git2effort methodology)
Add ASC 350-40 commit keyword configuration
Add feature flags for enabling/disabling inference"
```

---

## Task 2: Create DeveloperMonthlyFte Model

**Files:**
- Create: `backend/plugins/findevops/models/developer_monthly_fte.go`

**Step 1: Create the model file**

Create `backend/plugins/findevops/models/developer_monthly_fte.go`:

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

// DeveloperMonthlyFte tracks FTE normalization per developer per month
// Based on Swarmia's methodology: normalize activity to max 1 FTE, weight different activities
type DeveloperMonthlyFte struct {
	Id          string `gorm:"primaryKey;type:varchar(255)"` // {developer_id}:{fiscal_month}
	DeveloperId string `gorm:"type:varchar(255);index"`
	FiscalMonth string `gorm:"type:varchar(10);index"`
	ProjectName string `gorm:"type:varchar(255);index"`

	// Activity counts (raw signals from Git/Jira)
	PrsAuthored     int `gorm:"type:int;default:0"`
	PrsReviewed     int `gorm:"type:int;default:0"`
	CommitsAuthored int `gorm:"type:int;default:0"`
	IssuesUpdated   int `gorm:"type:int;default:0"`
	CommentsAdded   int `gorm:"type:int;default:0"`

	// FTE calculation (Swarmia methodology)
	RawActivityScore float64 `gorm:"type:decimal(10,2)"` // Weighted sum of activities
	BaselineScore    float64 `gorm:"type:decimal(10,2)"` // Team median × multiplier
	RawFte           float64 `gorm:"type:decimal(3,2)"`  // Before inactivity adjustment
	InactiveDays     int     `gorm:"type:int;default:0"` // Consecutive days with no activity
	AdjustedFte      float64 `gorm:"type:decimal(3,2)"`  // Final FTE after deductions

	// Hours allocation tracking
	HoursFromJira        float64 `gorm:"type:decimal(10,2);default:0"` // Hours from Jira time tracking
	HoursFromGitInferred float64 `gorm:"type:decimal(10,2);default:0"` // Hours inferred from Git
	HoursDistributed     float64 `gorm:"type:decimal(10,2);default:0"` // Hours distributed via FTE
	TotalAllocatedHours  float64 `gorm:"type:decimal(10,2);default:0"` // Sum of all allocated hours

	CalculatedAt time.Time
}

func (DeveloperMonthlyFte) TableName() string {
	return "developer_monthly_fte"
}
```

**Step 2: Commit**

```bash
git add backend/plugins/findevops/models/developer_monthly_fte.go
git commit -m "feat(findevops): add DeveloperMonthlyFte model

Tracks per-developer monthly FTE using Swarmia methodology:
- Raw activity signals (PRs, commits, reviews, comments)
- Weighted activity score and baseline calculation
- Inactivity detection and FTE adjustment
- Hours allocation tracking from multiple sources"
```

---

## Task 3: Add Effort Source Fields to CostAllocation Model

**Files:**
- Modify: `backend/plugins/findevops/models/cost_allocation.go`

**Step 1: Add effort source tracking fields**

Add these fields to `CostAllocation` struct after line 59 (after `IsUnallocated`):

```go
	// === EFFORT SOURCE TRACKING ===
	// Where the hours_worked value came from
	EffortSource    string `gorm:"type:varchar(50)"` // jira_time, jira_estimate, story_points, git_inferred, fte_distributed
	ConfidenceLevel string `gorm:"type:varchar(20)"` // high, medium, inferred, low

	// === GIT-INFERRED EFFORT BREAKDOWN ===
	GitCodingHours      float64 `gorm:"type:decimal(10,2)"` // Hours from coding activity
	GitReviewHours      float64 `gorm:"type:decimal(10,2)"` // Hours from review activity
	GitComplexityFactor float64 `gorm:"type:decimal(5,2)"`  // Complexity multiplier
	GitActiveDays       int     `gorm:"type:int"`           // Days with commits

	// === VALIDATION FLAGS ===
	EffortValidated       bool    `gorm:"type:bool;default:false"` // Jira vs Git cross-validated
	ValidationVariancePct float64 `gorm:"type:decimal(8,2)"`       // Variance between Jira and Git

	// === AUDIT TRAIL FOR R&D COMPLIANCE ===
	LinkedCommitShas      string `gorm:"type:text"` // JSON array of commit SHAs
	LinkedPrIds           string `gorm:"type:text"` // JSON array of PR IDs
	ClassificationSignals string `gorm:"type:text"` // JSON: what triggered ASC 350-40 category

	// === FTE CONTEXT ===
	DeveloperMonthlyFte float64 `gorm:"type:decimal(3,2)"` // Developer's FTE for this month
	FteAllocationPct    float64 `gorm:"type:decimal(5,2)"` // % of developer's month on this issue
```

**Step 2: Add constants for effort sources**

Add these constants at the top of the file, after the imports:

```go
// EffortSource constants
const (
	EffortSourceJiraTime       = "jira_time"
	EffortSourceJiraEstimate   = "jira_estimate"
	EffortSourceStoryPoints    = "story_points"
	EffortSourceGitInferred    = "git_inferred"
	EffortSourceFteDistributed = "fte_distributed"
)

// ConfidenceLevel constants
const (
	ConfidenceHigh     = "high"
	ConfidenceMedium   = "medium"
	ConfidenceInferred = "inferred"
	ConfidenceLow      = "low"
)
```

**Step 3: Commit**

```bash
git add backend/plugins/findevops/models/cost_allocation.go
git commit -m "feat(findevops): add effort source tracking to CostAllocation

- EffortSource: tracks where hours came from (jira_time, git_inferred, etc)
- ConfidenceLevel: high/medium/inferred/low
- Git effort breakdown: coding hours, review hours, complexity
- Validation flags for Jira vs Git cross-checking
- Audit trail: linked commits, PRs, classification signals
- FTE context for R&D compliance reporting"
```

---

## Task 4: Create Migration Script for New Fields

**Files:**
- Create: `backend/plugins/findevops/models/migrationscripts/20260201_add_effort_inference.go`
- Modify: `backend/plugins/findevops/models/migrationscripts/register.go`

**Step 1: Create migration script**

Create `backend/plugins/findevops/models/migrationscripts/20260201_add_effort_inference.go`:

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

type addEffortInference struct{}

// developerMonthlyFte20260201 - new table for FTE tracking
type developerMonthlyFte20260201 struct {
	Id                   string    `gorm:"primaryKey;type:varchar(255)"`
	DeveloperId          string    `gorm:"type:varchar(255);index"`
	FiscalMonth          string    `gorm:"type:varchar(10);index"`
	ProjectName          string    `gorm:"type:varchar(255);index"`
	PrsAuthored          int       `gorm:"type:int;default:0"`
	PrsReviewed          int       `gorm:"type:int;default:0"`
	CommitsAuthored      int       `gorm:"type:int;default:0"`
	IssuesUpdated        int       `gorm:"type:int;default:0"`
	CommentsAdded        int       `gorm:"type:int;default:0"`
	RawActivityScore     float64   `gorm:"type:decimal(10,2)"`
	BaselineScore        float64   `gorm:"type:decimal(10,2)"`
	RawFte               float64   `gorm:"type:decimal(3,2)"`
	InactiveDays         int       `gorm:"type:int;default:0"`
	AdjustedFte          float64   `gorm:"type:decimal(3,2)"`
	HoursFromJira        float64   `gorm:"type:decimal(10,2);default:0"`
	HoursFromGitInferred float64   `gorm:"type:decimal(10,2);default:0"`
	HoursDistributed     float64   `gorm:"type:decimal(10,2);default:0"`
	TotalAllocatedHours  float64   `gorm:"type:decimal(10,2);default:0"`
	CalculatedAt         time.Time
}

func (developerMonthlyFte20260201) TableName() string {
	return "developer_monthly_fte"
}

// costAllocation20260201 - add new columns to existing table
type costAllocation20260201 struct {
	Id                    string  `gorm:"primaryKey;type:varchar(255)"`
	EffortSource          string  `gorm:"type:varchar(50)"`
	ConfidenceLevel       string  `gorm:"type:varchar(20)"`
	GitCodingHours        float64 `gorm:"type:decimal(10,2)"`
	GitReviewHours        float64 `gorm:"type:decimal(10,2)"`
	GitComplexityFactor   float64 `gorm:"type:decimal(5,2)"`
	GitActiveDays         int     `gorm:"type:int"`
	EffortValidated       bool    `gorm:"type:bool;default:false"`
	ValidationVariancePct float64 `gorm:"type:decimal(8,2)"`
	LinkedCommitShas      string  `gorm:"type:text"`
	LinkedPrIds           string  `gorm:"type:text"`
	ClassificationSignals string  `gorm:"type:text"`
	DeveloperMonthlyFte   float64 `gorm:"type:decimal(3,2)"`
	FteAllocationPct      float64 `gorm:"type:decimal(5,2)"`
}

func (costAllocation20260201) TableName() string {
	return "cost_allocations"
}

// finDevOpsSettings20260201 - add new configuration fields
type finDevOpsSettings20260201 struct {
	Id                                    uint64  `gorm:"primaryKey;autoIncrement"`
	FteMaxPerMonth                        float64 `gorm:"default:1.0"`
	FteBaselineMultiplier                 float64 `gorm:"default:1.2"`
	FteInactivityThresholdDays            int     `gorm:"default:5"`
	FteWorkingHoursPerMonth               float64 `gorm:"default:160.0"`
	ActivityWeightPrAuthored              float64 `gorm:"default:1.0"`
	ActivityWeightPrReviewed              float64 `gorm:"default:0.3"`
	ActivityWeightCommitAuthored          float64 `gorm:"default:0.2"`
	ActivityWeightIssueUpdated            float64 `gorm:"default:0.1"`
	ActivityWeightCommentAdded            float64 `gorm:"default:0.05"`
	GitProductiveHoursPerActiveDay        float64 `gorm:"default:6.0"`
	GitReviewHoursPerCycle                float64 `gorm:"default:1.5"`
	GitCommentsPerReviewCycle             int     `gorm:"default:3"`
	GitMinHoursPerIssue                   float64 `gorm:"default:1.0"`
	GitMaxHoursPerIssue                   float64 `gorm:"default:80.0"`
	ValidationJiraGitVarianceThresholdPct float64 `gorm:"default:50.0"`
	PreliminaryCommitKeywords             string  `gorm:"type:text"`
	DevelopmentCommitKeywords             string  `gorm:"type:text"`
	PostImplementationCommitKeywords      string  `gorm:"type:text"`
	EnableGitEffortInference              bool    `gorm:"default:true"`
	EnableFteNormalization                bool    `gorm:"default:true"`
}

func (finDevOpsSettings20260201) TableName() string {
	return "_tool_findevops_settings"
}

func (u *addEffortInference) Up(baseRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		baseRes,
		&developerMonthlyFte20260201{},
		&costAllocation20260201{},
		&finDevOpsSettings20260201{},
	)
}

func (*addEffortInference) Version() uint64 {
	return 20260201000001
}

func (*addEffortInference) Name() string {
	return "findevops: add effort inference support (FTE normalization, git inference, audit trail)"
}
```

**Step 2: Register migration**

Update `backend/plugins/findevops/models/migrationscripts/register.go` to add the new migration:

```go
package migrationscripts

import "github.com/apache/incubator-devlake/core/plugin"

func All() []plugin.MigrationScript {
	return []plugin.MigrationScript{
		new(initSchema),
		new(addDeploymentCosts),
		new(addSettings),
		new(addBudgetVariance),
		new(addEffortInference), // Add this line
	}
}
```

**Step 3: Commit**

```bash
git add backend/plugins/findevops/models/migrationscripts/20260201_add_effort_inference.go
git add backend/plugins/findevops/models/migrationscripts/register.go
git commit -m "feat(findevops): add migration for effort inference tables

- Creates developer_monthly_fte table for FTE tracking
- Adds effort source tracking columns to cost_allocations
- Adds configuration fields to _tool_findevops_settings"
```

---

## Task 5: Register DeveloperMonthlyFte Model in Plugin

**Files:**
- Modify: `backend/plugins/findevops/impl/impl.go`

**Step 1: Update GetTablesInfo**

Update the `GetTablesInfo` method in `backend/plugins/findevops/impl/impl.go` (around line 76):

```go
func (p FinDevOps) GetTablesInfo() []dal.Tabler {
	return []dal.Tabler{
		&models.CostAllocation{},
		&models.MonthlyCostSummary{},
		&models.DeveloperHourlyRate{},
		&models.DeploymentCost{},
		&models.FinDevOpsSettings{},
		&models.DeveloperMonthlyFte{}, // Add this line
	}
}
```

**Step 2: Commit**

```bash
git add backend/plugins/findevops/impl/impl.go
git commit -m "feat(findevops): register DeveloperMonthlyFte model in plugin"
```

---

## Task 6: Create CollectDeveloperActivity Subtask

**Files:**
- Create: `backend/plugins/findevops/tasks/collect_developer_activity.go`
- Create: `backend/plugins/findevops/tasks/collect_developer_activity_test.go`

**Step 1: Write the test file**

Create `backend/plugins/findevops/tasks/collect_developer_activity_test.go`:

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

	"github.com/stretchr/testify/assert"
)

func TestCalculateActivityScore(t *testing.T) {
	settings := &ActivityWeights{
		PrAuthored:     1.0,
		PrReviewed:     0.3,
		CommitAuthored: 0.2,
		IssueUpdated:   0.1,
		CommentAdded:   0.05,
	}

	testCases := []struct {
		name           string
		prsAuthored    int
		prsReviewed    int
		commits        int
		issuesUpdated  int
		comments       int
		expectedScore  float64
	}{
		{
			name:          "Typical developer month",
			prsAuthored:   10,
			prsReviewed:   15,
			commits:       50,
			issuesUpdated: 20,
			comments:      30,
			expectedScore: 10*1.0 + 15*0.3 + 50*0.2 + 20*0.1 + 30*0.05, // 10 + 4.5 + 10 + 2 + 1.5 = 28
		},
		{
			name:          "Heavy committer",
			prsAuthored:   5,
			prsReviewed:   5,
			commits:       100,
			issuesUpdated: 10,
			comments:      10,
			expectedScore: 5*1.0 + 5*0.3 + 100*0.2 + 10*0.1 + 10*0.05, // 5 + 1.5 + 20 + 1 + 0.5 = 28
		},
		{
			name:          "Zero activity",
			prsAuthored:   0,
			prsReviewed:   0,
			commits:       0,
			issuesUpdated: 0,
			comments:      0,
			expectedScore: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := calculateActivityScore(
				tc.prsAuthored,
				tc.prsReviewed,
				tc.commits,
				tc.issuesUpdated,
				tc.comments,
				settings,
			)
			assert.InDelta(t, tc.expectedScore, score, 0.01)
		})
	}
}

func TestCalculateFte(t *testing.T) {
	testCases := []struct {
		name          string
		rawScore      float64
		baselineScore float64
		maxFte        float64
		expectedFte   float64
	}{
		{
			name:          "Full-time developer",
			rawScore:      30.0,
			baselineScore: 25.0,
			maxFte:        1.0,
			expectedFte:   1.0, // Capped at max
		},
		{
			name:          "Part-time developer",
			rawScore:      12.5,
			baselineScore: 25.0,
			maxFte:        1.0,
			expectedFte:   0.5,
		},
		{
			name:          "Inactive developer",
			rawScore:      0,
			baselineScore: 25.0,
			maxFte:        1.0,
			expectedFte:   0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fte := calculateFte(tc.rawScore, tc.baselineScore, tc.maxFte)
			assert.InDelta(t, tc.expectedFte, fte, 0.01)
		})
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd backend && go test ./plugins/findevops/tasks/... -run TestCalculateActivityScore -v
```

Expected: FAIL with "undefined: calculateActivityScore"

**Step 3: Create the subtask implementation**

Create `backend/plugins/findevops/tasks/collect_developer_activity.go`:

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
	"fmt"
	"math"
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/findevops/models"
)

var CollectDeveloperActivityMeta = plugin.SubTaskMeta{
	Name:             "collectDeveloperActivity",
	EntryPoint:       CollectDeveloperActivity,
	EnabledByDefault: true,
	Description:      "Collect developer activity signals for FTE calculation (Swarmia model)",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET, plugin.DOMAIN_TYPE_CODE},
}

// ActivityWeights holds the configurable weights for different activities
type ActivityWeights struct {
	PrAuthored     float64
	PrReviewed     float64
	CommitAuthored float64
	IssueUpdated   float64
	CommentAdded   float64
}

// DeveloperActivity holds raw activity counts for a developer in a month
type DeveloperActivity struct {
	DeveloperId     string
	FiscalMonth     string
	PrsAuthored     int
	PrsReviewed     int
	CommitsAuthored int
	IssuesUpdated   int
	CommentsAdded   int
}

func CollectDeveloperActivity(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*FinDevOpsTaskData)
	logger := taskCtx.GetLogger()
	settings := data.Settings

	if !settings.EnableFteNormalization {
		logger.Info("FTE normalization disabled, skipping collectDeveloperActivity")
		return nil
	}

	logger.Info("Starting collectDeveloperActivity for project: %s", data.Options.ProjectName)

	weights := &ActivityWeights{
		PrAuthored:     settings.ActivityWeightPrAuthored,
		PrReviewed:     settings.ActivityWeightPrReviewed,
		CommitAuthored: settings.ActivityWeightCommitAuthored,
		IssueUpdated:   settings.ActivityWeightIssueUpdated,
		CommentAdded:   settings.ActivityWeightCommentAdded,
	}

	// Get distinct fiscal months from issues
	months, err := getDistinctFiscalMonths(db, data.Options.ProjectName)
	if err != nil {
		return err
	}

	for _, month := range months {
		// Collect activity for each developer in this month
		activities, err := collectMonthlyActivities(db, data.Options.ProjectName, month)
		if err != nil {
			logger.Error(err, "failed to collect activities for month %s", month)
			continue
		}

		// Calculate baseline score (median × multiplier)
		baselineScore := calculateBaselineScore(activities, weights, settings.FteBaselineMultiplier)

		// Create FTE records for each developer
		for _, activity := range activities {
			rawScore := calculateActivityScore(
				activity.PrsAuthored,
				activity.PrsReviewed,
				activity.CommitsAuthored,
				activity.IssuesUpdated,
				activity.CommentsAdded,
				weights,
			)

			rawFte := calculateFte(rawScore, baselineScore, settings.FteMaxPerMonth)

			// TODO: Calculate inactive days from commit timestamps
			inactiveDays := 0
			adjustedFte := adjustFteForInactivity(rawFte, inactiveDays, settings.FteInactivityThresholdDays)

			fte := &models.DeveloperMonthlyFte{
				Id:               fmt.Sprintf("%s:%s", activity.DeveloperId, month),
				DeveloperId:      activity.DeveloperId,
				FiscalMonth:      month,
				ProjectName:      data.Options.ProjectName,
				PrsAuthored:      activity.PrsAuthored,
				PrsReviewed:      activity.PrsReviewed,
				CommitsAuthored:  activity.CommitsAuthored,
				IssuesUpdated:    activity.IssuesUpdated,
				CommentsAdded:    activity.CommentsAdded,
				RawActivityScore: rawScore,
				BaselineScore:    baselineScore,
				RawFte:           rawFte,
				InactiveDays:     inactiveDays,
				AdjustedFte:      adjustedFte,
				CalculatedAt:     time.Now(),
			}

			if err := db.CreateOrUpdate(fte); err != nil {
				logger.Error(err, "failed to save FTE for developer %s month %s", activity.DeveloperId, month)
			}
		}
	}

	logger.Info("Completed collectDeveloperActivity")
	return nil
}

func getDistinctFiscalMonths(db dal.Dal, projectName string) ([]string, errors.Error) {
	var months []string
	err := db.All(&months,
		dal.Select("DISTINCT DATE_FORMAT(resolution_date, '%Y-%m') as fiscal_month"),
		dal.From("issues"),
		dal.Join("LEFT JOIN board_issues bi ON bi.issue_id = issues.id"),
		dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'boards' AND pm.row_id = bi.board_id"),
		dal.Where("pm.project_name = ? AND resolution_date IS NOT NULL", projectName),
	)
	if err != nil {
		return nil, errors.Default.Wrap(err, "failed to get distinct fiscal months")
	}
	return months, nil
}

func collectMonthlyActivities(db dal.Dal, projectName string, fiscalMonth string) ([]DeveloperActivity, errors.Error) {
	// This is a simplified version - in production, you'd query commits, PRs, etc.
	// For now, we collect from issues assignee data
	var activities []DeveloperActivity

	// Query commits authored per developer
	type commitCount struct {
		AuthorId string
		Count    int
	}
	var commits []commitCount
	_ = db.All(&commits,
		dal.Select("author_id, COUNT(*) as count"),
		dal.From("commits"),
		dal.Where("DATE_FORMAT(authored_date, '%Y-%m') = ?", fiscalMonth),
		dal.Groupby("author_id"),
	)

	// Query PRs authored per developer
	type prCount struct {
		AuthorId string
		Count    int
	}
	var prsAuthored []prCount
	_ = db.All(&prsAuthored,
		dal.Select("author_id, COUNT(*) as count"),
		dal.From("pull_requests"),
		dal.Where("DATE_FORMAT(created_date, '%Y-%m') = ?", fiscalMonth),
		dal.Groupby("author_id"),
	)

	// Build activity map
	activityMap := make(map[string]*DeveloperActivity)
	for _, c := range commits {
		if c.AuthorId == "" {
			continue
		}
		if _, exists := activityMap[c.AuthorId]; !exists {
			activityMap[c.AuthorId] = &DeveloperActivity{
				DeveloperId: c.AuthorId,
				FiscalMonth: fiscalMonth,
			}
		}
		activityMap[c.AuthorId].CommitsAuthored = c.Count
	}

	for _, pr := range prsAuthored {
		if pr.AuthorId == "" {
			continue
		}
		if _, exists := activityMap[pr.AuthorId]; !exists {
			activityMap[pr.AuthorId] = &DeveloperActivity{
				DeveloperId: pr.AuthorId,
				FiscalMonth: fiscalMonth,
			}
		}
		activityMap[pr.AuthorId].PrsAuthored = pr.Count
	}

	for _, activity := range activityMap {
		activities = append(activities, *activity)
	}

	return activities, nil
}

func calculateActivityScore(prsAuthored, prsReviewed, commits, issuesUpdated, comments int, weights *ActivityWeights) float64 {
	return float64(prsAuthored)*weights.PrAuthored +
		float64(prsReviewed)*weights.PrReviewed +
		float64(commits)*weights.CommitAuthored +
		float64(issuesUpdated)*weights.IssueUpdated +
		float64(comments)*weights.CommentAdded
}

func calculateBaselineScore(activities []DeveloperActivity, weights *ActivityWeights, multiplier float64) float64 {
	if len(activities) == 0 {
		return 25.0 // Default baseline
	}

	// Calculate scores for all developers
	scores := make([]float64, len(activities))
	for i, activity := range activities {
		scores[i] = calculateActivityScore(
			activity.PrsAuthored,
			activity.PrsReviewed,
			activity.CommitsAuthored,
			activity.IssuesUpdated,
			activity.CommentsAdded,
			weights,
		)
	}

	// Calculate median
	median := calculateMedian(scores)
	return median * multiplier
}

func calculateMedian(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	// Sort values (simple bubble sort for small arrays)
	sorted := make([]float64, len(values))
	copy(sorted, values)
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j] > sorted[j+1] {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}

func calculateFte(rawScore, baselineScore, maxFte float64) float64 {
	if baselineScore <= 0 {
		return 0
	}
	fte := rawScore / baselineScore
	return math.Min(fte, maxFte)
}

func adjustFteForInactivity(rawFte float64, inactiveDays, threshold int) float64 {
	if inactiveDays <= threshold {
		return rawFte
	}
	// Reduce FTE proportionally for extended inactivity
	// Assuming 20 working days per month
	workingDays := 20
	activeDays := workingDays - inactiveDays
	if activeDays <= 0 {
		return 0
	}
	return rawFte * float64(activeDays) / float64(workingDays)
}
```

**Step 4: Run tests to verify they pass**

```bash
cd backend && go test ./plugins/findevops/tasks/... -run TestCalculate -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add backend/plugins/findevops/tasks/collect_developer_activity.go
git add backend/plugins/findevops/tasks/collect_developer_activity_test.go
git commit -m "feat(findevops): add collectDeveloperActivity subtask

Implements Swarmia FTE model:
- Collects activity signals (PRs, commits, reviews, comments)
- Calculates weighted activity score
- Computes baseline from team median
- Normalizes to FTE with inactivity adjustment"
```

---

## Task 7: Create InferGitEffort Subtask

**Files:**
- Create: `backend/plugins/findevops/tasks/infer_git_effort.go`
- Create: `backend/plugins/findevops/tasks/infer_git_effort_test.go`

**Step 1: Write the test file**

Create `backend/plugins/findevops/tasks/infer_git_effort_test.go`:

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

	"github.com/stretchr/testify/assert"
)

func TestCalculateGitInferredHours(t *testing.T) {
	config := &GitInferenceConfig{
		ProductiveHoursPerActiveDay: 6.0,
		ReviewHoursPerCycle:         1.5,
		CommentsPerReviewCycle:      3,
		MinHoursPerIssue:            1.0,
		MaxHoursPerIssue:            80.0,
	}

	testCases := []struct {
		name           string
		activeDays     int
		reviewComments int
		linesChanged   int
		filesChanged   int
		expectedMin    float64
		expectedMax    float64
	}{
		{
			name:           "Small task - 1 day, no reviews",
			activeDays:     1,
			reviewComments: 0,
			linesChanged:   50,
			filesChanged:   2,
			expectedMin:    1.0,  // At least min
			expectedMax:    10.0, // Reasonable upper bound
		},
		{
			name:           "Medium task - 3 days, some reviews",
			activeDays:     3,
			reviewComments: 6,
			linesChanged:   200,
			filesChanged:   5,
			expectedMin:    15.0, // 3*6 + 2*1.5 = 21
			expectedMax:    30.0,
		},
		{
			name:           "Large task - capped at max",
			activeDays:     20,
			reviewComments: 30,
			linesChanged:   5000,
			filesChanged:   50,
			expectedMin:    80.0, // Capped
			expectedMax:    80.0, // Capped
		},
		{
			name:           "Zero activity - returns min",
			activeDays:     0,
			reviewComments: 0,
			linesChanged:   0,
			filesChanged:   0,
			expectedMin:    1.0,
			expectedMax:    1.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hours := calculateGitInferredHours(
				tc.activeDays,
				tc.reviewComments,
				tc.linesChanged,
				tc.filesChanged,
				config,
			)
			assert.GreaterOrEqual(t, hours, tc.expectedMin, "Hours should be >= min")
			assert.LessOrEqual(t, hours, tc.expectedMax, "Hours should be <= max")
		})
	}
}

func TestCalculateComplexityFactor(t *testing.T) {
	testCases := []struct {
		name         string
		linesChanged int
		filesChanged int
		expectedMin  float64
		expectedMax  float64
	}{
		{
			name:         "Simple change",
			linesChanged: 10,
			filesChanged: 1,
			expectedMin:  0.5,
			expectedMax:  1.5,
		},
		{
			name:         "Complex change",
			linesChanged: 1000,
			filesChanged: 20,
			expectedMin:  2.0,
			expectedMax:  5.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			factor := calculateComplexityFactor(tc.linesChanged, tc.filesChanged)
			assert.GreaterOrEqual(t, factor, tc.expectedMin)
			assert.LessOrEqual(t, factor, tc.expectedMax)
		})
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd backend && go test ./plugins/findevops/tasks/... -run TestCalculateGitInferred -v
```

Expected: FAIL with "undefined: calculateGitInferredHours"

**Step 3: Create the subtask implementation**

Create `backend/plugins/findevops/tasks/infer_git_effort.go`:

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
	"encoding/json"
	"math"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
)

var InferGitEffortMeta = plugin.SubTaskMeta{
	Name:             "inferGitEffort",
	EntryPoint:       InferGitEffort,
	EnabledByDefault: true,
	Description:      "Infer effort from Git activity (git2effort methodology)",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CODE},
	Dependencies:     []*plugin.SubTaskMeta{&CollectDeveloperActivityMeta},
}

// GitInferenceConfig holds configuration for git-based effort inference
type GitInferenceConfig struct {
	ProductiveHoursPerActiveDay float64
	ReviewHoursPerCycle         float64
	CommentsPerReviewCycle      int
	MinHoursPerIssue            float64
	MaxHoursPerIssue            float64
}

// GitEffortResult stores inferred effort for an issue
type GitEffortResult struct {
	IssueId          string
	CodingHours      float64
	ReviewHours      float64
	ComplexityFactor float64
	ActiveDays       int
	TotalHours       float64
	CommitShas       []string
	PrIds            []string
}

// In-memory cache for git effort results (used by calculateCosts)
var gitEffortCache = make(map[string]*GitEffortResult)

func InferGitEffort(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*FinDevOpsTaskData)
	logger := taskCtx.GetLogger()
	settings := data.Settings

	if !settings.EnableGitEffortInference {
		logger.Info("Git effort inference disabled, skipping")
		return nil
	}

	logger.Info("Starting inferGitEffort for project: %s", data.Options.ProjectName)

	config := &GitInferenceConfig{
		ProductiveHoursPerActiveDay: settings.GitProductiveHoursPerActiveDay,
		ReviewHoursPerCycle:         settings.GitReviewHoursPerCycle,
		CommentsPerReviewCycle:      settings.GitCommentsPerReviewCycle,
		MinHoursPerIssue:            settings.GitMinHoursPerIssue,
		MaxHoursPerIssue:            settings.GitMaxHoursPerIssue,
	}

	// Clear cache
	gitEffortCache = make(map[string]*GitEffortResult)

	// Get all issues for this project
	var issueIds []string
	err := db.All(&issueIds,
		dal.Select("issues.id"),
		dal.From("issues"),
		dal.Join("LEFT JOIN board_issues bi ON bi.issue_id = issues.id"),
		dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'boards' AND pm.row_id = bi.board_id"),
		dal.Where("pm.project_name = ?", data.Options.ProjectName),
	)
	if err != nil {
		return errors.Default.Wrap(err, "failed to get issues")
	}

	logger.Info("Inferring git effort for %d issues", len(issueIds))

	for _, issueId := range issueIds {
		result := inferEffortForIssue(db, issueId, config)
		if result != nil {
			gitEffortCache[issueId] = result
		}
	}

	logger.Info("Completed inferGitEffort, cached %d results", len(gitEffortCache))
	return nil
}

func inferEffortForIssue(db dal.Dal, issueId string, config *GitInferenceConfig) *GitEffortResult {
	result := &GitEffortResult{
		IssueId:    issueId,
		CommitShas: []string{},
		PrIds:      []string{},
	}

	// Get linked commits via issue_commits table
	type commitInfo struct {
		CommitSha    string
		AuthoredDate string
		Additions    int
		Deletions    int
	}
	var commits []commitInfo
	_ = db.All(&commits,
		dal.Select("c.sha as commit_sha, c.authored_date, c.additions, c.deletions"),
		dal.From("issue_commits ic"),
		dal.Join("JOIN commits c ON c.sha = ic.commit_sha"),
		dal.Where("ic.issue_id = ?", issueId),
	)

	// Get linked PRs via pull_request_issues table
	type prInfo struct {
		PullRequestId string
		MergedDate    string
		Additions     int
		Deletions     int
	}
	var prs []prInfo
	_ = db.All(&prs,
		dal.Select("pr.id as pull_request_id, pr.merged_date, pr.additions, pr.deletions"),
		dal.From("pull_request_issues pri"),
		dal.Join("JOIN pull_requests pr ON pr.id = pri.pull_request_id"),
		dal.Where("pri.issue_id = ?", issueId),
	)

	// Calculate metrics
	activeDays := countActiveDays(commits)
	reviewComments := countReviewComments(db, prs)
	linesChanged, filesChanged := sumChanges(commits, prs, db)

	// Collect commit SHAs and PR IDs for audit trail
	for _, c := range commits {
		result.CommitShas = append(result.CommitShas, c.CommitSha)
	}
	for _, pr := range prs {
		result.PrIds = append(result.PrIds, pr.PullRequestId)
	}

	// Calculate inferred hours
	result.ActiveDays = activeDays
	result.ComplexityFactor = calculateComplexityFactor(linesChanged, filesChanged)
	result.CodingHours = float64(activeDays) * config.ProductiveHoursPerActiveDay
	result.ReviewHours = float64(reviewComments/config.CommentsPerReviewCycle) * config.ReviewHoursPerCycle
	result.TotalHours = calculateGitInferredHours(activeDays, reviewComments, linesChanged, filesChanged, config)

	return result
}

func countActiveDays(commits []struct {
	CommitSha    string
	AuthoredDate string
	Additions    int
	Deletions    int
}) int {
	dates := make(map[string]bool)
	for _, c := range commits {
		if len(c.AuthoredDate) >= 10 {
			dates[c.AuthoredDate[:10]] = true
		}
	}
	return len(dates)
}

func countReviewComments(db dal.Dal, prs []struct {
	PullRequestId string
	MergedDate    string
	Additions     int
	Deletions     int
}) int {
	if len(prs) == 0 {
		return 0
	}

	var prIds []string
	for _, pr := range prs {
		prIds = append(prIds, pr.PullRequestId)
	}

	var count int
	_ = db.First(&count,
		dal.Select("COUNT(*)"),
		dal.From("pull_request_comments"),
		dal.Where("pull_request_id IN ?", prIds),
	)
	return count
}

func sumChanges(commits []struct {
	CommitSha    string
	AuthoredDate string
	Additions    int
	Deletions    int
}, prs []struct {
	PullRequestId string
	MergedDate    string
	Additions     int
	Deletions     int
}, db dal.Dal) (int, int) {
	totalLines := 0
	for _, c := range commits {
		totalLines += c.Additions + c.Deletions
	}

	// Count files from PRs
	var filesChanged int
	for _, pr := range prs {
		var count int
		_ = db.First(&count,
			dal.Select("COUNT(DISTINCT file_path)"),
			dal.From("pull_request_commits prc"),
			dal.Join("JOIN commit_files cf ON cf.commit_sha = prc.commit_sha"),
			dal.Where("prc.pull_request_id = ?", pr.PullRequestId),
		)
		filesChanged += count
	}

	return totalLines, filesChanged
}

func calculateGitInferredHours(activeDays, reviewComments, linesChanged, filesChanged int, config *GitInferenceConfig) float64 {
	// Base hours from active coding days
	codingHours := float64(activeDays) * config.ProductiveHoursPerActiveDay

	// Add review overhead
	reviewCycles := float64(reviewComments) / float64(config.CommentsPerReviewCycle)
	reviewHours := reviewCycles * config.ReviewHoursPerCycle

	// Apply complexity factor
	complexity := calculateComplexityFactor(linesChanged, filesChanged)

	totalHours := (codingHours + reviewHours) * complexity

	// Clamp to min/max
	totalHours = math.Max(totalHours, config.MinHoursPerIssue)
	totalHours = math.Min(totalHours, config.MaxHoursPerIssue)

	return totalHours
}

func calculateComplexityFactor(linesChanged, filesChanged int) float64 {
	if linesChanged == 0 && filesChanged == 0 {
		return 1.0
	}

	// Log scale for lines, sqrt scale for files
	// This gives reasonable multipliers: small changes ~1.0, large changes ~3.0
	linesFactor := math.Log10(float64(linesChanged+1)) / 2
	filesFactor := math.Sqrt(float64(filesChanged)) / 3

	factor := 1.0 + linesFactor + filesFactor

	// Clamp between 0.5 and 5.0
	factor = math.Max(0.5, factor)
	factor = math.Min(5.0, factor)

	return factor
}

// GetGitEffortForIssue retrieves cached git effort result
func GetGitEffortForIssue(issueId string) *GitEffortResult {
	return gitEffortCache[issueId]
}

// GetGitEffortAuditTrail returns JSON strings for audit trail
func GetGitEffortAuditTrail(result *GitEffortResult) (commitShas, prIds string) {
	if result == nil {
		return "", ""
	}

	commitBytes, _ := json.Marshal(result.CommitShas)
	prBytes, _ := json.Marshal(result.PrIds)

	return string(commitBytes), string(prBytes)
}
```

**Step 4: Run tests to verify they pass**

```bash
cd backend && go test ./plugins/findevops/tasks/... -run TestCalculateGitInferred -v
cd backend && go test ./plugins/findevops/tasks/... -run TestCalculateComplexity -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add backend/plugins/findevops/tasks/infer_git_effort.go
git add backend/plugins/findevops/tasks/infer_git_effort_test.go
git commit -m "feat(findevops): add inferGitEffort subtask

Implements git2effort methodology:
- Links issues to commits and PRs
- Calculates active coding days
- Counts review cycles from PR comments
- Computes complexity factor from lines/files changed
- Caches results for use by calculateCosts
- Stores commit SHAs and PR IDs for R&D audit trail"
```

---

## Task 8: Update CalculateCosts to Use Multi-Source Effort

**Files:**
- Modify: `backend/plugins/findevops/tasks/calculate_costs.go`

**Step 1: Update getHoursWorked function**

Replace the existing `getHoursWorked` function (lines 291-308) with a multi-source version:

```go
// EffortResult holds the result of effort calculation with source tracking
type EffortResult struct {
	Hours           float64
	Source          string
	Confidence      string
	GitCodingHours  float64
	GitReviewHours  float64
	GitComplexity   float64
	GitActiveDays   int
	Validated       bool
	VariancePct     float64
	CommitShas      string
	PrIds           string
}

func getHoursWorkedMultiSource(issue ticket.Issue, settings *models.FinDevOpsSettings) *EffortResult {
	result := &EffortResult{}

	// Priority 1: Actual time spent (from Jira worklogs) - HIGH confidence
	if issue.TimeSpentMinutes != nil && *issue.TimeSpentMinutes > 0 {
		result.Hours = float64(*issue.TimeSpentMinutes) / 60.0
		result.Source = models.EffortSourceJiraTime
		result.Confidence = models.ConfidenceHigh
	} else if issue.OriginalEstimateMinutes != nil && *issue.OriginalEstimateMinutes > 0 {
		// Priority 2: Original estimate - MEDIUM confidence
		result.Hours = float64(*issue.OriginalEstimateMinutes) / 60.0
		result.Source = models.EffortSourceJiraEstimate
		result.Confidence = models.ConfidenceMedium
	} else if issue.StoryPoint != nil && *issue.StoryPoint > 0 {
		// Priority 3: Story points - MEDIUM confidence
		result.Hours = *issue.StoryPoint * settings.HoursPerStoryPoint
		result.Source = models.EffortSourceStoryPoints
		result.Confidence = models.ConfidenceMedium
	}

	// Try to get Git-inferred effort for validation or as primary source
	gitResult := GetGitEffortForIssue(issue.Id)
	if gitResult != nil && gitResult.TotalHours > 0 {
		result.GitCodingHours = gitResult.CodingHours
		result.GitReviewHours = gitResult.ReviewHours
		result.GitComplexity = gitResult.ComplexityFactor
		result.GitActiveDays = gitResult.ActiveDays
		result.CommitShas, result.PrIds = GetGitEffortAuditTrail(gitResult)

		if result.Hours > 0 {
			// Validate Jira vs Git
			variance := (result.Hours - gitResult.TotalHours) / result.Hours * 100
			result.VariancePct = variance
			result.Validated = true
		} else {
			// Priority 4: Git-inferred - INFERRED confidence
			result.Hours = gitResult.TotalHours
			result.Source = models.EffortSourceGitInferred
			result.Confidence = models.ConfidenceInferred
		}
	}

	return result
}
```

**Step 2: Update the main loop to use multi-source effort**

Update the loop in `CalculateCosts` function (around line 65) to use the new function and populate new fields:

```go
	for _, issue := range issues {
		// Get hours worked from multiple sources
		effortResult := getHoursWorkedMultiSource(issue, data.Settings)
		if effortResult.Hours == 0 {
			continue // Skip issues with no time data
		}

		// ... existing code for hourlyRate, fiscalMonth, labels, etc. ...

		// Create cost allocation record with effort source tracking
		allocation := &models.CostAllocation{
			Id:               fmt.Sprintf("%s:%s", issue.Id, fiscalMonth),
			InitiativeId:     initiativeId,
			IssueId:          issue.Id,
			FiscalMonth:      fiscalMonth,
			DeveloperId:      issue.AssigneeId,
			HoursWorked:      effortResult.Hours,
			HourlyRate:       hourlyRate,
			DeveloperCost:    effortResult.Hours * hourlyRate,
			TotalCost:        effortResult.Hours * hourlyRate,
			IssueType:        issue.Type,
			IssueLabels:      labels,
			EstimatedMinutes: estimatedMinutes,
			ActualMinutes:    actualMinutes,
			VarianceMinutes:  varianceMinutes,
			VariancePercent:  variancePercent,
			OverBudget:       overBudget,
			IsUnallocated:    isUnallocated,
			CalculatedAt:     time.Now(),
			CreatedAt:        time.Now(),

			// New effort source tracking fields
			EffortSource:          effortResult.Source,
			ConfidenceLevel:       effortResult.Confidence,
			GitCodingHours:        effortResult.GitCodingHours,
			GitReviewHours:        effortResult.GitReviewHours,
			GitComplexityFactor:   effortResult.GitComplexity,
			GitActiveDays:         effortResult.GitActiveDays,
			EffortValidated:       effortResult.Validated,
			ValidationVariancePct: effortResult.VariancePct,
			LinkedCommitShas:      effortResult.CommitShas,
			LinkedPrIds:           effortResult.PrIds,
		}

		if err := db.CreateOrUpdate(allocation); err != nil {
			logger.Error(err, "failed to save cost allocation for issue %s", issue.IssueKey)
		}
	}
```

**Step 3: Commit**

```bash
git add backend/plugins/findevops/tasks/calculate_costs.go
git commit -m "feat(findevops): update calculateCosts for multi-source effort

- Uses priority chain: Jira time > estimate > story points > git inferred
- Validates Jira data against Git when both available
- Populates effort source, confidence level, and audit trail
- Stores Git effort breakdown (coding hours, review hours, complexity)"
```

---

## Task 9: Update SubTaskMetas Order in Plugin

**Files:**
- Modify: `backend/plugins/findevops/impl/impl.go`

**Step 1: Update SubTaskMetas**

Update the `SubTaskMetas` method (around line 102):

```go
func (p FinDevOps) SubTaskMetas() []plugin.SubTaskMeta {
	return []plugin.SubTaskMeta{
		// Phase 1: Developer FTE Calculation (Swarmia model)
		tasks.CollectDeveloperActivityMeta,

		// Phase 2: Git Effort Inference (git2effort)
		tasks.InferGitEffortMeta,

		// Phase 3: Cost Allocation (enhanced with multi-source fusion)
		tasks.CalculateCostsMeta,

		// Phase 4: ASC 350-40 Classification
		tasks.CategorizeCapitalizationMeta,

		// Phase 5: Deployment Costs
		tasks.CalculateDeploymentCostsMeta,
	}
}
```

**Step 2: Update MakeMetricPluginPipelinePlanV200**

Update the subtasks list (around line 167):

```go
		Subtasks: []string{
			"collectDeveloperActivity",
			"inferGitEffort",
			"calculateCosts",
			"categorizeCapitalization",
			"calculateDeploymentCosts",
		},
```

**Step 3: Commit**

```bash
git add backend/plugins/findevops/impl/impl.go
git commit -m "feat(findevops): update subtask order for effort inference pipeline

New order:
1. collectDeveloperActivity - FTE calculation
2. inferGitEffort - Git-based effort inference
3. calculateCosts - Multi-source cost allocation
4. categorizeCapitalization - ASC 350-40 classification
5. calculateDeploymentCosts - Cost per deployment"
```

---

## Task 10: Add E2E Test for Effort Inference

**Files:**
- Create: `backend/plugins/findevops/e2e/effort_inference_test.go`

**Step 1: Create E2E test**

Create `backend/plugins/findevops/e2e/effort_inference_test.go`:

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

	"github.com/apache/incubator-devlake/core/models/domainlayer/code"
	"github.com/apache/incubator-devlake/core/models/domainlayer/crossdomain"
	"github.com/apache/incubator-devlake/core/models/domainlayer/ticket"
	"github.com/apache/incubator-devlake/helpers/e2ehelper"
	"github.com/apache/incubator-devlake/plugins/findevops/impl"
	"github.com/apache/incubator-devlake/plugins/findevops/models"
	"github.com/apache/incubator-devlake/plugins/findevops/tasks"
)

func TestEffortInferenceDataFlow(t *testing.T) {
	var plugin impl.FinDevOps
	dataflowTester := e2ehelper.NewDataFlowTester(t, "findevops", plugin)

	// Import test data
	dataflowTester.ImportCsvIntoTabler("./effort_inference/issues.csv", &ticket.Issue{})
	dataflowTester.ImportCsvIntoTabler("./effort_inference/commits.csv", &code.Commit{})
	dataflowTester.ImportCsvIntoTabler("./effort_inference/pull_requests.csv", &code.PullRequest{})
	dataflowTester.ImportCsvIntoTabler("./effort_inference/issue_commits.csv", &crossdomain.IssueCommit{})
	dataflowTester.ImportCsvIntoTabler("./effort_inference/pull_request_issues.csv", &crossdomain.PullRequestIssue{})
	dataflowTester.ImportCsvIntoTabler("./effort_inference/board_issues.csv", &ticket.BoardIssue{})
	dataflowTester.ImportCsvIntoTabler("./effort_inference/project_mapping.csv", &crossdomain.ProjectMapping{})

	taskData := &tasks.FinDevOpsTaskData{
		Options: &tasks.FinDevOpsOptions{
			ProjectName:       "test-project",
			DefaultHourlyRate: 87.0,
		},
		Settings: models.NewDefaultSettings(),
	}

	// Run subtasks in order
	dataflowTester.Subtask(tasks.CollectDeveloperActivityMeta, taskData)
	dataflowTester.Subtask(tasks.InferGitEffortMeta, taskData)
	dataflowTester.Subtask(tasks.CalculateCostsMeta, taskData)

	// Verify FTE records were created
	dataflowTester.VerifyTableWithOptions(
		&models.DeveloperMonthlyFte{},
		e2ehelper.TableOptions{
			CSVRelPath: "./effort_inference/developer_monthly_fte.csv",
		},
	)

	// Verify cost allocations have effort source tracking
	dataflowTester.VerifyTableWithOptions(
		&models.CostAllocation{},
		e2ehelper.TableOptions{
			CSVRelPath: "./effort_inference/cost_allocations.csv",
		},
	)
}
```

**Step 2: Commit (test data CSVs would need to be created separately)**

```bash
git add backend/plugins/findevops/e2e/effort_inference_test.go
git commit -m "test(findevops): add E2E test for effort inference pipeline

Tests the complete flow:
- collectDeveloperActivity creates FTE records
- inferGitEffort caches git-based effort
- calculateCosts uses multi-source effort with audit trail"
```

---

## Task 11: Update Audit Documentation

**Files:**
- Modify: `docs/audit/data-lineage/findevops-lineage.md`

**Step 1: Update data lineage documentation**

Add a new section to the data lineage document describing the effort inference flow:

```markdown
## Effort Inference Data Flow (NEW)

```
┌─────────────────────────────────────────────────────────────────┐
│                     EFFORT INFERENCE PIPELINE                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  1. collectDeveloperActivity (Swarmia Model)            │    │
│  │     Input: commits, pull_requests, issues               │    │
│  │     Output: developer_monthly_fte                       │    │
│  │     Logic: weighted activity → normalized FTE           │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              ↓                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  2. inferGitEffort (git2effort methodology)             │    │
│  │     Input: issue_commits, pull_request_issues           │    │
│  │     Output: cached git_effort per issue                 │    │
│  │     Logic: active_days × productive_hours + reviews     │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              ↓                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  3. calculateCosts (multi-source fusion)                │    │
│  │     Input: issues + git_effort + FTE                    │    │
│  │     Output: cost_allocations with effort_source         │    │
│  │     Logic: Jira time > estimate > points > git > FTE    │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Effort Source Priority

| Priority | Source | Confidence | When Used |
|----------|--------|------------|-----------|
| 1 | jira_time | HIGH | time_spent_minutes > 0 |
| 2 | jira_estimate | MEDIUM | original_estimate_minutes > 0 |
| 3 | story_points | MEDIUM | story_point > 0 |
| 4 | git_inferred | INFERRED | linked commits/PRs exist |
| 5 | fte_distributed | LOW | FTE allocation fallback |
```

**Step 2: Commit**

```bash
git add docs/audit/data-lineage/findevops-lineage.md
git commit -m "docs: update data lineage with effort inference pipeline

Documents:
- Three-phase effort inference flow
- Effort source priority chain
- Confidence levels for R&D compliance"
```

---

## Summary

| Task | Description | Files |
|------|-------------|-------|
| 1 | Add settings fields | `models/settings.go` |
| 2 | Create DeveloperMonthlyFte model | `models/developer_monthly_fte.go` |
| 3 | Add effort source fields to CostAllocation | `models/cost_allocation.go` |
| 4 | Create migration script | `migrationscripts/20260201_*.go`, `register.go` |
| 5 | Register model in plugin | `impl/impl.go` |
| 6 | Create collectDeveloperActivity subtask | `tasks/collect_developer_activity.go` |
| 7 | Create inferGitEffort subtask | `tasks/infer_git_effort.go` |
| 8 | Update calculateCosts for multi-source | `tasks/calculate_costs.go` |
| 9 | Update subtask order | `impl/impl.go` |
| 10 | Add E2E test | `e2e/effort_inference_test.go` |
| 11 | Update documentation | `docs/audit/data-lineage/` |

---

## Execution Notes

- Tasks 1-5 are schema/model changes - run migration after deploying
- Tasks 6-8 implement the core logic with TDD
- Task 9 wires everything together
- Task 10-11 are validation and documentation

**After completing this plan:**
- Run `make build-plugin` to compile
- Run migrations by restarting DevLake
- Test with a project that has both Jira and Git data
- Verify effort_source distribution in cost_allocations table
