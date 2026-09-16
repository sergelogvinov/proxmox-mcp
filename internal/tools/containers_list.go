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

// ContainersListResult is the structured output of the proxmox_containers_list tool.
type ContainersListResult struct {
	Cluster    string         `json:"cluster" jsonschema:"Name of the cluster"`
	Count      int            `json:"count" jsonschema:"Number of containers"`
	Containers []GuestSummary `json:"containers,omitempty" jsonschema:"List of containers"`
}

type containersListInput struct {
	Cluster string `json:"cluster" jsonschema:"Cluster name"`
}

// RegisterContainersList registers the containers list tool.
func (t *ProxmoxTools) RegisterContainersList(srv *mcp.Server) {
	mcp.AddTool(srv,
		&mcp.Tool{
			Name:        "proxmox_containers_list",
			Description: "List LXC containers in a Proxmox cluster, including status and resource usage.",
			Annotations: &mcp.ToolAnnotations{
				IdempotentHint: true,
				ReadOnlyHint:   true,
				OpenWorldHint:  new(true),
			},
		},
		t.handlerContainersList,
	)
}

func (t *ProxmoxTools) handlerContainersList(ctx context.Context, req *mcp.CallToolRequest, input containersListInput) (*mcp.CallToolResult, any, error) {
	result, err := t.ContainersList(ctx, input.Cluster, authorizationToken(req))
	if err != nil {
		return nil, ContainersListResult{}, err
	}

	return &mcp.CallToolResult{Content: []mcp.Content{
		&mcp.TextContent{Text: formatter.ToText(result)},
	}}, *result, nil
}

// ContainersList returns LXC containers from the Proxmox cluster.
func (t *ProxmoxTools) ContainersList(ctx context.Context, cluster, authToken string) (*ContainersListResult, error) {
	guests, err := t.listGuests(ctx, cluster, authToken, "lxc")
	if err != nil {
		return nil, err
	}

	return &ContainersListResult{
		Cluster:    cluster,
		Count:      len(guests),
		Containers: guests,
	}, nil
}
