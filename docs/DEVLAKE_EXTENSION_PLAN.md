# DevLake Engineering Management Extension Plan

**Project Goal**: Extend Apache DevLake with capabilities similar to Jellyfish/Swarmia for AI Impact, Business Alignment, Capacity Planning, and FinDevOps.

**Team Size**: ~20 developers
**Timeline**: 8-10 weeks for Phase 1
**Date Created**: 2026-01-28

---

## Executive Summary

This plan extends DevLake to provide:
1. **AI Impact Analysis** - Detect AI-assisted code and measure productivity impact (no direct tool integration needed)
2. **Business Alignment** - Map engineering work to strategic business initiatives via Jira Epics
3. **Capacity & Scenario Planning** - Forecast initiative completion and model team changes
4. **FinDevOps** - Software capitalization tracking for US GAAP ASC 350-40 compliance

**Key Insight**: Following Jellyfish's approach, AI detection is done by analyzing commit/PR patterns (vendor-agnostic) rather than integrating with individual AI tools. This is simpler and more maintainable.

---

## Architecture Overview

### New Components

```
/backend/
├── core/models/domainlayer/
│   ├── businessinitiative.go          # NEW: Strategic initiatives domain model
│   ├── ai_usage_signal.go             # NEW: AI detection results
│   └── cost_allocation.go             # NEW: Cost tracking for GAAP compliance
│
├── plugins/
│   ├── businessmetrics/               # NEW: Business goal extraction & alignment
│   │   ├── impl/impl.go
│   │   ├── tasks/extract_business_goals.go
│   │   ├── tasks/calculate_alignment.go
│   │   └── models/migrationscripts/
│   │
│   ├── aidetector/                    # NEW: AI usage pattern detection
│   │   ├── impl/impl.go
│   │   ├── tasks/analyze_commit_patterns.go
│   │   ├── tasks/analyze_pr_characteristics.go
│   │   ├── tasks/score_ai_confidence.go
│   │   └── models/migrationscripts/
│   │
│   ├── findevops/                     # NEW: Cost allocation & capitalization
│   │   ├── impl/impl.go
│   │   ├── tasks/calculate_costs.go
│   │   ├── tasks/categorize_capitalization.go
│   │   ├── api/cost_reports.go
│   │   └── models/migrationscripts/
│   │
│   └── capacityplanner/               # NEW: Velocity & forecasting
│       ├── impl/impl.go
│       ├── tasks/calculate_velocity.go
│       ├── tasks/forecast_completion.go
│       ├── api/forecasting.go
│       └── models/migrationscripts/
│
/grafana/dashboards/engineering-management/
├── ai-impact.json                     # NEW: AI adoption & productivity
├── business-alignment.json            # NEW: Strategic goal tracking
├── capacity-planning.json             # NEW: Velocity & forecasting
└── cost-allocation.json               # NEW: Financial reporting
```

---

## Phase 1: Core Infrastructure (Weeks 1-6)

### 1.1 Business Alignment Module (Weeks 1-2)

**Objective**: Link all engineering work to strategic business initiatives.

#### What Gets Built

**Files to create**:
```
/backend/core/models/domainlayer/businessinitiative.go
/backend/core/models/migrationscripts/20260201_business_initiative.go
/backend/plugins/businessmetrics/impl/impl.go
/backend/plugins/businessmetrics/tasks/extract_business_goals.go
/backend/plugins/businessmetrics/tasks/calculate_alignment.go
/backend/plugins/businessmetrics/models/migrationscripts/20260201_init.go
/backend/plugins/businessmetrics/e2e/business_alignment_test.go
```

**Domain Model**:
```go
// BusinessInitiative represents a strategic business goal
type BusinessInitiative struct {
    Id              string
    Name            string
    JiraEpicKey     string    // e.g., "PROJ-123"
    GoalType        string    // revenue, efficiency, compliance, innovation
    FiscalQuarter   string    // "2026-Q1"
    TargetDate      *time.Time
    Status          string    // planned, active, completed, cancelled
    CreatedAt       time.Time
}

// WorkAllocation links work items to initiatives
type WorkAllocation struct {
    Id                  string
    InitiativeId        string
    EntityType          string    // issue, pull_request, commit
    EntityId            string
    DeveloperId         string
    StoryPoints         int
    EstimatedHours      float64
    ActualHours         float64
    CreatedAt           time.Time
}
```

**How It Works**:
1. **Extract Task** reads Jira Epics from `_tool_jira_board_issues` table
2. Parses Epic labels/custom fields to extract business goal metadata
3. Creates `BusinessInitiative` records in domain layer
4. **Alignment Task** links Issues → PRs → Commits to initiatives
5. Uses existing DevLake relationships (`issue_commits`, `pull_request_issues`)
6. Creates `WorkAllocation` records showing time spent per initiative

