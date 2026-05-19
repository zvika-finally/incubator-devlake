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
        "This will run Devlake's connectUserAccountsExact subtask, which overwrites user_accounts based on a name+email heuristic. Manual entries in the mapping table will be replaced.",
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
        {latest && <Link to={`/advanced/pipeline/${latest.id}`}>View pipeline {latest.id} →</Link>}
      </Space>
    </Card>
  );
};
