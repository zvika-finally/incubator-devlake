# Enhanced Metrics - Remaining Work Plan

## Overview

This document outlines the remaining work needed to complete the Enhanced Metrics implementation. The backend models, tasks, and migrations are complete. What remains is:

1. **Config-UI Settings Pages** - UI for configuring each plugin
2. **Config-UI Connection Pages** - UI for Cursor and Claude Code connections
3. **Optional Plugin Loading** - Make AI tool plugins load only when configured
4. **Unit Tests** - Test coverage for new functionality

---

## 1. Make AI Tool Plugins Optional

### Problem
The `cursor` and `claudecode` plugins should only be loaded/active when connections are configured. Users without Cursor Business or Claude Code Team shouldn't see errors.

### Implementation

#### 1.1 Plugin Registration Changes
**Files to modify:**
- `backend/server/services/init.go` (or wherever plugins are registered)

**Approach:** These plugins should be registered but not fail if no connections exist. The current implementation already supports this - connections are optional, and tasks only run when a connectionId is provided.

#### 1.2 Blueprint Integration (Optional)
**Files to create:**
- `backend/plugins/cursor/impl/impl.go` - Add `PluginBlueprintV200` interface if needed
- `backend/plugins/claudecode/impl/impl.go` - Add `PluginBlueprintV200` interface if needed

**Note:** Since these are data source plugins (not metric plugins), they're already opt-in via connections.

---

## 2. Config-UI Settings Pages

### 2.1 AI Detector Settings Page
**Route:** `/plugins/aidetector/settings`

**Files to create:**
- `config-ui/src/pages/plugins/aidetector/settings.tsx`
- `config-ui/src/pages/plugins/aidetector/index.tsx` (if not exists)

**Settings to expose:**
```typescript
interface AIDetectorSettings {
  confidenceThreshold: number;      // Slider: 0-100, default 70
  churnWindowDays: number;          // Number input: default 30
  enableChurnTracking: boolean;     // Toggle: default true
  excludeAuthors: string[];         // Text field: comma-separated bot accounts
}
```

**UI Components:**
- Slider for confidence threshold with explanation
- Number input for churn window
- Toggle for churn tracking
- Text area for excluded authors

---

### 2.2 Business Metrics Settings Page
**Route:** `/plugins/businessmetrics/settings`

**Files to create:**
- `config-ui/src/pages/plugins/businessmetrics/settings.tsx`

**Settings to expose:**
```typescript
interface BusinessMetricsSettings {
  investmentLabelPrefix: string;    // Text: default "investment:"
  stageLabelPrefix: string;         // Text: default "stage:"
  goalLabelPrefix: string;          // Text: default "goal:"
  // DORA thresholds
  eliteDeployFreq: number;
  eliteLeadTimeHours: number;
  eliteCFR: number;
  eliteMTTRHours: number;
}
```

---

### 2.3 Working Agreements Page
**Route:** `/plugins/businessmetrics/agreements/:projectName`

**Files to create:**
- `config-ui/src/pages/plugins/businessmetrics/agreements.tsx`
- `config-ui/src/pages/plugins/businessmetrics/agreements-list.tsx`
- `config-ui/src/components/working-agreement-form.tsx`

**UI Components:**
- Table listing current agreements with edit/delete
- Add Agreement modal/form
- Agreement type dropdown (pr_merge_time, review_turnaround, wip_limit, issues_in_progress)
- Threshold value input
- Unit selector (days, hours, count)
- Alert enabled toggle

**API Integration:**
- GET `/plugins/businessmetrics/agreements/:projectName`
- POST `/plugins/businessmetrics/agreements/:projectName`
- PUT `/plugins/businessmetrics/agreements/:projectName/:agreementType`
- DELETE `/plugins/businessmetrics/agreements/:projectName/:agreementType`
- GET `/plugins/businessmetrics/violations/:projectName`
- GET `/plugins/businessmetrics/compliance/:projectName`

