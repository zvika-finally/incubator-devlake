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
	"strconv"
	"strings"
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/aimeasure/models"
)

var ComputeSlackSignalsMeta = plugin.SubTaskMeta{
	Name:             "computeSlackSignals",
	EntryPoint:       ComputeSlackSignals,
	EnabledByDefault: true,
	Description:      "Aggregate per-engineer per-week Slack participation by channel category",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET}, // Slack is communication-layer; ticket is the closest existing domain
}

// ResolveSlackEngineer returns the DevLake account_id for a Slack user, falling
// back to a synthetic "slack:<user>" id when unmapped. Empty user → empty id.
func ResolveSlackEngineer(overrides map[string]string, slackUserId string) string {
	if slackUserId == "" {
		return ""
	}
	if a, ok := overrides["slack:"+slackUserId]; ok {
		return a
	}
	return "slack:" + slackUserId
}

// IsThreadParticipation returns true when a message is a reply within a thread
// (not the top-level message and not the thread root).
func IsThreadParticipation(ts, threadTs string) bool {
	return threadTs != "" && threadTs != ts
}

// SlackTsToTime parses Slack's "<unix>.<usec>" timestamp format to a UTC time.
func SlackTsToTime(ts string) (time.Time, error) {
	dot := strings.Index(ts, ".")
	secs := ts
	if dot > 0 {
		secs = ts[:dot]
	}
	n, err := strconv.ParseInt(secs, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(n, 0).UTC(), nil
}

func ComputeSlackSignals(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	logger := taskCtx.GetLogger()
	now := time.Now().UTC()

	// Load channel→category mapping (one query).
	type chanCat struct {
		ChannelKey string `gorm:"column:channel_key"`
		Category   string `gorm:"column:category"`
	}
	var rawMapping []chanCat
	if err := db.All(&rawMapping, dal.From("aimeasure_slack_channel_categories")); err != nil {
		return errors.Default.Wrap(err, "failed to load slack channel categories")
	}
	mapping := map[string]models.ChannelCategory{}
	for _, m := range rawMapping {
		mapping[m.ChannelKey] = models.ChannelCategory(m.Category)
	}

	// Load identity overrides for Slack (one query).
	type override struct {
		SourceId  string `gorm:"column:source_id"`
		AccountId string `gorm:"column:account_id"`
	}
	var rawOverrides []override
	if err := db.All(&rawOverrides,
		dal.From("aimeasure_account_overrides"),
		dal.Where("source_system = ?", "slack"),
	); err != nil {
		return errors.Default.Wrap(err, "failed to load identity overrides")
	}
	overrides := map[string]string{}
	for _, o := range rawOverrides {
		overrides["slack:"+o.SourceId] = o.AccountId
	}

	// Stream the messages joined to channel names. Use a cursor to bound memory
	// on large workspaces.
	type slackMsg struct {
		User        string `gorm:"column:user"`
		Ts          string `gorm:"column:ts"`
		ThreadTs    string `gorm:"column:thread_ts"`
		ChannelId   string `gorm:"column:channel_id"`
		ChannelName string `gorm:"column:channel_name"`
	}
	cursor, err := db.Cursor(
		dal.Select("m.user AS user, m.ts AS ts, m.thread_ts AS thread_ts, m.channel_id AS channel_id, c.name AS channel_name"),
		dal.From("_tool_slack_channel_messages m"),
		dal.Join("LEFT JOIN _tool_slack_channels c ON c.channel_id = m.channel_id AND c.connection_id = m.connection_id"),
		dal.Where("m.user IS NOT NULL AND m.user != '' AND (m.subtype IS NULL OR m.subtype = '' OR m.subtype = 'thread_broadcast')"),
	)
	if err != nil {
		return errors.Default.Wrap(err, "failed to query slack messages")
	}
	defer cursor.Close()

	type slackKey struct {
		Engineer string
		Week     time.Time
		Category models.ChannelCategory
	}
	type slackAgg struct {
		Messages   int
		Threaded   int
		AfterHours int
	}
	buckets := map[slackKey]*slackAgg{}

	for cursor.Next() {
		var m slackMsg
		if err := db.Fetch(cursor, &m); err != nil {
			return errors.Default.Wrap(err, "row scan failed on slack message")
		}
		engineer := ResolveSlackEngineer(overrides, m.User)
		if engineer == "" {
			continue
		}
		when, terr := SlackTsToTime(m.Ts)
		if terr != nil {
			logger.Warn(terr, "failed to parse slack ts %q; skipping", m.Ts)
			continue
		}
		cat := CategorizeChannel(mapping, m.ChannelName, m.ChannelId)
		key := slackKey{Engineer: engineer, Week: PeriodWeekStart(when), Category: cat}
		a, ok := buckets[key]
		if !ok {
			a = &slackAgg{}
			buckets[key] = a
		}
		a.Messages++
		if IsThreadParticipation(m.Ts, m.ThreadTs) {
			a.Threaded++
		}
		if IsAfterHours(when) {
			a.AfterHours++
		}
	}

	count := 0
	for k, agg := range buckets {
		row := &models.EngineerSlackSignals{
			EngineerId:               k.Engineer,
			PeriodWeek:               k.Week,
			ChannelCategory:          k.Category,
			MessageCount:             agg.Messages,
			ThreadParticipationCount: agg.Threaded,
			AfterHoursMessageCount:   agg.AfterHours,
			AfterHoursRatio:          SafeRatio(agg.AfterHours, agg.Messages),
			ComputedAt:               now,
		}
		if err := db.CreateOrUpdate(row); err != nil {
			return errors.Default.Wrap(err, "failed to upsert engineer_slack_signals row")
		}
		count++
	}
	logger.Info("computeSlackSignals wrote %d (engineer, week, category) buckets", count)
	return nil
}
