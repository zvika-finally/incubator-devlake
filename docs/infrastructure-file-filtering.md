# Infrastructure File Filtering for Code Churn Analysis

**Date:** February 1, 2026
**Issue:** AI Code Churn metrics showing extreme values (34,400%) due to infrastructure file churn
**Solution:** Filter infrastructure files during churn calculation

---

## Problem Statement

### Root Cause
The AI Code Churn analysis was including infrastructure and configuration files (.github/, package.json, etc.) in churn calculations. These files:

1. **Have naturally high churn** - CI/CD configs and package files get iteratively improved across many PRs
2. **Are shared infrastructure** - Changes affect multiple features, not individual PR code quality
3. **Create extreme outliers** - 32% of PRs had 1000%+ churn ratios

### Example from Production
```
PR #28: Added 3 lines to .github/workflows/ci-cd.yml
30-day churn: 1,032 lines modified across 8 .github/ files
Churn ratio: 34,400% (!)
```

### Impact on Metrics
```
Distribution of Non-AI PRs (106 total):
- 38 PRs: 0% churn
- 11 PRs: 1-49% churn
- 6 PRs: 50-99% churn
- 13 PRs: 100-199% churn
- 4 PRs: 200-499% churn
- 34 PRs: 1000%+ churn (avg 9,826%)

Average displayed: 3,186% (meaningless due to outliers)
```

---

## Solution Architecture

### 1. Configurable Exclusion Patterns

**File:** `backend/plugins/aidetector/models/settings.go`

Added field to `AIDetectorSettings`:
```go
// File exclusion patterns for churn analysis (JSON array)
ExcludeFilePatterns string `json:"excludeFilePatterns" gorm:"type:text"`
```

**Default Patterns:**
```json
[
  ".github/",
  "package.json",
  "package-lock.json",
  "yarn.lock",
  "pnpm-lock.yaml",
  "Dockerfile",
  "docker-compose.yml",
  ".gitignore",
  ".eslintrc",
  "tsconfig.json"
]
```

### 2. File Filtering Logic

**File:** `backend/plugins/aidetector/tasks/analyze_code_churn.go`

Added two helper functions:

#### filterInfrastructureFiles()
- Parses exclusion patterns from settings
- Filters file list before churn calculation
- Logs number of excluded files for debugging

#### shouldExcludeFile()
- Matches file paths against patterns
- Supports:
  - **Exact match**: `package.json` matches `package.json`
  - **Directory prefix**: `.github/` matches `.github/workflows/ci.yml`
  - **Filename anywhere**: `Dockerfile` matches `docker/Dockerfile`

### 3. Integration Point

Updated `AnalyzeCodeChurn()` function:
```go
// Get files changed in this PR (excluding infrastructure files)
files := getPRFiles(db, pr.Id)
files = filterInfrastructureFiles(files, data.Settings, logger)
if len(files) == 0 {
    continue // Skip PRs that only touched infrastructure
}
```

### 4. Database Migration

**File:** `backend/plugins/aidetector/models/migrationscripts/20260201_add_exclude_patterns.go`

- Version: `20260201000001`
- Adds `exclude_file_patterns` column to `_tool_aidetector_settings` table
- Sets default exclusion patterns for existing projects

### 5. Dashboard Documentation

**File:** `grafana/dashboards/AIDetection.json`

Updated Code Churn description:
```markdown
**Code Churn** measures how much code is modified after initial merge.

**Note:** Infrastructure files (.github/, package.json, config files)
are excluded from churn analysis to prevent skewing metrics.
```

---

## Configuration

### Per-Project Customization

Exclusion patterns can be customized via API:

```bash
curl -X PUT http://localhost:8080/plugins/aidetector/settings/MyProject \
  -H "Content-Type: application/json" \
  -d '{
    "excludeFilePatterns": "[\"custom/path/\",\"*.config.js\"]"
  }'
```

### Adding New Patterns

Common additional patterns:
```json
[
  "*.md",              // Documentation
  "*.test.js",         // Tests (if you want to exclude)
  "jest.config.js",    // Test config
  ".prettierrc",       // Code formatting
  ".editorconfig",     // Editor config
  "renovate.json",     // Dependency bot config
  ".vscode/",          // IDE settings
  ".idea/"             // IDE settings
]
```

---

## Expected Results After Deployment

### Before Filtering
```sql
Non-AI Code: 3,186% average churn (meaningless)
AI Code: 98% average churn
```

### After Filtering
Expected realistic distribution:
```
Non-AI Code: ~15-30% average churn
AI Code: ~10-20% average churn
```

