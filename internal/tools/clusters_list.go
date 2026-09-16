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
	"slices"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sergelogvinov/proxmox-mcp/pkg/formatter"
)

// ClustersListResult is the structured output of the proxmox_clusters_list tool.
type ClustersListResult struct {
	Clusters []string `json:"clusters,omitempty" jsonschema:"Clusters names"`
	Count    int      `json:"count" jsonschema:"Number of configured clusters"`
}

// RegisterClustersList registers the clusters list tool.
func (t *ProxmoxTools) RegisterClustersList(srv *mcp.Server) {
	mcp.AddTool(srv,
		&mcp.Tool{
			Name:        "proxmox_clusters_list",
			Description: "List the Proxmox clusters configured in the MCP server.",
			Annotations: &mcp.ToolAnnotations{
				IdempotentHint: true,
				ReadOnlyHint:   true,
				OpenWorldHint:  new(false),
			},
		},
		t.handlerClustersList,
	)
}

func (t *ProxmoxTools) handlerClustersList(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
	clusters := t.pool.GetClusters()
	slices.Sort(clusters)

	result := &ClustersListResult{
		Clusters: clusters,
		Count:    len(clusters),
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: formatter.ToText(result)},
		},
	}, *result, nil
}
