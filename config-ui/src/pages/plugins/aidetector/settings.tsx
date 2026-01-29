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
import { Card, Form, Slider, InputNumber, Switch, Input, Button, message, Typography, Space, Divider } from 'antd';
import { SaveOutlined, ReloadOutlined } from '@ant-design/icons';

import { request } from '@/utils';

const { Title, Text, Paragraph } = Typography;
const { TextArea } = Input;

interface AIDetectorSettings {
  confidenceThreshold: number;
  churnWindowDays: number;
  enableChurnTracking: boolean;
  excludeAuthors: string;
}

const defaultSettings: AIDetectorSettings = {
  confidenceThreshold: 70,
  churnWindowDays: 30,
  enableChurnTracking: true,
  excludeAuthors: '',
};

export const AIDetectorSettings = () => {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);

  const fetchSettings = async () => {
    setLoading(true);
    try {
      const data = await request('/plugins/aidetector/settings');
      form.setFieldsValue({
        ...defaultSettings,
        ...data,
        excludeAuthors: Array.isArray(data.excludeAuthors)
          ? data.excludeAuthors.join(', ')
          : data.excludeAuthors || '',
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

  const handleSave = async (values: AIDetectorSettings) => {
    setSaving(true);
    try {
      const payload = {
        ...values,
        excludeAuthors: values.excludeAuthors
          ? values.excludeAuthors.split(',').map((s: string) => s.trim()).filter(Boolean)
          : [],
      };
      await request('/plugins/aidetector/settings', {
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
      <Title level={2}>AI Detector Settings</Title>
      <Paragraph type="secondary">
        Configure how AI-assisted code contributions are detected and analyzed.
      </Paragraph>

      <Card loading={loading}>
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSave}
          initialValues={defaultSettings}
        >
          <Title level={4}>Detection Settings</Title>

          <Form.Item
            name="confidenceThreshold"
            label="Confidence Threshold"
            tooltip="Minimum confidence level (0-100) to classify a commit as AI-assisted"
          >
            <Slider
              min={0}
              max={100}
              marks={{
                0: '0%',
                50: '50%',
                70: '70% (Default)',
                100: '100%',
              }}
            />
          </Form.Item>

          <Form.Item
            name="excludeAuthors"
            label="Excluded Authors"
            tooltip="Comma-separated list of authors to exclude from AI detection (e.g., bot accounts)"
          >
            <TextArea
              rows={3}
              placeholder="dependabot[bot], renovate[bot], github-actions[bot]"
            />
          </Form.Item>

          <Divider />

          <Title level={4}>Code Churn Analysis</Title>
          <Paragraph type="secondary">
            Track how AI-assisted code changes over time after initial commit.
          </Paragraph>

          <Form.Item
            name="enableChurnTracking"
            label="Enable Churn Tracking"
            valuePropName="checked"
          >
            <Switch />
          </Form.Item>

          <Form.Item
            name="churnWindowDays"
            label="Churn Analysis Window (Days)"
            tooltip="Number of days after merge to track code changes"
          >
            <InputNumber min={1} max={365} style={{ width: 200 }} />
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

export default AIDetectorSettings;
