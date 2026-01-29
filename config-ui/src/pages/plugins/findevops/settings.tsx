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

import { useState, useEffect } from 'react';
import { Card, Form, InputNumber, Input, Select, Button, message, Typography, Space, Divider, Collapse } from 'antd';
import { SaveOutlined, ReloadOutlined, DollarOutlined } from '@ant-design/icons';

import { request } from '@/utils';

const { Title, Paragraph } = Typography;
const { TextArea } = Input;
const { Panel } = Collapse;
const { Option } = Select;

interface FinDevOpsSettings {
  defaultHourlyRate: number;
  capitalizationFramework: string;
  fiscalYearStartMonth: number;
  unallocatedCostThreshold: number;
  hoursPerStoryPoint: number;
  preliminaryLabels: string;
  postImplementationLabels: string;
}

const defaultSettings: FinDevOpsSettings = {
  defaultHourlyRate: 87.0,
  capitalizationFramework: 'asc_350_40_stages',
  fiscalYearStartMonth: 1,
  unallocatedCostThreshold: 10,
  hoursPerStoryPoint: 4.0,
  preliminaryLabels: 'research, spike, discovery',
  postImplementationLabels: 'maintenance, support, bug-fix',
};

const months = [
  { value: 1, label: 'January' },
  { value: 2, label: 'February' },
  { value: 3, label: 'March' },
  { value: 4, label: 'April' },
  { value: 5, label: 'May' },
  { value: 6, label: 'June' },
  { value: 7, label: 'July' },
  { value: 8, label: 'August' },
  { value: 9, label: 'September' },
  { value: 10, label: 'October' },
  { value: 11, label: 'November' },
  { value: 12, label: 'December' },
];

export const FinDevOpsSettings = () => {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);

  const fetchSettings = async () => {
    setLoading(true);
    try {
      const data = await request('/plugins/findevops/settings');
      form.setFieldsValue({
        ...defaultSettings,
        ...data,
        preliminaryLabels: Array.isArray(data.preliminaryLabels)
          ? data.preliminaryLabels.join(', ')
          : data.preliminaryLabels || defaultSettings.preliminaryLabels,
        postImplementationLabels: Array.isArray(data.postImplementationLabels)
          ? data.postImplementationLabels.join(', ')
          : data.postImplementationLabels || defaultSettings.postImplementationLabels,
      });
    } catch (err) {
      form.setFieldsValue(defaultSettings);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchSettings();
  }, []);

  const handleSave = async (values: FinDevOpsSettings) => {
    setSaving(true);
    try {
      const payload = {
        ...values,
        preliminaryLabels: values.preliminaryLabels
          ? values.preliminaryLabels.split(',').map((s: string) => s.trim()).filter(Boolean)
          : [],
        postImplementationLabels: values.postImplementationLabels
          ? values.postImplementationLabels.split(',').map((s: string) => s.trim()).filter(Boolean)
          : [],
      };
      await request('/plugins/findevops/settings', {
        method: 'PUT',
        data: payload,
      });
      message.success('Settings saved successfully');
    } catch (err) {
      message.error('Failed to save settings');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div style={{ padding: 24, maxWidth: 800 }}>
      <Title level={2}>
        <DollarOutlined /> FinDevOps Settings
      </Title>
      <Paragraph type="secondary">
        Configure cost allocation, capitalization rules, and financial reporting parameters.
      </Paragraph>

      <Card loading={loading}>
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSave}
          initialValues={defaultSettings}
        >
          <Title level={4}>Cost Calculation</Title>

          <Form.Item
            name="defaultHourlyRate"
            label="Default Hourly Rate ($)"
            tooltip="Blended hourly rate for cost calculations when team-specific rates are unavailable"
          >
            <InputNumber
              min={0}
              max={1000}
              step={0.5}
              precision={2}
              prefix="$"
              style={{ width: 200 }}
            />
          </Form.Item>

          <Form.Item
            name="hoursPerStoryPoint"
            label="Hours per Story Point"
            tooltip="Average hours of effort per story point for cost estimation"
          >
            <InputNumber min={0.5} max={40} step={0.5} style={{ width: 200 }} />
          </Form.Item>

          <Form.Item
            name="unallocatedCostThreshold"
            label="Unallocated Cost Alert Threshold (%)"
            tooltip="Alert when unallocated costs exceed this percentage"
          >
            <InputNumber min={0} max={100} style={{ width: 200 }} addonAfter="%" />
          </Form.Item>

          <Divider />

          <Title level={4}>Capitalization (ASC 350-40)</Title>

          <Form.Item
            name="capitalizationFramework"
            label="Capitalization Framework"
            tooltip="Method for determining capitalizable software development costs"
          >
            <Select style={{ width: 300 }}>
              <Option value="asc_350_40_stages">ASC 350-40 (Stage-Based)</Option>
              <Option value="asc_350_40_probable">ASC 350-40 (Probable Success)</Option>
            </Select>
          </Form.Item>

          <Collapse>
            <Panel header="Label Mappings for Capitalization" key="labels">
              <Form.Item
                name="preliminaryLabels"
                label="Preliminary Stage Labels (Expense)"
                tooltip="Comma-separated labels for work that should be expensed (preliminary stage)"
              >
                <TextArea
                  rows={2}
                  placeholder="research, spike, discovery, prototype"
                />
              </Form.Item>

              <Form.Item
                name="postImplementationLabels"
                label="Post-Implementation Labels (Expense)"
                tooltip="Comma-separated labels for post-implementation work (maintenance, support)"
              >
                <TextArea
                  rows={2}
                  placeholder="maintenance, support, bug-fix, training"
                />
              </Form.Item>
            </Panel>
          </Collapse>

          <Divider />

          <Title level={4}>Fiscal Calendar</Title>

          <Form.Item
            name="fiscalYearStartMonth"
            label="Fiscal Year Start Month"
            tooltip="First month of your organization's fiscal year"
          >
            <Select style={{ width: 200 }}>
              {months.map((m) => (
                <Option key={m.value} value={m.value}>
                  {m.label}
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Divider />

          <Form.Item>
            <Space>
              <Button
                type="primary"
                htmlType="submit"
                icon={<SaveOutlined />}
                loading={saving}
              >
                Save Settings
              </Button>
              <Button
                icon={<ReloadOutlined />}
                onClick={fetchSettings}
                loading={loading}
              >
                Reset
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
};

export default FinDevOpsSettings;
