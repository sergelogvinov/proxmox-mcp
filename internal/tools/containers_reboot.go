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

// RegisterContainersReboot registers the container reboot tool.
func (t *ProxmoxTools) RegisterContainersReboot(srv *mcp.Server) {
	mcp.AddTool(srv,
		&mcp.Tool{
			Name:        "proxmox_containers_reboot",
			Description: "Gracefully reboot an LXC container and return the task identifier.",
			Annotations: &mcp.ToolAnnotations{
				DestructiveHint: new(true),
				IdempotentHint:  false,
				ReadOnlyHint:    false,
				OpenWorldHint:   new(true),
			},
		},
		t.handlerContainersReboot,
	)
}

func (t *ProxmoxTools) handlerContainersReboot(ctx context.Context, req *mcp.CallToolRequest, input guestRebootInput) (*mcp.CallToolResult, any, error) {
	result, err := t.ContainersReboot(ctx, input.Cluster, input.Node, input.VMID, authorizationToken(req))
	if err != nil {
		return nil, GuestRebootResult{}, err
	}

	return &mcp.CallToolResult{Content: []mcp.Content{
		&mcp.TextContent{Text: formatter.ToText(result)},
	}}, *result, nil
}

// ContainersReboot gracefully reboots an LXC container.
func (t *ProxmoxTools) ContainersReboot(ctx context.Context, cluster, node string, vmid int, authToken string) (*GuestRebootResult, error) {
	px, err := t.pool.GetProxmoxClusterWithToken(cluster, authToken)
	if err != nil {
		return nil, err
	}

	upid, err := px.Nodes().LXC().Reboot(ctx, node, vmid, nil)
	if err != nil {
		return nil, err
	}

	return &GuestRebootResult{Cluster: cluster, Node: node, VMID: vmid, UPID: upid}, nil
}
