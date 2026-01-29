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
import { Card, Form, InputNumber, Input, Button, message, Typography, Space, Divider, Alert } from 'antd';
import { SaveOutlined, ReloadOutlined } from '@ant-design/icons';

import { request } from '@/utils';

const { Title, Paragraph } = Typography;
const { TextArea } = Input;

interface CapacityPlannerSettings {
  velocitySprintCount: number;
  sprintDurationWeeks: number;
  monteCarloIterations: number;
  rampUpWeeks: number;
  activeStatuses: string;
}

const defaultSettings: CapacityPlannerSettings = {
  velocitySprintCount: 6,
  sprintDurationWeeks: 2,
  monteCarloIterations: 1000,
  rampUpWeeks: 8,
  activeStatuses: '["In Progress", "In Review", "Testing"]',
};

export const CapacityPlannerSettings = () => {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);

  const fetchSettings = async () => {
    setLoading(true);
    try {
      const data = await request('/plugins/capacityplanner/settings');
      form.setFieldsValue({
        ...defaultSettings,
        ...data,
        activeStatuses: Array.isArray(data.activeStatuses)
          ? JSON.stringify(data.activeStatuses, null, 2)
          : data.activeStatuses || defaultSettings.activeStatuses,
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

  const handleSave = async (values: CapacityPlannerSettings) => {
    setSaving(true);
    try {
      let activeStatuses;
      try {
        activeStatuses = JSON.parse(values.activeStatuses);
      } catch {
        message.error('Invalid JSON format for Active Statuses');
        setSaving(false);
        return;
      }

      await request('/plugins/capacityplanner/settings', {
        method: 'PUT',
        data: {
          ...values,
          activeStatuses,
        },
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
      <Title level={2}>Capacity Planner Settings</Title>
      <Paragraph type="secondary">
        Configure velocity calculation, Monte Carlo simulations, and flow efficiency analysis.
      </Paragraph>

      <Card loading={loading}>
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSave}
          initialValues={defaultSettings}
        >
          <Title level={4}>Velocity Calculation</Title>

          <Form.Item
            name="velocitySprintCount"
            label="Sprint Count for Velocity"
            tooltip="Number of past sprints to include in velocity calculation"
          >
            <InputNumber min={1} max={20} style={{ width: 200 }} />
          </Form.Item>

          <Form.Item
            name="sprintDurationWeeks"
            label="Sprint Duration (Weeks)"
            tooltip="Duration of each sprint in weeks"
          >
            <InputNumber min={1} max={8} style={{ width: 200 }} />
          </Form.Item>

          <Divider />

          <Title level={4}>Monte Carlo Forecasting</Title>

          <Form.Item
            name="monteCarloIterations"
            label="Simulation Iterations"
            tooltip="Number of Monte Carlo simulation iterations for forecasting"
          >
            <InputNumber min={100} max={100000} step={100} style={{ width: 200 }} />
          </Form.Item>

          <Alert
            message="Higher iterations increase accuracy but take longer to compute"
            type="info"
            showIcon
            style={{ marginBottom: 16 }}
          />

          <Divider />

          <Title level={4}>Brooks's Law Modeling</Title>

          <Form.Item
            name="rampUpWeeks"
            label="New Developer Ramp-Up (Weeks)"
            tooltip="Weeks for new developers to reach full productivity (Brooks's Law)"
          >
            <InputNumber min={1} max={26} style={{ width: 200 }} />
          </Form.Item>

          <Divider />

          <Title level={4}>Flow Efficiency</Title>

          <Form.Item
            name="activeStatuses"
            label="Active Work Statuses (JSON Array)"
            tooltip="Issue statuses that count as active work time (vs wait time)"
          >
            <TextArea
              rows={4}
              placeholder='["In Progress", "In Review", "Testing"]'
            />
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

export default CapacityPlannerSettings;
