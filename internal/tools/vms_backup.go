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

// VMsBackupResult is the structured output of the proxmox_vms_backup tool.
type VMsBackupResult struct {
	Cluster string `json:"cluster" jsonschema:"Name of the cluster"`
	Node    string `json:"node" jsonschema:"Node hosting the virtual machine"`
	VMID    int    `json:"vmid" jsonschema:"Virtual machine ID"`
	UPID    string `json:"upid" jsonschema:"Backup task identifier"`
}

type vmsBackupInput struct {
	Cluster  string `json:"cluster" jsonschema:"Cluster name"`
	Node     string `json:"node,omitempty" jsonschema:"Node hosting the virtual machine; discovered automatically when omitted"`
	VMID     int    `json:"vmid" jsonschema:"Virtual machine ID"`
	Storage  string `json:"storage,omitempty" jsonschema:"Target backup storage"`
	Mode     string `json:"mode,omitempty" jsonschema:"Backup mode: snapshot, suspend, or stop"`
	Compress string `json:"compress,omitempty" jsonschema:"Compression: 0, gzip, lzo, or zstd"`
}

// RegisterVMsBackup registers the virtual machine backup tool.
func (t *ProxmoxTools) RegisterVMsBackup(srv *mcp.Server) {
	mcp.AddTool(srv,
		&mcp.Tool{
			Name:        "proxmox_vms_backup",
			Description: "Start a backup of one QEMU virtual machine and return the asynchronous task identifier.",
			Annotations: &mcp.ToolAnnotations{
				DestructiveHint: new(true),
				IdempotentHint:  false,
				ReadOnlyHint:    false,
				OpenWorldHint:   new(true),
			},
		},
		t.handlerVMsBackup,
	)
}

func (t *ProxmoxTools) handlerVMsBackup(ctx context.Context, req *mcp.CallToolRequest, input vmsBackupInput) (*mcp.CallToolResult, any, error) {
	result, err := t.VMsBackup(ctx, input.Cluster, input.Node, input.VMID, input.Storage, input.Mode, input.Compress, authorizationToken(req))
	if err != nil {
		return nil, VMsBackupResult{}, err
	}

	return &mcp.CallToolResult{Content: []mcp.Content{
		&mcp.TextContent{Text: formatter.ToText(result)},
	}}, *result, nil
}

// VMsBackup starts an asynchronous backup of a QEMU virtual machine.
func (t *ProxmoxTools) VMsBackup(ctx context.Context, cluster, node string, vmid int, storage, mode, compress, authToken string) (*VMsBackupResult, error) {
	if vmid <= 0 {
		return nil, fmt.Errorf("virtual machine ID must be greater than zero")
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
			if resource.Type == "qemu" && resource.VMID == vmid {
				node = resource.Node

				break
			}
		}
		if node == "" {
			return nil, fmt.Errorf("virtual machine %d not found in cluster %q", vmid, cluster)
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

	return &VMsBackupResult{Cluster: cluster, Node: node, VMID: vmid, UPID: upid}, nil
}

func validateBackupMode(mode string) error {
	switch vzdump.Mode(mode) {
	case "", vzdump.ModeSnapshot, vzdump.ModeSuspend, vzdump.ModeStop:
		return nil
	default:
		return fmt.Errorf("invalid backup mode %q: must be snapshot, suspend, or stop", mode)
	}
}

func validateBackupCompression(compress string) error {
	switch vzdump.Compress(compress) {
	case "", vzdump.CompressNone, vzdump.CompressGzip, vzdump.CompressLZO, vzdump.CompressZstd:
		return nil
	default:
		return fmt.Errorf("invalid backup compression %q: must be 0, gzip, lzo, or zstd", compress)
	}
}
