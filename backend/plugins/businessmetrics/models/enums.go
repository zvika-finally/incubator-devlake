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

package models

// BusinessCapability values define the type of business capability
// an initiative supports
const (
	CapabilityCoreProduct    = "core_product"
	CapabilityGrowth         = "growth"
	CapabilityMonetization   = "monetization"
	CapabilityPlatform       = "platform"
	CapabilityInfrastructure = "infrastructure"
	CapabilityInternalTools  = "internal_tools"
	CapabilityCompliance     = "compliance"
)

// RevenueImpact values classify how an initiative impacts revenue
const (
	RevenueImpactDirect     = "direct"      // Directly generates revenue
	RevenueImpactEnabling   = "enabling"    // Enables revenue-generating features
	RevenueImpactSupporting = "supporting"  // Supports revenue indirectly
	RevenueImpactCostCenter = "cost_center" // Pure cost (compliance, maintenance)
)

// HealthLevel values for team health scoring
const (
	HealthLevelElite  = "elite"  // Score >= 80
	HealthLevelHigh   = "high"   // Score >= 60
	HealthLevelMedium = "medium" // Score >= 40
	HealthLevelLow    = "low"    // Score < 40
)

// ValidBusinessCapabilities returns all valid capability values
func ValidBusinessCapabilities() []string {
	return []string{
		CapabilityCoreProduct,
		CapabilityGrowth,
		CapabilityMonetization,
		CapabilityPlatform,
		CapabilityInfrastructure,
		CapabilityInternalTools,
		CapabilityCompliance,
	}
}

// ValidRevenueImpacts returns all valid revenue impact values
func ValidRevenueImpacts() []string {
	return []string{
		RevenueImpactDirect,
		RevenueImpactEnabling,
		RevenueImpactSupporting,
		RevenueImpactCostCenter,
	}
}
