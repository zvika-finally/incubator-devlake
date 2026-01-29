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
import { useParams } from 'react-router-dom';
import {
  Card,
  Table,
  Button,
  Modal,
  Form,
  Input,
  InputNumber,
  Select,
  Switch,
  Space,
  Tag,
  message,
  Typography,
  Popconfirm,
  Tabs,
  Alert,
  Statistic,
  Row,
  Col,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  CheckCircleOutlined,
  WarningOutlined,
  CloseCircleOutlined,
} from '@ant-design/icons';

import { request } from '@/utils';

const { Title, Paragraph } = Typography;
const { Option } = Select;

interface WorkingAgreement {
  id: string;
  projectName: string;
  agreementType: string;
  thresholdValue: number;
  unit: string;
  alertEnabled: boolean;
  createdAt: string;
  updatedAt: string;
}

interface Violation {
  id: string;
  agreementType: string;
  entityType: string;
  entityId: string;
  actualValue: number;
  thresholdValue: number;
  violatedAt: string;
  isResolved: boolean;
}

interface ComplianceSummary {
  agreementType: string;
  totalChecked: number;
  totalViolations: number;
  complianceRate: number;
  periodStart: string;
  periodEnd: string;
}

const agreementTypeOptions = [
  { value: 'pr_merge_time', label: 'PR Merge Time', defaultUnit: 'hours' },
  { value: 'review_turnaround', label: 'Review Turnaround', defaultUnit: 'hours' },
  { value: 'wip_limit', label: 'WIP Limit', defaultUnit: 'count' },
  { value: 'issues_in_progress', label: 'Issues In Progress', defaultUnit: 'count' },
];

const unitOptions = ['hours', 'days', 'count'];

