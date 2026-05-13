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

import { useState, useEffect } from 'react';
import { Tabs, Card, Form, InputNumber, Input, Checkbox, Button, Space, Divider, message, Tooltip } from 'antd';
import { InfoCircleOutlined } from '@ant-design/icons';

import API from '@/api';
import type {
  AIDetectorSettings,
  BusinessMetricsSettings,
  CapacityPlannerSettings,
  FinDevOpsSettings,
} from '@/api/metric-settings';
import { operator } from '@/utils';
import type { IProject } from '@/types';

interface Props {
  project: IProject;
}

export const MetricSettingsPanel = ({ project }: Props) => {
  const [activeTab, setActiveTab] = useState('aidetector');

  return (
    <Card>
      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        items={[
          {
            key: 'aidetector',
            label: 'AI Detector',
            children: <AIDetectorSettingsForm projectName={project.name} />,
          },
          {
            key: 'businessmetrics',
            label: 'Business Metrics',
            children: <BusinessMetricsSettingsForm projectName={project.name} />,
          },
          {
            key: 'capacityplanner',
            label: 'Capacity Planner',
            children: <CapacityPlannerSettingsForm projectName={project.name} />,
          },
          {
            key: 'findevops',
            label: 'FinDevOps',
            children: <FinDevOpsSettingsForm projectName={project.name} />,
          },
        ]}
      />
    </Card>
  );
};

// AI Detector Settings Form
const AIDetectorSettingsForm = ({ projectName }: { projectName: string }) => {
  const [form] = Form.useForm<AIDetectorSettings>();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    const loadSettings = async () => {
      setLoading(true);
      try {
        const settings = await API.metricSettings.get<AIDetectorSettings>('aidetector', projectName);
        form.setFieldsValue(settings);
      } catch {
        // Use defaults if no settings exist
      }
      setLoading(false);
    };
    loadSettings();
  }, [projectName, form]);

  const handleSave = async () => {
    const values = await form.validateFields();
    const [success] = await operator(() => API.metricSettings.update('aidetector', projectName, values), {
      setOperating: setSaving,
      formatMessage: () => 'AI Detector settings saved successfully.',
    });
    if (!success) {
      message.error('Failed to save settings');
    }
  };

  const handleReset = async () => {
    const [success] = await operator(() => API.metricSettings.remove('aidetector', projectName), {
      setOperating: setSaving,
      formatMessage: () => 'Settings reset to defaults.',
    });
    if (success) {
      // Reload defaults
      const settings = await API.metricSettings.get<AIDetectorSettings>('aidetector', projectName);
      form.setFieldsValue(settings);
    }
  };

  return (
    <Form form={form} layout="vertical" disabled={loading}>
      <Divider orientation="left">Confidence Thresholds (0-100)</Divider>
      <Space wrap>
        <Form.Item
          name="confidenceTrailer"
          label={
            <span>
              Git Trailer Confidence{' '}
              <Tooltip title="Confidence for Co-Authored-By trailers">
                <InfoCircleOutlined />
              </Tooltip>
            </span>
          }
        >
          <InputNumber min={0} max={100} />
        </Form.Item>
        <Form.Item
          name="confidenceBody"
          label={
            <span>
              Body/Message Confidence{' '}
              <Tooltip title="Confidence for PR body or commit message markers">
                <InfoCircleOutlined />
              </Tooltip>
            </span>
          }
        >
          <InputNumber min={0} max={100} />
        </Form.Item>
        <Form.Item name="confidenceGeneric" label="Generic AI Confidence">
          <InputNumber min={0} max={100} />
        </Form.Item>
        <Form.Item name="confidenceEmail" label="AI Email Confidence">
          <InputNumber min={0} max={100} />
        </Form.Item>
      </Space>

      <Divider orientation="left">Detection Settings</Divider>
      <Form.Item
        name="detectionThreshold"
        label="Detection Threshold"
        tooltip="Minimum confidence to flag as AI-assisted"
      >
        <InputNumber min={0} max={100} />
      </Form.Item>

      <Divider orientation="left">Scoring Weights (should sum to 100)</Divider>
      <Space>
        <Form.Item name="explicitSignalWeight" label="Explicit Signal Weight">
          <InputNumber min={0} max={100} />
        </Form.Item>
        <Form.Item name="behavioralSignalWeight" label="Behavioral Signal Weight">
          <InputNumber min={0} max={100} />
        </Form.Item>
        <Form.Item name="prPatternWeight" label="PR Pattern Weight">
          <InputNumber min={0} max={100} />
        </Form.Item>
      </Space>

      <Divider orientation="left">Options</Divider>
      <Form.Item name="analyzeHistorical" valuePropName="checked">
        <Checkbox>Analyze Historical PRs</Checkbox>
      </Form.Item>

      <Form.Item name="customToolPatterns" label="Custom Tool Patterns (JSON)">
        <Input.TextArea rows={3} placeholder='[{"tool": "my_tool", "patterns": ["pattern1"]}]' />
      </Form.Item>

      <Space>
        <Button type="primary" loading={saving} onClick={handleSave}>
          Save Settings
        </Button>
        <Button onClick={handleReset}>Reset to Defaults</Button>
      </Space>
    </Form>
  );
};

