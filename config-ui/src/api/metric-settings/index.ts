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

import { request } from '@/utils';

// Generic metric settings API for metric plugins (aidetector, businessmetrics, capacityplanner, findevops)

export const get = <T>(plugin: string, projectName: string): Promise<T> =>
  request(`/plugins/${plugin}/settings/${encodeURIComponent(projectName)}`);

export const update = <T>(plugin: string, projectName: string, data: Partial<T>): Promise<T> =>
  request(`/plugins/${plugin}/settings/${encodeURIComponent(projectName)}`, {
    method: 'put',
    data,
  });

export const remove = (plugin: string, projectName: string): Promise<{ message: string }> =>
  request(`/plugins/${plugin}/settings/${encodeURIComponent(projectName)}`, {
    method: 'delete',
  });

export const list = <T>(plugin: string): Promise<T[]> => request(`/plugins/${plugin}/settings`);

// Type definitions for settings

export interface AIDetectorSettings {
  id?: number;
  projectName?: string;
  confidenceTrailer: number;
  confidenceBody: number;
  confidenceGeneric: number;
  confidenceEmail: number;
  detectionThreshold: number;
  explicitSignalWeight: number;
  behavioralSignalWeight: number;
  prPatternWeight: number;
  customToolPatterns: string;
  analyzeHistorical: boolean;
}

export interface BusinessMetricsSettings {
  id?: number;
  projectName?: string;
  eliteDeployFreq: number;
  eliteLeadTimeHours: number;
  eliteCfr: number;
  eliteMttrHours: number;
  eliteThreshold: number;
  highThreshold: number;
  mediumThreshold: number;
  investmentLabelPrefix: string;
  stageLabelPrefix: string;
  businessValueWeights: string;
}

export interface CapacityPlannerSettings {
  id?: number;
  projectName?: string;
  monteCarloIterations: number;
  velocityVariance: number;
  defaultVelocity: number;
  sprintDurationWeeks: number;
  velocitySprintCount: number;
  rampUpWeeks: number;
  newHireProductivity: number;
  channelOverhead: number;
  defaultDeveloperCost: number;
  roiTimeHorizonMonths: number;
  forecastPercentiles: string;
}

export interface FinDevOpsSettings {
  id?: number;
  projectName?: string;
  defaultHourlyRate: number;
  roleRates: string;
  hoursPerStoryPoint: number;
  capitalizationFramework: string;
  preliminaryLabels: string;
  postImplementationLabels: string;
  preliminaryTypes: string;
  developmentTypes: string;
  postImplementationTypes: string;
}
