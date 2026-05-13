# Getting Started - DevLake Extension Prerequisites

This document outlines everything that needs to happen **before writing any code** for the Engineering Management Platform extension.

---

## Phase 0: Discovery & Setup (Week 0)

### Task 1: Understand Current Jira Structure

**Goal**: Figure out how business goals are currently tracked so we can extract them automatically.

**Action Items**:
1. [ ] Log into your Jira instance
2. [ ] Navigate to a few Epic issues
3. [ ] Document:
   - What fields exist on Epics? (Screenshot or list custom fields)
   - How are business goals indicated? (Epic name format? Labels? Custom field?)
   - Example Epics:
     ```
     Epic Key: PROJ-123
     Epic Name: "Mobile Redesign Q1 2026"
     Labels: ["mobile", "q1-2026", "goal:revenue"]
     Custom Fields: ?
     ```

**Questions to Answer**:
- [ ] Is there a "Business Goal" custom field on Epics?
- [ ] Do Epic labels follow a pattern? (e.g., `goal:revenue`, `goal:efficiency`)
- [ ] How do you know if an Epic is for Q1 vs Q2? (Date field? Label? Name convention?)
- [ ] What Epic types exist? (Are all strategic initiatives Epics? Or are some Stories?)

**Deliverable**: Document in this format:
```
Epic Structure:
- Business goal indicated by: [label "goal:*" / custom field "Business Goal" / Epic name prefix]
- Fiscal quarter indicated by: [label "q1-2026" / custom field "Target Quarter" / Epic name suffix]
- Example Epic:
  - Key: PROJ-123
  - Name: Mobile Redesign Q1 2026
  - Labels: goal:revenue, mobile, q1-2026
  - Custom Field "Strategic Goal": Revenue Growth
```

---

### Task 2: Export Developer Hourly Rates

**Goal**: Get the data needed for cost calculations.

**Action Items**:
1. [ ] Create CSV file: `hourly_rates.csv`
2. [ ] Format:
   ```csv
   developer_email,hourly_rate,cost_center,effective_date
   dev1@company.com,120.00,Engineering,2026-01-01
   dev2@company.com,135.00,Engineering,2026-01-01
   dev3@company.com,110.00,Engineering,2026-01-01
   ```
3. [ ] Include ALL ~20 developers on the team
4. [ ] Store in `/backend/config/` (do NOT commit to Git - add to `.gitignore`)

**Questions to Answer**:
- [ ] Are rates the same for all developers? (If yes, can use a default rate)
- [ ] Should we include contractor rates separately? (If yes, add cost_center column)
- [ ] Do rates change throughout the year? (If yes, need effective_date)

**Deliverable**: CSV file with all developer rates

---

### Task 3: Define Capitalization Rules

**Goal**: Establish clear rules for what work is capitalizable (for US GAAP ASC 350-40 compliance).

**Action Items**:
1. [ ] Meet with finance team or engineering leadership
2. [ ] Categorize work types:

**Basic Framework** (start with this, adjust as needed):
```yaml
US GAAP ASC 350-40 Categorization:

CAPITALIZABLE (Development Stage):
- Issue Types: Story, New Feature, Enhancement
- Exclude Labels: research, spike, proof-of-concept
- Rationale: Building new functionality = asset

EXPENSE (Preliminary Stage):
- Labels: research, spike, investigation, feasibility, discovery
- Rationale: Research phase, not production-ready

EXPENSE (Post-Implementation):
- Issue Types: Bug, Defect, Hotfix
- Labels: maintenance, support, bug-fix
- Rationale: Keeping existing system running

EXPENSE (Operations):
- Labels: support, customer-help, ops, incident
- Rationale: Day-to-day operations, not development
```

3. [ ] Review with finance team: "Does this match our accounting policy?"
4. [ ] Document any special cases:
   - [ ] Are all Epics capitalizable?
   - [ ] Are refactoring tasks capitalizable or expense?
   - [ ] Are test-only stories capitalizable?

