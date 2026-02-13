---
  Enhanced Effort Inference Model (DevLake-Conformant)

  1. Settings Model Updates

  File: backend/plugins/findevops/models/settings.go

  package models

  // FinDevOpsSettings stores project-scoped configuration
  type FinDevOpsSettings struct {
      Id          uint64 `json:"id" gorm:"primaryKey;autoIncrement"`
      ProjectName string `json:"projectName" gorm:"type:varchar(255);uniqueIndex"`

      // === EXISTING FIELDS ===
      DefaultHourlyRate        float64 `json:"defaultHourlyRate" gorm:"default:87.0"`
      HoursPerStoryPoint       float64 `json:"hoursPerStoryPoint" gorm:"default:4.0"`
      CapitalizationFramework  string  `json:"capitalizationFramework" gorm:"type:varchar(50);default:'asc_350_40_stages'"`
      PreliminaryLabels        string  `json:"preliminaryLabels" gorm:"type:text"`
      PostImplementationLabels string  `json:"postImplementationLabels" gorm:"type:text"`
      PreliminaryTypes         string  `json:"preliminaryTypes" gorm:"type:text"`
      DevelopmentTypes         string  `json:"developmentTypes" gorm:"type:text"`
      PostImplementationTypes  string  `json:"postImplementationTypes" gorm:"type:text"`

      // === NEW: FTE NORMALIZATION (Swarmia Model) ===
      FteMaxPerMonth            float64 `json:"fteMaxPerMonth" gorm:"default:1.0"`
      FteBaselineMultiplier     float64 `json:"fteBaselineMultiplier" gorm:"default:1.2"`
      FteInactivityThresholdDays int    `json:"fteInactivityThresholdDays" gorm:"default:5"`
      FteWorkingHoursPerMonth   float64 `json:"fteWorkingHoursPerMonth" gorm:"default:160.0"`

      // === NEW: ACTIVITY WEIGHTS (Swarmia Model) ===
      ActivityWeightPrAuthored     float64 `json:"activityWeightPrAuthored" gorm:"default:1.0"`
      ActivityWeightPrReviewed     float64 `json:"activityWeightPrReviewed" gorm:"default:0.3"`
      ActivityWeightCommitAuthored float64 `json:"activityWeightCommitAuthored" gorm:"default:0.2"`
      ActivityWeightIssueUpdated   float64 `json:"activityWeightIssueUpdated" gorm:"default:0.1"`
      ActivityWeightCommentAdded   float64 `json:"activityWeightCommentAdded" gorm:"default:0.05"`

      // === NEW: GIT INFERENCE (git2effort methodology) ===
      GitProductiveHoursPerActiveDay float64 `json:"gitProductiveHoursPerActiveDay" gorm:"default:6.0"`
      GitReviewHoursPerCycle         float64 `json:"gitReviewHoursPerCycle" gorm:"default:1.5"`
      GitCommentsPerReviewCycle      int     `json:"gitCommentsPerReviewCycle" gorm:"default:3"`
      GitMinHoursPerIssue            float64 `json:"gitMinHoursPerIssue" gorm:"default:1.0"`
      GitMaxHoursPerIssue            float64 `json:"gitMaxHoursPerIssue" gorm:"default:80.0"`

      // === NEW: VALIDATION THRESHOLDS ===
      ValidationJiraGitVarianceThresholdPct float64 `json:"validationJiraGitVarianceThresholdPct" gorm:"default:50.0"`

      // === NEW: ASC 350-40 COMMIT KEYWORDS ===
      PreliminaryCommitKeywords        string `json:"preliminaryCommitKeywords" gorm:"type:text"`
      DevelopmentCommitKeywords        string `json:"developmentCommitKeywords" gorm:"type:text"`
      PostImplementationCommitKeywords string `json:"postImplementationCommitKeywords" gorm:"type:text"`

      // === NEW: EFFORT INFERENCE ENABLED ===
      EnableGitEffortInference bool `json:"enableGitEffortInference" gorm:"default:true"`
      EnableFteNormalization   bool `json:"enableFteNormalization" gorm:"default:true"`
  }

  func (FinDevOpsSettings) TableName() string {
      return "_tool_findevops_settings"
  }

  func (s *FinDevOpsSettings) GetProjectName() string {
      return s.ProjectName
  }

  func (s *FinDevOpsSettings) SetProjectName(name string) {
      s.ProjectName = name
  }

  // NewDefaultSettings returns settings with all defaults
  func NewDefaultSettings() *FinDevOpsSettings {
      return &FinDevOpsSettings{
          // Existing defaults
          DefaultHourlyRate:       87.0,
          HoursPerStoryPoint:      4.0,
          CapitalizationFramework: "asc_350_40_stages",
          PreliminaryLabels:       `["research","spike","investigation","poc","feasibility"]`,
          PostImplementationLabels: `["bug","hotfix","maintenance","support","fix"]`,
          PreliminaryTypes:        `["Spike","Research","POC","Discovery"]`,
          DevelopmentTypes:        `["Story","Feature","Enhancement","Task","Requirement"]`,
          PostImplementationTypes: `["Bug","Defect","Hotfix","Support","Incident"]`,

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

  ---
  2. New Data Models

  File: backend/plugins/findevops/models/developer_monthly_fte.go

  package models

  import "time"

  // DeveloperMonthlyFte tracks FTE normalization per developer per month (Swarmia model)
  type DeveloperMonthlyFte struct {
      Id           string `gorm:"primaryKey;type:varchar(255)"` // {developer_id}:{fiscal_month}
      DeveloperId  string `gorm:"type:varchar(255);index"`
      FiscalMonth  string `gorm:"type:varchar(10);index"`
      ProjectName  string `gorm:"type:varchar(255);index"`

      // Activity counts (raw signals)
      PrsAuthored     int `gorm:"type:int;default:0"`
      PrsReviewed     int `gorm:"type:int;default:0"`
      CommitsAuthored int `gorm:"type:int;default:0"`
      IssuesUpdated   int `gorm:"type:int;default:0"`
      CommentsAdded   int `gorm:"type:int;default:0"`

      // FTE calculation (Swarmia methodology)
      RawActivityScore float64 `gorm:"type:decimal(10,2)"`
      BaselineScore    float64 `gorm:"type:decimal(10,2)"` // Team median × multiplier
      RawFte           float64 `gorm:"type:decimal(3,2)"`  // Before inactivity adjustment
      InactiveDays     int     `gorm:"type:int;default:0"` // Consecutive days with no activity
      AdjustedFte      float64 `gorm:"type:decimal(3,2)"`  // Final FTE after deductions

      // Hours allocation tracking
      HoursFromJira        float64 `gorm:"type:decimal(10,2);default:0"`
      HoursFromGitInferred float64 `gorm:"type:decimal(10,2);default:0"`
      HoursDistributed     float64 `gorm:"type:decimal(10,2);default:0"`
      TotalAllocatedHours  float64 `gorm:"type:decimal(10,2);default:0"`

      CalculatedAt time.Time
  }

  func (DeveloperMonthlyFte) TableName() string {
      return "developer_monthly_fte"
  }

  File: backend/plugins/findevops/models/cost_allocation.go (enhanced)

  package models

  import "time"

  type CostAllocation struct {
      // === EXISTING FIELDS ===
      Id                     string    `gorm:"primaryKey;type:varchar(255)"`
      InitiativeId           string    `gorm:"type:varchar(255);index"`
      IssueId                string    `gorm:"type:varchar(255);index"`
      IssueKey               string    `gorm:"type:varchar(255)"`
      IssueType              string    `gorm:"type:varchar(100)"`
      IssueLabels            string    `gorm:"type:text"`
      FiscalMonth            string    `gorm:"type:varchar(10);index"`
      DeveloperId            string    `gorm:"type:varchar(255);index"`
      ProjectName            string    `gorm:"type:varchar(255);index"`

      HoursWorked            float64   `gorm:"type:decimal(10,2)"`
      HourlyRate             float64   `gorm:"type:decimal(10,2)"`
      DeveloperCost          float64   `gorm:"type:decimal(12,2)"`
      AIToolCost             float64   `gorm:"type:decimal(12,2)"`
      TotalCost              float64   `gorm:"type:decimal(12,2)"`

      CapitalizationCategory string    `gorm:"type:varchar(50)"`
      ProjectPhase           string    `gorm:"type:varchar(50)"`
      CapitalizationPercent  int       `gorm:"type:int"`
      CategoryReason         string    `gorm:"type:varchar(255)"`

      EstimatedMinutes       int64     `gorm:"type:bigint"`
      ActualMinutes          int64     `gorm:"type:bigint"`
      VarianceMinutes        int64     `gorm:"type:bigint"`
      VariancePercent        float64   `gorm:"type:decimal(8,2)"`
      OverBudget             bool      `gorm:"type:bool"`
      IsUnallocated          bool      `gorm:"type:bool;index"`

      CalculatedAt           time.Time
      CreatedAt              time.Time

      // === NEW: EFFORT SOURCE TRACKING ===
      EffortSource     string  `gorm:"type:varchar(50)"` // jira_time, jira_estimate, story_points, git_inferred, fte_distributed
      ConfidenceLevel  string  `gorm:"type:varchar(20)"` // high, medium, inferred, low

      // === NEW: GIT-INFERRED EFFORT BREAKDOWN ===
      GitCodingHours      float64 `gorm:"type:decimal(10,2)"`
      GitReviewHours      float64 `gorm:"type:decimal(10,2)"`
      GitComplexityFactor float64 `gorm:"type:decimal(5,2)"`
      GitActiveDays       int     `gorm:"type:int"`

      // === NEW: VALIDATION FLAGS ===
      EffortValidated       bool    `gorm:"type:bool;default:false"`
      ValidationVariancePct float64 `gorm:"type:decimal(8,2)"` // Jira vs Git variance

      // === NEW: AUDIT TRAIL FOR R&D COMPLIANCE ===
      LinkedCommitShas      string `gorm:"type:text"` // JSON array of commit SHAs
      LinkedPrIds           string `gorm:"type:text"` // JSON array of PR IDs
      ClassificationSignals string `gorm:"type:text"` // JSON: what triggered ASC 350-40 category

      // === NEW: FTE CONTEXT ===
      DeveloperMonthlyFte float64 `gorm:"type:decimal(3,2)"`
      FteAllocationPct    float64 `gorm:"type:decimal(5,2)"` // % of developer's month on this issue
  }

  func (CostAllocation) TableName() string {
      return "cost_allocations"
  }

  // EffortSource constants
  const (
      EffortSourceJiraTime      = "jira_time"
      EffortSourceJiraEstimate  = "jira_estimate"
      EffortSourceStoryPoints   = "story_points"
      EffortSourceGitInferred   = "git_inferred"
      EffortSourceFteDistributed = "fte_distributed"
  )

  // ConfidenceLevel constants
  const (
      ConfidenceHigh     = "high"
      ConfidenceMedium   = "medium"
      ConfidenceInferred = "inferred"
      ConfidenceLow      = "low"
  )

  ---
  3. Migration Script

  File: backend/plugins/findevops/models/migrationscripts/20260201_add_effort_inference.go

  package migrationscripts

  import (
      "github.com/apache/incubator-devlake/core/context"
      "github.com/apache/incubator-devlake/core/errors"
      "github.com/apache/incubator-devlake/helpers/migrationhelper"
  )

  type addEffortInference struct{}

  // DeveloperMonthlyFte - new table
  type developerMonthlyFte20260201 struct {
      Id                   string  `gorm:"primaryKey;type:varchar(255)"`
      DeveloperId          string  `gorm:"type:varchar(255);index"`
      FiscalMonth          string  `gorm:"type:varchar(10);index"`
      ProjectName          string  `gorm:"type:varchar(255);index"`
      PrsAuthored          int     `gorm:"type:int;default:0"`
      PrsReviewed          int     `gorm:"type:int;default:0"`
      CommitsAuthored      int     `gorm:"type:int;default:0"`
      IssuesUpdated        int     `gorm:"type:int;default:0"`
      CommentsAdded        int     `gorm:"type:int;default:0"`
      RawActivityScore     float64 `gorm:"type:decimal(10,2)"`
      BaselineScore        float64 `gorm:"type:decimal(10,2)"`
      RawFte               float64 `gorm:"type:decimal(3,2)"`
      InactiveDays         int     `gorm:"type:int;default:0"`
      AdjustedFte          float64 `gorm:"type:decimal(3,2)"`
      HoursFromJira        float64 `gorm:"type:decimal(10,2);default:0"`
      HoursFromGitInferred float64 `gorm:"type:decimal(10,2);default:0"`
      HoursDistributed     float64 `gorm:"type:decimal(10,2);default:0"`
      TotalAllocatedHours  float64 `gorm:"type:decimal(10,2);default:0"`
  }

  func (developerMonthlyFte20260201) TableName() string {
      return "developer_monthly_fte"
  }

  // CostAllocation - add new columns
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

  // Settings - add new configuration fields
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
      return "findevops: add effort inference support (FTE, git inference, audit trail)"
  }

  ---
  4. Register Migration and Models

  File: backend/plugins/findevops/models/migrationscripts/register.go

  package migrationscripts

  import "github.com/apache/incubator-devlake/core/plugin"

  func All() []plugin.MigrationScript {
      return []plugin.MigrationScript{
          new(initSchema),           // 20260129000003
          new(addDeploymentCosts),   // 20260130000003
          new(addSettings),          // 20260130000004
          new(addBudgetVariance),    // 20260131000001
          new(addEffortInference),   // 20260201000001 ← NEW
      }
  }

  File: backend/plugins/findevops/impl/impl.go (update GetTablesInfo)

  func (p FinDevOps) GetTablesInfo() []dal.Tabler {
      return []dal.Tabler{
          &models.CostAllocation{},
          &models.MonthlyCostSummary{},
          &models.DeveloperHourlyRate{},
          &models.DeploymentCost{},
          &models.FinDevOpsSettings{},
          &models.DeveloperMonthlyFte{}, // ← NEW
      }
  }

  ---
  5. New Subtasks

  File: backend/plugins/findevops/tasks/collect_developer_activity.go

  package tasks

  import (
      "github.com/apache/incubator-devlake/core/dal"
      "github.com/apache/incubator-devlake/core/errors"
      "github.com/apache/incubator-devlake/core/plugin"
  )

  var CollectDeveloperActivityMeta = plugin.SubTaskMeta{
      Name:             "collectDeveloperActivity",
      EntryPoint:       CollectDeveloperActivity,
      EnabledByDefault: true,
      Description:      "Collect developer activity signals for FTE calculation",
      DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
  }

  func CollectDeveloperActivity(taskCtx plugin.SubTaskContext) errors.Error {
      // Query commits, PRs, reviews, comments per developer per month
      // Store raw activity counts
      // This feeds into calculateDeveloperFte
  }

  File: backend/plugins/findevops/tasks/calculate_developer_fte.go

  package tasks

  var CalculateDeveloperFteMeta = plugin.SubTaskMeta{
      Name:             "calculateDeveloperFte",
      EntryPoint:       CalculateDeveloperFte,
      EnabledByDefault: true,
      Description:      "Calculate FTE normalization per developer (Swarmia model)",
      DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
      Dependencies:     []*plugin.SubTaskMeta{&CollectDeveloperActivityMeta},
  }

  func CalculateDeveloperFte(taskCtx plugin.SubTaskContext) errors.Error {
      // Apply activity weights from settings
      // Calculate baseline score (team median × multiplier)
      // Normalize to max FTE
      // Detect and deduct inactivity periods
      // Store in developer_monthly_fte table
  }

  File: backend/plugins/findevops/tasks/infer_git_effort.go

  package tasks

  var InferGitEffortMeta = plugin.SubTaskMeta{
      Name:             "inferGitEffort",
      EntryPoint:       InferGitEffort,
      EnabledByDefault: true,
      Description:      "Infer effort from Git activity (git2effort methodology)",
      DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
      Dependencies:     []*plugin.SubTaskMeta{&CalculateDeveloperFteMeta},
  }

  func InferGitEffort(taskCtx plugin.SubTaskContext) errors.Error {
      // For each issue:
      //   1. Find linked PRs via pull_request_issues
      //   2. Find linked commits via issue_commits
      //   3. Calculate coding_hours = active_days × productive_hours
      //   4. Calculate review_hours = review_cycles × hours_per_cycle
      //   5. Calculate complexity factor from lines_changed, files_touched
      //   6. Store git_effort temporarily for use in calculateCostAllocations
  }

  ---
  6. Updated Subtask Order

  File: backend/plugins/findevops/impl/impl.go (update SubTaskMetas)

  func (p FinDevOps) SubTaskMetas() []plugin.SubTaskMeta {
      return []plugin.SubTaskMeta{
          // Phase 1: Developer FTE Calculation (Swarmia model)
          tasks.CollectDeveloperActivityMeta,
          tasks.CalculateDeveloperFteMeta,

          // Phase 2: Git Effort Inference (git2effort)
          tasks.InferGitEffortMeta,

          // Phase 3: Cost Allocation (enhanced with multi-source fusion)
          tasks.CalculateCostsMeta,  // Now uses Jira + Git + FTE

          // Phase 4: ASC 350-40 Classification (enhanced with commit signals)
          tasks.CategorizeCapitalizationMeta,

          // Phase 5: Aggregation
          tasks.CalculateDeploymentCostsMeta,
      }
  }

  ---
  Summary: What This Gives You
  ┌──────────────────────┬────────────────────────────────────────────┬────────────────────────────────────────────────────┐
  │       Feature        │               Implementation               │                      Benefit                       │
  ├──────────────────────┼────────────────────────────────────────────┼────────────────────────────────────────────────────┤
  │ FTE Normalization    │ developer_monthly_fte table + settings     │ Prevents gaming, accounts for work style (Swarmia) │
  ├──────────────────────┼────────────────────────────────────────────┼────────────────────────────────────────────────────┤
  │ Git Effort Inference │ inferGitEffort subtask                     │ 95%+ coverage vs 13% (git2effort)                  │
  ├──────────────────────┼────────────────────────────────────────────┼────────────────────────────────────────────────────┤
  │ Multi-Source Fusion  │ Enhanced calculateCosts                    │ Best data wins, with validation                    │
  ├──────────────────────┼────────────────────────────────────────────┼────────────────────────────────────────────────────┤
  │ Audit Trail          │ classification_signals, linked_commit_shas │ R&D tax credit compliant                           │
  ├──────────────────────┼────────────────────────────────────────────┼────────────────────────────────────────────────────┤
  │ Configurable         │ All parameters in FinDevOpsSettings        │ UI-adjustable per project                          │
  ├──────────────────────┼────────────────────────────────────────────┼────────────────────────────────────────────────────┤
  │ Confidence Levels    │ effort_source, confidence_level columns    │ Transparency for finance                           │
  └──────────────────────┴────────────────────────────────────────────┴────────────────────────────────────────────────────┘
  ---