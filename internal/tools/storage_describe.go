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
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sergelogvinov/proxmox-mcp/pkg/formatter"
)

// StorageDescribeResult is the structured output of the proxmox_storage_describe tool.
type StorageDescribeResult struct {
	Cluster string                  `json:"cluster" jsonschema:"Name of the cluster"`
	Storage StorageSummary          `json:"storage" jsonschema:"Storage details"`
	Node    string                  `json:"node,omitempty" jsonschema:"Node used to list storage content"`
	Count   int                     `json:"count,omitempty" jsonschema:"Number of storage content entries"`
	Content []StorageContentSummary `json:"content,omitempty" jsonschema:"Storage content on the selected node"`
}

// storageDescribeInput is the input of the proxmox_storage_describe tool.
type storageDescribeInput struct {
	Cluster string `json:"cluster" jsonschema:"Cluster name"`
	Name    string `json:"name" jsonschema:"Storage name"`
	Node    string `json:"node,omitempty" jsonschema:"Optional node name used to list storage content"`
}

// RegisterStorageDescribe registers the storage describe tool.
func (t *ProxmoxTools) RegisterStorageDescribe(srv *mcp.Server) {
	mcp.AddTool(srv,
		&mcp.Tool{
			Name:        "proxmox_storage_describe",
			Description: "Describe a Proxmox storage. When a node is specified, also list the storage content visible on that node.",
			Annotations: &mcp.ToolAnnotations{
				IdempotentHint: true,
				ReadOnlyHint:   true,
				OpenWorldHint:  new(true),
			},
		},
		t.handlerStorageDescribe,
	)
}

func (t *ProxmoxTools) handlerStorageDescribe(ctx context.Context, req *mcp.CallToolRequest, input storageDescribeInput) (*mcp.CallToolResult, any, error) {
	result, err := t.StorageDescribe(ctx, input.Cluster, input.Name, input.Node, authorizationToken(req))
	if err != nil {
		return nil, StorageDescribeResult{}, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: formatter.ToText(result)},
		},
	}, *result, nil
}

// StorageDescribe returns a storage summary and, optionally, its content on a node.
func (t *ProxmoxTools) StorageDescribe(ctx context.Context, cluster, storageName, node, authToken string) (*StorageDescribeResult, error) {
	list, err := t.StorageList(ctx, cluster, authToken)
	if err != nil {
		return nil, err
	}

	var summary *StorageSummary
	for i := range list.Storages {
		if list.Storages[i].Name == storageName {
			summary = &list.Storages[i]
			break
		}
	}
	if summary == nil {
		return nil, fmt.Errorf("storage %q not found in cluster %q", storageName, cluster)
	}

	result := &StorageDescribeResult{
		Cluster: cluster,
		Storage: *summary,
		Node:    node,
	}
	if node == "" {
		return result, nil
	}

	px, err := t.pool.GetProxmoxClusterWithToken(cluster, authToken)
	if err != nil {
		return nil, err
	}

	volumes, err := px.Nodes(node).Storage().Content(storageName).List(ctx, nil)
	if err != nil {
		return nil, err
	}

	result.Content = make([]StorageContentSummary, 0, len(volumes))
	for _, volume := range volumes {
		size := volume.Size
		if size == 0 {
			size = volume.ApproximateSize
		}

		result.Content = append(result.Content, StorageContentSummary{
			VolumeID:  strings.TrimPrefix(volume.VolID, storageName+":"),
			VMID:      volume.VMID,
			Format:    volume.Format,
			Size:      size,
			Used:      volume.Used,
			CreatedAt: formatStorageContentTime(volume.CTime),
			Notes:     volume.Notes,
			Protected: volume.Protected,
		})
	}
	slices.SortFunc(result.Content, func(a, b StorageContentSummary) int {
		return compareStrings(a.VolumeID, b.VolumeID)
	})
	result.Count = len(result.Content)

	return result, nil
}

func compareStrings(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func formatStorageContentTime(timestamp int64) string {
	if timestamp == 0 {
		return ""
	}

	return time.Unix(timestamp, 0).UTC().Format(time.RFC3339)
}
