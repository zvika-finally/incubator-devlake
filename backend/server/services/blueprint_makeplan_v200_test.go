/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package services

import (
	"encoding/json"
	"testing"

	coreModels "github.com/apache/incubator-devlake/core/models"
	"github.com/apache/incubator-devlake/core/models/domainlayer"
	"github.com/apache/incubator-devlake/core/models/domainlayer/code"
	"github.com/apache/incubator-devlake/core/models/domainlayer/ticket"
	"github.com/apache/incubator-devlake/core/plugin"
	mockplugin "github.com/apache/incubator-devlake/mocks/core/plugin"
	"github.com/apache/incubator-devlake/plugins/org/tasks"
	"github.com/stretchr/testify/assert"
)

func TestMakePlanV200(t *testing.T) {
	const projectName = "TestMakePlanV200-project"
	githubName := "TestMakePlanV200-github" // mimic github
	// mock github plugin as a data source plugin
	githubConnId := uint64(1)
	githubScopes := []*coreModels.BlueprintScope{
		{ScopeId: "github:GithubRepo:1:123"},
		{ScopeId: "github:GithubRepo:1:321"},
	}
	githubOutputPlan := coreModels.PipelinePlan{
		{
			{Plugin: githubName, Options: map[string]interface{}{"name": "apache/incubator-devlake"}},
			{Plugin: "gitextractor", Options: map[string]interface{}{"url": "http://gihub.com/apache/incubator-devlake.git"}},
		},
		{
			{Plugin: githubName, Options: map[string]interface{}{"name": "apache/incubator-devlake-website"}},
			{Plugin: "gitextractor", Options: map[string]interface{}{"url": "http://gihub.com/apache/incubator-devlake-website.git"}},
		},
	}
	githubOutputScopes := []plugin.Scope{
		&code.Repo{DomainEntity: domainlayer.DomainEntity{Id: "github:GithubRepo:1:123"}, Name: "apache/incubator-devlake"},
		&ticket.Board{DomainEntity: domainlayer.DomainEntity{Id: "github:GithubRepo:1:123"}, Name: "apache/incubator-devlake"},
	}
	github := new(mockplugin.CompositeDataSourcePluginBlueprintV200)
	github.On("MakeDataSourcePipelinePlanV200", githubConnId, githubScopes).Return(githubOutputPlan, githubOutputScopes, nil)

	// mock dora plugin as a metric plugin
	doraName := "TestMakePlanV200-dora"
	doraOutputPlan := coreModels.PipelinePlan{
		{
			{Plugin: "refdiff", Subtasks: []string{"calculateProjectDeploymentCommitsDiff"}, Options: map[string]interface{}{"projectName": projectName}},
			{Plugin: doraName},
		},
	}
	dora := new(mockplugin.CompositeMetricPluginBlueprintV200)
	dora.On("MakeMetricPluginPipelinePlanV200", projectName, json.RawMessage("{}")).Return(doraOutputPlan, nil)

	// mock org plugin
	org := new(mockplugin.CompositeProjectMapper)
	orgPlan := coreModels.PipelinePlan{
		{
			{Plugin: "org", Subtasks: []string{"setProjectMapping"}, Options: map[string]interface{}{"projectMappings": []interface{}{tasks.NewProjectMapping(projectName, githubOutputScopes)}}},
		},
	}
	org.On("MapProject", projectName, githubOutputScopes).Return(orgPlan, nil)

	// expectation, establish expectation before any code being launch to avoid unwanted modification
	expectedPlan := make(coreModels.PipelinePlan, 0)
	expectedPlan = append(expectedPlan, orgPlan...)
	expectedPlan = append(expectedPlan, githubOutputPlan...)
	expectedPlan = append(expectedPlan, doraOutputPlan...)

	// plugin registration
	plugin.RegisterPlugin(githubName, github)
	plugin.RegisterPlugin(doraName, dora)
	plugin.RegisterPlugin("org", org)

	// put them together and call GeneratePlanJsonV200
	connections := []*coreModels.BlueprintConnection{
		{PluginName: githubName, ConnectionId: githubConnId, Scopes: githubScopes},
	}
	metrics := map[string]json.RawMessage{
		doraName: nil,
	}

	plan, err := GeneratePlanJsonV200(projectName, connections, metrics, false)
	assert.Nil(t, err)

	assert.Equal(t, expectedPlan, plan)
}

