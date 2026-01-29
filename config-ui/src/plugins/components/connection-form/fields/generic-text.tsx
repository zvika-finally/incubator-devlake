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

import { useEffect } from 'react';
import { Input, InputNumber } from 'antd';

import { Block } from '@/components';

interface Props {
  type: 'create' | 'update';
  fieldKey: string;
  label?: string;
  subLabel?: string;
  placeholder?: string;
  required?: boolean;
  inputType?: 'text' | 'password' | 'number';
  defaultValue?: string | number;
  initialValue: string | number;
  value: string | number;
  error: string;
  setValue: (value: string | number | undefined) => void;
  setError: (value: string | undefined) => void;
}

export const ConnectionGenericText = ({
  type,
  fieldKey,
  label,
  subLabel,
  placeholder,
  required = false,
  inputType = 'text',
  defaultValue,
  initialValue,
  value,
  setValue,
  setError,
}: Props) => {
  useEffect(() => {
    if (type === 'create') {
      setValue(initialValue ?? defaultValue);
    }
  }, [type, initialValue, defaultValue]);

  useEffect(() => {
    if (required && !value) {
      setError(`${label || fieldKey} is required`);
    } else {
      setError(undefined);
    }
  }, [value, required, label, fieldKey]);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setValue(e.target.value);
  };

  const handleNumberChange = (val: number | null) => {
    setValue(val ?? undefined);
  };

  const displayLabel = label || fieldKey.charAt(0).toUpperCase() + fieldKey.slice(1).replace(/([A-Z])/g, ' $1');

  if (inputType === 'number') {
    return (
      <Block title={displayLabel} description={subLabel} required={required}>
        <InputNumber
          style={{ width: 386 }}
          placeholder={placeholder || `Enter ${displayLabel.toLowerCase()}`}
          value={value as number}
          onChange={handleNumberChange}
        />
      </Block>
    );
  }

  if (inputType === 'password') {
    return (
      <Block title={displayLabel} description={subLabel} required={required}>
        <Input.Password
          style={{ width: 386 }}
          placeholder={type === 'update' ? '********' : placeholder || `Enter ${displayLabel.toLowerCase()}`}
          value={value as string}
          onChange={handleChange}
        />
      </Block>
    );
  }

  return (
    <Block title={displayLabel} description={subLabel} required={required}>
      <Input
        style={{ width: 386 }}
        placeholder={placeholder || `Enter ${displayLabel.toLowerCase()}`}
        value={value as string}
        onChange={handleChange}
      />
    </Block>
  );
};
