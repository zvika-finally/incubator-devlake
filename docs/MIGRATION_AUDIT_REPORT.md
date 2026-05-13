# Migration Files Audit Report
**Date:** 2026-01-29
**Audited By:** Claude Code (Automated Audit)

## Executive Summary

This audit reviewed all plugin migration files across the Apache DevLake codebase to ensure compliance with the guidelines specified in AGENTS.md and backend/DevelopmentManual.md.

### Critical Findings:
- **5 plugins** have unregistered migration scripts that will NOT run at server startup
- **1 plugin** (issue_trace) is completely missing its register.go file
- **1 plugin** (issue_trace) has migration files with non-compliant naming convention
- **1 test file** incorrectly placed in migration directory

---

## Detailed Findings

### 🔴 Critical Issues (Must Fix Immediately)

#### 1. Issue_trace Plugin - Missing register.go
**Location:** `backend/plugins/issue_trace/models/migrationscripts/`
**Severity:** CRITICAL

**Problem:**
- No `register.go` file exists
- Migration `2024_05_30_new_issue_table.go` will never execute
- Naming convention violation: uses underscores instead of `20240530` format

**Impact:** Database tables `issue_status_history` and `issue_assignee_history` will never be created.

**Recommended Fix:**
```go
// Create: backend/plugins/issue_trace/models/migrationscripts/register.go
package migrationscripts

import "github.com/apache/incubator-devlake/core/plugin"

func All() []plugin.MigrationScript {
	return []plugin.MigrationScript{
		new(NewIssueTable),
	}
}
```

Also rename the file:
- **From:** `2024_05_30_new_issue_table.go`
- **To:** `20240530_new_issue_table.go`

---

#### 2. Gitlab Plugin - Unregistered Migration
**Location:** `backend/plugins/gitlab/models/migrationscripts/`
**File:** `20240904_remove_mr_review_fields.go`
**Severity:** HIGH

**Problem:** Migration script exists but is NOT registered in `register.go`

**Current State:**
- 30 migration files
- 29 registered in `register.go`

**Missing Registration:** `new(removeMrEnricherFields)`

**Impact:** The `first_comment_time` and `review_rounds` columns will not be removed from `_tool_gitlab_merge_requests` table.

**Recommended Fix:**
Add to `register.go` after line with `addGitlabAssigneeAndReviewerPrimaryKey`:
```go
new(removeMrEnricherFields),
```

---

#### 3. Bitbucket Plugin - Misplaced Test File
**Location:** `backend/plugins/bitbucket/models/migrationscripts/`
**File:** `20251001_add_api_token_auth_test.go`
**Severity:** MEDIUM

**Problem:** Unit test file incorrectly placed in migrations directory and counted as migration

**Current State:**
- 21 files in directory (including 1 test file)
- 20 actual migrations registered correctly

**Recommended Actions:**
1. Move test file to: `backend/plugins/bitbucket/models/migrationscripts/test/` OR
2. Rename to standard Go test convention alongside the migration file
3. Update `.gitignore` or file organization to prevent future confusion

**Note:** This is NOT a functional issue but violates directory structure conventions.

---

#### 4. Q_dev Plugin - Unregistered Migration
**Location:** `backend/plugins/q_dev/models/migrationscripts/`
**File:** `20250709_delete_user_metrics.go`
**Severity:** HIGH

**Problem:** Migration script exists but is NOT registered in `register.go`

**Current State:**
- 8 migration files
- 7 registered in `register.go`

**Missing Registration:** `new(deleteUserMetrics)`

**Impact:** The `_tool_q_dev_user_metrics` table will not be dropped as intended.

**Recommended Fix:**
Add to `register.go` after `addDisplayNameFields` and before `addMissingMetrics`:
```go
new(deleteUserMetrics),
```

---

#### 5. Testmo Plugin - Unregistered Migration
**Location:** `backend/plugins/testmo/models/migrationscripts/`
**File:** `20250629_add_scope_config_id.go`
**Severity:** HIGH

**Problem:** Migration script exists but is NOT registered in `register.go`

**Current State:**
- 5 migration files
- 4 registered in `register.go`
- Note: Two files with similar names on same date (`20250629_add_scope_config_id.go` and `20250629_add_scope_config_id_to_projects.go`)

**Missing Registration:** `new(addScopeConfigIdToProject)`

**Impact:** The `scope_config_id` field will not be added to `_tool_testmo_projects` table.

**Recommended Fix:**
The register.go has `addScopeConfigIdToProjects` but is missing the singular version. Review both files and add:
```go
new(addScopeConfigIdToProject),
```

---

## ✅ Compliance Status by Plugin

### Plugins with NO Issues Found:
- ae
- aidetector
- argocd
- azuredevops_go
- bamboo
- bitbucket_server
- businessmetrics
- capacityplanner
- circleci
- claudecode
- cursor
- customize
- dbt
- dora
- feishu
- findevops
- gitee
- gitextractor
- github
- github_graphql
- icla
- jenkins
- jira
- linker (no migrations)
- opsgenie
- org (no migrations)
- pagerduty
- refdiff (no migrations)
- slack
- sonarqube
- starrocks (no migrations)
- tapd
- teambition
- trello
- webhook
- zentao