---

### 2.4 Capacity Planner Settings Page
**Route:** `/plugins/capacityplanner/settings`

**Files to create:**
- `config-ui/src/pages/plugins/capacityplanner/settings.tsx`

**Settings to expose:**
```typescript
interface CapacityPlannerSettings {
  velocitySprintCount: number;      // Number: default 6
  sprintDurationWeeks: number;      // Number: default 2
  monteCarloIterations: number;     // Number: default 1000
  rampUpWeeks: number;              // Number: default 8 (Brooks's Law)
  // Flow efficiency active statuses
  activeStatuses: string[];         // Text area: JSON array
}
```

---

### 2.5 FinDevOps Settings Page
**Route:** `/plugins/findevops/settings`

**Files to create:**
- `config-ui/src/pages/plugins/findevops/settings.tsx`

**Settings to expose:**
```typescript
interface FinDevOpsSettings {
  defaultHourlyRate: number;        // Currency input: default 87.00
  capitalizationFramework: string;  // Dropdown: 'asc_350_40_stages' | 'asc_350_40_probable'
  fiscalYearStartMonth: number;     // Month selector: default 1
  unallocatedCostThreshold: number; // Percentage: default 10%
  hoursPerStoryPoint: number;       // Number: default 4.0
  // Label mappings
  preliminaryLabels: string[];
  postImplementationLabels: string[];
}
```

---

## 3. Config-UI Connection Pages

### 3.1 Cursor Connection Page
**Route:** `/connections/cursor`

**Files to create:**
- `config-ui/src/pages/connections/cursor/index.tsx`
- `config-ui/src/pages/connections/cursor/create.tsx`
- `config-ui/src/pages/connections/cursor/detail.tsx`

**Connection Form Fields:**
```typescript
interface CursorConnectionForm {
  name: string;                     // Text: required
  teamId: string;                   // Text: required, Cursor team ID
  apiKey: string;                   // Password field: required, masked
  rateLimitPerSecond: number;       // Number: default 5
}
```

**UI Requirements:**
- Standard DevLake connection list pattern
- Create/Edit form with validation
- Test connection button (optional - would need backend endpoint)
- API key masking in display

**API Integration:**
- GET `/plugins/cursor/connections`
- POST `/plugins/cursor/connections`
- GET `/plugins/cursor/connections/:connectionId`
- PATCH `/plugins/cursor/connections/:connectionId`
- DELETE `/plugins/cursor/connections/:connectionId`

---

### 3.2 Claude Code Connection Page
**Route:** `/connections/claudecode`

**Files to create:**
- `config-ui/src/pages/connections/claudecode/index.tsx`
- `config-ui/src/pages/connections/claudecode/create.tsx`
- `config-ui/src/pages/connections/claudecode/detail.tsx`

**Connection Form Fields:**
```typescript
interface ClaudeCodeConnectionForm {
  name: string;                     // Text: required
  organizationId: string;           // Text: required
  adminApiKey: string;              // Password field: required (sk-ant-admin-...)
  rateLimitPerSecond: number;       // Number: default 5
}
```

---

## 4. Navigation and Routing Updates

### 4.1 Add Plugin Routes
**File to modify:** `config-ui/src/routes/index.tsx` (or equivalent)

Add routes for:
- `/plugins/aidetector/settings`
- `/plugins/businessmetrics/settings`
- `/plugins/businessmetrics/agreements/:projectName`
- `/plugins/capacityplanner/settings`
- `/plugins/findevops/settings`

### 4.2 Add Connection Routes
**File to modify:** `config-ui/src/routes/index.tsx`

Add routes for:
- `/connections/cursor/*`
- `/connections/claudecode/*`

### 4.3 Update Navigation Menu
**File to modify:** `config-ui/src/components/sidebar.tsx` (or equivalent)