func TestGeneratePlanJsonV200_MetricPluginDependencies(t *testing.T) {
	const projectName = "TestMetricDeps-project"

	// Setup mock data source plugin
	githubName := "TestMetricDeps-github"
	githubConnId := uint64(1)
	githubScopes := []*coreModels.BlueprintScope{
		{ScopeId: "github:GithubRepo:1:123"},
	}
	githubOutputPlan := coreModels.PipelinePlan{
		{{Plugin: githubName, Options: map[string]interface{}{"name": "test/repo"}}},
	}
	githubOutputScopes := []plugin.Scope{
		&code.Repo{DomainEntity: domainlayer.DomainEntity{Id: "github:GithubRepo:1:123"}, Name: "test/repo"},
	}
	github := new(mockplugin.CompositeDataSourcePluginBlueprintV200)
	github.On("MakeDataSourcePipelinePlanV200", githubConnId, githubScopes).Return(githubOutputPlan, githubOutputScopes, nil)

	// Setup mock metric plugins with dependencies
	// Create a type that implements both MetricPluginBlueprintV200 and PluginMetric
	type MockMetricPlugin struct {
		*mockplugin.CompositeMetricPluginBlueprintV200
		*mockplugin.PluginMetric
	}

	// dora has no dependencies
	doraName := "TestMetricDeps-dora"
	doraOutputPlan := coreModels.PipelinePlan{
		{{Plugin: doraName, Options: map[string]interface{}{"stage": "dora"}}},
	}
	doraBlueprintMock := new(mockplugin.CompositeMetricPluginBlueprintV200)
	doraBlueprintMock.On("MakeMetricPluginPipelinePlanV200", projectName, json.RawMessage("{}")).Return(doraOutputPlan, nil)
	doraMetricMock := new(mockplugin.PluginMetric)
	doraMetricMock.On("RunAfter").Return([]string{}, nil)
	dora := &MockMetricPlugin{doraBlueprintMock, doraMetricMock}

	// businessmetrics depends on dora
	bizName := "TestMetricDeps-businessmetrics"
	bizOutputPlan := coreModels.PipelinePlan{
		{{Plugin: bizName, Options: map[string]interface{}{"stage": "businessmetrics"}}},
	}
	bizBlueprintMock := new(mockplugin.CompositeMetricPluginBlueprintV200)
	bizBlueprintMock.On("MakeMetricPluginPipelinePlanV200", projectName, json.RawMessage("{}")).Return(bizOutputPlan, nil)
	bizMetricMock := new(mockplugin.PluginMetric)
	bizMetricMock.On("RunAfter").Return([]string{doraName}, nil)
	biz := &MockMetricPlugin{bizBlueprintMock, bizMetricMock}

	// capacityplanner has no dependencies (reads only from domain tables)
	capName := "TestMetricDeps-capacityplanner"
	capOutputPlan := coreModels.PipelinePlan{
		{{Plugin: capName, Options: map[string]interface{}{"stage": "capacityplanner"}}},
	}
	capBlueprintMock := new(mockplugin.CompositeMetricPluginBlueprintV200)
	capBlueprintMock.On("MakeMetricPluginPipelinePlanV200", projectName, json.RawMessage("{}")).Return(capOutputPlan, nil)
	capMetricMock := new(mockplugin.PluginMetric)
	capMetricMock.On("RunAfter").Return([]string{}, nil)
	cap := &MockMetricPlugin{capBlueprintMock, capMetricMock}

	// mock org plugin
	org := new(mockplugin.CompositeProjectMapper)
	orgPlan := coreModels.PipelinePlan{
		{{Plugin: "org", Options: map[string]interface{}{"projectMappings": []interface{}{tasks.NewProjectMapping(projectName, githubOutputScopes)}}}},
	}
	org.On("MapProject", projectName, githubOutputScopes).Return(orgPlan, nil)

	// Register plugins
	plugin.RegisterPlugin(githubName, github)
	plugin.RegisterPlugin(doraName, dora)
	plugin.RegisterPlugin(bizName, biz)
	plugin.RegisterPlugin(capName, cap)
	plugin.RegisterPlugin("org", org)

	// Test 1: Plugins should execute in correct dependency order
	t.Run("correct dependency order", func(t *testing.T) {
		connections := []*coreModels.BlueprintConnection{
			{PluginName: githubName, ConnectionId: githubConnId, Scopes: githubScopes},
		}
		// Add plugins in random order to verify sorting works
		metrics := map[string]json.RawMessage{
			capName:   json.RawMessage("{}"),
			doraName:  json.RawMessage("{}"),
			bizName:   json.RawMessage("{}"),
		}

		plan, err := GeneratePlanJsonV200(projectName, connections, metrics, false)
		assert.Nil(t, err)
		assert.NotNil(t, plan)

		// Expected plan: org -> github (parallel) -> capacityplanner -> dora -> businessmetrics (sequential)
		// Note: capacityplanner and dora both have no dependencies, so they're sorted alphabetically
		// Verify metric plugins are in correct sequential order
		expectedPlan := make(coreModels.PipelinePlan, 0)
		expectedPlan = append(expectedPlan, orgPlan...)
		expectedPlan = append(expectedPlan, githubOutputPlan...)
		expectedPlan = append(expectedPlan, capOutputPlan...)
		expectedPlan = append(expectedPlan, doraOutputPlan...)
		expectedPlan = append(expectedPlan, bizOutputPlan...)

		assert.Equal(t, expectedPlan, plan)
	})

	// Test 2: Missing dependency should be rejected
	t.Run("missing dependency error", func(t *testing.T) {
		connections := []*coreModels.BlueprintConnection{
			{PluginName: githubName, ConnectionId: githubConnId, Scopes: githubScopes},
		}
		// Enable capacityplanner without businessmetrics (its dependency)
		// Also need to register a plugin that depends on something not enabled
		missingDepName := "TestMetricDeps-missingdep"
		missingDepBlueprintMock := new(mockplugin.CompositeMetricPluginBlueprintV200)
		missingDepBlueprintMock.On("MakeMetricPluginPipelinePlanV200", projectName, json.RawMessage("{}")).Return(coreModels.PipelinePlan{}, nil)
		missingDepMetricMock := new(mockplugin.PluginMetric)
		missingDepMetricMock.On("RunAfter").Return([]string{bizName}, nil) // Depends on bizName which won't be enabled
		missingDep := &MockMetricPlugin{missingDepBlueprintMock, missingDepMetricMock}
		plugin.RegisterPlugin(missingDepName, missingDep)

		metrics := map[string]json.RawMessage{
			doraName:       json.RawMessage("{}"),
			missingDepName: json.RawMessage("{}"),
		}

		_, err := GeneratePlanJsonV200(projectName, connections, metrics, false)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "depends on")
		assert.Contains(t, err.Error(), "not enabled")
	})
}

