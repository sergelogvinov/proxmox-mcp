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

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sergelogvinov/proxmox-mcp/pkg/formatter"
)

// ClustersDescribeResult is the structured output of the proxmox_clusters_describe tool.
type ClustersDescribeResult struct {
	Cluster string `json:"cluster" jsonschema:"Cluster name (see proxmox_clusters_list)"`
	Version string `json:"version" jsonschema:"Proxmox version of the cluster"`
}

// clustersDescribeInput is the input of the proxmox_clusters_describe tool.
type clustersDescribeInput struct {
	Cluster string `json:"cluster" jsonschema:"Cluster name (region)"`
}

// RegisterClustersDescribe registers the clusters describe tool.
func (t *ProxmoxTools) RegisterClustersDescribe(srv *mcp.Server) {
	mcp.AddTool(srv,
		&mcp.Tool{
			Name:        "proxmox_clusters_describe",
			Description: "Describe a Proxmox cluster: returns the Proxmox version of the cluster in the given region.",
			Annotations: &mcp.ToolAnnotations{
				IdempotentHint: true,
				ReadOnlyHint:   true,
				OpenWorldHint:  new(true),
			},
		},
		t.handlerClustersDescribe,
	)
}

func (t *ProxmoxTools) handlerClustersDescribe(ctx context.Context, req *mcp.CallToolRequest, input clustersDescribeInput) (*mcp.CallToolResult, any, error) {
	result, err := t.ClustersDescribe(ctx, input.Cluster, authorizationHeader(req))
	if err != nil {
		return nil, ClustersDescribeResult{}, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: formatter.ToText(result)},
		},
	}, *result, nil
}

// ClustersDescribe returns details of the Proxmox cluster in the given region.
func (t *ProxmoxTools) ClustersDescribe(ctx context.Context, region, authHeader string) (*ClustersDescribeResult, error) {
	version, err := t.pool.GetClusterVersion(ctx, region, authHeader)
	if err != nil {
		return nil, err
	}

	return &ClustersDescribeResult{
		Cluster: region,
		Version: version,
	}, nil
}