**Deliverable**: Finalized categorization rules document

---

### Task 4: Review DevLake Data Availability

**Goal**: Confirm existing DevLake installation has the data we need.

**Action Items**:
1. [ ] Access DevLake database (ask DevOps for credentials)
2. [ ] Run these queries to check data:

```sql
-- Check if Jira data exists
SELECT COUNT(*) FROM _tool_jira_board_issues WHERE type = 'Epic';
-- Expected: Should see your Jira Epics

-- Check if GitHub/GitLab data exists
SELECT COUNT(*) FROM pull_requests WHERE merged_date >= DATE_SUB(NOW(), INTERVAL 3 MONTH);
-- Expected: Should see recent PRs

-- Check if commits are linked to issues
SELECT COUNT(*) FROM issue_commits;
-- Expected: Should see linkages

-- Check if PRs are linked to issues
SELECT COUNT(*) FROM pull_request_issues;
-- Expected: Should see linkages
```

3. [ ] Document any gaps:
   - [ ] Are Epics being collected?
   - [ ] Are commits linked to Jira issues?
   - [ ] How far back does data go? (Need at least 3 months for velocity trends)

**Questions to Answer**:
- [ ] Is DevLake already running and collecting data? (Or do we need to set it up first?)
- [ ] What data sources are configured? (Jira? GitHub? GitLab? Both?)
- [ ] When was last successful data collection? (Check for staleness)

**Deliverable**: Data availability report with any gaps noted

---

### Task 5: Set Up Development Environment

**Goal**: Get a local DevLake instance running for testing.