// Business Metrics Settings Form
const BusinessMetricsSettingsForm = ({ projectName }: { projectName: string }) => {
  const [form] = Form.useForm<BusinessMetricsSettings>();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    const loadSettings = async () => {
      setLoading(true);
      try {
        const settings = await API.metricSettings.get<BusinessMetricsSettings>('businessmetrics', projectName);
        form.setFieldsValue(settings);
      } catch {
        // Use defaults
      }
      setLoading(false);
    };
    loadSettings();
  }, [projectName, form]);

  const handleSave = async () => {
    const values = await form.validateFields();
    const [success] = await operator(() => API.metricSettings.update('businessmetrics', projectName, values), {
      setOperating: setSaving,
      formatMessage: () => 'Business Metrics settings saved successfully.',
    });
    if (!success) {
      message.error('Failed to save settings');
    }
  };

  const handleReset = async () => {
    const [success] = await operator(() => API.metricSettings.remove('businessmetrics', projectName), {
      setOperating: setSaving,
      formatMessage: () => 'Settings reset to defaults.',
    });
    if (success) {
      const settings = await API.metricSettings.get<BusinessMetricsSettings>('businessmetrics', projectName);
      form.setFieldsValue(settings);
    }
  };

  return (
    <Form form={form} layout="vertical" disabled={loading}>
      <Divider orientation="left">DORA Elite Benchmarks</Divider>
      <Space wrap>
        <Form.Item name="eliteDeployFreq" label="Elite Deploy Frequency (per day)">
          <InputNumber min={0} step={0.1} />
        </Form.Item>
        <Form.Item name="eliteLeadTimeHours" label="Elite Lead Time (hours)">
          <InputNumber min={0} step={1} />
        </Form.Item>
        <Form.Item name="eliteCfr" label="Elite Change Failure Rate (%)">
          <InputNumber min={0} max={100} step={0.1} />
        </Form.Item>
        <Form.Item name="eliteMttrHours" label="Elite MTTR (hours)">
          <InputNumber min={0} step={0.1} />
        </Form.Item>
      </Space>

      <Divider orientation="left">Health Level Thresholds (score out of 100)</Divider>
      <Space>
        <Form.Item name="eliteThreshold" label="Elite Threshold">
          <InputNumber min={0} max={100} />
        </Form.Item>
        <Form.Item name="highThreshold" label="High Threshold">
          <InputNumber min={0} max={100} />
        </Form.Item>
        <Form.Item name="mediumThreshold" label="Medium Threshold">
          <InputNumber min={0} max={100} />
        </Form.Item>
      </Space>

      <Divider orientation="left">Label Prefixes</Divider>
      <Space>
        <Form.Item name="investmentLabelPrefix" label="Investment Label Prefix">
          <Input placeholder="investment:" />
        </Form.Item>
        <Form.Item name="stageLabelPrefix" label="Stage Label Prefix">
          <Input placeholder="stage:" />
        </Form.Item>
      </Space>

      <Space>
        <Button type="primary" loading={saving} onClick={handleSave}>
          Save Settings
        </Button>
        <Button onClick={handleReset}>Reset to Defaults</Button>
      </Space>
    </Form>
  );
};

