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

// NodesListResult is the structured output of the proxmox_nodes_list tool.
type NodesListResult struct {
	Cluster string        `json:"cluster" jsonschema:"Name of the cluster"`
	Count   int           `json:"count" jsonschema:"Number of nodes"`
	Nodes   []NodeSummary `json:"nodes,omitempty" jsonschema:"List of nodes"`
}

// nodesListInput is the input of the proxmox_nodes_list tool.
type nodesListInput struct {
	Cluster string `json:"cluster" jsonschema:"Cluster name"`
}

// RegisterNodesList registers the nodes list tool.
func (t *ProxmoxTools) RegisterNodesList(srv *mcp.Server) {
	mcp.AddTool(srv,
		&mcp.Tool{
			Name:        "proxmox_nodes_list",
			Description: "List the nodes in a Proxmox cluster, including each node's status and resource usage.",
			Annotations: &mcp.ToolAnnotations{
				IdempotentHint: true,
				ReadOnlyHint:   true,
				OpenWorldHint:  new(true),
			},
		},
		t.handlerNodesList,
	)
}

func (t *ProxmoxTools) handlerNodesList(ctx context.Context, req *mcp.CallToolRequest, input nodesListInput) (*mcp.CallToolResult, any, error) {
	result, err := t.NodesList(ctx, input.Cluster, authorizationToken(req))
	if err != nil {
		return nil, NodesListResult{}, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: formatter.ToText(result)},
		},
	}, *result, nil
}

// NodesList returns the nodes in the Proxmox cluster in the given cluster.
func (t *ProxmoxTools) NodesList(ctx context.Context, cluster, authToken string) (*NodesListResult, error) {
	px, err := t.pool.GetProxmoxClusterWithToken(cluster, authToken)
	if err != nil {
		return nil, err
	}

	resources, err := px.Cluster().Resources().Get(ctx, proxmoxcluster.ResourceTypeNode)
	if err != nil {
		return nil, err
	}

	nodes := make([]NodeSummary, 0, len(resources))
	for _, resource := range resources {
		nodes = append(nodes, newNodeSummary(resource))
	}

	slices.SortFunc(nodes, func(a, b NodeSummary) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return &NodesListResult{
		Cluster: cluster,
		Nodes:   nodes,
		Count:   len(nodes),
	}, nil
}

func newNodeSummary(resource proxmoxcluster.Resource) NodeSummary {
	name := resource.Node
	if name == "" {
		name = resource.Name
	}

	return NodeSummary{
		Name:   name,
		Status: resource.Status,
		Resources: fmt.Sprintf(
			"cpu=%d (used=%.0f%%), memory=%dGiB (used=%dGiB), system storage=%dGiB (used=%dGiB)",
			resource.MaxCPU,
			resource.CPU*100,
			resource.MaxMem/(1024*1024*1024),
			resource.Mem/(1024*1024*1024),
			resource.MaxDisk/(1024*1024*1024),
			resource.Disk/(1024*1024*1024),
		),
	}
}