**Database Schema**:
```sql
CREATE TABLE domain_business_initiatives (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(500) NOT NULL,
    jira_epic_key VARCHAR(100) NOT NULL,
    goal_type VARCHAR(50),
    fiscal_quarter VARCHAR(10),
    target_date TIMESTAMP,
    status VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE domain_work_allocations (
    id VARCHAR(255) PRIMARY KEY,
    initiative_id VARCHAR(255) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id VARCHAR(255) NOT NULL,
    developer_id VARCHAR(255),
    story_points INT,
    estimated_hours DECIMAL(10,2),
    actual_hours DECIMAL(10,2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (initiative_id) REFERENCES domain_business_initiatives(id)
);
```

---

### 1.2 AI Impact Detection (Weeks 3-4)

**Objective**: Detect AI-assisted code using pattern analysis (mimicking Jellyfish's approach).

#### What Gets Built

**Files to create**:
```
/backend/core/models/domainlayer/ai_usage_signal.go
/backend/core/models/migrationscripts/20260215_ai_usage.go
/backend/plugins/aidetector/impl/impl.go
/backend/plugins/aidetector/tasks/analyze_commit_patterns.go
/backend/plugins/aidetector/tasks/analyze_pr_characteristics.go
/backend/plugins/aidetector/tasks/score_ai_confidence.go
/backend/plugins/aidetector/models/migrationscripts/20260215_init.go
/backend/plugins/aidetector/e2e/ai_detection_test.go
```

**Domain Model**:
```go
type AIUsageSignal struct {
    Id                  string
    PullRequestId       string
    CommitSha           string
    AIConfidenceScore   int       // 0-100
    DetectedTool        string    // "unknown", "copilot", "cursor", "claude"
    VelocityImpact      float64   // multiplier vs developer baseline
    CycleTimeReduction  float64   // hours saved
    CodeDuplicationFlag bool
    PatternSignatures   string    // JSON: detected patterns
    DetectedAt          time.Time
}
```

**How It Works** (based on [Jellyfish's methodology](https://jellyfish.co/platform/jellyfish-ai-impact/)):

1. **Pattern Analysis** - Analyzes existing commits/PRs for AI signals:
   - **Rapid commit velocity**: Multiple commits within minutes (AI generates code fast)
   - **PR size anomaly**: AI-assisted PRs are [18% larger on average](https://jellyfish.co/blog/ai-assisted-pull-requests-are-18-larger/)
   - **Lines per minute**: Unusually high code production rate
   - **Code duplication**: AI often generates similar structures
   - **Cycle time breakdown**: Short coding time vs review time ratio

2. **Scoring Algorithm**:
```go
func CalculateAIConfidence(pr *PullRequest, commits []*Commit) int {
    score := 0

    // Rapid commits (max 30 points)
    if avgTimeBetween(commits) < 5*time.Minute {
        score += 30
    }

    // High lines/minute (max 25 points)
    if linesPerMinute(pr) > 20 {
        score += 25
    }

    // PR size anomaly (max 20 points)
    if pr.Additions > developerBaseline*1.15 {
        score += 20
    }

    // Code duplication (max 15 points)
    score += detectDuplication(pr)

    // Generic commit messages (max 10 points)
    if hasGenericMessages(commits) {
        score += 10
    }

    return min(score, 100)
}
```

3. **No Integration Required**: Works with existing Git/GitHub data - no Cursor/Claude APIs needed

**Database Schema**:
```sql
CREATE TABLE domain_ai_usage_signals (
    id VARCHAR(255) PRIMARY KEY,
    pull_request_id VARCHAR(255),
    commit_sha VARCHAR(255),
    ai_confidence_score INT CHECK (ai_confidence_score BETWEEN 0 AND 100),
    detected_tool VARCHAR(50),
    velocity_impact DECIMAL(5,2),
    cycle_time_reduction DECIMAL(10,2),
    code_duplication_flag BOOLEAN DEFAULT FALSE,
    pattern_signatures TEXT,
    detected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (pull_request_id) REFERENCES pull_requests(id)
);
```

**References**:
- [Jellyfish AI Impact Dashboard](https://jellyfish.co/platform/jellyfish-ai-impact/)
- [51% of PRs Use AI (May 2025)](https://jellyfish.co/blog/ai-impact-data-june-2025/)

---

### 1.3 FinDevOps Cost Allocation (Weeks 5-6)

**Objective**: Calculate development costs for US GAAP ASC 350-40 compliance.

#### What Gets Built

**Files to create**:
```
/backend/core/models/domainlayer/cost_allocation.go
/backend/core/models/migrationscripts/20260220_cost_allocation.go
/backend/plugins/findevops/impl/impl.go
/backend/plugins/findevops/tasks/calculate_costs.go
/backend/plugins/findevops/tasks/categorize_capitalization.go
/backend/plugins/findevops/api/cost_reports.go
/backend/plugins/findevops/models/migrationscripts/20260220_init.go
/backend/plugins/findevops/e2e/cost_allocation_test.go
```

**Domain Model**:
```go
type CostAllocation struct {
    Id                      string
    InitiativeId            string
    FiscalMonth             string    // "2026-01"
    DeveloperCosts          float64   // hours * hourly_rate
    AIToolCosts             float64   // estimated AI costs
    TotalCost               float64
    CapitalizationCategory  string    // "capitalizable", "expense"
    ProjectPhase            string    // "preliminary", "development", "post_implementation"
    CalculatedAt            time.Time
}

type DeveloperHourlyRate struct {
    DeveloperId       string
    DeveloperEmail    string
    HourlyRate        float64
    CostCenter        string
    EffectiveDate     time.Time
}
```

**How It Works**:
1. Reads work allocations from `businessmetrics` plugin
2. Joins with hourly rates (from CSV import or config)
3. Calculates: `actual_hours * hourly_rate` per developer
4. Applies US GAAP ASC 350-40 categorization rules
5. Generates monthly cost reports for finance team

**RECOMMENDED: ASC 350-40 Three-Stage Model** (Industry Standard)

This approach is used by Swarmia, Jellyfish, and complies with US GAAP accounting standards.

```yaml
# Stage-Based Capitalization (Current Standard until 2028)
# Based on ASC 350-40 and updated FASB ASU 2025-06

stages:
  preliminary_stage:
    description: "Planning, research, determining feasibility"
    capitalization_rule: "100% EXPENSE"
    examples:
      - Research & spikes
      - Feasibility studies
      - Requirements gathering
      - Technology evaluations
      - Initial planning
    jira_detection:
      - issue_type: ["Spike", "Research", "Discovery"]
      - labels: ["research", "spike", "investigation", "feasibility", "discovery", "proof-of-concept"]
      - status: ["To Do", "Backlog"] # Before development starts

  application_development_stage:
    description: "Building new functionality - software ready for intended use"
    capitalization_rule: "100% CAPITALIZE (with exceptions)"
    capitalizable_costs:
      - "Direct external costs (contractors, third-party services)"
      - "Employee compensation directly tied to coding, testing, deployment"
      - "Travel costs for developers working on project"
    expensed_costs:
      - "Training costs"
      - "Data migration/conversion"
      - "General overhead"
    examples:
      - New features adding functionality
      - Enhancements that enable new capabilities
      - Platform improvements (when adding functionality)
      - Architectural upgrades (when enabling new features)
    jira_detection:
      - issue_type: ["Story", "Feature", "Enhancement"]
      - exclude_labels: ["bug", "hotfix", "maintenance", "support", "training", "migration"]
      - status: ["In Progress", "In Review", "Testing", "Done"]
      - criteria: "Must add NEW functionality that didn't exist before"

  post_implementation_stage:
    description: "Maintaining existing functionality, fixing defects"
    capitalization_rule: "100% EXPENSE"
    examples:
      - Bug fixes
      - Routine maintenance
      - Performance optimization (without new features)
      - Technical debt cleanup
      - Production support
      - User training
      - Documentation updates
    jira_detection:
      - issue_type: ["Bug", "Defect", "Hotfix", "Support", "Task"]
      - labels: ["bug", "hotfix", "maintenance", "support", "training", "ktlo", "tech-debt"]
      - criteria: "Does NOT add new functionality"

# Special Categories
special_categories:
  enhancements_and_upgrades:
    description: "Modifications to existing software"
    capitalization_rule: "CAPITALIZE if adds NEW functionality, EXPENSE if maintains existing"
    decision_criteria:
      - question: "Does this enable the software to perform tasks it couldn't do before?"
      - if_yes: "CAPITALIZE (Application Development Stage)"
      - if_no: "EXPENSE (Post-Implementation Stage)"
    examples_capitalizable:
      - "Adding OAuth2 authentication (new login method)"
      - "Adding export to PDF feature (new capability)"
      - "Implementing real-time notifications (new feature)"
    examples_expense:
      - "Fixing broken authentication flow (maintaining existing feature)"
      - "Improving PDF export performance (maintaining existing feature)"
      - "Fixing notification delivery bugs (maintaining existing feature)"

  ktlo_keep_the_lights_on:
    description: "Routine operations maintaining current state"
    capitalization_rule: "100% EXPENSE"
    industry_benchmark: "20-30% of engineering time (healthy target)"
    examples:
      - Monitoring and alerting
      - Routine patches and updates
      - Infrastructure maintenance
      - Security patches (without new security features)
      - Performance monitoring
      - Incident response

  management_and_overhead:
    description: "Non-direct development activities"
    capitalization_rule: "100% EXPENSE"
    examples:
      - Sprint planning
      - Retrospectives
      - 1:1s and team meetings
      - Process improvement
      - Hiring and onboarding
      - General administration
```

**Implementation: Automatic Categorization Rules**

DevLake will automatically categorize work using these rules (following Jellyfish/Swarmia approach):

```yaml
capitalization_rules:
  # Rule 1: Stage Detection (Primary)
  - name: "Detect Application Development Stage (CAPITALIZE)"
    priority: 1
    conditions:
      ALL_OF:
        - issue_type: ["Story", "Feature", "Enhancement", "Epic"]
        - status: ["In Progress", "In Review", "Testing", "Done", "Closed"]
        - NOT labels: ["research", "spike", "bug", "hotfix", "maintenance", "support", "training"]
    result: "capitalizable"
    stage: "application_development"
    capitalization_percent: 100

  # Rule 2: Enhancement Assessment (Requires Functionality Check)
  - name: "Enhancement with New Functionality (CAPITALIZE)"
    priority: 2
    conditions:
      ALL_OF:
        - issue_type: ["Enhancement", "Improvement"]
        - NOT labels: ["bug-fix", "performance-only", "refactor-only"]
      MANUAL_REVIEW_IF:
        - description_contains: ["fix", "improve", "optimize"]
        - description_not_contains: ["new", "add", "enable", "support"]
    result: "capitalizable_if_new_functionality"
    stage: "application_development"
    capitalization_percent: 100
    note: "Flag for review if unclear whether new functionality is added"

  # Rule 3: Preliminary Stage (EXPENSE)
  - name: "Research and Planning (EXPENSE)"
    priority: 3
    conditions:
      ANY_OF:
        - issue_type: ["Spike", "Research", "Discovery"]
        - labels: ["research", "spike", "investigation", "feasibility", "discovery", "poc", "proof-of-concept"]
        - status: ["To Do", "Backlog"]  # Pre-development
    result: "expense"
    stage: "preliminary"
    capitalization_percent: 0

  # Rule 4: Post-Implementation Stage (EXPENSE)
  - name: "Maintenance and Bug Fixes (EXPENSE)"
    priority: 4
    conditions:
      ANY_OF:
        - issue_type: ["Bug", "Defect", "Hotfix", "Support"]
        - labels: ["bug", "hotfix", "maintenance", "ktlo", "support", "incident"]
    result: "expense"
    stage: "post_implementation"
    capitalization_percent: 0

  # Rule 5: Overhead Activities (EXPENSE)
  - name: "Management and Overhead (EXPENSE)"
    priority: 5
    conditions:
      ANY_OF:
        - labels: ["management", "admin", "overhead", "meeting", "training", "onboarding", "hiring"]
        - issue_type: ["Admin", "Process"]
    result: "expense"
    stage: "overhead"
    capitalization_percent: 0

  # Rule 6: Data Migration (EXPENSE per ASC 350-40)
  - name: "Data Migration and Conversion (EXPENSE)"
    priority: 6
    conditions:
      ANY_OF:
        - labels: ["migration", "data-conversion", "data-migration"]
        - title_contains: ["migrate", "migration", "data conversion"]
    result: "expense"
    stage: "post_implementation"
    capitalization_percent: 0

  # Rule 7: Training (EXPENSE per ASC 350-40)
  - name: "Training Activities (EXPENSE)"
    priority: 7
    conditions:
      ANY_OF:
        - labels: ["training", "documentation", "user-training"]
        - issue_type: ["Training", "Documentation"]
    result: "expense"
    stage: "overhead"
    capitalization_percent: 0

# Default Rule (if no match)
default_rule:
  name: "Unclassified Work (FLAG FOR REVIEW)"
  result: "review_required"
  capitalization_percent: 0
  action: "Flag issue for manual classification"
```

**Cost Calculation Formula**:
```
For each Jira issue:
1. Determine Stage: Apply rules above in priority order
2. Get Time Spent: Sum worklogs for issue
3. Get Developer Rates: Lookup hourly rate by assignee
4. Calculate Cost: time_hours × hourly_rate
5. Apply Capitalization:
   - If stage = "application_development": cost × 100% = capitalizable
   - If stage = "preliminary" or "post_implementation": cost × 0% = expense

Total Capitalizable Cost = SUM(all capitalizable costs)
Total Expense = SUM(all expense costs)
Capitalization Rate = Total Capitalizable / (Total Capitalizable + Total Expense)
```

**API Endpoints**:
```
GET  /api/findevops/costs?initiative_id=X&fiscal_month=2026-01
GET  /api/findevops/gaap-report?fiscal_quarter=2026-Q1
GET  /api/findevops/capitalization-summary?fiscal_quarter=2026-Q1
POST /api/findevops/export-csv?fiscal_month=2026-01
POST /api/findevops/audit-report?fiscal_year=2026
```

**Audit-Ready Reporting**:
Following [Jellyfish's approach](https://jellyfish.co/library/software-capitalization-benefits/), generate granular reports that:
- Show work item → time → cost → capitalization decision chain
- Include justification for each classification
- Provide drill-down by Epic, developer, project
- Track changes to classifications over time (audit trail)

---

### 1.3.1 Future-Proofing: FASB ASU 2025-06 (Effective 2028)

**Important Update**: In September 2025, FASB issued ASU 2025-06, which modernizes software capitalization guidance.

**Key Changes** (from [FASB ASU 2025-06](https://dart.deloitte.com/USDART/home/publications/deloitte/heads-up/2025/fasb-asu-amends-software-costs-guidance)):
1. **Eliminates Three-Stage Model**: No more preliminary/development/post-implementation stages
2. **New Criteria**: Capitalize when:
   - Management authorizes and commits funding
   - **AND** it's "probable" the project will be completed
3. **Significant Development Uncertainty**: Cannot capitalize if software has "technological innovations or novel, unique, or unproven functions"

**Impact**: More costs may be expensed under the new guidance.

**Implementation Timeline**: Effective for fiscal years beginning after December 15, 2027.

**Our Approach**:
- Build with current three-stage model (2026-2027)
- Design system to easily switch to "probable-to-complete" model (2028+)
- Add `capitalization_framework` config: `"asc_350_40_stages"` or `"asc_350_40_probable"`

**Preparation**:
```yaml
# Future framework (2028+)
probable_to_complete_model:
  criteria:
    - management_committed: true  # Epic approved and funded
    - probable_to_complete: true  # No significant development uncertainty
    - exclude_if:
        - significant_uncertainty: true  # Novel/unproven technology
        - labels: ["experimental", "r&d", "novel-tech", "unproven"]
```

---

### 1.4 Capacity & Scenario Planning (Weeks 5-6)

**Objective**: Forecast initiative completion and model team changes.

#### What Gets Built

**Files to create**:
```
/backend/core/models/domainlayer/capacity_metric.go
/backend/core/models/migrationscripts/20260220_capacity.go
/backend/plugins/capacityplanner/impl/impl.go
/backend/plugins/capacityplanner/tasks/calculate_velocity.go
/backend/plugins/capacityplanner/tasks/forecast_completion.go
/backend/plugins/capacityplanner/api/forecasting.go
/backend/plugins/capacityplanner/models/migrationscripts/20260220_init.go
/backend/plugins/capacityplanner/e2e/capacity_test.go
```

**Domain Model**:
```go
type TeamVelocity struct {
    TeamId          string
    SprintId        string
    FiscalWeek      string
    StoryPoints     int
    CommitCount     int
    PRCount         int
    AvgCycleTime    float64
    CalculatedAt    time.Time
}

type InitiativeForecast struct {
    InitiativeId            string
    RemainingStoryPoints    int
    AvgVelocity             float64
    EstimatedSprints        int
    EstimatedCompletionDate time.Time
    ConfidenceLevel         string
}
```

**How It Works**:
1. Aggregates historical velocity by team (last 4-6 sprints)
2. Calculates remaining work per initiative
3. Forecasts: `remaining_points / avg_velocity = estimated_sprints`
4. What-if scenarios: Models impact of team size changes

**API Endpoints**:
```
GET /api/capacity/forecast?initiative_id=X
GET /api/capacity/scenario?initiative_id=X&team_size_delta=+2
GET /api/capacity/skills-analysis
```

---

## Phase 2: Visualization (Weeks 7-8)

### Grafana Dashboards

#### 2.1 AI Impact Dashboard

**Panels**:
1. **AI Adoption Rate** - "51% of PRs use AI"
2. **AI Adoption Trend** - Monthly % over time
3. **Velocity Impact** - "AI-assisted work is 28% faster"
4. **AI Adoption by Team** - Heatmap by developer/team
5. **AI Confidence Distribution** - Histogram of scores

**Queries**:
```sql
-- AI adoption rate
SELECT COUNT(*) * 100.0 / (SELECT COUNT(*) FROM pull_requests)
FROM domain_ai_usage_signals
WHERE ai_confidence_score > 70;

-- Velocity comparison
SELECT
    CASE WHEN ai_confidence_score > 70 THEN 'AI-Assisted' ELSE 'Non-AI' END as type,
    AVG(cycle_time_hours)
FROM domain_ai_usage_signals s
JOIN pull_requests p ON s.pull_request_id = p.id
GROUP BY type;
```

#### 2.2 Business Alignment Dashboard

**Panels**:
1. **Time Allocation Pie Chart** - "45% Revenue, 30% Efficiency, 25% Compliance"
2. **Initiative Progress Bars** - Completion % per initiative
3. **Misalignment Warnings** - Over/under-resourced initiatives
4. **Quarterly Goal Tracking** - Timeline vs target dates

#### 2.3 Capacity Planning Dashboard

**Panels**:
1. **Team Velocity Trend** - Last 12 sprints
2. **Initiative Forecasts** - "Epic X: 6 weeks remaining"
3. **Skill Distribution** - Backend/Frontend capacity
4. **Scenario Comparison** - Impact of team changes

#### 2.4 Cost Allocation Dashboard

**Panels**:
1. **Monthly Costs** - Total per initiative
2. **Capitalizable vs Expense** - US GAAP breakdown
3. **Cost per Story Point** - Efficiency trends
4. **Budget vs Actual** - Spending alerts

---

## Prerequisites & Setup

### Data Prerequisites (NEEDED BEFORE CODING)

1. **Jira Epic Structure**:
   - [ ] How are business goals tracked? (Custom field? Label format?)
   - [ ] Example Epic label: "Mobile Redesign Q1" or "goal:revenue"?
   - [ ] What issue types are used? (Story, Bug, Epic, etc.)

2. **Developer Hourly Rates**:
   - [ ] Provide CSV: `developer_email,hourly_rate,cost_center`
   - [ ] Example:
   ```csv
   dev1@company.com,120.00,Engineering
   dev2@company.com,135.00,Engineering
   ```

3. **Capitalization Rules**:
   - [ ] Which work types are capitalizable?
   - [ ] Features = capitalizable? Bugs = expense? Research = expense?

### Environment Setup

- [ ] Go 1.21+ installed
- [ ] PostgreSQL 13+ (DevLake's database)
- [ ] Access to existing DevLake instance
- [ ] Grafana 9.0+ for dashboards

---

## Implementation Checklist

### Week 1-2: Business Alignment
- [ ] Create `businessmetrics` plugin directory structure
- [ ] Implement `BusinessInitiative` domain model
- [ ] Write migrations for new tables
- [ ] Implement Epic extraction task
- [ ] Implement work allocation task
- [ ] Write E2E tests
- [ ] Verify: Can query "Show commits for Initiative X"

### Week 3-4: AI Impact Detection
- [ ] Create `aidetector` plugin directory structure
- [ ] Implement `AIUsageSignal` domain model
- [ ] Write migrations
- [ ] Implement pattern analysis tasks
- [ ] Implement scoring algorithm
- [ ] Tune thresholds with historical data
- [ ] Manual validation: Test 50 PRs for accuracy
- [ ] Verify: Dashboard shows "% PRs with AI"

### Week 5-6: FinDevOps & Capacity
- [ ] Create `findevops` plugin
- [ ] Implement cost allocation logic
- [ ] Import hourly rates CSV
- [ ] Implement capitalization rules engine
- [ ] Create cost report API endpoints
- [ ] Create `capacityplanner` plugin
- [ ] Implement velocity calculation
- [ ] Implement forecasting algorithms
- [ ] Create scenario API endpoints
- [ ] Verify: Generate Q1 2026 cost report

### Week 7-8: Dashboards
- [ ] Create AI Impact dashboard JSON
- [ ] Create Business Alignment dashboard JSON
- [ ] Create Capacity Planning dashboard JSON
- [ ] Create Cost Allocation dashboard JSON
- [ ] Test query performance
- [ ] User acceptance testing

### Week 9-10: Launch
- [ ] Full E2E test with production data
- [ ] Performance tuning
- [ ] Documentation
- [ ] Training for engineering managers
- [ ] Finance team demo
- [ ] Production deployment

---

## Open Questions (TO ANSWER BEFORE STARTING)

### Jira Structure (Based on Screenshot Analysis)

**Epic FA-6534 Observed Fields**:
- ✅ **Standard Fields**: Assignee, Reporter, Labels, Team, Story Points, Priority, Due date, Sprint
- ✅ **Labels Format**: Using labels like `cards-production-support` for categorization
- ✅ **Child Work Items**: Epic contains multiple stories/tasks (FA-7482, FA-6571, FA-6535, FA-6536)
- ✅ **Epic Structure**: Title format "Sports Facilities Management LLC - Onboarding Product & Operations"
- ❓ **Custom Fields**: "More fields" section was collapsed - need to see what's there

**DECISION - ADD THESE CUSTOM FIELDS TO JIRA EPICS**:

**Recommended Custom Fields** (User can add these):
1. ✅ **Business Goal** (Text field, 255 chars)
   - Example: "Improve customer retention", "Expand mobile capabilities"

2. ✅ **Investment Category** (Select: single choice)
   - Options: New Business Value, KTLO, Platform/Infrastructure, Tech Debt, Production Support, Research & Development

3. ✅ **Strategic Theme** (Text field)
   - Example: "Mobile Redesign Q1", "API Modernization"

4. ✅ **Fiscal Quarter** (Text field)
   - Example: "2026-Q1"

5. ✅ **Business Value** (Select: single)
   - Options: High, Medium, Low

6. ✅ **Customer Impact** (Text area)
   - Description of customer benefit

7. ✅ **R&D Subcategory** (Select: single) - Only for R&D items
   - Options: Bookkeeping, Engineering Management, Platform, BizOps/Data

**Implementation Note**: DevLake will read these custom fields via Jira API using field IDs (e.g., `customfield_10001`). We'll need to map field names to IDs during setup.

**Alternative**: If adding custom fields is too complex, we can use **Labels** with a convention:
- `investment:KTLO`, `investment:R&D`
- `rd-subcat:Platform`, `rd-subcat:BizOps`
- `goal:customer-retention`, `theme:mobile-q1`

❓ **Team Names**: Please provide list of team names (for capacity planning)

### Developer Rates & Cost Data

**CONFIRMED**:
1. ✅ **Rate Model**: Use **blended average** rates (user has individual rates but prefers simplification)
   - Suggested blended rates:
     ```csv
     role,hourly_rate,cost_center
     Senior Engineer,120.00,Engineering
     Staff Engineer,135.00,Engineering
     Engineer,100.00,Engineering
     Designer,110.00,Design
     Product Manager,125.00,Product
     ```
   - **Implementation**: DevLake will apply rate based on role in Jira (Assignee → Role lookup)

2. ✅ **Team Size**: ~20 people total
   - ❓ Breakdown by team? (For capacity planning)
   - ❓ Roles: How many Engineers vs Designers vs PMs?

**Alternative Approach**: If role data is not in Jira, we can:
- Use a single blended rate for all developers (e.g., $115/hr average)
- Store role mapping in a separate config file (email → role → rate)

### Capitalization Rules

**CONFIRMED** (from user's existing model):
1. ✅ **Investment Categories & Capitalization %**:
   - **KTLO (Maintenance)**: 0% capitalizable (100% expense)
   - **Management**: 0% capitalizable (100% expense)
   - **Production Support**: 0% capitalizable (100% expense)
   - **Research & Development**: Varies by subcategory:
     - Bookkeeping: 0%
     - Engineering Management: 15%
     - Platform: 25%
     - BizOps/Data: 10%

2. ✅ **Example from User**:
   ```
   Time Allocation: 50% R&D, 35% Management, 15% Production Support, 0% KTLO
   R&D Breakdown: 50% Platform, 30% EM, 20% BizOps/Data
   Result: 25% × 50% + 15% × 30% + 10% × 20% = 19% total capitalization
   ```

3. **Implementation Note**:
   - Epic's "Investment Category" field determines primary category
   - If category = "R&D", use "R&D Subcategory" field to determine capitalization %
   - Time spent on Epic (from Jira worklogs) × rate × capitalization % = capitalizable amount

### AI Tools

**CONFIRMED**:
1. ✅ AI tools in use:
   - **Claude Code** - Anthropic's AI coding assistant
   - **Cursor** - AI-first code editor
   - **Codex (GitHub Copilot)** - GitHub's AI pair programmer

2. ❓ Do developers add any commit conventions when using AI?
   - Need to check: Do commit messages mention AI tools?
   - Do PRs have labels like `ai-assisted`?
   - **Recommendation**: If not, we'll rely on pattern detection (Jellyfish approach)

### Process Questions

**NEEDED FROM USER**:
1. Who reviews cost allocation before finance submission?
2. How often should forecasts update? (Weekly? Per sprint?)
3. Should scenario planning be self-service or managed?
4. What's your sprint cadence? (2 weeks?)

### Technical Decisions

1. Should we backfill AI detection on last 6 months of PRs? (Recommend: Yes, for baseline)
2. What's acceptable false positive rate for AI detection? (Suggest: 70% confidence threshold)
3. Cache velocity calculations or compute on-demand? (Recommend: Cache, refresh daily)

---

## Success Metrics (3 months post-launch)

- [ ] Can identify % of work using AI (match Jellyfish's accuracy)
- [ ] Can quantify velocity impact of AI tools
- [ ] 100% of work linked to business initiatives
- [ ] Leadership reviews time allocation monthly
- [ ] Forecasts accurate within ±2 sprints
- [ ] Finance uses cost reports for capitalization
- [ ] Audit trail passes external audit

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| AI detection accuracy too low | Medium | Tune algorithm, manual validation |
| Jira Epic structure incompatible | High | Discovery phase before coding |
| Finance rejects cost categorization | High | Early review with finance |
| Dashboard performance issues | Medium | Database indexes, caching |
| Developer pushback on AI tracking | Low | Communicate: Pattern detection, not surveillance |

---

## Comparison: Old Approach vs. Recommended Approach

### Your Original Percentage-Based Model
```
Approach: Fixed percentages by category
- KTLO: 0%
- Management: 0%
- R&D → Platform: 25%
- R&D → EM: 15%
- R&D → BizOps: 10%

Pros: Simple, easy to understand
Cons:
- Not GAAP-compliant (arbitrary percentages)
- Doesn't distinguish between building NEW features vs maintaining existing
- Hard to defend in audit (why is Platform 25% vs 15%?)
- Doesn't account for work nature (bug vs feature)
```

### Recommended Three-Stage Model
```
Approach: Automatic categorization by development stage
- Preliminary Stage: 0% (research, planning)
- Application Development: 100% (NEW functionality)
- Post-Implementation: 0% (maintenance, bugs)

Pros:
- GAAP ASC 350-40 compliant ✅
- Automated via Jira issue type/labels ✅
- Audit-defensible (clear rules) ✅
- Matches Jellyfish/Swarmia approach ✅
- Distinguishes NEW features from maintenance ✅

Cons:
- Requires accurate Jira categorization
- Some work may need manual review
```

**Winner**: **Three-Stage Model** - Industry standard, audit-ready, automated

---

## Industry Benchmarks

From research, here's what healthy teams look like:

| Metric | Benchmark | Source |
|--------|-----------|--------|
| **KTLO Allocation** | 20-30% of eng time | [Uplevel](https://uplevelteam.com/blog/ktlo-in-software-development) |
| **New Features (SaaS)** | 40-60% of output | Research composite |
| **Innovation vs Maintenance** | 80/20 ratio (leaders) | Industry best practice |
| **Capitalization Rate** | Varies 20-60% | Depends on product maturity |
| **Audit Pass Rate** | 100% with automation | [Jellyfish case study](https://jellyfish.co/library/software-capitalization-benefits/) |

**Your Target** (recommended):
- KTLO: 25% (healthy maintenance)
- New Features: 60% (innovation-focused)
- Management: 15% (process & planning)
- Expected Capitalization Rate: **50-60%** (60% features × 100% capitalizable)

---

## References

### Accounting Standards
- [FASB ASU 2025-06: Software Cost Accounting Updates](https://dart.deloitte.com/USDART/home/publications/deloitte/heads-up/2025/fasb-asu-amends-software-costs-guidance)
- [ASC 350-40: Internal-Use Software Accounting](https://finquery.com/blog/asc-350-internal-use-software-accounting-fasb/)
- [US GAAP Software Capitalization Guidance](https://www.eisneramper.com/insights/technology/capitalizing-internal-use-software-0123/)
- [KPMG: FASB Software Cost Accounting](https://kpmg.com/us/en/frv/reference-library/2025/fasb-issues-final-asu-on-software-cost-accounting.html)

### Industry Approaches
- [Jellyfish: Software Capitalization Benefits](https://jellyfish.co/library/software-capitalization-benefits/)
- [Jellyfish: Capitalize External-Use Software](https://jellyfish.co/library/external-use-software-capitalization/)
- [Swarmia: Capitalize Software Development Costs (Docs)](https://help.swarmia.com/capitalize-software-development-costs)
- [Swarmia: Hitchhiker's Guide to Capitalizing Costs](https://www.swarmia.com/blog/capitalizing-software-development-costs/)
- [DX: Mastering Software Capitalization for Engineering Leaders](https://getdx.com/blog/software-capitalization/)
- [Waydev: Understanding Capitalization of Software Development](https://waydev.co/software-capitalization/)

### AI Impact
- [Jellyfish: AI Impact Dashboard](https://jellyfish.co/platform/jellyfish-ai-impact/)
- [Jellyfish: AI Use in Engineering Up 260% YoY](https://jellyfish.co/blog/ai-impact-data-june-2025/)
- [Jellyfish: AI-Assisted PRs Are 18% Larger](https://jellyfish.co/blog/ai-assisted-pull-requests-are-18-larger/)

### KTLO Best Practices
- [Uplevel: KTLO in Software Development - A Leader's Guide](https://uplevelteam.com/blog/ktlo-in-software-development)
- [OpenReplay: KTLO Explained - Metrics and Best Practices](https://blog.openreplay.com/ktlo-explained-metrics-and-best-practices-for-software-teams/)
- [Axify: What Is KTLO? Keeping the Lights On Explained](https://axify.io/blog/what-is-ktlo)

### DevLake Development
- [DevLake Plugin Development Guide](https://devlake.apache.org/docs/DeveloperManuals/PluginImplementation)

---

**Document Version**: 1.0
**Last Updated**: 2026-01-28
**Next Review**: Weekly during implementation
