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
	"strconv"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	proxmoxcluster "github.com/sergelogvinov/go-proxmox-rest/cluster"
	"github.com/sergelogvinov/go-proxmox-rest/nodes/vzdump"
	"github.com/sergelogvinov/proxmox-mcp/pkg/formatter"
)

// ContainersBackupResult is the structured output of the proxmox_containers_backup tool.
type ContainersBackupResult struct {
	Cluster string `json:"cluster" jsonschema:"Name of the cluster"`
	Node    string `json:"node" jsonschema:"Node hosting the container"`
	VMID    int    `json:"vmid" jsonschema:"Container ID"`
	UPID    string `json:"upid" jsonschema:"Backup task identifier"`
}

type containersBackupInput struct {
	Cluster  string `json:"cluster" jsonschema:"Cluster name"`
	Node     string `json:"node,omitempty" jsonschema:"Node hosting the container; discovered automatically when omitted"`
	VMID     int    `json:"vmid" jsonschema:"Container ID"`
	Storage  string `json:"storage,omitempty" jsonschema:"Target backup storage"`
	Mode     string `json:"mode,omitempty" jsonschema:"Backup mode: snapshot, suspend, or stop"`
	Compress string `json:"compress,omitempty" jsonschema:"Compression: 0, gzip, lzo, or zstd"`
}

// RegisterContainersBackup registers the container backup tool.
func (t *ProxmoxTools) RegisterContainersBackup(srv *mcp.Server) {
	mcp.AddTool(srv,
		&mcp.Tool{
			Name:        "proxmox_containers_backup",
			Description: "Start a backup of one LXC container and return the asynchronous task identifier.",
			Annotations: &mcp.ToolAnnotations{
				DestructiveHint: new(true),
				IdempotentHint:  false,
				ReadOnlyHint:    false,
				OpenWorldHint:   new(true),
			},
		},
		t.handlerContainersBackup,
	)
}

func (t *ProxmoxTools) handlerContainersBackup(ctx context.Context, req *mcp.CallToolRequest, input containersBackupInput) (*mcp.CallToolResult, any, error) {
	result, err := t.ContainersBackup(ctx, input.Cluster, input.Node, input.VMID, input.Storage, input.Mode, input.Compress, authorizationToken(req))
	if err != nil {
		return nil, ContainersBackupResult{}, err
	}

	return &mcp.CallToolResult{Content: []mcp.Content{
		&mcp.TextContent{Text: formatter.ToText(result)},
	}}, *result, nil
}

// ContainersBackup starts an asynchronous backup of an LXC container.
func (t *ProxmoxTools) ContainersBackup(ctx context.Context, cluster, node string, vmid int, storage, mode, compress, authToken string) (*ContainersBackupResult, error) {
	if vmid <= 0 {
		return nil, fmt.Errorf("container ID must be greater than zero")
	}
	if err := validateBackupMode(mode); err != nil {
		return nil, err
	}
	if err := validateBackupCompression(compress); err != nil {
		return nil, err
	}

	px, err := t.pool.GetProxmoxClusterWithToken(cluster, authToken)
	if err != nil {
		return nil, err
	}
	if node == "" {
		resources, err := px.Cluster().Resources().List(ctx, proxmoxcluster.ListFilter{Type: proxmoxcluster.ResourceTypeVM})
		if err != nil {
			return nil, err
		}

		for _, resource := range resources {
			if resource.Type == "lxc" && resource.VMID == vmid {
				node = resource.Node

				break
			}
		}
		if node == "" {
			return nil, fmt.Errorf("container %d not found in cluster %q", vmid, cluster)
		}
	}

	upid, err := px.Nodes(node).VZDump().Create(ctx, &vzdump.Options{
		VMID:     []string{strconv.Itoa(vmid)},
		Storage:  storage,
		Mode:     vzdump.Mode(mode),
		Compress: vzdump.Compress(compress),
	})
	if err != nil {
		return nil, err
	}

	return &ContainersBackupResult{Cluster: cluster, Node: node, VMID: vmid, UPID: upid}, nil
}
