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
    <Typography.Title level={3} style={{ marginBottom: 0 }}>
      Team Configuration
    </Typography.Title>
    <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
      Define canonical users, map them to per-source accounts (GitHub, Jira, etc.), and run the auto-mapping heuristic.
      Matches Devlake's documented Team Configuration flow:&nbsp;
      <a href="https://devlake.apache.org/docs/Configuration/TeamConfiguration" target="_blank" rel="noreferrer">
        devlake.apache.org docs
      </a>
      .
    </Typography.Paragraph>
    <UsersSection />
    <MappingSection />
    <AutoMapSection />
  </Space>
);