export const WorkingAgreementsPage = () => {
  const { projectName } = useParams<{ projectName: string }>();
  const [agreements, setAgreements] = useState<WorkingAgreement[]>([]);
  const [violations, setViolations] = useState<Violation[]>([]);
  const [compliance, setCompliance] = useState<ComplianceSummary[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingAgreement, setEditingAgreement] = useState<WorkingAgreement | null>(null);
  const [form] = Form.useForm();

  const fetchData = async () => {
    if (!projectName) return;
    setLoading(true);
    try {
      const [agreementsData, violationsData, complianceData] = await Promise.all([
        request(`/plugins/businessmetrics/agreements/${projectName}`),
        request(`/plugins/businessmetrics/violations/${projectName}`),
        request(`/plugins/businessmetrics/compliance/${projectName}`),
      ]);
      setAgreements(agreementsData || []);
      setViolations(violationsData || []);
      setCompliance(complianceData || []);
    } catch (err) {
      message.error('Failed to fetch data');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, [projectName]);

  const handleSave = async (values: any) => {
    try {
      await request(`/plugins/businessmetrics/agreements/${projectName}`, {
        method: 'POST',
        data: {
          ...values,
          projectName,
        },
      });
      message.success('Agreement saved successfully');
      setModalVisible(false);
      form.resetFields();
      setEditingAgreement(null);
      fetchData();
    } catch (err) {
      message.error('Failed to save agreement');
    }
  };

  const handleDelete = async (agreementType: string) => {
    try {
      await request(`/plugins/businessmetrics/agreements/${projectName}/${agreementType}`, {
        method: 'DELETE',
      });
      message.success('Agreement deleted');
      fetchData();
    } catch (err) {
      message.error('Failed to delete agreement');
    }
  };

  const openEditModal = (agreement: WorkingAgreement) => {
    setEditingAgreement(agreement);
    form.setFieldsValue(agreement);
    setModalVisible(true);
  };

  const openCreateModal = () => {
    setEditingAgreement(null);
    form.resetFields();
    setModalVisible(true);
  };

  const agreementColumns = [
    {
      title: 'Type',
      dataIndex: 'agreementType',
      key: 'agreementType',
      render: (type: string) => {
        const opt = agreementTypeOptions.find((o) => o.value === type);
        return opt?.label || type;
      },
    },
    {
      title: 'Threshold',
      key: 'threshold',
      render: (_: any, record: WorkingAgreement) => (
        <span>
          {record.thresholdValue} {record.unit}
        </span>
      ),
    },
    {
      title: 'Alert',
      dataIndex: 'alertEnabled',
      key: 'alertEnabled',
      render: (enabled: boolean) => (
        <Tag color={enabled ? 'green' : 'default'}>{enabled ? 'Enabled' : 'Disabled'}</Tag>
      ),
    },
    {
      title: 'Actions',
      key: 'actions',
      render: (_: any, record: WorkingAgreement) => (
        <Space>
          <Button type="link" icon={<EditOutlined />} onClick={() => openEditModal(record)}>
            Edit
          </Button>
          <Popconfirm
            title="Delete this agreement?"
            onConfirm={() => handleDelete(record.agreementType)}
          >
            <Button type="link" danger icon={<DeleteOutlined />}>
              Delete
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const violationColumns = [
    {
      title: 'Agreement',
      dataIndex: 'agreementType',
      key: 'agreementType',
      render: (type: string) => {
        const opt = agreementTypeOptions.find((o) => o.value === type);
        return opt?.label || type;
      },
    },
    {
      title: 'Entity',
      key: 'entity',
      render: (_: any, record: Violation) => (
        <span>
          {record.entityType}: {record.entityId}
        </span>
      ),
    },
    {
      title: 'Actual vs Threshold',
      key: 'values',
      render: (_: any, record: Violation) => (
        <span style={{ color: '#f5222d' }}>
          {record.actualValue} / {record.thresholdValue}
        </span>
      ),
    },
    {
      title: 'Violated At',
      dataIndex: 'violatedAt',
      key: 'violatedAt',
      render: (date: string) => new Date(date).toLocaleDateString(),
    },
    {
      title: 'Status',
      dataIndex: 'isResolved',
      key: 'isResolved',
      render: (resolved: boolean) =>
        resolved ? (
          <Tag icon={<CheckCircleOutlined />} color="success">
            Resolved
          </Tag>
        ) : (
          <Tag icon={<WarningOutlined />} color="warning">
            Active
          </Tag>
        ),
    },
  ];

  const overallCompliance =
    compliance.length > 0
      ? compliance.reduce((acc, c) => acc + c.complianceRate, 0) / compliance.length
      : 100;

  return (
    <div style={{ padding: 24 }}>
      <Title level={2}>Working Agreements: {projectName}</Title>
      <Paragraph type="secondary">
        Define team working agreements and track compliance over time.
      </Paragraph>

      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title="Overall Compliance"
              value={overallCompliance}
              precision={1}
              suffix="%"
              valueStyle={{ color: overallCompliance >= 80 ? '#3f8600' : '#cf1322' }}
              prefix={overallCompliance >= 80 ? <CheckCircleOutlined /> : <CloseCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="Active Agreements" value={agreements.length} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="Active Violations"
              value={violations.filter((v) => !v.isResolved).length}
              valueStyle={{ color: violations.filter((v) => !v.isResolved).length > 0 ? '#cf1322' : '#3f8600' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="Total Violations" value={violations.length} />
          </Card>
        </Col>
      </Row>

      <Tabs
        items={[
          {
            key: 'agreements',
            label: 'Agreements',
            children: (
              <Card
                title="Working Agreements"
                extra={
                  <Button type="primary" icon={<PlusOutlined />} onClick={openCreateModal}>
                    Add Agreement
                  </Button>
                }
              >
                <Table
                  dataSource={agreements}
                  columns={agreementColumns}
                  rowKey="id"
                  loading={loading}
                />
              </Card>
            ),
          },
          {
            key: 'violations',
            label: `Violations (${violations.filter((v) => !v.isResolved).length})`,
            children: (
              <Card title="Agreement Violations">
                {violations.filter((v) => !v.isResolved).length > 0 && (
                  <Alert
                    message="Active Violations Detected"
                    description="Review and address the following agreement violations."
                    type="warning"
                    showIcon
                    style={{ marginBottom: 16 }}
                  />
                )}
                <Table
                  dataSource={violations}
                  columns={violationColumns}
                  rowKey="id"
                  loading={loading}
                />
              </Card>
            ),
          },
          {
            key: 'compliance',
            label: 'Compliance History',
            children: (
              <Card title="Compliance Summary">
                <Table
                  dataSource={compliance}
                  columns={[
                    {
                      title: 'Agreement',
                      dataIndex: 'agreementType',
                      render: (type: string) => {
                        const opt = agreementTypeOptions.find((o) => o.value === type);
                        return opt?.label || type;
                      },
                    },
                    { title: 'Checked', dataIndex: 'totalChecked' },
                    { title: 'Violations', dataIndex: 'totalViolations' },
                    {
                      title: 'Compliance Rate',
                      dataIndex: 'complianceRate',
                      render: (rate: number) => (
                        <Tag color={rate >= 80 ? 'green' : rate >= 60 ? 'orange' : 'red'}>
                          {rate.toFixed(1)}%
                        </Tag>
                      ),
                    },
                    {
                      title: 'Period',
                      key: 'period',
                      render: (_: any, record: ComplianceSummary) =>
                        `${new Date(record.periodStart).toLocaleDateString()} - ${new Date(record.periodEnd).toLocaleDateString()}`,
                    },
                  ]}
                  rowKey={(r) => `${r.agreementType}-${r.periodStart}`}
                  loading={loading}
                />
              </Card>
            ),
          },
        ]}
      />

      <Modal
        title={editingAgreement ? 'Edit Agreement' : 'Add Agreement'}
        open={modalVisible}
        onCancel={() => {
          setModalVisible(false);
          form.resetFields();
          setEditingAgreement(null);
        }}
        footer={null}
      >
        <Form form={form} layout="vertical" onFinish={handleSave}>
          <Form.Item
            name="agreementType"
            label="Agreement Type"
            rules={[{ required: true, message: 'Please select an agreement type' }]}
          >
            <Select
              placeholder="Select type"
              onChange={(value) => {
                const opt = agreementTypeOptions.find((o) => o.value === value);
                if (opt) {
                  form.setFieldValue('unit', opt.defaultUnit);
                }
              }}
            >
              {agreementTypeOptions.map((opt) => (
                <Option key={opt.value} value={opt.value}>
                  {opt.label}
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            name="thresholdValue"
            label="Threshold Value"
            rules={[{ required: true, message: 'Please enter a threshold value' }]}
          >
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name="unit"
            label="Unit"
            rules={[{ required: true, message: 'Please select a unit' }]}
          >
            <Select placeholder="Select unit">
              {unitOptions.map((u) => (
                <Option key={u} value={u}>
                  {u}
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item name="alertEnabled" label="Enable Alerts" valuePropName="checked">
            <Switch />
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                Save
              </Button>
              <Button onClick={() => setModalVisible(false)}>Cancel</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default WorkingAgreementsPage;