// Capacity Planner Settings Form
const CapacityPlannerSettingsForm = ({ projectName }: { projectName: string }) => {
  const [form] = Form.useForm<CapacityPlannerSettings>();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    const loadSettings = async () => {
      setLoading(true);
      try {
        const settings = await API.metricSettings.get<CapacityPlannerSettings>('capacityplanner', projectName);
        form.setFieldsValue(settings);
      } catch {
        // Use defaults
      }
      setLoading(false);
    };
    loadSettings();
  }, [projectName, form]);

  const handleSave = async () => {
    const values = await form.validateFields();
    const [success] = await operator(() => API.metricSettings.update('capacityplanner', projectName, values), {
      setOperating: setSaving,
      formatMessage: () => 'Capacity Planner settings saved successfully.',
    });
    if (!success) {
      message.error('Failed to save settings');
    }
  };

  const handleReset = async () => {
    const [success] = await operator(() => API.metricSettings.remove('capacityplanner', projectName), {
      setOperating: setSaving,
      formatMessage: () => 'Settings reset to defaults.',
    });
    if (success) {
      const settings = await API.metricSettings.get<CapacityPlannerSettings>('capacityplanner', projectName);
      form.setFieldsValue(settings);
    }
  };

  return (
    <Form form={form} layout="vertical" disabled={loading}>
      <Divider orientation="left">Monte Carlo Simulation</Divider>
      <Space wrap>
        <Form.Item name="monteCarloIterations" label="Iterations">
          <InputNumber min={100} max={10000} step={100} />
        </Form.Item>
        <Form.Item name="velocityVariance" label="Velocity Variance (0-1)">
          <InputNumber min={0} max={1} step={0.05} />
        </Form.Item>
        <Form.Item name="defaultVelocity" label="Default Velocity (story points/week)">
          <InputNumber min={1} step={1} />
        </Form.Item>
      </Space>

      <Divider orientation="left">Sprint Settings</Divider>
      <Space>
        <Form.Item name="sprintDurationWeeks" label="Sprint Duration (weeks)">
          <InputNumber min={1} max={8} />
        </Form.Item>
        <Form.Item name="velocitySprintCount" label="Sprints for Velocity Calculation">
          <InputNumber min={1} max={20} />
        </Form.Item>
      </Space>

      <Divider orientation="left">Brooks's Law Model</Divider>
      <Space wrap>
        <Form.Item name="rampUpWeeks" label="Ramp-up Weeks">
          <InputNumber min={0} step={0.5} />
        </Form.Item>
        <Form.Item name="newHireProductivity" label="New Hire Productivity (0-1)">
          <InputNumber min={0} max={1} step={0.1} />
        </Form.Item>
        <Form.Item name="channelOverhead" label="Communication Overhead (0-1)">
          <InputNumber min={0} max={1} step={0.05} />
        </Form.Item>
      </Space>

      <Divider orientation="left">ROI Calculation</Divider>
      <Space>
        <Form.Item name="defaultDeveloperCost" label="Default Developer Cost (annual)">
          <InputNumber min={0} step={1000} formatter={(value) => `$ ${value}`} />
        </Form.Item>
        <Form.Item name="roiTimeHorizonMonths" label="ROI Time Horizon (months)">
          <InputNumber min={1} max={60} />
        </Form.Item>
      </Space>

      <Space>
        <Button type="primary" loading={saving} onClick={handleSave}>
          Save Settings
        </Button>
        <Button onClick={handleReset}>Reset to Defaults</Button>
      </Space>
    </Form>
  );
};

