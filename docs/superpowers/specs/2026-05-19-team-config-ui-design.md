# Team Configuration UI Design

**Date:** 2026-05-19
**Status:** Approved
**Repo:** incubator-devlake (config-ui)

## Goal

Add a `/team-config` page to the Devlake config-ui that lets admins manage `users`, `user_account_mapping`, and trigger the `connectUserAccountsExact` auto-mapping pipeline — matching the procedure documented at https://devlake.apache.org/docs/Configuration/TeamConfiguration.

## Motivation

The 2.x redesign of this Devlake fork removed the prior Teams UI. The `org` plugin's CSV REST endpoints still work, but in finally's deployment the OAuth2 / ALB layer blocks Bearer-token-only callers — only browser sessions with the `X-Remote-Groups` header pass. That makes the documented `curl ... PUT users.csv` flow unusable from outside the browser.

A page rendered inside the config-ui inherits the user's authenticated session automatically, so all the org-plugin endpoints "just work" without further auth wiring. This unblocks ongoing team management (adding/removing engineers, fixing auto-mapping mistakes) without DevOps involvement.

## Scope (V1)

One new route `/team-config` with three vertically stacked sections:

1. **Users**
   - Drag-and-drop CSV upload (`PUT /api/plugins/org/users.csv`)
   - "Download template" button (`GET /api/plugins/org/users.csv?fake_data=true`)
   - "Download current" button (`GET /api/plugins/org/users.csv`)
   - Read-only preview table of current `users` (Id, Name, Email, TeamIds)
2. **User-Account Mapping**
   - Same controls against `/api/plugins/org/user_account_mapping.csv`
   - Preview shows current `user_accounts` rows joined with account names where possible
3. **Auto-Mapping**
   - Single button "Run mapping algorithm" → `POST /api/pipelines` with
     ```json
     { "name": "team-config-auto-map-<timestamp>", "plan": [[{ "plugin": "org", "subtasks": ["connectUserAccountsExact"] }]] }
     ```
   - Status indicator (idle / running / succeeded / failed)
   - Link to the resulting pipeline detail page once started

Sidebar nav entry: "Team Config" with `UsergroupAddOutlined` icon, slotted between "Connections" and "Projects".

## Out of scope (V1)

- Inline form editors per row — full-CSV upload is the documented flow
- Teams CSV / `team_users` — no current teams to manage
- Audit log / change history
- Suggesting candidate mappings from the live `accounts` table
- Filtering / pagination on the preview tables

These are easy follow-ups once V1 ships.

## Files

**Create:**
- `config-ui/src/routes/team-config/index.ts` — route export
- `config-ui/src/routes/team-config/team-config.tsx` — page composing three sections
- `config-ui/src/routes/team-config/sections/users-section.tsx`
- `config-ui/src/routes/team-config/sections/mapping-section.tsx`
- `config-ui/src/routes/team-config/sections/auto-map-section.tsx`
- `config-ui/src/routes/team-config/api.ts` — axios helpers for the org-plugin endpoints + pipelines POST

**Modify:**
- `config-ui/src/routes/index.ts` — re-export the new route
- `config-ui/src/routes/layout/layout.tsx` (or wherever the sidebar lives) — add the nav entry

## API contract

All endpoints are existing — we are NOT modifying the backend. We are consuming:

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/plugins/org/users.csv` | Current users (CSV text) |
| GET | `/api/plugins/org/users.csv?fake_data=true` | Template |
| PUT | `/api/plugins/org/users.csv` | Replace users (multipart `file` field) |
| GET | `/api/plugins/org/user_account_mapping.csv` | Current mappings (CSV text) |
| PUT | `/api/plugins/org/user_account_mapping.csv` | Replace mappings (multipart `file` field) |
| POST | `/api/pipelines` | Trigger auto-mapping pipeline |

CSV header format (gocsv, case-sensitive struct-field-name match — verified empirically):
- users.csv: `Id,Name,Email,TeamIds` (TeamIds optional)
- user_account_mapping.csv: `Id,UserId` (minimum; the handler unmarshals into `account` struct — extra columns are tolerated)

## UX details

- Each upload section shows: drop zone (or file picker), "Upload" button (disabled until a file is selected), success/error toast on response.
- Preview tables fetch on mount and after every successful upload. Limit to first 100 rows in V1.
- Auto-Mapping button is non-destructive but expensive; show a confirmation modal "This will overwrite user_accounts based on name/email heuristics. Continue?"
- All loading/error states use Ant Design's `<Spin>`, `<Alert>`, and `notification.error()`.

## Error handling

- 4xx/5xx responses from the API → show the response body in a notification banner (Devlake's API returns `{"success":false,"message":"..."}` on failure)
- Empty current-state CSVs → render "No data yet — upload a file to get started"
- Pipeline POST is async; the UI surfaces the pipeline ID + a link to follow it

## Testing

- Component tests for each section using existing testing setup (whatever's already configured — looks like none currently, so manual smoke-test in a browser)
- E2E manual checklist: upload bad CSV → error toast; upload good CSV → success; click run-mapping → pipeline ID returned; refresh page → previews populated
- The config-ui has no Vitest suite today; we won't add one in this PR

## Deployment

After merge, rebuild + redeploy the `devlake_ui` image via `finally-internal/scripts/build-and-push-devlake.sh --ui-only`. Lives at `https://devlake.internal.finally.com/team-config`.
