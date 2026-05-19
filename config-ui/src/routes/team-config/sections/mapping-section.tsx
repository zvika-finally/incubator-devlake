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

import { useEffect, useState } from 'react';
import { Card, Upload, Button, Table, Space, notification, Spin } from 'antd';
import { InboxOutlined, DownloadOutlined, UploadOutlined } from '@ant-design/icons';
import { saveAs } from 'file-saver';

import { getMappingCsv, putMappingCsv, parseCsv } from '../api';

type MappingRow = { Id: string; UserId: string };

export const MappingSection = () => {
  const [rows, setRows] = useState<MappingRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [pending, setPending] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);

  const refresh = async () => {
    setLoading(true);
    try {
      const csv = await getMappingCsv();
      setRows(parseCsv(csv) as MappingRow[]);
    } catch (err) {
      notification.error({ message: 'Failed to load mappings', description: String((err as Error).message) });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    refresh();
  }, []);

  const downloadCurrent = async () => {
    try {
      const csv = await getMappingCsv();
      saveAs(new Blob([csv], { type: 'text/csv' }), 'user_account_mapping.csv');
    } catch (err) {
      notification.error({ message: 'Download failed', description: String((err as Error).message) });
    }
  };

  const downloadTemplate = async () => {
    try {
      const csv = await getMappingCsv(true);
      saveAs(new Blob([csv], { type: 'text/csv' }), 'user_account_mapping-template.csv');
    } catch (err) {
      notification.error({ message: 'Template download failed', description: String((err as Error).message) });
    }
  };

  const upload = async () => {
    if (!pending) return;
    setUploading(true);
    try {
      await putMappingCsv(pending);
      notification.success({ message: 'user_account_mapping.csv uploaded' });
      setPending(null);
      await refresh();
    } catch (err) {
      notification.error({ message: 'Upload failed', description: String((err as Error).message) });
    } finally {
      setUploading(false);
    }
  };

  return (
    <Card title="User ↔ Account Mapping" extra={
      <Space>
        <Button icon={<DownloadOutlined />} onClick={downloadTemplate}>Template</Button>
        <Button icon={<DownloadOutlined />} onClick={downloadCurrent}>Current</Button>
      </Space>
    }>
      <Upload.Dragger
        accept=".csv"
        multiple={false}
        beforeUpload={(file) => {
          setPending(file);
          return false;
        }}
        fileList={pending ? [{ uid: '1', name: pending.name, status: 'done' as const }] : []}
        onRemove={() => setPending(null)}
      >
        <p className="ant-upload-drag-icon"><InboxOutlined /></p>
        <p className="ant-upload-text">Click or drag a user_account_mapping.csv file here</p>
        <p className="ant-upload-hint">Headers must be Id,UserId (case-sensitive). Id is the source account_id; UserId is the canonical user.</p>
      </Upload.Dragger>
      <div style={{ marginTop: 12, marginBottom: 24 }}>
        <Button type="primary" icon={<UploadOutlined />} disabled={!pending} loading={uploading} onClick={upload}>
          Upload
        </Button>
      </div>
      <Spin spinning={loading}>
        <Table
          rowKey="Id"
          dataSource={rows}
          size="small"
          pagination={{ pageSize: 50 }}
          columns={[
            { title: 'Account ID', dataIndex: 'Id', key: 'Id' },
            { title: 'User ID', dataIndex: 'UserId', key: 'UserId' },
          ]}
          locale={{ emptyText: 'No mappings yet — upload a CSV or run the auto-mapping below.' }}
        />
      </Spin>
    </Card>
  );
};
