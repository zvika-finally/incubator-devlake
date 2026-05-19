# Team Configuration UI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `/team-config` page to the Devlake config-ui that manages `users` + `user_account_mapping` and triggers the `connectUserAccountsExact` auto-mapping pipeline.

**Architecture:** A single React route composed of three independent section components. All sections call existing `org` plugin REST endpoints — no backend changes. Auth comes for free from the browser session (same-origin call inside an already-authenticated config-ui session).

**Tech Stack:** React 18 + TypeScript + Vite + Ant Design 5 + axios + react-router-dom (browser router).

---

## File map

**Create:**
- `config-ui/src/routes/team-config/index.ts`
- `config-ui/src/routes/team-config/team-config.tsx`
- `config-ui/src/routes/team-config/api.ts`
- `config-ui/src/routes/team-config/sections/users-section.tsx`
- `config-ui/src/routes/team-config/sections/mapping-section.tsx`
- `config-ui/src/routes/team-config/sections/auto-map-section.tsx`

**Modify:**
- `config-ui/src/routes/index.ts` — re-export the new page
- `config-ui/src/routes/layout/config.tsx` — add menu entry
- `config-ui/src/app/routrer.tsx` — register the route

---

## Conventions

- All new files start with the Apache 2.0 license header (copy from any existing file in `config-ui/src/routes/`).
- TypeScript strict mode is on; never use `any` — use `unknown` and narrow.
- HTTP errors that come back as `{success:false, message:"..."}` should surface the `message` in an Ant Design `notification.error`.
- Use `notification.success` for successful uploads.
- File uploads use Ant Design `Upload.Dragger` with `beforeUpload={() => false}` so the file stays client-side until the user clicks an explicit "Upload" button.
- API base path is `/api/plugins/org/...` and `/api/pipelines`. No `/rest/` because we want the cookie-auth path (the user is already SSO'd in via OAuth2 proxy when they load this page).

---

## Task 1: API helpers

**Files:**
- Create: `config-ui/src/routes/team-config/api.ts`

- [ ] **Step 1: Write the file**

```typescript
/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

import axios from 'axios';

const ORG = '/api/plugins/org';

export const getUsersCsv = (fakeData = false): Promise<string> =>
  axios
    .get(`${ORG}/users.csv`, { params: fakeData ? { fake_data: 'true' } : undefined, responseType: 'text' })
    .then((r) => r.data);

export const getMappingCsv = (fakeData = false): Promise<string> =>
  axios
    .get(`${ORG}/user_account_mapping.csv`, {
      params: fakeData ? { fake_data: 'true' } : undefined,
      responseType: 'text',
    })
    .then((r) => r.data);

export const putUsersCsv = async (file: File): Promise<void> => {
  const form = new FormData();
  form.append('file', file);
  await axios.put(`${ORG}/users.csv`, form);
};

export const putMappingCsv = async (file: File): Promise<void> => {
  const form = new FormData();
  form.append('file', file);
  await axios.put(`${ORG}/user_account_mapping.csv`, form);
};

export type PipelineRef = { id: number; name: string };

export const runAutoMapping = async (): Promise<PipelineRef> => {
  const body = {
    name: `team-config-auto-map-${new Date().toISOString().replace(/[:.]/g, '-')}`,
    plan: [[{ plugin: 'org', subtasks: ['connectUserAccountsExact'] }]],
  };
  const { data } = await axios.post('/api/pipelines', body);
  return { id: data.id, name: data.name };
};

/** Parse a CSV string into an array of row objects keyed by the header. */
export const parseCsv = (csv: string): Record<string, string>[] => {
  const lines = csv
    .split(/\r?\n/)
    .filter((l) => l.trim().length > 0);
  if (lines.length < 2) return [];
  const headers = splitCsvLine(lines[0]);
  return lines.slice(1).map((line) => {
    const cells = splitCsvLine(line);
    return Object.fromEntries(headers.map((h, i) => [h, cells[i] ?? '']));
  });
};

const splitCsvLine = (line: string): string[] => {
  const out: string[] = [];
  let cur = '';
  let inQuotes = false;
  for (let i = 0; i < line.length; i++) {
    const ch = line[i];
    if (inQuotes) {
      if (ch === '"' && line[i + 1] === '"') {
        cur += '"';
        i++;
      } else if (ch === '"') {
        inQuotes = false;
      } else {
        cur += ch;
      }
    } else if (ch === '"') {
      inQuotes = true;
    } else if (ch === ',') {
      out.push(cur);
      cur = '';
    } else {
      cur += ch;
    }
  }
  out.push(cur);
  return out;
};
```

- [ ] **Step 2: Commit**

```bash
git add config-ui/src/routes/team-config/api.ts
git commit -m "feat(team-config-ui): API helpers for org plugin endpoints"
```

---

## Task 2: Users section

**Files:**
- Create: `config-ui/src/routes/team-config/sections/users-section.tsx`

- [ ] **Step 1: Write the component**

```typescript
/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

import { useEffect, useState } from 'react';
import { Card, Upload, Button, Table, Space, notification, Spin } from 'antd';
import { InboxOutlined, DownloadOutlined, UploadOutlined } from '@ant-design/icons';
import { saveAs } from 'file-saver';

import { getUsersCsv, putUsersCsv, parseCsv } from '../api';

type UserRow = { Id: string; Name: string; Email: string; TeamIds: string };

export const UsersSection = () => {
  const [rows, setRows] = useState<UserRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [pending, setPending] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);

  const refresh = async () => {
    setLoading(true);
    try {
      const csv = await getUsersCsv();
      setRows(parseCsv(csv) as UserRow[]);
    } catch (err) {
      notification.error({ message: 'Failed to load users', description: String((err as Error).message) });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    refresh();
  }, []);

  const downloadCurrent = async () => {
    try {
      const csv = await getUsersCsv();
      saveAs(new Blob([csv], { type: 'text/csv' }), 'users.csv');
    } catch (err) {
      notification.error({ message: 'Download failed', description: String((err as Error).message) });
    }
  };

  const downloadTemplate = async () => {
    try {
      const csv = await getUsersCsv(true);
      saveAs(new Blob([csv], { type: 'text/csv' }), 'users-template.csv');
    } catch (err) {
      notification.error({ message: 'Template download failed', description: String((err as Error).message) });
    }
  };

  const upload = async () => {
    if (!pending) return;
    setUploading(true);
    try {
      await putUsersCsv(pending);
      notification.success({ message: 'users.csv uploaded' });
      setPending(null);
      await refresh();
    } catch (err) {
      notification.error({ message: 'Upload failed', description: String((err as Error).message) });
    } finally {
      setUploading(false);
    }
  };

  return (
    <Card title="Users" extra={
      <Space>
        <Button icon={<DownloadOutlined />} onClick={downloadTemplate}>Template</Button>
        <Button icon={<DownloadOutlined />} onClick={downloadCurrent}>Current</Button>
      </Space>
    }>
      <Upload.Dragger
        accept=".csv"
        multiple={false}
        beforeUpload={(file) => {
          setPending(file);
          return false;
        }}
        fileList={pending ? [{ uid: '1', name: pending.name, status: 'done' as const }] : []}
        onRemove={() => setPending(null)}
      >
        <p className="ant-upload-drag-icon"><InboxOutlined /></p>
        <p className="ant-upload-text">Click or drag a users.csv file here</p>
        <p className="ant-upload-hint">Headers must be Id,Name,Email,TeamIds (case-sensitive).</p>
      </Upload.Dragger>
      <div style={{ marginTop: 12, marginBottom: 24 }}>
        <Button type="primary" icon={<UploadOutlined />} disabled={!pending} loading={uploading} onClick={upload}>
          Upload
        </Button>
      </div>
      <Spin spinning={loading}>
        <Table
          rowKey="Id"
          dataSource={rows}
          size="small"
          pagination={{ pageSize: 50 }}
          columns={[
            { title: 'Id', dataIndex: 'Id', key: 'Id' },
            { title: 'Name', dataIndex: 'Name', key: 'Name' },
            { title: 'Email', dataIndex: 'Email', key: 'Email' },
            { title: 'Team IDs', dataIndex: 'TeamIds', key: 'TeamIds' },
          ]}
          locale={{ emptyText: 'No users yet — upload a CSV to get started.' }}
        />
      </Spin>
    </Card>
  );
};
```

- [ ] **Step 2: Commit**

```bash
git add config-ui/src/routes/team-config/sections/users-section.tsx
git commit -m "feat(team-config-ui): users CSV section"
```

---

## Task 3: Mapping section

**Files:**
- Create: `config-ui/src/routes/team-config/sections/mapping-section.tsx`

- [ ] **Step 1: Write the component**

Mirror the Users section. Differences: calls `getMappingCsv` / `putMappingCsv`, columns are `Id`, `UserId`, header hint says "Headers must be `Id,UserId` (case-sensitive)".

```typescript
/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

import { useEffect, useState } from 'react';
import { Card, Upload, Button, Table, Space, notification, Spin } from 'antd';
import { InboxOutlined, DownloadOutlined, UploadOutlined } from '@ant-design/icons';
import { saveAs } from 'file-saver';

import { getMappingCsv, putMappingCsv, parseCsv } from '../api';

type MappingRow = { Id: string; UserId: string };

export const MappingSection = () => {
  const [rows, setRows] = useState<MappingRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [pending, setPending] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);

  const refresh = async () => {
    setLoading(true);
    try {
      const csv = await getMappingCsv();
      setRows(parseCsv(csv) as MappingRow[]);
    } catch (err) {
      notification.error({ message: 'Failed to load mappings', description: String((err as Error).message) });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    refresh();
  }, []);

  const downloadCurrent = async () => {
    try {
      const csv = await getMappingCsv();
      saveAs(new Blob([csv], { type: 'text/csv' }), 'user_account_mapping.csv');
    } catch (err) {
      notification.error({ message: 'Download failed', description: String((err as Error).message) });
    }
  };

  const downloadTemplate = async () => {
    try {
      const csv = await getMappingCsv(true);
      saveAs(new Blob([csv], { type: 'text/csv' }), 'user_account_mapping-template.csv');
    } catch (err) {
      notification.error({ message: 'Template download failed', description: String((err as Error).message) });
    }
  };

  const upload = async () => {
    if (!pending) return;
    setUploading(true);
    try {
      await putMappingCsv(pending);
      notification.success({ message: 'user_account_mapping.csv uploaded' });
      setPending(null);
      await refresh();
    } catch (err) {
      notification.error({ message: 'Upload failed', description: String((err as Error).message) });
    } finally {
      setUploading(false);
    }
  };

  return (
    <Card title="User ↔ Account Mapping" extra={
      <Space>
        <Button icon={<DownloadOutlined />} onClick={downloadTemplate}>Template</Button>
        <Button icon={<DownloadOutlined />} onClick={downloadCurrent}>Current</Button>
      </Space>
    }>
      <Upload.Dragger
        accept=".csv"
        multiple={false}
        beforeUpload={(file) => {
          setPending(file);
          return false;
        }}
        fileList={pending ? [{ uid: '1', name: pending.name, status: 'done' as const }] : []}
        onRemove={() => setPending(null)}
      >
        <p className="ant-upload-drag-icon"><InboxOutlined /></p>
        <p className="ant-upload-text">Click or drag a user_account_mapping.csv file here</p>
        <p className="ant-upload-hint">Headers must be Id,UserId (case-sensitive). Id is the source account_id; UserId is the canonical user.</p>
      </Upload.Dragger>
      <div style={{ marginTop: 12, marginBottom: 24 }}>
        <Button type="primary" icon={<UploadOutlined />} disabled={!pending} loading={uploading} onClick={upload}>
          Upload
        </Button>
      </div>
      <Spin spinning={loading}>
        <Table
          rowKey="Id"
          dataSource={rows}
          size="small"
          pagination={{ pageSize: 50 }}
          columns={[
            { title: 'Account ID', dataIndex: 'Id', key: 'Id' },
            { title: 'User ID', dataIndex: 'UserId', key: 'UserId' },
          ]}
          locale={{ emptyText: 'No mappings yet — upload a CSV or run the auto-mapping below.' }}
        />
      </Spin>
    </Card>
  );
};
```

- [ ] **Step 2: Commit**

```bash
git add config-ui/src/routes/team-config/sections/mapping-section.tsx
git commit -m "feat(team-config-ui): user_account_mapping CSV section"
```

---

## Task 4: Auto-mapping section

**Files:**
- Create: `config-ui/src/routes/team-config/sections/auto-map-section.tsx`

- [ ] **Step 1: Write the component**

```typescript
/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

import { useState } from 'react';
import { Card, Button, Modal, Alert, Space, notification } from 'antd';
import { PlayCircleOutlined } from '@ant-design/icons';
import { Link } from 'react-router-dom';

import { runAutoMapping, PipelineRef } from '../api';

export const AutoMapSection = () => {
  const [running, setRunning] = useState(false);
  const [latest, setLatest] = useState<PipelineRef | null>(null);

  const trigger = () => {
    Modal.confirm({
      title: 'Run auto-mapping?',
      content:
        'This will run Devlake\'s connectUserAccountsExact subtask, which overwrites user_accounts based on a name+email heuristic. Manual entries in the mapping table will be replaced.',
      okText: 'Run',
      onOk: async () => {
        setRunning(true);
        try {
          const ref = await runAutoMapping();
          setLatest(ref);
          notification.success({ message: `Pipeline ${ref.id} started`, description: ref.name });
        } catch (err) {
          notification.error({ message: 'Pipeline trigger failed', description: String((err as Error).message) });
        } finally {
          setRunning(false);
        }
      },
    });
  };

  return (
    <Card title="Auto-Mapping">
      <Alert
        type="info"
        showIcon
        message="Match accounts to users automatically"
        description="Runs Devlake's connectUserAccountsExact subtask. The heuristic matches by exact email then by full name across all source accounts (GitHub, Jira, etc.) to populate the user_accounts table."
        style={{ marginBottom: 16 }}
      />
      <Space>
        <Button type="primary" icon={<PlayCircleOutlined />} loading={running} onClick={trigger}>
          Run mapping algorithm
        </Button>
        {latest && (
          <Link to={`/advanced/pipeline/${latest.id}`}>View pipeline {latest.id} →</Link>
        )}
      </Space>
    </Card>
  );
};
```

- [ ] **Step 2: Commit**

```bash
git add config-ui/src/routes/team-config/sections/auto-map-section.tsx
git commit -m "feat(team-config-ui): auto-mapping pipeline trigger section"
```

---

## Task 5: Page + route registration + nav entry

**Files:**
- Create: `config-ui/src/routes/team-config/team-config.tsx`
- Create: `config-ui/src/routes/team-config/index.ts`
- Modify: `config-ui/src/routes/index.ts`
- Modify: `config-ui/src/app/routrer.tsx`
- Modify: `config-ui/src/routes/layout/config.tsx`

- [ ] **Step 1: Create the page**

```typescript
// config-ui/src/routes/team-config/team-config.tsx
/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

import { Space, Typography } from 'antd';

import { UsersSection } from './sections/users-section';
import { MappingSection } from './sections/mapping-section';
import { AutoMapSection } from './sections/auto-map-section';

export const TeamConfig = () => (
  <Space direction="vertical" size="large" style={{ width: '100%' }}>
    <Typography.Title level={3} style={{ marginBottom: 0 }}>Team Configuration</Typography.Title>
    <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
      Define canonical users, map them to per-source accounts (GitHub, Jira, etc.), and run the
      auto-mapping heuristic. Matches Devlake's documented Team Configuration flow:&nbsp;
      <a href="https://devlake.apache.org/docs/Configuration/TeamConfiguration" target="_blank" rel="noreferrer">
        devlake.apache.org docs
      </a>.
    </Typography.Paragraph>
    <UsersSection />
    <MappingSection />
    <AutoMapSection />
  </Space>
);
```

- [ ] **Step 2: Create the route export**

```typescript
// config-ui/src/routes/team-config/index.ts
/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

export { TeamConfig } from './team-config';
```

- [ ] **Step 3: Add to the routes barrel**

Modify `config-ui/src/routes/index.ts` — add `export * from './team-config';` alongside the other exports.

- [ ] **Step 4: Register the route**

Modify `config-ui/src/app/routrer.tsx`:
1. Add `TeamConfig` to the named-imports from `@/routes`.
2. Add a new child route under the layout entry (next to `keys`):
   ```typescript
   {
     path: 'team-config',
     element: <TeamConfig />,
   },
   ```

- [ ] **Step 5: Add the sidebar nav entry**

Modify `config-ui/src/routes/layout/config.tsx`:
1. Add `UsergroupAddOutlined` to the imports from `@ant-design/icons`.
2. Insert this entry into `menuItems` between Connections and Advanced:
   ```typescript
   {
     key: `${PATH_PREFIX}/team-config`,
     label: 'Team Config',
     icon: <UsergroupAddOutlined />,
   },
   ```

- [ ] **Step 6: Commit**

```bash
git add config-ui/src/routes/team-config/ config-ui/src/routes/index.ts config-ui/src/app/routrer.tsx config-ui/src/routes/layout/config.tsx
git commit -m "feat(team-config-ui): page composition + route + sidebar entry"
```

---

## Task 6: Verify TypeScript + lint

**Files:** none

- [ ] **Step 1: Run TS check + ESLint**

```bash
cd config-ui && yarn lint 2>&1 | tail -30
```

Expected: exit code 0. If TS errors come up, fix them in the offending file and re-run.

- [ ] **Step 2: Run a local Vite build to make sure it compiles**

```bash
cd config-ui && yarn build 2>&1 | tail -20
```

Expected: build succeeds, output in `config-ui/dist/`.

- [ ] **Step 3: If anything was fixed in step 1 or 2, commit**

```bash
git commit -am "chore(team-config-ui): lint / type fixes from CI run"
```

(Skip if nothing changed.)

---

## Task 7: Push branch + open PR

**Files:** none

- [ ] **Step 1: Push**

```bash
git push -u origin team-config-ui
```

- [ ] **Step 2: Open the PR via gh**

```bash
gh pr create --base main --head team-config-ui \
  --title "feat(config-ui): Team Configuration page" \
  --body "$(cat <<'EOF'
## Summary

Adds a \`/team-config\` page to the Devlake config-ui that manages \`users\`, \`user_account_mapping\`, and triggers the \`connectUserAccountsExact\` auto-mapping pipeline — matching the procedure at https://devlake.apache.org/docs/Configuration/TeamConfiguration.

## Why now

The 2.x redesign of this fork removed the prior Teams UI. The org plugin's CSV REST endpoints still work, but in finally's deployment the OAuth2/ALB layer blocks Bearer-token-only callers. A page rendered inside the config-ui inherits the existing browser session, so all endpoints work without further auth wiring.

## Scope (V1)

- Users CSV upload + download (template + current)
- user_account_mapping CSV upload + download
- Auto-mapping pipeline trigger with confirm modal
- Preview tables for current state
- Sidebar entry "Team Config" between Connections and Advanced

Out of scope (follow-ups): inline row editors, teams.csv section, audit log, suggested mappings from the live \`accounts\` table.

## Test plan

- [x] \`yarn lint\` passes
- [x] \`yarn build\` succeeds
- [ ] Deploy via \`build-and-push-devlake.sh --ui-only\` and smoke-test in a browser:
  - Upload \`users.csv\` → preview refreshes
  - Upload \`user_account_mapping.csv\` → preview refreshes
  - Click "Run mapping algorithm" → pipeline ID surfaced, link to detail works
  - Error path: upload an invalid CSV → error notification shows the API message

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

- [ ] **Step 3: Print the PR URL for the user**

---

## Self-review

- Each file gets a license header.
- Three sections share patterns (download/upload/preview); they're intentionally NOT extracted into a shared component for V1 because the asymmetry between users / mapping / auto-map is large enough that abstraction would obscure rather than help. Revisit if a 4th section is added.
- The `parseCsv` helper is small enough to ship inline; it correctly handles quoted fields and escaped quotes, which the Devlake fixture format uses.
- All API calls are same-origin so no CORS / auth header wrangling needed.
- The router file is misspelled `routrer.tsx` in the codebase — preserved that typo in the modify list.
