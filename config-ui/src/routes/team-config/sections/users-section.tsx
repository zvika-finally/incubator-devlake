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

import { getUsersCsv, putUsersCsv, parseCsv } from '../api';

type UserRow = { Id: string; Name: string; Email: string; TeamIds: string };

export const UsersSection = () => {
  const [rows, setRows] = useState<UserRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [pending, setPending] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);

  const refresh = async () => {
    setLoading(true);
    try {
      const csv = await getUsersCsv();
      setRows(parseCsv(csv) as UserRow[]);
    } catch (err) {
      notification.error({ message: 'Failed to load users', description: String((err as Error).message) });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    refresh();
  }, []);

  const downloadCurrent = async () => {
    try {
      const csv = await getUsersCsv();
      saveAs(new Blob([csv], { type: 'text/csv' }), 'users.csv');
    } catch (err) {
      notification.error({ message: 'Download failed', description: String((err as Error).message) });
    }
  };

  const downloadTemplate = async () => {
    try {
      const csv = await getUsersCsv(true);
      saveAs(new Blob([csv], { type: 'text/csv' }), 'users-template.csv');
    } catch (err) {
      notification.error({ message: 'Template download failed', description: String((err as Error).message) });
    }
  };

  const upload = async () => {
    if (!pending) return;
    setUploading(true);
    try {
      await putUsersCsv(pending);
      notification.success({ message: 'users.csv uploaded' });
      setPending(null);
      await refresh();
    } catch (err) {
      notification.error({ message: 'Upload failed', description: String((err as Error).message) });
    } finally {
      setUploading(false);
    }
  };

  return (
    <Card title="Users" extra={
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
        <p className="ant-upload-text">Click or drag a users.csv file here</p>
        <p className="ant-upload-hint">Headers must be Id,Name,Email,TeamIds (case-sensitive).</p>
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
            { title: 'Id', dataIndex: 'Id', key: 'Id' },
            { title: 'Name', dataIndex: 'Name', key: 'Name' },
            { title: 'Email', dataIndex: 'Email', key: 'Email' },
            { title: 'Team IDs', dataIndex: 'TeamIds', key: 'TeamIds' },
          ]}
          locale={{ emptyText: 'No users yet — upload a CSV to get started.' }}
        />
      </Spin>
    </Card>
  );
};