Infrastructure PRs (only .github/ changes) will show 0 churn (correctly skipped).

---

## Deployment Steps

1. **Build Plugin:**
   ```bash
   cd backend
   go build -buildmode=plugin -o bin/plugins/aidetector/aidetector.so plugins/aidetector/*.go
   ```

2. **Restart DevLake Server:**
   - Migration runs automatically on startup
   - Adds `exclude_file_patterns` column with defaults

3. **Re-run Analysis:**
   ```bash
   curl -X POST http://localhost:8080/pipelines \
     -H "Content-Type: application/json" \
     -d '{
       "name": "Recalculate AI Metrics with Filtering",
       "plan": [[{
         "plugin": "aidetector",
         "options": {"projectName": "Expense Management"}
       }]]
     }'
   ```

4. **Verify Results:**
   ```sql
   SELECT is_ai_assisted,
          COUNT(*) as total,
          AVG(churn_ratio30_days) as avg_churn,
          MIN(churn_ratio30_days) as min_churn,
          MAX(churn_ratio30_days) as max_churn
   FROM ai_churn_metrics
   GROUP BY is_ai_assisted;
   ```

---

## Files Modified

### New Files
1. `backend/plugins/aidetector/models/migrationscripts/20260201_add_exclude_patterns.go`
   - Migration to add exclude_file_patterns column

### Modified Files
1. `backend/plugins/aidetector/models/settings.go`
   - Added ExcludeFilePatterns field
   - Updated NewDefaultSettings() with default patterns

2. `backend/plugins/aidetector/tasks/analyze_code_churn.go`
   - Added filterInfrastructureFiles() helper
   - Added shouldExcludeFile() pattern matcher
   - Integrated filtering into main analysis flow

3. `backend/plugins/aidetector/models/migrationscripts/register.go`
   - Registered new migration script

4. `grafana/dashboards/AIDetection.json`
   - Updated Code Churn description panel

### Rebuilt
- `backend/bin/plugins/aidetector/aidetector.so` (37MB, 2026-02-01 21:08)

---

## Testing

### Manual Verification

Test the filtering logic:
```sql
-- Check if infrastructure PRs are excluded
SELECT pull_request_key, file_paths
FROM ai_churn_metrics
WHERE file_paths LIKE '%.github/%'
   OR file_paths LIKE '%package.json%';
-- Should return 0 rows after re-running with filtering
```

### Expected Behavior

Infrastructure-only PRs:
- **Before:** churn_ratio30_days = 344.0 (34,400%)
- **After:** Not in ai_churn_metrics table (no application code)

Mixed PRs (app code + infrastructure):
- **Before:** churn includes all files
- **After:** churn only from application code files

---

## Rationale

### Why Filter at Calculation Time?

**Benefits:**
1. **Accurate from source** - Data is correct when created
2. **Consistent** - All downstream queries automatically correct
3. **Configurable** - Per-project patterns via settings
4. **Auditable** - Exclusion logged, can track what was filtered

**Alternatives Considered:**
- ❌ Dashboard-level filtering - Inconsistent, harder to maintain
- ❌ Post-processing - Complex, error-prone
- ✅ **Calculation-time filtering** - Clean, reusable, maintainable

### Industry Best Practices

Most code quality tools (SonarQube, Code Climate) exclude:
- Configuration files
- Build scripts
- CI/CD definitions
- Package lock files
- Generated code

This aligns our metrics with industry standards.

---

## Monitoring

### Check Exclusion Impact

```sql
-- See how many PRs were filtered out
SELECT
    COUNT(*) as total_merged_prs,
    (SELECT COUNT(*) FROM ai_churn_metrics) as prs_with_churn,
    COUNT(*) - (SELECT COUNT(*) FROM ai_churn_metrics) as filtered_out
FROM pull_requests pr
JOIN project_mapping pm ON pm.row_id = pr.base_repo_id AND pm.table = 'repos'
WHERE pr.merged_date IS NOT NULL
  AND pm.project_name = 'Expense Management';
```

### Debug Logging

Enable debug logs to see filtering:
```
logger.Debug("Excluded %d infrastructure files from churn analysis", excludedCount)
```

---

## Future Enhancements

1. **Glob Pattern Support** - Use filepath.Match() for wildcards
2. **Regex Patterns** - More complex matching (e.g., `test.*\.js$`)
3. **File Size Threshold** - Exclude very large files (generated assets)
4. **Language-Specific Defaults** - Different patterns for Go/Python/JS projects
5. **UI Configuration** - Manage patterns via Config UI instead of API

---

**Status:** ✅ **Implementation Complete - Ready for Deployment**
