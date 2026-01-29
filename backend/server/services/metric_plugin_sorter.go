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
	"fmt"
	"sort"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
)

// sortMetricPluginsByDependencies performs topological sort on metric plugins
// Returns ordered list of plugin names or error if circular dependency detected
func sortMetricPluginsByDependencies(
	pluginNames []string,
) ([]string, errors.Error) {
	// Build dependency graph
	dependencyMap := make(map[string][]string)

	for _, name := range pluginNames {
		p, err := plugin.GetPlugin(name)
		if err != nil {
			return nil, errors.Default.Wrap(err,
				fmt.Sprintf("failed to get plugin %s", name))
		}

		// Get dependencies via RunAfter()
		if metric, ok := p.(plugin.PluginMetric); ok {
			deps, err := metric.RunAfter()
			if err != nil {
				return nil, errors.Default.Wrap(err,
					fmt.Sprintf("failed to get RunAfter for plugin %s", name))
			}
			dependencyMap[name] = deps
		} else {
			// Non-metric plugins have no dependencies
			dependencyMap[name] = []string{}
		}
	}

	// Validate all dependencies exist in the enabled plugin set
	for name, deps := range dependencyMap {
		for _, dep := range deps {
			if _, exists := dependencyMap[dep]; !exists {
				return nil, errors.Default.New(
					fmt.Sprintf("plugin %s depends on %s which is not enabled", name, dep))
			}
		}
	}

	// Perform topological sort using Kahn's algorithm
	sorted, err := topologicalSort(dependencyMap)
	if err != nil {
		return nil, errors.Default.Wrap(err, "circular dependency detected in metric plugins")
	}

	return sorted, nil
}

// topologicalSort performs Kahn's algorithm for stable topological ordering
// Returns sorted list or error if cycle detected
func topologicalSort(dependencyMap map[string][]string) ([]string, error) {
	var result []string
	inDegree := make(map[string]int)

	// Initialize all nodes with zero in-degree
	for node := range dependencyMap {
		if _, exists := inDegree[node]; !exists {
			inDegree[node] = 0
		}
	}

	// Calculate in-degrees (number of dependencies each node has)
	// If A depends on B, then A has an in-degree contribution from B
	for node, deps := range dependencyMap {
		inDegree[node] += len(deps)
	}

	// Process nodes with zero in-degree
	for len(dependencyMap) > 0 {
		// Find nodes with zero in-degree
		var zeroInDegree []string
		for node := range dependencyMap {
			if inDegree[node] == 0 {
				zeroInDegree = append(zeroInDegree, node)
			}
		}

		// Cycle detection
		if len(zeroInDegree) == 0 {
			return nil, fmt.Errorf("circular dependency detected")
		}

		// Stable sort for deterministic output
		sort.Strings(zeroInDegree)

		// Add to result and remove from graph
		for _, node := range zeroInDegree {
			result = append(result, node)

			// Decrease in-degree for nodes that depend on this node
			for remainingNode, deps := range dependencyMap {
				for i, dep := range deps {
					if dep == node {
						// Remove this dependency
						dependencyMap[remainingNode] = append(deps[:i], deps[i+1:]...)
						inDegree[remainingNode]--
						break
					}
				}
			}

			delete(dependencyMap, node)
			delete(inDegree, node)
		}
	}

	return result, nil
}