Add menu items for new plugins under Connections section:
- Cursor (with Cursor logo)
- Claude Code (with Anthropic logo)

---

## 5. Unit Tests

### 5.1 Backend Tests

**Files to create:**

#### FinDevOps Tests
- `backend/plugins/findevops/e2e/calculate_costs_test.go`
  - Test unallocated cost detection
  - Test budget variance calculation
  - Test monthly summary aggregation

#### BusinessMetrics Tests
- `backend/plugins/businessmetrics/e2e/check_agreements_test.go`
  - Test PR merge time violation detection
  - Test WIP limit calculation
  - Test compliance summary generation

#### CapacityPlanner Tests
- `backend/plugins/capacityplanner/e2e/calculate_flow_efficiency_test.go`
  - Test flow efficiency calculation from status transitions
  - Test category assignment (excellent/good/average/poor)
  - Test percentile calculations

#### AIDetector Tests
- `backend/plugins/aidetector/e2e/analyze_code_churn_test.go`
  - Test churn calculation after merge
  - Test AI vs non-AI comparison
  - Test churn ratio computation

#### Cursor Plugin Tests
- `backend/plugins/cursor/e2e/collect_metrics_test.go`
  - Test API response parsing
  - Test metric storage
  - Mock API responses

#### Claude Code Plugin Tests
- `backend/plugins/claudecode/e2e/collect_metrics_test.go`
  - Test API response parsing
  - Test metric storage
  - Mock API responses

---

## 6. Implementation Order

### Phase 1: Make Plugins Production-Ready (1 day)
1. Verify cursor/claudecode plugins load without errors when no connections exist
2. Add proper error handling for API failures
3. Add rate limiting support

### Phase 2: Settings Pages (2 days)
1. AI Detector settings page
2. Business Metrics settings page
3. Working Agreements page
4. Capacity Planner settings page
5. FinDevOps settings page

### Phase 3: Connection Pages (1.5 days)
1. Cursor connection page
2. Claude Code connection page
3. Navigation updates

### Phase 4: Unit Tests (1 day)
1. Backend unit tests for new calculations
2. API response parsing tests

### Phase 5: Integration Testing (0.5 day)
1. End-to-end testing with real data
2. Dashboard verification

---

## 7. File Summary

### New Config-UI Files (~15 files)
```
config-ui/src/pages/plugins/
├── aidetector/
│   └── settings.tsx
├── businessmetrics/
│   ├── settings.tsx
│   ├── agreements.tsx
│   └── agreements-list.tsx
├── capacityplanner/
│   └── settings.tsx
└── findevops/
    └── settings.tsx

config-ui/src/pages/connections/
├── cursor/
│   ├── index.tsx
│   ├── create.tsx
│   └── detail.tsx
└── claudecode/
    ├── index.tsx
    ├── create.tsx
    └── detail.tsx

config-ui/src/components/
└── working-agreement-form.tsx
```

### New Backend Test Files (~6 files)
```
backend/plugins/
├── findevops/e2e/calculate_costs_test.go
├── businessmetrics/e2e/check_agreements_test.go
├── capacityplanner/e2e/calculate_flow_efficiency_test.go
├── aidetector/e2e/analyze_code_churn_test.go
├── cursor/e2e/collect_metrics_test.go
└── claudecode/e2e/collect_metrics_test.go
```

---

## 8. Estimated Effort

| Phase | Effort |
|-------|--------|
| Make Plugins Production-Ready | 1 day |
| Settings Pages | 2 days |
| Connection Pages | 1.5 days |
| Unit Tests | 1 day |
| Integration Testing | 0.5 day |
| **Total** | **6 days** |

---

## 9. Dependencies

### Config-UI Dependencies
The config-ui likely already has these, but verify:
- Ant Design components
- React Router
- API client utilities
- Form validation (likely built-in with Ant Design)

### Backend Dependencies
Already satisfied by existing plugins.
