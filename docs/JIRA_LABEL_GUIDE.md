# Jira Label Guide for Engineering Metrics

**Start using these labels NOW** - DevLake will automatically detect and categorize your work for business alignment and cost capitalization reporting.

---

## Required Labels (Add to Epics & Stories)

### 1. **Investment Category** (What type of work is this?)

Add ONE of these labels to every Epic/Story:

| Label | Description | When to Use |
|-------|-------------|-------------|
| `investment:business` | New features, revenue-generating work | Building new product capabilities |
| `investment:ktlo` | Keep The Lights On - maintenance | Bug fixes, routine maintenance, keeping systems running |
| `investment:platform` | Infrastructure improvements | Platform upgrades, DevOps, tooling |
| `investment:techdebt` | Technical debt reduction | Refactoring, code cleanup, architecture improvements |
| `investment:support` | Production support | Incident response, customer support escalations |
| `investment:rd` | Research & Development | Spikes, feasibility studies, prototypes |

### 2. **Development Stage** (For capitalization compliance)

Add ONE of these labels to every Epic/Story:

| Label | Description | Capitalizable? | When to Use |
|-------|-------------|----------------|-------------|
| `stage:development` | Building NEW functionality | ✅ 100% | Adding features that didn't exist before |
| `stage:maintenance` | Maintaining existing features | ❌ 0% | Bug fixes, performance tweaks, keeping things working |
| `stage:research` | Planning and feasibility | ❌ 0% | Spikes, research, POCs, investigations |

**Key Rule**: `stage:development` = **NEW functionality ONLY**. If it's not adding new capabilities, it's maintenance.

---

## Optional Labels (Recommended for Better Insights)

### 3. **Business Goal** (Strategic alignment)

| Label | Description |
|-------|-------------|
| `goal:revenue` | Drives revenue growth |
| `goal:efficiency` | Improves operational efficiency |
| `goal:compliance` | Regulatory or compliance requirement |
| `goal:innovation` | Innovation initiative |
| `goal:customer-retention` | Improves customer retention |

### 4. **Strategic Theme** (Quarterly initiatives)

Examples:
- `theme:mobile-redesign-q1`
- `theme:api-modernization`
- `theme:platform-migration`

### 5. **Fiscal Period**

Examples:
- `q:2026-q1`
- `q:2026-q2`

---

## Label Examples

### Example 1: New Feature Development
```
Epic: "Mobile App Redesign"
Labels:
  - investment:business
  - stage:development
  - goal:revenue
  - theme:mobile-redesign-q1
  - q:2026-q1

Result: 100% capitalizable, counts toward Business investment
```

### Example 2: Bug Fix
```
Story: "Fix login authentication error"
Labels:
  - investment:ktlo
  - stage:maintenance

Result: 0% capitalizable (expense), counts toward KTLO
```

### Example 3: Platform Upgrade (New Functionality)
```
Epic: "Add OAuth2 Authentication"
Labels:
  - investment:platform
  - stage:development
  - goal:efficiency

Result: 100% capitalizable (adding NEW auth method)
```

### Example 4: Platform Maintenance
```
Story: "Update PostgreSQL to v15"
Labels:
  - investment:platform
  - stage:maintenance

Result: 0% capitalizable (maintaining existing infrastructure)
```

### Example 5: Research Spike
```
Spike: "Investigate Redis caching options"
Labels:
  - investment:rd
  - stage:research

Result: 0% capitalizable (preliminary research)
```

---

## Automatic Detection (If You Forget Labels)

DevLake will automatically infer labels based on issue type:

| Issue Type | Auto-inferred Stage | Auto-inferred Investment |
|------------|---------------------|--------------------------|
| Bug, Hotfix | `stage:maintenance` | `investment:ktlo` |
| Story, Feature | `stage:development` | `investment:business` |
| Spike, Research | `stage:research` | `investment:rd` |

**But it's better to add labels explicitly for accuracy!**

---

## How to Add Labels in Jira

1. Open an Epic or Story
2. Click the "Labels" field
3. Type the label (e.g., `investment:business`)
4. Press Enter
5. Add the stage label (e.g., `stage:development`)
6. Save

---

## Best Practices

### ✅ Do This:
- Add at least TWO labels: one `investment:*` and one `stage:*`
- Be honest about `stage:development` - only use for NEW features
- Add labels to Epics (they'll cascade insights to child stories)
- Use consistent themes across quarters (e.g., `theme:mobile-*`)

### ❌ Don't Do This:
- Don't mix `investment:business` with `stage:maintenance` (contradictory)
- Don't use `stage:development` for bug fixes (that's maintenance)
- Don't forget labels entirely (auto-detection is less accurate)

---

## Dashboard Impact

Once DevLake collects your labeled issues, you'll see:

### Business Alignment Dashboard
- **Investment Distribution**: "45% Business, 30% KTLO, 15% Platform, 10% Tech Debt"
- **Work by Goal**: "60% Revenue initiatives, 25% Efficiency, 15% Compliance"
- **Theme Progress**: "Mobile Redesign: 75% complete"

### Cost Capitalization Dashboard
- **Monthly Costs**: "$50,000 total (60% capitalizable, 40% expense)"
- **Capitalization Rate**: "60% this quarter (up from 50% last quarter)"
- **Audit Report**: CSV export for finance team

### Capacity Planning Dashboard
- **Velocity by Category**: "Business work: 40 pts/sprint, KTLO: 15 pts/sprint"
- **Forecast**: "Platform Migration theme will complete in 6 weeks"

---

## Start Labeling Now!

1. **Backfill Recent Work** (last 3 months):
   - Add labels to recent Epics and Stories
   - Prioritize Epics (they roll up child story data)

2. **Train Your Team**:
   - Share this guide with the team
   - Make it part of sprint planning
   - Add label checklist to Definition of Done

3. **Monitor Adoption**:
   - DevLake will show "% of work with labels"
   - Aim for 80%+ labeled work

---

## Questions?

**Q: What if I'm not sure which stage to use?**
A: Ask: "Does this add NEW functionality?" If yes → `stage:development`. If no → `stage:maintenance`.

**Q: Can one Epic have multiple investment labels?**
A: No - choose the PRIMARY category. Most work fits cleanly into one bucket.

**Q: Do I need to label every story, or just Epics?**
A: Label both, but Epics are more important. DevLake aggregates child story data to the Epic level.

**Q: What about work that's 50% new features, 50% maintenance?**
A: Split into separate Epics/Stories if possible. Otherwise, use the PRIMARY purpose as the label.

---

**Once DevLake's new plugins are deployed, your labeled work will automatically generate dashboards and reports!**
