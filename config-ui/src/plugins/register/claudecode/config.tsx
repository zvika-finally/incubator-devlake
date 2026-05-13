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
 */

import { IPluginConfig } from '@/types';

import Icon from './assets/icon.svg?react';

export const ClaudeCodeConfig: IPluginConfig = {
  plugin: 'claudecode',
  name: 'Claude Code',
  icon: ({ color }) => <Icon fill={color} />,
  sort: 26,
  isBeta: true,
  connection: {
    docLink: 'https://docs.anthropic.com/en/docs/claude-code',
    initialValues: {
      rateLimitPerSecond: 5,
    },
    fields: [
      'name',
      {
        key: 'adminApiKey',
        label: 'Admin API Key',
        subLabel: 'Admin API key (sk-ant-admin-...) from Console → Settings → Admin Keys. Only org admins can create these.',
        type: 'password',
        required: true,
      },
      {
        key: 'rateLimitPerSecond',
        label: 'Rate Limit',
        subLabel: 'API requests per second (default: 5)',
        inputType: 'number',
        defaultValue: 5,
      },
    ],
  },
  dataScope: {
    title: 'Organizations',
  },
};
