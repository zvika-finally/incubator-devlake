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
import { useNavigate } from 'react-router-dom';
import { Flex, Space, Card, Modal, Input, Checkbox, Button } from 'antd';

import API from '@/api';
import { Block, HelpTooltip, Message } from '@/components';
import { PATHS } from '@/config';
import { IProject } from '@/types';
import { operator } from '@/utils';

import * as S from './styled';

const RegexPrIssueDefaultValue = '(?mi)(Closes)[\\s]*.*(((and )?#\\d+[ ]*)+)';

interface Props {
  project: IProject;
  onRefresh: () => void;
}

export const SettingsPanel = ({ project, onRefresh }: Props) => {
  const [name, setName] = useState('');
  const [dora, setDora] = useState({
    enable: false,
  });
  const [linker, setLinker] = useState({
    enable: false,
    prToIssueRegexp: '',
  });
  const [issueTrace, setIssueTrace] = useState({
    enable: false,
  });
  const [aidetector, setAidetector] = useState({
    enable: false,
  });
  const [businessmetrics, setBusinessmetrics] = useState({
    enable: false,
  });
  const [capacityplanner, setCapacityplanner] = useState({
    enable: false,
  });
  const [findevops, setFindevops] = useState({
    enable: false,
  });
  const [aimeasure, setAimeasure] = useState({
    enable: false,
  });
  const [operating, setOperating] = useState(false);
  const [open, setOpen] = useState(false);

  const navigate = useNavigate();

  useEffect(() => {
    const dora = project.metrics.find((ms) => ms.pluginName === 'dora');
    const linker = project.metrics.find((ms) => ms.pluginName === 'linker');
    const issueTrace = project.metrics.find((ms) => ms.pluginName === 'issue_trace');
    const aidetector = project.metrics.find((ms) => ms.pluginName === 'aidetector');
    const businessmetrics = project.metrics.find((ms) => ms.pluginName === 'businessmetrics');
    const capacityplanner = project.metrics.find((ms) => ms.pluginName === 'capacityplanner');
    const findevops = project.metrics.find((ms) => ms.pluginName === 'findevops');
    const aimeasure = project.metrics.find((ms) => ms.pluginName === 'aimeasure');

    setName(project.name);
    setDora({
      enable: dora?.enable ?? false,
    });
    setLinker({
      enable: linker?.enable ?? false,
      prToIssueRegexp: linker?.pluginOption?.prToIssueRegexp ?? RegexPrIssueDefaultValue,
    });
    setIssueTrace({
      enable: issueTrace?.enable ?? false,
    });
    setAidetector({
      enable: aidetector?.enable ?? false,
    });
    setBusinessmetrics({
      enable: businessmetrics?.enable ?? false,
    });
    setCapacityplanner({
      enable: capacityplanner?.enable ?? false,
    });
    setFindevops({
      enable: findevops?.enable ?? false,
    });
    setAimeasure({
      enable: aimeasure?.enable ?? false,
    });
  }, [project]);

  const handleUpdate = async () => {
    const [success] = await operator(
      () =>
        API.project.update(project.name, {
          name,
          description: '',
          metrics: [
            {
              pluginName: 'dora',
              pluginOption: {},
              enable: dora.enable,
            },
            {
              pluginName: 'linker',
              pluginOption: {
                prToIssueRegexp: linker.prToIssueRegexp,
              },
              enable: linker.enable,
            },
            {
              pluginName: 'issue_trace',
              pluginOption: {},
              enable: issueTrace.enable,
            },
            {
              pluginName: 'aidetector',
              pluginOption: {},
              enable: aidetector.enable,
            },
            {
              pluginName: 'businessmetrics',
              pluginOption: {},
              enable: businessmetrics.enable,
            },
            {
              pluginName: 'capacityplanner',
              pluginOption: {},
              enable: capacityplanner.enable,
            },
            {
              pluginName: 'findevops',
              pluginOption: {},
              enable: findevops.enable,
            },
            {
              pluginName: 'aimeasure',
              pluginOption: {},
              enable: aimeasure.enable,
            },
          ],
        }),
      {
        setOperating,
      },
    );

    if (success) {
      onRefresh();
      navigate(PATHS.PROJECT(name), {
        state: {
          tabId: 'settings',
        },
      });
    }
  };

  const handleShowDeleteDialog = () => {
    setOpen(true);
  };

  const handleHideDeleteDialog = () => {
    setOpen(false);
  };

  const handleDelete = async () => {
    const [success] = await operator(() => API.project.remove(project.name), {
      setOperating,
      formatMessage: () => 'Delete project successful.',
    });

    if (success) {
      navigate(PATHS.PROJECTS());
    }
  };

  return (
    <Flex vertical>
      <Space direction="vertical" size="large">
        <Card>
          <Block title="Project Name" description="Edit your project name with letters, numbers, -, _ or /" required>
            <Input style={{ width: 386 }} value={name} onChange={(e) => setName(e.target.value)} />
          </Block>
          <Block
            title={
              <Checkbox checked={dora.enable} onChange={(e) => setDora({ enable: e.target.checked })}>
                Enable DORA Metrics
              </Checkbox>
            }
            description="DORA metrics are four widely-adopted metrics for measuring software delivery performance."
          />
          <Block
            title={
              <Checkbox checked={linker.enable} onChange={(e) => setLinker({ ...linker, enable: e.target.checked })}>
                Associate pull requests with issues
              </Checkbox>
            }
            description={
              <span>
                Parse the issue key with the regex from the title and description of the pull requests in this project.
                <HelpTooltip
                  overlayInnerStyle={{ width: 500 }}
                  content={
                    <>
                      <div>
                        Example 1 - If your PR title or description contains a Jira issue key in the format 'Closes
                        [DI-123](www.yourdomain.atlassian.net/browse/di-123)', please use the following regex template:{' '}
                        (?mi)Closes[\s]*.*(((and)?https://\S+.atlassian.net/browse/\S+[ ]*)+)
                      </div>
                      <div>
                        Example 2 - If your PR title or description contains a GitHub issue key in the format 'Resolves
                        www.github.com/namespace/repo_name/issues/123)', please use the following regex template:{' '}
                        (?mi)Resolves[\s]*.*(((and)?https://github.com/%s/issues/\d+[ ]*)+)
                      </div>
                    </>
                  }
                />
              </span>
            }
          >
            {linker.enable && (
              <Input
                style={{ width: 600 }}
                placeholder={RegexPrIssueDefaultValue}
                value={linker.prToIssueRegexp}
                onChange={(e) => setLinker({ ...linker, prToIssueRegexp: e.target.value })}
              />
            )}
          </Block>
          <Block
            title={
              <Checkbox checked={issueTrace.enable} onChange={(e) => setIssueTrace({ enable: e.target.checked })}>
                Enable issue trace
              </Checkbox>
            }
            description="Parse the issue status and assignee history from issue changelogs. Currently, only Jira issues are supported."
          />
          <Block
            title={
              <Checkbox checked={aidetector.enable} onChange={(e) => setAidetector({ enable: e.target.checked })}>
                Enable AI Detector
              </Checkbox>
            }
            description="Detect AI-assisted code by analyzing commit and PR patterns (Jellyfish-style approach)."
          />
          <Block
            title={
              <Checkbox checked={businessmetrics.enable} onChange={(e) => setBusinessmetrics({ enable: e.target.checked })}>
                Enable Business Metrics
              </Checkbox>
            }
            description="Extract business initiatives from Jira Epics and calculate work alignment."
          />
          <Block
            title={
              <Checkbox checked={capacityplanner.enable} onChange={(e) => setCapacityplanner({ enable: e.target.checked })}>
                Enable Capacity Planner
              </Checkbox>
            }
            description="Calculate team velocity and forecast initiative completion dates."
          />
          <Block
            title={
              <Checkbox checked={findevops.enable} onChange={(e) => setFindevops({ enable: e.target.checked })}>
                Enable FinDevOps
              </Checkbox>
            }
            description="Calculate development costs and categorize for US GAAP ASC 350-40 capitalization compliance."
          />
          <Block
            title={
              <Checkbox checked={aimeasure.enable} onChange={(e) => setAimeasure({ enable: e.target.checked })}>
                Enable AI Measure
              </Checkbox>
            }
            description="Classify PRs by AI cohort, track defect/composition signals, and surface verification-effort + dark-matter metrics (Phase B: Slack participation, after-hours, sentiment proxy)."
          />
          <Block>
            <Button type="primary" loading={operating} disabled={!name} onClick={handleUpdate}>
              Save
            </Button>
          </Block>
        </Card>
        <Flex justify="center">
          <Button type="primary" danger onClick={handleShowDeleteDialog}>
            Delete Project
          </Button>
        </Flex>
      </Space>
      <Modal
        open={open}
        width={820}
        centered
        title="Are you sure you want to delete this Project?"
        okText="Confirm"
        okButtonProps={{
          loading: operating,
        }}
        onCancel={handleHideDeleteDialog}
        onOk={handleDelete}
      >
        <S.DialogBody>
          <Message content="This operation cannot be undone. Deleting this project will remove all associated project settings and data. This action does not delete any data connections or the data collected through them." />
        </S.DialogBody>
      </Modal>
    </Flex>
  );
};
