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

// VMsListResult is the structured output of the proxmox_vms_list tool.
type VMsListResult struct {
	Cluster string         `json:"cluster" jsonschema:"Name of the cluster"`
	Count   int            `json:"count" jsonschema:"Number of virtual machines"`
	VMs     []GuestSummary `json:"vms,omitempty" jsonschema:"List of virtual machines"`
}

type vmsListInput struct {
	Cluster string `json:"cluster" jsonschema:"Cluster name"`
}

// RegisterVMsList registers the virtual machines list tool.
func (t *ProxmoxTools) RegisterVMsList(srv *mcp.Server) {
	mcp.AddTool(srv,
		&mcp.Tool{
			Name:        "proxmox_vms_list",
			Description: "List QEMU virtual machines in a Proxmox cluster, including status and resource usage.",
			Annotations: &mcp.ToolAnnotations{
				IdempotentHint: true,
				ReadOnlyHint:   true,
				OpenWorldHint:  new(true),
			},
		},
		t.handlerVMsList,
	)
}

func (t *ProxmoxTools) handlerVMsList(ctx context.Context, req *mcp.CallToolRequest, input vmsListInput) (*mcp.CallToolResult, any, error) {
	result, err := t.VMsList(ctx, input.Cluster, authorizationToken(req))
	if err != nil {
		return nil, VMsListResult{}, err
	}

	return &mcp.CallToolResult{Content: []mcp.Content{
		&mcp.TextContent{Text: formatter.ToText(result)},
	}}, *result, nil
}

// VMsList returns QEMU virtual machines from the Proxmox cluster.
func (t *ProxmoxTools) VMsList(ctx context.Context, cluster, authToken string) (*VMsListResult, error) {
	guests, err := t.listGuests(ctx, cluster, authToken, "qemu")
	if err != nil {
		return nil, err
	}

	return &VMsListResult{Cluster: cluster, Count: len(guests), VMs: guests}, nil
}
