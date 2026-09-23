/*
Copyright 2026 Serge Logvinov.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tools

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	proxmoxcluster "github.com/sergelogvinov/go-proxmox-rest/cluster"
	"github.com/sergelogvinov/proxmox-mcp/pkg/formatter"
)

const defaultEventsLimit = 10

// EventsListResult is the structured output of the proxmox_events_list tool.
type EventsListResult struct {
	Cluster string         `json:"cluster" jsonschema:"Name of the cluster"`
	Count   int            `json:"count" jsonschema:"Number of returned task events"`
	Events  []EventSummary `json:"events,omitempty" jsonschema:"Recent task events"`
}

// eventsListInput is the input of the proxmox_events_list tool.
type eventsListInput struct {
	Cluster string `json:"cluster" jsonschema:"Cluster name"`
	Limit   int    `json:"limit,omitempty" jsonschema:"Maximum number of task events to return (default 10)"`
}

// RegisterEventsList registers the events list tool.
func (t *ProxmoxTools) RegisterEventsList(srv *mcp.Server) {
	mcp.AddTool(srv,
		&mcp.Tool{
			Name:        "proxmox_events_list",
			Description: "List recent Proxmox task events across all nodes in a cluster. Returns 10 events by default.",
			Annotations: &mcp.ToolAnnotations{
				IdempotentHint: true,
				ReadOnlyHint:   true,
				OpenWorldHint:  new(true),
			},
		},
		t.handlerEventsList,
	)
}

func (t *ProxmoxTools) handlerEventsList(ctx context.Context, req *mcp.CallToolRequest, input eventsListInput) (*mcp.CallToolResult, any, error) {
	result, err := t.EventsList(ctx, input.Cluster, input.Limit, authorizationToken(req))
	if err != nil {
		return nil, EventsListResult{}, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: formatter.ToText(result)},
		},
	}, *result, nil
}

// EventsList returns recent task events from the Proxmox cluster in the given cluster.
func (t *ProxmoxTools) EventsList(ctx context.Context, cluster string, limit int, authToken string) (*EventsListResult, error) {
	if limit < 0 {
		return nil, fmt.Errorf("limit must not be negative")
	}
	if limit == 0 {
		limit = defaultEventsLimit
	}

	px, err := getProxmoxClusterWithToken(t.pool, cluster, authToken)
	if err != nil {
		return nil, err
	}

	tasks, err := px.Cluster().Tasks(ctx)
	if err != nil {
		return nil, err
	}

	slices.SortStableFunc(tasks, func(a, b proxmoxcluster.Task) int {
		return compareDescending(a.StartTime, b.StartTime)
	})
	if len(tasks) > limit {
		tasks = tasks[:limit]
	}

	events := make([]EventSummary, 0, len(tasks))
	for _, task := range tasks {
		status := task.Status
		if task.EndTime == 0 && status == "" {
			status = "running"
		}

		events = append(events, EventSummary{
			Node:      task.Node,
			Type:      task.Type,
			User:      task.User,
			StartTime: formatEventTime(task.StartTime),
			Duration:  formatEventDuration(task.StartTime, task.EndTime),
			Status:    status,
		})
	}

	return &EventsListResult{
		Cluster: cluster,
		Count:   len(events),
		Events:  events,
	}, nil
}

func compareDescending(a, b int64) int {
	switch {
	case a > b:
		return -1
	case a < b:
		return 1
	default:
		return 0
	}
}

func formatEventTime(timestamp int64) string {
	if timestamp == 0 {
		return ""
	}

	return time.Unix(timestamp, 0).UTC().Format(time.RFC3339)
}

func formatEventDuration(startTime, endTime int64) string {
	if startTime == 0 || endTime == 0 || endTime < startTime {
		return ""
	}

	return (time.Duration(endTime-startTime) * time.Second).String()
}
