# Dashboard Algorithms & Metrics Guide

**Version:** 1.0
**Last Updated:** 2026-01-29
**Purpose:** User-friendly explanation of all dashboard calculations

---

## Table of Contents

1. [AI-Assisted Development Dashboard](#ai-assisted-development-dashboard)
2. [Business Alignment & Team Health Dashboard](#business-alignment--team-health-dashboard)
3. [Capacity Planning & Forecasting Dashboard](#capacity-planning--forecasting-dashboard)
4. [FinDevOps - Cost & Capitalization Dashboard](#findevops---cost--capitalization-dashboard)

---

## AI-Assisted Development Dashboard

### 📊 Metrics Explained

#### **AI Confidence Score** (0-100%)
**What it measures:** How likely a PR was created with AI assistance

**How it's calculated:**
```
Base signals (each worth points):
- Explicit markers (Co-authored-by, #cursor, #claude-code) → High confidence
- PR characteristics (large changes, quick reviews) → Medium confidence
- Code patterns (AI-style comments, formatting) → Low confidence

Final Score = Sum of signal weights / Total possible points × 100
```

**Confidence Levels:**
- **High (≥70%):** Strong evidence of AI usage
- **Medium (40-69%):** Likely AI-assisted
- **Low (<40%):** Possibly human-written or minimal AI

#### **Code Churn** (Percentage)
**What it measures:** How much code gets modified after initial merge

**Formula:**
```
Churn Rate = (Lines Added + Lines Deleted in Follow-up Commits) / Total Lines × 100
```

**How we calculate it:**
1. Track initial PR merge (baseline)
2. Monitor all changes to those files for 30 days
3. Count additions/deletions as churn
4. Compare AI vs Non-AI PRs

**Healthy Range:**
- **Low Churn (<50%):** Stable, high-quality code
- **Medium Churn (50-100%):** Some refinement needed
- **High Churn (>100%):** May indicate quality issues

**Research:**
- GitClear study found AI code has ~41% higher churn
- Lower churn = more stable, maintainable code

---

## Business Alignment & Team Health Dashboard

### 📊 Team Health Score (0-100)

**Purpose:** Measure engineering team performance against DORA elite benchmarks

**Formula:**
```
Total Score = Deploy Freq Score + Lead Time Score + CFR Score + MTTR Score
(Max 25 points each = 100 total)
```

#### **Component Calculations:**

| Metric | Elite Benchmark | Score Formula | Max Points |
|--------|----------------|---------------|------------|
| **Deploy Frequency** | 1/day | `min(25, (actual_freq / 1.0) × 25)` | 25 |
| **Lead Time** | 24 hours | `min(25, (24.0 / actual_hours) × 25)` | 25 |
| **Change Failure Rate** | 5% | `min(25, (5.0 / actual_cfr%) × 25)` | 25 |
| **MTTR** | 1 hour | `min(25, (1.0 / actual_hours) × 25)` | 25 |

#### **Health Levels:**

| Score Range | Level | Interpretation |
|------------|-------|----------------|
| **≥80** | 🟢 Elite | World-class performance |
| **60-79** | 🟡 High | Strong performance |
| **40-59** | 🟠 Medium | Room for improvement |
| **<40** | 🔴 Low | Needs attention |

#### **Example Calculation:**

If your team has:
- Deploy Freq: **7.5/day** → Score: 25 (maxed out, 7.5 > 1.0)
- Lead Time: **3.27 hours** → Score: 25 (maxed out, better than 24h)
- CFR: **17.95%** → Score: 6.98 ≈ **7** (5 / 17.95 × 25)
- MTTR: **4 hours** → Score: 6.25 ≈ **6** (1 / 4 × 25)

**Total:** 25 + 25 + 7 + 6 = **63 points** → 🟡 **High** health level

---

### 📊 Business Value Score (0-100)

**Purpose:** Quantify business impact of engineering work

**Formula:**
```
Score = 50 (base) + Revenue Impact Weight + Efficiency Bonus
```

#### **Revenue Impact Weights:**

| Category | Description | Weight | Example |
|----------|-------------|--------|---------|
| **Direct** | Generates revenue | +30 | Payment processing, checkout |
| **Enabling** | Enables revenue | +20 | Analytics, A/B testing |
| **Supporting** | Supports business | +10 | Admin tools, reporting |
| **Cost Center** | Pure cost | +0 | Internal tooling |

#### **Efficiency Bonuses:**

| Revenue Type | Efficiency Ratio | Bonus |
|-------------|-----------------|-------|
| Direct | Story Points / Budget ≥ 5 | +20 |
| Enabling | Story Points / Budget ≥ 2 | +15 |
| Supporting | Story Points / Budget ≥ 1 | +10 |
| Cost Center | Story Points / Budget > 0 | +5 |

#### **Example:**

Project: "Mobile Checkout Redesign"
- Revenue Impact: **Direct** → +30
- Story Points: 89, Budget: 13 weeks
- Efficiency Ratio: 89/13 = 6.85 (≥5) → +20

**Score:** 50 + 30 + 20 = **100 points** (maximum)

---

## Capacity Planning & Forecasting Dashboard

### 📊 Monte Carlo Simulation

**Purpose:** Probabilistic project completion forecasting

**How it works:**

1. **Input Data:**
   - Historical sprint velocities (last 6 sprints)
   - Remaining story points
   - Velocity variance (default: 25%)

2. **Simulation Process:**
   ```
   Run 1000 iterations:
     For each iteration:
       remaining_points = total - completed
       days = 0

       while remaining_points > 0:
         # Simulate velocity with random variance
         velocity = random_gaussian(avg_velocity, stddev × 0.25)
         remaining_points -= max(1, velocity)
         days += 7  # One sprint

       record(completion_days)
   ```

3. **Output Percentiles:**
   - **P50 (50th percentile):** 50% chance of finishing by this date
   - **P90 (90th percentile):** 90% confidence (recommended for planning)
   - **P95 (95th percentile):** Very conservative estimate

#### **Example:**

Project with 120 remaining story points, average velocity = 15/sprint:

```
P50: 8 sprints (56 days)   → 50% confident
P90: 11 sprints (77 days)  → 90% confident (use this!)
P95: 12 sprints (84 days)  → 95% confident
```

**Best Practice:** Use P90 for commitments, P50 for internal planning

---

### 📊 Brooks's Law Model

**Purpose:** Calculate impact of team size changes on productivity

**The Law:**
> "Adding manpower to a late software project makes it later"
> — Fred Brooks, The Mythical Man-Month

**Formula:**
```
Communication Channels = n × (n - 1) / 2

Where n = team size
```

**Team Size Impact:**

| Team Size | Channels | Communication Overhead |
|-----------|----------|----------------------|
| 3 | 3 | Low |
| 5 | 10 | Moderate |
| 7 | 21 | High |
| 10 | 45 | Very High |

**Hiring Impact:**
```
New hire productivity during ramp-up (8 weeks):
- Weeks 1-4: 25% productive
- Weeks 5-8: 50% productive
- Week 9+: 100% productive

Overhead cost:
overhead_factor = 1 - (channel_delta / (current_channels + 1)) × 0.1
effective_capacity = base_capacity × overhead_factor
```

**Example:**

Team of 5 adding 2 new members:
- Current channels: 10
- New channels: 21
- Channel delta: +11
- Overhead: 1 - (11 / 11) × 0.1 = 0.90 (10% productivity loss)
- First 8 weeks: New hires at 50% avg + 10% team overhead

**Lesson:** Hiring doesn't immediately increase velocity!

---

### 📊 ROI Calculation (AI Tools)

**Purpose:** Justify investment in AI coding tools

**Formula:**
```
Annual Benefit = Direct + Productivity + Quality

Direct Benefit = hours_saved/week × team_size × 52 × hourly_cost

Productivity Benefit = team_hours × (gain%/100) × hourly_cost

Quality Benefit = team_hours × 20% × (improvement%/100) × hourly_cost

ROI = ((benefit × 3) - (cost × 3)) / (cost × 3) × 100
```

**Assumptions:**
- Hourly cost: $87 (default, adjust for your team)
- Quality improvement: Assumed 20% of development time saved from fewer bugs
- 3-year timeframe for ROI

**Example:**

Team: 10 developers, AI tool saves 5 hours/week/dev
- Direct: 5 × 10 × 52 × $87 = **$226,200/year**
- Productivity gain (10%): 2080 hours × 10% × $87 = **$18,096/year**
- Quality improvement (5%): 2080 hours × 20% × 5% × $87 = **$1,808/year**

**Total Annual Benefit:** $246,104
**Tool Cost:** $30/dev/month × 10 × 12 = $3,600/year

**3-Year ROI:** (($246K × 3) - ($3.6K × 3)) / ($3.6K × 3) × 100 = **6,745%**

---

## FinDevOps - Cost & Capitalization Dashboard

### 📊 ASC 350-40 Categorization

**Purpose:** Comply with US GAAP accounting for software capitalization

**The Rule:**
> Only development stage costs can be capitalized. Preliminary and post-implementation costs must be expensed.

**Formula:**
```
Total Cost = Hours Worked × Hourly Rate

Capitalizable Cost = Sum of "development" phase costs
Expensed Cost = Sum of "preliminary" + "post_implementation" costs

Capitalization Rate = (Capitalizable / Total) × 100
```

---

### 📊 Three Development Stages

#### **1. Preliminary Stage** (❌ NOT Capitalizable)

**Accounting Treatment:** **Expense immediately**

**What counts:**
- ✓ Planning and feasibility studies
- ✓ Architecture and design
- ✓ Proof-of-concept (POC)
- ✓ Research spikes
- ✓ Requirements gathering
- ✓ Technology evaluation

**How we detect:**
- Issue Type: `Spike`, `Research`
- Labels: `planning`, `design`, `poc`, `proof-of-concept`, `feasibility`, `discovery`, `research`

**Example Issues:**
- "Research authentication options for new API"
- "Design microservices architecture for payment system"
- "POC: Evaluate Redis vs Memcached for caching"

---

#### **2. Development Stage** (✅ CAPITALIZABLE)

**Accounting Treatment:** **Capitalize and amortize**

**What counts:**
- ✓ Coding and implementation
- ✓ New features and functionality
- ✓ Database schema creation
- ✓ API development
- ✓ Testing new functionality
- ✓ Integration work
- ✓ Enhancements to existing features

**How we detect:**
- Issue Type: `Story`, `Task`, `Enhancement`, `Feature` (default)
- Labels: `feature`, `enhancement`, `implementation`
- **This is the default** - any work that doesn't match preliminary or post-implementation

**Example Issues:**
- "Implement OAuth2 login flow"
- "Add payment processing API integration"
- "Create user dashboard with analytics"
- "Build notification system"

---

#### **3. Post-Implementation Stage** (❌ NOT Capitalizable)

**Accounting Treatment:** **Expense immediately**

**What counts:**
- ✓ Bug fixes and defects
- ✓ Maintenance and support
- ✓ Training and documentation updates
- ✓ Performance tuning (unless substantial upgrade)
- ✓ Security patches
- ✓ Hotfixes

**How we detect:**
- Issue Type: `Bug`, `Defect`
- Labels: `maintenance`, `support`, `training`, `hotfix`, `patch`, `bugfix`

**Example Issues:**
- "Fix login timeout after 30 minutes"
- "Update SSL certificate"
- "Patch security vulnerability in auth library"
- "Fix memory leak in API endpoint"

---

### 📊 Cost Per Deployment

**Purpose:** Measure delivery efficiency

**Formula:**
```
Cost Per Deployment = Total Cost (time window) / Deployment Count (time window)
```

**Time Windows:**
- **7-day (weekly):** Short-term efficiency
- **30-day (monthly):** Monthly trends
- **90-day (quarterly):** Long-term patterns

**How it's calculated:**

1. **Sum all development costs in window:**
   ```sql
   SUM(cost_allocations.total_cost)
   WHERE calculated_at BETWEEN window_start AND window_end
   ```

2. **Count successful deployments:**
   ```sql
   COUNT(cicd_deployment_commits)
   WHERE result = 'SUCCESS'
   AND finished_date BETWEEN window_start AND window_end
   ```

3. **Divide:**
   ```
   Cost Per Deploy = Total Cost / Deployment Count
   ```

**Example:**

90-day window:
- Total Cost: $64,000
- Deployments: 541
- **Cost Per Deploy:** $64,000 / 541 = **$118.30**

**Interpretation:**
- **Lower is better** - more efficient delivery pipeline
- Compare across time windows to spot trends
- Track month-over-month to measure improvement

**Good Benchmarks:**
- **<$500/deploy:** Efficient continuous delivery
- **$500-$2000/deploy:** Moderate efficiency
- **>$2000/deploy:** May indicate deployment bottlenecks

---

### 📊 Budget Variance Tracking

**Purpose:** Compare estimated vs actual costs

**Formula:**
```
Variance = (Estimated - Actual) / Estimated × 100

Positive variance = Under budget (good!)
Negative variance = Over budget (investigate)
```

**Data Sources:**
- **Estimated:** Jira `Original Estimate` field (minutes)
- **Actual:** Jira `Time Spent` field (minutes)

**Example:**

Issue: "Implement payment API"
- Estimated: 40 hours (2400 minutes)
- Actual: 52 hours (3120 minutes)
- **Variance:** (2400 - 3120) / 2400 × 100 = **-30%** (30% over budget)

**Dashboard Metrics:**
- **Total Estimated Cost:** Sum of all estimates × hourly rate
- **Total Actual Cost:** Sum of all actuals × hourly rate
- **Budget Variance:** Overall percentage
- **Over Budget Issue Count:** # of issues exceeding estimate

---

## 📖 Quick Reference

### Key Formulas at a Glance

| Dashboard | Metric | Formula |
|-----------|--------|---------|
| **AI Detection** | Confidence Score | Signal weights / Total × 100 |
| **AI Detection** | Code Churn | (Additions + Deletions) / Total Lines × 100 |
| **Team Health** | Total Score | Deploy + LeadTime + CFR + MTTR (max 100) |
| **Business Value** | Value Score | 50 + Revenue Weight + Efficiency Bonus |
| **Capacity** | Monte Carlo | 1000 simulations with 25% variance |
| **Capacity** | Brooks's Law | n × (n-1) / 2 channels |
| **Capacity** | ROI | ((Benefit×3) - (Cost×3)) / (Cost×3) × 100 |
| **FinDevOps** | Capitalization Rate | Capitalizable / Total × 100 |
| **FinDevOps** | Cost Per Deploy | Total Cost / Deployment Count |

---

## 🎯 Best Practices

1. **Use P90 (90th percentile)** for project commitments, not P50
2. **Team Health Score ≥60** is good, ≥80 is elite
3. **Capitalization Rate 50-70%** is typical for feature-heavy teams
4. **Lower Cost Per Deploy** indicates efficient CI/CD
5. **Code Churn <50%** suggests high-quality initial code
6. **Monitor trends over time** - single snapshots can be misleading

---

## 📞 Questions?

- **ASC 350-40 Compliance:** Consult your finance team for interpretation
- **DORA Benchmarks:** See [DORA State of DevOps Reports](https://dora.dev)
- **Monte Carlo Method:** Based on software estimation research by Mike Cohn
- **Brooks's Law:** From "The Mythical Man-Month" by Frederick Brooks

---

**Document Location:** `/docs/DASHBOARD_ALGORITHMS_GUIDE.md`
**Related:** `/docs/BUG_FIXES_DASHBOARD_METRICS.md` (technical implementation)