func TestGeneratePlanJsonV200_CircularDependency(t *testing.T) {
	const projectName = "TestCircularDeps-project"

	// Setup minimal data source
	githubName := "TestCircularDeps-github"
	githubConnId := uint64(1)
	githubScopes := []*coreModels.BlueprintScope{
		{ScopeId: "github:GithubRepo:1:123"},
	}
	githubOutputPlan := coreModels.PipelinePlan{
		{{Plugin: githubName}},
	}
	githubOutputScopes := []plugin.Scope{
		&code.Repo{DomainEntity: domainlayer.DomainEntity{Id: "github:GithubRepo:1:123"}},
	}
	github := new(mockplugin.CompositeDataSourcePluginBlueprintV200)
	github.On("MakeDataSourcePipelinePlanV200", githubConnId, githubScopes).Return(githubOutputPlan, githubOutputScopes, nil)

	// Setup plugins with circular dependency: A -> B -> C -> A
	type MockMetricPlugin struct {
		*mockplugin.CompositeMetricPluginBlueprintV200
		*mockplugin.PluginMetric
	}

	pluginAName := "TestCircular-A"
	pluginABlueprintMock := new(mockplugin.CompositeMetricPluginBlueprintV200)
	pluginABlueprintMock.On("MakeMetricPluginPipelinePlanV200", projectName, json.RawMessage("{}")).Return(coreModels.PipelinePlan{{{Plugin: pluginAName}}}, nil)
	pluginAMetricMock := new(mockplugin.PluginMetric)
	pluginAMetricMock.On("RunAfter").Return([]string{"TestCircular-C"}, nil)
	pluginA := &MockMetricPlugin{pluginABlueprintMock, pluginAMetricMock}

	pluginBName := "TestCircular-B"
	pluginBBlueprintMock := new(mockplugin.CompositeMetricPluginBlueprintV200)
	pluginBBlueprintMock.On("MakeMetricPluginPipelinePlanV200", projectName, json.RawMessage("{}")).Return(coreModels.PipelinePlan{{{Plugin: pluginBName}}}, nil)
	pluginBMetricMock := new(mockplugin.PluginMetric)
	pluginBMetricMock.On("RunAfter").Return([]string{pluginAName}, nil)
	pluginB := &MockMetricPlugin{pluginBBlueprintMock, pluginBMetricMock}

	pluginCName := "TestCircular-C"
	pluginCBlueprintMock := new(mockplugin.CompositeMetricPluginBlueprintV200)
	pluginCBlueprintMock.On("MakeMetricPluginPipelinePlanV200", projectName, json.RawMessage("{}")).Return(coreModels.PipelinePlan{{{Plugin: pluginCName}}}, nil)
	pluginCMetricMock := new(mockplugin.PluginMetric)
	pluginCMetricMock.On("RunAfter").Return([]string{pluginBName}, nil)
	pluginC := &MockMetricPlugin{pluginCBlueprintMock, pluginCMetricMock}

	org := new(mockplugin.CompositeProjectMapper)
	org.On("MapProject", projectName, githubOutputScopes).Return(coreModels.PipelinePlan{}, nil)

	// Register plugins
	plugin.RegisterPlugin(githubName, github)
	plugin.RegisterPlugin(pluginAName, pluginA)
	plugin.RegisterPlugin(pluginBName, pluginB)
	plugin.RegisterPlugin(pluginCName, pluginC)
	plugin.RegisterPlugin("org", org)

	connections := []*coreModels.BlueprintConnection{
		{PluginName: githubName, ConnectionId: githubConnId, Scopes: githubScopes},
	}
	metrics := map[string]json.RawMessage{
		pluginAName: json.RawMessage("{}"),
		pluginBName: json.RawMessage("{}"),
		pluginCName: json.RawMessage("{}"),
	}

	_, err := GeneratePlanJsonV200(projectName, connections, metrics, false)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "circular dependency")
}
