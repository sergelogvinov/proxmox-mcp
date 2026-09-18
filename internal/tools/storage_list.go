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
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	proxmoxcluster "github.com/sergelogvinov/go-proxmox-rest/cluster"
	"github.com/sergelogvinov/proxmox-mcp/pkg/formatter"
)

// StorageListResult is the structured output of the proxmox_storage_list tool.
type StorageListResult struct {
	Cluster  string           `json:"cluster" jsonschema:"Name of the cluster"`
	Count    int              `json:"count" jsonschema:"Number of storage resources"`
	Storages []StorageSummary `json:"storages,omitempty" jsonschema:"List of storage resources"`
}

// storageListInput is the input of the proxmox_storage_list tool.
type storageListInput struct {
	Cluster string `json:"cluster" jsonschema:"Cluster name"`
}

// RegisterStorageList registers the storage list tool.
func (t *ProxmoxTools) RegisterStorageList(srv *mcp.Server) {
	mcp.AddTool(srv,
		&mcp.Tool{
			Name:        "proxmox_storage_list",
			Description: "List storage resources in a Proxmox cluster, including status, type, content, and capacity usage.",
			Annotations: &mcp.ToolAnnotations{
				IdempotentHint: true,
				ReadOnlyHint:   true,
				OpenWorldHint:  new(true),
			},
		},
		t.handlerStorageList,
	)
}

func (t *ProxmoxTools) handlerStorageList(ctx context.Context, req *mcp.CallToolRequest, input storageListInput) (*mcp.CallToolResult, any, error) {
	result, err := t.StorageList(ctx, input.Cluster, authorizationToken(req))
	if err != nil {
		return nil, StorageListResult{}, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: formatter.ToText(result)},
		},
	}, *result, nil
}

// StorageList returns storage resources from the Proxmox cluster in the given cluster.
func (t *ProxmoxTools) StorageList(ctx context.Context, cluster, authToken string) (*StorageListResult, error) {
	px, err := t.pool.GetProxmoxClusterWithToken(cluster, authToken)
	if err != nil {
		return nil, err
	}

	resources, err := px.Cluster().Resources().List(ctx, proxmoxcluster.ListFilter{Type: proxmoxcluster.ResourceTypeStorage})
	if err != nil {
		return nil, err
	}

	type storageKey struct {
		name     string
		typeName string
		content  string
		shared   bool
	}

	type storageGroup struct {
		summary  StorageSummary
		statuses map[string][]string
		capacity int64
		used     int64
		hasStats bool
	}

	grouped := make(map[storageKey]*storageGroup, len(resources))
	for _, resource := range resources {
		name := resource.Storage
		if name == "" {
			name = resource.Name
		}

		key := storageKey{
			name:     name,
			typeName: resource.PluginType,
			content:  resource.StorageContent,
			shared:   resource.Shared == 1,
		}
		status := resource.Status
		if status == "" {
			status = "unknown"
		}

		if group, ok := grouped[key]; ok {
			if resource.Node != "" && !slices.Contains(group.statuses[status], resource.Node) {
				group.statuses[status] = append(group.statuses[status], resource.Node)
			}
			if resource.MaxDisk > 0 && (!key.shared || !group.hasStats) {
				if key.shared {
					group.capacity = resource.MaxDisk
					group.used = resource.Disk
				} else {
					group.capacity += resource.MaxDisk
					group.used += resource.Disk
				}
				group.hasStats = true
			}

			continue
		}

		group := &storageGroup{
			summary: StorageSummary{
				Name:    name,
				Type:    resource.PluginType,
				Content: resource.StorageContent,
				Shared:  resource.Shared == 1,
			},
			statuses: make(map[string][]string),
		}
		if resource.Node != "" {
			group.statuses[status] = []string{resource.Node}
		}
		if resource.MaxDisk > 0 {
			group.capacity = resource.MaxDisk
			group.used = resource.Disk
			group.hasStats = true
		}
		grouped[key] = group
	}

	storages := make([]StorageSummary, 0, len(grouped))
	for _, group := range grouped {
		for status, nodes := range group.statuses {
			slices.Sort(nodes)
			if status == "available" {
				group.summary.Available = nodes
				continue
			}
			if group.summary.Other == nil {
				group.summary.Other = make(map[string][]string)
			}
			group.summary.Other[status] = nodes
		}
		group.summary.Resources = fmt.Sprintf(
			"capacity=%dGiB (used=%dGiB)",
			group.capacity/(1024*1024*1024),
			group.used/(1024*1024*1024),
		)
		storages = append(storages, group.summary)
	}

	slices.SortFunc(storages, func(a, b StorageSummary) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return &StorageListResult{
		Cluster:  cluster,
		Count:    len(storages),
		Storages: storages,
	}, nil
}
