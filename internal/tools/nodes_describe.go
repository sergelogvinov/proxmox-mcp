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

	"github.com/modelcontextprotocol/go-sdk/mcp"
	proxmoxcluster "github.com/sergelogvinov/go-proxmox-rest/cluster"
	"github.com/sergelogvinov/proxmox-mcp/pkg/formatter"
)

// NodesDescribeResult is the structured output of the proxmox_nodes_describe tool.
type NodesDescribeResult struct {
	Cluster       string            `json:"cluster" jsonschema:"Name of the cluster"`
	Name          string            `json:"name" jsonschema:"Node name"`
	Status        string            `json:"status" jsonschema:"Node status"`
	Resources     string            `json:"resources" jsonschema:"Node resources (CPU, Memory, Storage)"`
	Uptime        string            `json:"uptime" jsonschema:"Node uptime in days and hours"`
	Version       NodeVersion       `json:"version" jsonschema:"Node version"`
	CurrentKernel NodeCurrentKernel `json:"current_kernel" jsonschema:"Current kernel"`
	BootInfo      NodeBootInfo      `json:"boot_info" jsonschema:"Boot information"`
}

// nodesDescribeInput is the input of the proxmox_nodes_describe tool.
type nodesDescribeInput struct {
	Cluster string `json:"cluster" jsonschema:"Cluster name"`
	Node    string `json:"node" jsonschema:"Node name"`
}

// RegisterNodesDescribe registers the nodes describe tool.
func (t *ProxmoxTools) RegisterNodesDescribe(srv *mcp.Server) {
	mcp.AddTool(srv,
		&mcp.Tool{
			Name:        "proxmox_nodes_describe",
			Description: "Describe a Proxmox node: returns its status, resources, uptime, version, current kernel, and boot information.",
			Annotations: &mcp.ToolAnnotations{
				IdempotentHint: true,
				ReadOnlyHint:   true,
				OpenWorldHint:  new(true),
			},
		},
		t.handlerNodesDescribe,
	)
}

func (t *ProxmoxTools) handlerNodesDescribe(ctx context.Context, req *mcp.CallToolRequest, input nodesDescribeInput) (*mcp.CallToolResult, any, error) {
	result, err := t.NodesDescribe(ctx, input.Cluster, input.Node, authorizationToken(req))
	if err != nil {
		return nil, NodesDescribeResult{}, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: formatter.ToText(result)},
		},
	}, *result, nil
}

// NodesDescribe returns details of a node in the Proxmox cluster in the given cluster.
func (t *ProxmoxTools) NodesDescribe(ctx context.Context, cluster, node, authToken string) (*NodesDescribeResult, error) {
	px, err := t.pool.GetProxmoxClusterWithToken(cluster, authToken)
	if err != nil {
		return nil, err
	}

	resources, err := px.Cluster().Resources().Get(ctx, proxmoxcluster.ResourceTypeNode)
	if err != nil {
		return nil, err
	}

	var summary NodeSummary
	for _, resource := range resources {
		name := resource.Node
		if name == "" {
			name = resource.Name
		}
		if name == node {
			summary = newNodeSummary(resource)

			break
		}
	}

	status, err := px.Nodes().Status(ctx, node)
	if err != nil {
		return nil, err
	}

	version, err := px.Nodes().Version(ctx, node)
	if err != nil {
		return nil, err
	}

	result := &NodesDescribeResult{
		Cluster:   cluster,
		Name:      node,
		Status:    summary.Status,
		Resources: summary.Resources,
		Uptime:    formatUptime(status.Uptime),
		Version: NodeVersion{
			Release: version.Release,
			Version: version.Version,
		},
	}

	if status.CurrentKernel != nil {
		result.CurrentKernel = NodeCurrentKernel{
			Sysname: status.CurrentKernel.Sysname,
			Release: status.CurrentKernel.Release,
			Version: status.CurrentKernel.Version,
			Machine: status.CurrentKernel.Machine,
		}
	}
	if status.BootInfo != nil {
		result.BootInfo = NodeBootInfo{
			Mode:       status.BootInfo.Mode,
			SecureBoot: status.BootInfo.SecureBoot,
		}
	}

	return result, nil
}

func formatUptime(seconds int64) string {
	totalHours := seconds / (60 * 60)

	return fmt.Sprintf("%dd%dh", totalHours/24, totalHours%24)
}