**Action Items**:
1. [ ] Clone DevLake repo (you've already done this)
2. [ ] Install dependencies:
   ```bash
   cd /Users/zvikabadalov/workspaces/dev/incubator-devlake/backend
   make go-dep    # Install Go tools
   make dep       # Install dependencies
   ```
3. [ ] Set up local database:
   ```bash
   # Start PostgreSQL via Docker
   docker run -d --name devlake-postgres \
     -e POSTGRES_PASSWORD=devlake \
     -e POSTGRES_DB=devlake \
     -p 5432:5432 \
     postgres:13
   ```
4. [ ] Configure connection to your Jira/GitHub:
   ```bash
   # Create .env file
   cp backend/env.example backend/.env
   # Edit .env to add your Jira/GitHub credentials
   ```
5. [ ] Run DevLake locally:
   ```bash
   cd backend
   make build    # Build plugins and server
   make run      # Start server
   ```
6. [ ] Verify: Visit `http://localhost:8080` and configure a connection

**Deliverable**: Working local DevLake instance

---

### Task 6: Validate AI Detection Approach

**Goal**: Test that AI pattern detection will work with your team's data.

**Action Items**:
1. [ ] Manually review 20 recent PRs:
   - 10 you KNOW used AI tools (Cursor, Claude)
   - 10 you KNOW didn't use AI
2. [ ] For each PR, note:
   - Number of commits
   - Time between commits
   - PR size (lines added)
   - Cycle time (first commit → merge)
3. [ ] Look for patterns:
   - [ ] Are AI-assisted PRs faster? (Shorter cycle time?)
   - [ ] Are AI-assisted PRs larger? (More lines changed?)
   - [ ] Do AI-assisted PRs have more rapid commits? (< 15 min between commits?)

**Example Analysis**:
```
PR #1234 (AI-assisted with Cursor):
- Commits: 5
- Time between commits: 8 min average
- PR size: 450 lines added
- Cycle time: 2.5 hours
- Pattern: Rapid commits, large PR

PR #1235 (Non-AI):
- Commits: 3
- Time between commits: 45 min average
- PR size: 180 lines added
- Cycle time: 4 hours
- Pattern: Slower commits, smaller PR
```

**Deliverable**: Validation report confirming AI detection is feasible

---

## Phase 0.5: Create Project Structure (Week 0)

Once prerequisites are complete, set up the directory structure:

### Task 7: Create Plugin Directories

**Action Items**:
```bash
cd /Users/zvikabadalov/workspaces/dev/incubator-devlake/backend

# Create businessmetrics plugin
mkdir -p plugins/businessmetrics/{impl,api,tasks,models/migrationscripts,e2e}
touch plugins/businessmetrics/impl/impl.go
touch plugins/businessmetrics/tasks/extract_business_goals.go
touch plugins/businessmetrics/tasks/calculate_alignment.go

# Create aidetector plugin
mkdir -p plugins/aidetector/{impl,tasks,models/migrationscripts,e2e}
touch plugins/aidetector/impl/impl.go
touch plugins/aidetector/tasks/analyze_commit_patterns.go
touch plugins/aidetector/tasks/analyze_pr_characteristics.go
touch plugins/aidetector/tasks/score_ai_confidence.go

# Create findevops plugin
mkdir -p plugins/findevops/{impl,api,tasks,models/migrationscripts,e2e}
touch plugins/findevops/impl/impl.go
touch plugins/findevops/api/cost_reports.go
touch plugins/findevops/tasks/calculate_costs.go
touch plugins/findevops/tasks/categorize_capitalization.go

# Create capacityplanner plugin
mkdir -p plugins/capacityplanner/{impl,api,tasks,models/migrationscripts,e2e}
touch plugins/capacityplanner/impl/impl.go
touch plugins/capacityplanner/api/forecasting.go
touch plugins/capacityplanner/tasks/calculate_velocity.go
touch plugins/capacityplanner/tasks/forecast_completion.go

# Create domain models
mkdir -p core/models/domainlayer
touch core/models/domainlayer/businessinitiative.go
touch core/models/domainlayer/ai_usage_signal.go
touch core/models/domainlayer/cost_allocation.go
touch core/models/domainlayer/capacity_metric.go
```

---

## Decision Log

As you answer the questions above, document decisions here:

### Business Goal Tracking Decision
**Date**: ___________
**Decision**: Business goals are tracked in Jira via [Epic labels "goal:*" / custom field "Business Goal" / other]
**Format**:
```
Example Epic: PROJ-123
- Business Goal: [field value or label]
- Fiscal Quarter: [field value or label]
```

### Capitalization Rules Decision
**Date**: ___________
**Decision**: Approved by [Finance Lead Name]
**Rules**: See Task 3 above (link to finalized rules document)

### AI Detection Threshold Decision
**Date**: ___________
**Decision**: AI confidence score threshold set to [70%] based on validation testing
**Reasoning**: [From Task 6 - validation showed X% accuracy at this threshold]

### Hourly Rate Approach Decision
**Date**: ___________
**Decision**:
- [ ] Use individual developer rates (CSV with 20 rows)
- [ ] Use team average rate (single value: $____/hour)
- [ ] Use role-based rates (Senior: $___, Mid: $___, Junior: $___)

---

## Next Steps After Prerequisites

Once all tasks above are complete:

1. **Week 1**: Start coding `businessmetrics` plugin
   - Follow implementation plan in `DEVLAKE_EXTENSION_PLAN.md` Section 1.1
   - Begin with domain model and migrations
   - Test with real Jira data from your instance

2. **Week 3**: Start coding `aidetector` plugin
   - Implement pattern analysis
   - Tune scoring algorithm with your team's PRs
   - Validate accuracy against manual review

3. **Week 5**: Start coding `findevops` and `capacityplanner` plugins
   - Use hourly rates from Task 2
   - Apply capitalization rules from Task 3
   - Generate first cost report

---

## Contact & Questions

As you work through prerequisites, track questions here:

**Questions for Engineering Leadership**:
- [ ] Question 1: ___________
- [ ] Question 2: ___________

**Questions for Finance Team**:
- [ ] Question 1: ___________
- [ ] Question 2: ___________

**Blockers**:
- [ ] Blocker 1: ___________
- [ ] Blocker 2: ___________

---

**Document Version**: 1.0
**Last Updated**: 2026-01-28
**Status**: 🔴 Prerequisites in progress
