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

/**
 * Extract a useful error message from an axios or generic error.
 *
 * Devlake's API returns `{success: false, message: "..."}` on failure, exposed
 * at `err.response.data.message`. Plain `err.message` only surfaces the
 * generic axios "Request failed with status code N" — useless for diagnosis.
 *
 * Also handles 428 Precondition Required by redirecting to `/db-migrate`,
 * matching the interceptor on the shared `request()` axios instance. We
 * can't reuse that instance directly because it returns `resp.data` and
 * doesn't expose `responseType: 'text'`.
 */
export const extractErrorMessage = (err: unknown): string => {
  if (axios.isAxiosError(err)) {
    if (err.response?.status === 428) {
      window.location.replace('/db-migrate');
      return 'Schema migration required — redirecting…';
    }
    const apiMsg = (err.response?.data as { message?: string } | undefined)?.message;
    return apiMsg ?? err.message;
  }
  return err instanceof Error ? err.message : String(err);
};

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
  const lines = csv.split(/\r?\n/).filter((l) => l.trim().length > 0);
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
