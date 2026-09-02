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

// Package tools implements the MCP tool handlers, grouped by domain:
// cluster, vm, container, storage, backup.
package tools

import (
	"context"
	"slices"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ClustersListResult is the structured output of the proxmox_clusters_list tool.
type ClustersListResult struct {
	// Clusters is the sorted list of configured Proxmox cluster names (regions).
	Clusters []string `json:"clusters,omitempty" jsonschema:"Clusters names (regions)"`
	// Count is the number of configured clusters.
	Count int `json:"count" jsonschema:"Number of configured clusters"`
}

// registerClusterTools registers the cluster management tools.
func (t *ProxmoxTools) registerClusterTools(srv *mcp.Server) {
	mcp.AddTool(srv,
		&mcp.Tool{
			Name:        "proxmox_clusters_list",
			Description: "List the Proxmox clusters (regions) configured in the MCP server.",
		},
		func(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
			result, err := t.ClustersList()
			if err != nil {
				return nil, ClustersListResult{}, err
			}

			return &mcp.CallToolResult{}, *result, nil
		},
	)
}

// ClustersList returns the list of configured Proxmox clusters (regions).
func (t *ProxmoxTools) ClustersList() (*ClustersListResult, error) {
	clusters := t.pool.GetRegions()
	slices.Sort(clusters)

	return &ClustersListResult{
		Clusters: clusters,
		Count:    len(clusters),
	}, nil
}
