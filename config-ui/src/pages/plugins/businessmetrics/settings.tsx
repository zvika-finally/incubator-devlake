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
import { Card, Form, Input, InputNumber, Button, message, Typography, Space, Divider, Collapse } from 'antd';
import { SaveOutlined, ReloadOutlined } from '@ant-design/icons';

import { request } from '@/utils';

const { Title, Paragraph } = Typography;
const { Panel } = Collapse;

interface BusinessMetricsSettings {
  investmentLabelPrefix: string;
  stageLabelPrefix: string;
  goalLabelPrefix: string;
  eliteDeployFreq: number;
  eliteLeadTimeHours: number;
  eliteCFR: number;
  eliteMTTRHours: number;
}

const defaultSettings: BusinessMetricsSettings = {
  investmentLabelPrefix: 'investment:',
  stageLabelPrefix: 'stage:',
  goalLabelPrefix: 'goal:',
  eliteDeployFreq: 7,
  eliteLeadTimeHours: 24,
  eliteCFR: 5,
  eliteMTTRHours: 1,
};

export const BusinessMetricsSettings = () => {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);

  const fetchSettings = async () => {
    setLoading(true);
    try {
      const data = await request('/plugins/businessmetrics/settings');
      form.setFieldsValue({ ...defaultSettings, ...data });
    } catch (err) {
      form.setFieldsValue(defaultSettings);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchSettings();
  }, []);

  const handleSave = async (values: BusinessMetricsSettings) => {
    setSaving(true);
    try {
      await request('/plugins/businessmetrics/settings', {
        method: 'PUT',
        data: values,
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
      <Title level={2}>Business Metrics Settings</Title>
      <Paragraph type="secondary">
        Configure label prefixes for categorizing work items and DORA performance thresholds.
      </Paragraph>

      <Card loading={loading}>
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSave}
          initialValues={defaultSettings}
        >
          <Title level={4}>Label Prefixes</Title>
          <Paragraph type="secondary">
            Define prefixes used to categorize issues and PRs for business alignment tracking.
          </Paragraph>

          <Form.Item
            name="investmentLabelPrefix"
            label="Investment Category Prefix"
            tooltip="Label prefix for investment categories (e.g., 'investment:feature', 'investment:tech-debt')"
          >
            <Input placeholder="investment:" />
          </Form.Item>

          <Form.Item
            name="stageLabelPrefix"
            label="Stage Prefix"
            tooltip="Label prefix for workflow stages"
          >
            <Input placeholder="stage:" />
          </Form.Item>

          <Form.Item
            name="goalLabelPrefix"
            label="Business Goal Prefix"
            tooltip="Label prefix for linking to business goals"
          >
            <Input placeholder="goal:" />
          </Form.Item>

          <Divider />

          <Collapse>
            <Panel header="DORA Elite Performer Thresholds" key="dora">
              <Paragraph type="secondary">
                Define thresholds for DORA elite performer classification.
              </Paragraph>

              <Form.Item
                name="eliteDeployFreq"
                label="Elite Deploy Frequency (deploys per week)"
                tooltip="Minimum deployments per week for elite classification"
              >
                <InputNumber min={1} max={100} style={{ width: 200 }} />
              </Form.Item>

              <Form.Item
                name="eliteLeadTimeHours"
                label="Elite Lead Time (hours)"
                tooltip="Maximum lead time in hours for elite classification"
              >
                <InputNumber min={1} max={720} style={{ width: 200 }} />
              </Form.Item>

              <Form.Item
                name="eliteCFR"
                label="Elite Change Failure Rate (%)"
                tooltip="Maximum change failure rate percentage for elite classification"
              >
                <InputNumber min={0} max={100} style={{ width: 200 }} />
              </Form.Item>

              <Form.Item
                name="eliteMTTRHours"
                label="Elite MTTR (hours)"
                tooltip="Maximum mean time to recovery in hours for elite classification"
              >
                <InputNumber min={0.1} max={168} step={0.5} style={{ width: 200 }} />
              </Form.Item>
            </Panel>
          </Collapse>

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

export default BusinessMetricsSettings;
