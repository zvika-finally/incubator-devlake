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

package tasks

import (
	"encoding/json"
	"strings"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/aidetector/models"
)

var ScoreAIConfidenceMeta = plugin.SubTaskMeta{
	Name:             "scoreAIConfidence",
	EntryPoint:       ScoreAIConfidence,
	EnabledByDefault: true,
	Description:      "Calculate final AI confidence score by combining all pattern signals",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CODE},
}

// PatternSignature stores detailed breakdown of AI detection patterns
type PatternSignature struct {
	// Explicit signals (HIGH confidence)
	ExplicitDetection string `json:"explicit_detection"`
	ExplicitTools     string `json:"explicit_tools,omitempty"`
	ExplicitPatterns  string `json:"explicit_patterns,omitempty"`
	// Behavioral signals
	RapidCommits       string `json:"rapid_commits"`
	PRSizeAnomaly      string `json:"pr_size_anomaly"`
	HighLinesPerMinute string `json:"high_lines_per_minute"`
	GenericMessages    string `json:"generic_messages"`
	DetectionReason    string `json:"detection_reason"`
}

func ScoreAIConfidence(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*AIDetectorTaskData)
	logger := taskCtx.GetLogger()

	logger.Info("Starting scoreAIConfidence for project: %s", data.Options.ProjectName)

	confidenceThreshold := GetEffectiveConfidenceThreshold(data)

	// Get signals scoped to the requested project
	var signals []models.AIUsageSignal
	err := db.All(&signals,
		dal.Select("ai_usage_signals.*"),
		dal.From(&models.AIUsageSignal{}),
		dal.Join("LEFT JOIN pull_requests pr ON pr.id = ai_usage_signals.pull_request_id"),
		dal.Join("LEFT JOIN project_mapping pm ON pm.table = 'repos' AND pm.row_id = pr.base_repo_id"),
		dal.Where("pm.project_name = ?", data.Options.ProjectName),
	)
	if err != nil {
		return errors.Default.Wrap(err, "failed to query signals")
	}

	aiAssistedCount := 0
	for i := range signals {
		signal := &signals[i]

		// Calculate total confidence score (max 100)
		// Scoring breakdown:
		//
		// EXPLICIT SIGNALS (HIGH confidence):
		// - Git trailers / PR body markers: max 70 points
		//
		// BEHAVIORAL SIGNALS (lower confidence):
		// - Rapid commits: max 30 points
		// - Lines per minute: max 25 points
		// - PR size anomaly: max 20 points
		// - Code duplication: max 15 points (not implemented yet)
		// - Generic messages: max 10 points
		//
		// Note: Explicit signals alone can trigger detection (70+ = highly confident)
		//       Behavioral signals require multiple indicators
		totalScore := signal.ExplicitSignalScore +
			signal.RapidCommitScore +
			signal.LinesPerMinuteScore +
			signal.PRSizeScore +
			signal.DuplicationScore +
			signal.GenericMessageScore

		// Cap at 100
		if totalScore > 100 {
			totalScore = 100
		}
		signal.AIConfidenceScore = totalScore

		// Determine detected tool (heuristic - could be enhanced)
		signal.DetectedTool = detectTool(signal)

		// Build pattern signatures JSON
		sig := buildPatternSignature(signal)
		sigJSON, _ := json.Marshal(sig)
		signal.PatternSignatures = string(sigJSON)

		// Update signal
		if err := db.Update(signal); err != nil {
			logger.Error(err, "failed to update signal %s", signal.Id)
			continue
		}

		if totalScore >= confidenceThreshold {
			aiAssistedCount++
		}
	}

	// Log summary statistics
	if len(signals) > 0 {
		aiPercent := float64(aiAssistedCount) * 100.0 / float64(len(signals))
		logger.Info("AI Detection Summary: %d/%d PRs (%.1f%%) flagged as AI-assisted (threshold: %d%%)",
			aiAssistedCount, len(signals), aiPercent, confidenceThreshold)
	}

	logger.Info("Completed scoreAIConfidence")
	return nil
}

func detectTool(signal *models.AIUsageSignal) string {
	// Priority 1: Explicit tool detection (highest confidence)
	// If we found explicit markers like git trailers or PR body markers,
	// use that tool directly
	if signal.ExplicitToolDetected && signal.ExplicitTools != "" {
		// Return first explicitly detected tool
		tools := strings.Split(signal.ExplicitTools, ",")
		if len(tools) > 0 {
			return tools[0]
		}
	}

	// Priority 2: Heuristic tool detection based on behavioral patterns
	// This is approximate - actual tool detection requires explicit signals

	// Very rapid commits + high line output suggests Cursor or Claude Code
	if signal.RapidCommitScore >= 30 && signal.LinesPerMinuteScore >= 20 {
		return "cursor_or_claude_likely"
	}

	// Moderate rapid commits might indicate Copilot (inline suggestions)
	if signal.RapidCommitScore >= 20 {
		return "copilot_likely"
	}

	// High confidence but not rapid = might be AI-assisted but not autocomplete
	if signal.AIConfidenceScore >= 50 {
		return "ai_assisted_unknown"
	}

	return "unknown"
}

func buildPatternSignature(signal *models.AIUsageSignal) PatternSignature {
	sig := PatternSignature{}

	// Explicit detection (highest confidence)
	if signal.ExplicitToolDetected {
		sig.ExplicitDetection = "confirmed"
		sig.ExplicitTools = signal.ExplicitTools
		sig.ExplicitPatterns = signal.ExplicitPatterns
	} else {
		sig.ExplicitDetection = "none"
	}

	// Rapid commits analysis
	if signal.RapidCommitScore >= 30 {
		sig.RapidCommits = "very_rapid"
	} else if signal.RapidCommitScore >= 20 {
		sig.RapidCommits = "rapid"
	} else if signal.RapidCommitScore >= 10 {
		sig.RapidCommits = "moderate"
	} else {
		sig.RapidCommits = "normal"
	}

	// PR size analysis
	if signal.PRSizeScore >= 20 {
		sig.PRSizeAnomaly = "significantly_larger"
	} else if signal.PRSizeScore >= 10 {
		sig.PRSizeAnomaly = "larger_than_baseline"
	} else {
		sig.PRSizeAnomaly = "normal"
	}

	// Lines per minute
	if signal.LinesPerMinuteScore >= 20 {
		sig.HighLinesPerMinute = "very_high"
	} else if signal.LinesPerMinuteScore >= 10 {
		sig.HighLinesPerMinute = "elevated"
	} else {
		sig.HighLinesPerMinute = "normal"
	}

	// Generic messages
	if signal.GenericMessageScore >= 7 {
		sig.GenericMessages = "mostly_generic"
	} else if signal.GenericMessageScore >= 3 {
		sig.GenericMessages = "some_generic"
	} else {
		sig.GenericMessages = "descriptive"
	}

	// Build detection reason (explicit signals take priority)
	reasons := []string{}
	if signal.ExplicitToolDetected {
		reasons = append(reasons, "explicit_marker")
	}
	if signal.RapidCommitScore >= 20 {
		reasons = append(reasons, "rapid_commits")
	}
	if signal.PRSizeScore >= 10 {
		reasons = append(reasons, "large_pr")
	}
	if signal.LinesPerMinuteScore >= 10 {
		reasons = append(reasons, "high_velocity")
	}
	if len(reasons) > 0 {
		sig.DetectionReason = strings.Join(reasons, "+")
	} else {
		sig.DetectionReason = "low_confidence"
	}

	return sig
}