// FinDevOps Settings Form
const FinDevOpsSettingsForm = ({ projectName }: { projectName: string }) => {
  const [form] = Form.useForm<FinDevOpsSettings>();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    const loadSettings = async () => {
      setLoading(true);
      try {
        const settings = await API.metricSettings.get<FinDevOpsSettings>('findevops', projectName);
        form.setFieldsValue(settings);
      } catch {
        // Use defaults
      }
      setLoading(false);
    };
    loadSettings();
  }, [projectName, form]);

  const handleSave = async () => {
    const values = await form.validateFields();
    const [success] = await operator(() => API.metricSettings.update('findevops', projectName, values), {
      setOperating: setSaving,
      formatMessage: () => 'FinDevOps settings saved successfully.',
    });
    if (!success) {
      message.error('Failed to save settings');
    }
  };

  const handleReset = async () => {
    const [success] = await operator(() => API.metricSettings.remove('findevops', projectName), {
      setOperating: setSaving,
      formatMessage: () => 'Settings reset to defaults.',
    });
    if (success) {
      const settings = await API.metricSettings.get<FinDevOpsSettings>('findevops', projectName);
      form.setFieldsValue(settings);
    }
  };

  return (
    <Form form={form} layout="vertical" disabled={loading}>
      <Divider orientation="left">Cost Settings</Divider>
      <Space>
        <Form.Item name="defaultHourlyRate" label="Default Hourly Rate">
          <InputNumber min={0} step={1} formatter={(value) => `$ ${value}`} />
        </Form.Item>
        <Form.Item name="hoursPerStoryPoint" label="Hours per Story Point">
          <InputNumber min={0.5} step={0.5} />
        </Form.Item>
      </Space>

      <Form.Item name="roleRates" label="Role Rates (JSON)" tooltip='{"engineer": 72, "senior": 96, "staff": 120}'>
        <Input.TextArea rows={2} placeholder='{"engineer": 72, "seniorEngineer": 96, "staffEngineer": 120}' />
      </Form.Item>

      <Divider orientation="left">Capitalization Framework</Divider>
      <Form.Item name="capitalizationFramework" label="Framework">
        <Input placeholder="asc_350_40_stages" />
      </Form.Item>

      <Divider orientation="left">ASC 350-40 Stage Labels (JSON arrays)</Divider>
      <Form.Item name="preliminaryLabels" label="Preliminary Stage Labels">
        <Input.TextArea rows={2} placeholder='["research", "spike", "investigation"]' />
      </Form.Item>
      <Form.Item name="postImplementationLabels" label="Post-Implementation Labels">
        <Input.TextArea rows={2} placeholder='["bug", "hotfix", "maintenance"]' />
      </Form.Item>

      <Divider orientation="left">Issue Type Mappings (JSON arrays)</Divider>
      <Form.Item name="preliminaryTypes" label="Preliminary Types">
        <Input placeholder='["Spike", "Research", "Discovery"]' />
      </Form.Item>
      <Form.Item name="developmentTypes" label="Development Types">
        <Input placeholder='["Story", "Feature", "Enhancement"]' />
      </Form.Item>
      <Form.Item name="postImplementationTypes" label="Post-Implementation Types">
        <Input placeholder='["Bug", "Defect", "Hotfix"]' />
      </Form.Item>

      <Space>
        <Button type="primary" loading={saving} onClick={handleSave}>
          Save Settings
        </Button>
        <Button onClick={handleReset}>Reset to Defaults</Button>
      </Space>
    </Form>
  );
};
