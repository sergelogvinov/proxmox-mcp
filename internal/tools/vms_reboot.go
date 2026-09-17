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

// GuestRebootResult is the structured output of a guest reboot tool.
type GuestRebootResult struct {
	Cluster string `json:"cluster" jsonschema:"Name of the cluster"`
	Node    string `json:"node" jsonschema:"Node hosting the guest"`
	VMID    int    `json:"vmid" jsonschema:"Guest ID"`
	UPID    string `json:"upid" jsonschema:"Reboot task identifier"`
}

type guestRebootInput struct {
	Cluster string `json:"cluster" jsonschema:"Cluster name"`
	Node    string `json:"node" jsonschema:"Node hosting the guest"`
	VMID    int    `json:"vmid" jsonschema:"Guest ID"`
}

// RegisterVMsReboot registers the virtual machine reboot tool.
func (t *ProxmoxTools) RegisterVMsReboot(srv *mcp.Server) {
	mcp.AddTool(srv,
		&mcp.Tool{
			Name:        "proxmox_vms_reboot",
			Description: "Gracefully reboot a QEMU virtual machine and return the task identifier.",
			Annotations: &mcp.ToolAnnotations{
				DestructiveHint: new(true),
				IdempotentHint:  false,
				ReadOnlyHint:    false,
				OpenWorldHint:   new(true),
			},
		},
		t.handlerVMsReboot,
	)
}

func (t *ProxmoxTools) handlerVMsReboot(ctx context.Context, req *mcp.CallToolRequest, input guestRebootInput) (*mcp.CallToolResult, any, error) {
	result, err := t.VMsReboot(ctx, input.Cluster, input.Node, input.VMID, authorizationToken(req))
	if err != nil {
		return nil, GuestRebootResult{}, err
	}

	return &mcp.CallToolResult{Content: []mcp.Content{
		&mcp.TextContent{Text: formatter.ToText(result)},
	}}, *result, nil
}

// VMsReboot gracefully reboots a QEMU virtual machine.
func (t *ProxmoxTools) VMsReboot(ctx context.Context, cluster, node string, vmid int, authToken string) (*GuestRebootResult, error) {
	px, err := t.pool.GetProxmoxClusterWithToken(cluster, authToken)
	if err != nil {
		return nil, err
	}

	upid, err := px.Nodes(node).Qemu().Reboot(ctx, vmid, nil)
	if err != nil {
		return nil, err
	}

	return &GuestRebootResult{Cluster: cluster, Node: node, VMID: vmid, UPID: upid}, nil
}