### Plugins with Issues:
1. **issue_trace** - Critical (missing register.go)
2. **gitlab** - High (1 unregistered migration)
3. **bitbucket** - Medium (test file misplacement)
4. **q_dev** - High (1 unregistered migration)
5. **testmo** - High (1 unregistered migration)

---

## Validation Checklist

### ✅ Validated (Passing)
- [x] All migration files follow `YYYYMMDD_description.go` naming format (except issue_trace)
- [x] All registered migrations implement `script.Script` interface correctly
- [x] All migrations have proper `Up()`, `Version()`, and `Name()` methods
- [x] Apache 2.0 license headers present on all files
- [x] Migration versions follow timestamp format

### ❌ Issues Found (Failing)
- [ ] All migration scripts are registered in `register.go` - **5 plugins have gaps**
- [ ] All plugins with migrations have `register.go` - **issue_trace missing**
- [ ] No test files in migration directories - **bitbucket has one**
- [ ] All migration filenames follow strict YYYYMMDD format - **issue_trace uses underscores**

---

## Recommendations

### Immediate Actions Required:
1. **Create `register.go` for issue_trace plugin** (blocking)
2. **Register 4 missing migrations** in gitlab, q_dev, testmo
3. **Rename issue_trace migration file** to follow naming convention
4. **Move or reorganize bitbucket test file**

### Process Improvements:
1. **Add automated CI check** for migration registration
   - Similar to existing `plugins/table_info_test.go`
   - Validate file count matches registration count
   - Prevent future regressions

2. **Create migration registration linter**
   - Run as pre-commit hook
   - Flag unregistered migrations immediately

3. **Update documentation**
   - Add explicit examples in DevelopmentManual.md
   - Emphasize that forgetting registration causes silent failures

4. **Template for new migrations**
   - Provide migration template file
   - Include automatic register.go update reminder

---

## Testing Recommendations

After applying fixes, run:

```bash
# From repo root
make build           # Verify all plugins compile
make unit-test       # Run unit tests
make migration-script-lint  # Validate migrations (if available)

# Test migrations manually
docker-compose -f docker-compose-dev.yml up mysql
make dev
# Check logs for migration execution
```

---

## Appendix: Plugin Migration Counts

| Plugin | Migration Files | Registered | Status |
|--------|----------------|------------|--------|
| ae | 1 | 1 | ✅ |
| aidetector | 2 | 2 | ✅ |
| argocd | 2 | 2 | ✅ |
| azuredevops_go | 5 | 5 | ✅ |
| bamboo | 7 | 7 | ✅ |
| bitbucket | 21 | 20 | ⚠️ (1 test file) |
| bitbucket_server | 9 | 9 | ✅ |
| businessmetrics | 2 | 2 | ✅ |
| capacityplanner | 2 | 2 | ✅ |
| circleci | 5 | 5 | ✅ |
| claudecode | 1 | 1 | ✅ |
| cursor | 1 | 1 | ✅ |
| customize | 1 | 1 | ✅ |
| dora | 5 | 5 | ✅ |
| feishu | 2 | 2 | ✅ |
| findevops | 2 | 2 | ✅ |
| gitee | 18 | 18 | ✅ |
| github | 33 | 33 | ✅ |
| github_graphql | 4 | 4 | ✅ |
| gitlab | 30 | 29 | ❌ |
| icla | 1 | 1 | ✅ |
| issue_trace | 1 | 0 | ❌ (no register.go) |
| jenkins | 9 | 9 | ✅ |
| jira | 31 | 31 | ✅ |
| linker | 0 | 0 | ✅ |
| opsgenie | 4 | 4 | ✅ |
| pagerduty | 2 | 2 | ✅ |
| q_dev | 8 | 7 | ❌ |
| slack | 2 | 2 | ✅ |
| sonarqube | 7 | 7 | ✅ |
| tapd | 25 | 25 | ✅ |
| teambition | 18 | 18 | ✅ |
| testmo | 5 | 4 | ❌ |
| trello | 2 | 2 | ✅ |
| webhook | 3 | 3 | ✅ |
| zentao | 22 | 22 | ✅ |

**Total Plugins Audited:** 41
**Plugins with Issues:** 5
**Compliance Rate:** 87.8%

---

## Conclusion

The codebase generally follows migration guidelines well, with 87.8% of plugins fully compliant. However, the 5 plugins with issues represent critical functionality gaps where migrations will silently fail to execute at runtime.

The most critical issue is the **issue_trace plugin** which has no migration registration mechanism at all. This should be addressed immediately to prevent data integrity issues.

Adding automated CI validation for migration registration (similar to the existing table_info_test.go) would prevent these issues from occurring in the future.
