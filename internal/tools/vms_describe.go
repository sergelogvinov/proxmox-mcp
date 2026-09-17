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
	"slices"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	proxmoxcluster "github.com/sergelogvinov/go-proxmox-rest/cluster"
	proxmoxstorage "github.com/sergelogvinov/go-proxmox-rest/nodes/storage"
	"github.com/sergelogvinov/proxmox-mcp/pkg/formatter"
)

// VMsDescribeResult is the structured output of the proxmox_vms_describe tool.
type VMsDescribeResult struct {
	Cluster           string                  `json:"cluster" jsonschema:"Name of the cluster"`
	Node              string                  `json:"node" jsonschema:"Node hosting the virtual machine"`
	VMID              int                     `json:"vmid" jsonschema:"Virtual machine ID"`
	Name              string                  `json:"name" jsonschema:"Virtual machine name"`
	Status            string                  `json:"status" jsonschema:"Virtual machine status"`
	Uptime            string                  `json:"uptime,omitempty" jsonschema:"Virtual machine uptime"`
	CPUUsage          float64                 `json:"cpu_usage,omitempty" jsonschema:"Current CPU usage as a fraction"`
	CPUs              float64                 `json:"cpus" jsonschema:"Maximum usable CPU count"`
	Memory            int64                   `json:"memory,omitempty" jsonschema:"Current memory usage in bytes"`
	MaxMemory         int64                   `json:"max_memory" jsonschema:"Maximum memory in bytes"`
	MaxDisk           int64                   `json:"disk" jsonschema:"Root disk size in bytes"`
	DiskRead          int64                   `json:"disk_read,omitempty" jsonschema:"Bytes read from block devices since start"`
	DiskWrite         int64                   `json:"disk_write,omitempty" jsonschema:"Bytes written to block devices since start"`
	NetworkIn         int64                   `json:"network_in,omitempty" jsonschema:"Network bytes received since start"`
	NetworkOut        int64                   `json:"network_out,omitempty" jsonschema:"Network bytes sent since start"`
	Tags              []string                `json:"tags,omitempty" jsonschema:"Virtual machine tags"`
	RunningMachine    string                  `json:"running_machine,omitempty" jsonschema:"Running QEMU machine type"`
	RunningQEMU       string                  `json:"running_qemu,omitempty" jsonschema:"Running QEMU version"`
	Template          bool                    `json:"template" jsonschema:"Virtual machine is a template"`
	HA                map[string]any          `json:"ha,omitempty" jsonschema:"High availability service status"`
	Serial            bool                    `json:"serial,omitempty" jsonschema:"Serial device is configured"`
	AgentEnabled      bool                    `json:"agent_enabled,omitempty" jsonschema:"QEMU Agent is enabled"`
	AgentAlive        bool                    `json:"agent_alive,omitempty" jsonschema:"QEMU Agent is responding"`
	NetworkInterfaces []GuestNetworkInterface `json:"network_interfaces,omitempty" jsonschema:"Guest network interfaces reported by the QEMU guest agent"`
	Backups           []StorageContentSummary `json:"backups,omitempty" jsonschema:"Backups available for the virtual machine"`
}

type vmsDescribeInput struct {
	Cluster string `json:"cluster" jsonschema:"Cluster name"`
	Node    string `json:"node,omitempty" jsonschema:"Node hosting the virtual machine; discovered automatically when omitted"`
	VMID    int    `json:"vmid" jsonschema:"Virtual machine ID"`
}

// RegisterVMsDescribe registers the virtual machine describe tool.
func (t *ProxmoxTools) RegisterVMsDescribe(srv *mcp.Server) {
	mcp.AddTool(srv,
		&mcp.Tool{
			Name:        "proxmox_vms_describe",
			Description: "Describe a QEMU virtual machine, including current status and guest network interfaces when its QEMU guest agent is responding.",
			Annotations: &mcp.ToolAnnotations{
				IdempotentHint: true,
				ReadOnlyHint:   true,
				OpenWorldHint:  new(true),
			},
		},
		t.handlerVMsDescribe,
	)
}

func (t *ProxmoxTools) handlerVMsDescribe(ctx context.Context, req *mcp.CallToolRequest, input vmsDescribeInput) (*mcp.CallToolResult, any, error) {
	result, err := t.VMsDescribe(ctx, input.Cluster, input.Node, input.VMID, authorizationToken(req))
	if err != nil {
		return nil, VMsDescribeResult{}, err
	}

	return &mcp.CallToolResult{Content: []mcp.Content{
		&mcp.TextContent{Text: formatter.ToText(result)},
	}}, *result, nil
}

// VMsDescribe returns the current status of a QEMU virtual machine.
func (t *ProxmoxTools) VMsDescribe(ctx context.Context, cluster, node string, vmid int, authToken string) (*VMsDescribeResult, error) {
	px, err := t.pool.GetProxmoxClusterWithToken(cluster, authToken)
	if err != nil {
		return nil, err
	}
	if node == "" {
		resources, err := px.Cluster().Resources().Get(ctx, proxmoxcluster.ResourceTypeVM)
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

	status, err := px.Nodes(node).Qemu().Status(ctx, vmid)
	if err != nil {
		return nil, err
	}

	tags := strings.FieldsFunc(status.Tags, func(r rune) bool { return r == ',' || r == ';' })
	for i := range tags {
		tags[i] = strings.TrimSpace(tags[i])
	}
	slices.Sort(tags)

	result := &VMsDescribeResult{
		Cluster:        cluster,
		Node:           node,
		VMID:           status.VMID,
		Name:           status.Name,
		Status:         string(status.Status),
		CPUUsage:       status.CPU,
		CPUs:           status.CPUs,
		Memory:         status.Mem,
		MaxMemory:      status.MaxMem,
		MaxDisk:        status.MaxDisk,
		DiskRead:       status.DiskRead,
		DiskWrite:      status.DiskWrite,
		NetworkIn:      status.NetIn,
		NetworkOut:     status.NetOut,
		Tags:           tags,
		RunningMachine: status.RunningMachine,
		RunningQEMU:    status.RunningQemu,
		Template:       status.Template,
		HA:             status.HA,
		Serial:         status.Serial,
		AgentEnabled:   status.Agent,
	}
	if status.Uptime > 0 {
		result.Uptime = formatUptime(status.Uptime)
	}

	if status.Agent && status.Status == "running" {
		agent := px.Nodes(node).Qemu().Agent()
		if err := agent.Ping(ctx, vmid); err == nil {
			result.AgentAlive = true

			interfaces, err := agent.NetworkGetInterfaces(ctx, vmid)
			if err == nil {
				result.NetworkInterfaces = make([]GuestNetworkInterface, 0, len(interfaces))
				for _, networkInterface := range interfaces {
					addresses := make([]string, 0, len(networkInterface.IPAddresses))
					for _, address := range networkInterface.IPAddresses {
						addresses = append(addresses, fmt.Sprintf("%s/%d", address.IPAddress, address.Prefix))
					}
					slices.Sort(addresses)

					result.NetworkInterfaces = append(result.NetworkInterfaces, GuestNetworkInterface{
						Name:            networkInterface.Name,
						HardwareAddress: networkInterface.HardwareAddress,
						IPAddresses:     addresses,
					})
				}
				slices.SortFunc(result.NetworkInterfaces, func(a, b GuestNetworkInterface) int {
					return strings.Compare(a.Name, b.Name)
				})
			}
		}
	}

	storages, err := px.Nodes(node).Storage().List(ctx, &proxmoxstorage.ListOptions{
		Content: []string{"backup"},
	})
	if err != nil {
		return nil, err
	}

	for _, storage := range storages {
		volumes, err := px.Nodes(node).Storage().Content().List(ctx, storage.Storage, &proxmoxstorage.ContentListOptions{
			Content: "backup",
			VMID:    vmid,
		})
		if err != nil {
			return nil, err
		}

		for _, volume := range volumes {
			size := volume.Size
			if size == 0 {
				size = volume.ApproximateSize
			}

			result.Backups = append(result.Backups, StorageContentSummary{
				VolumeID:  volume.VolID,
				VMID:      volume.VMID,
				Format:    volume.Format,
				Size:      size,
				Used:      volume.Used,
				CreatedAt: formatStorageContentTime(volume.CTime),
				Notes:     volume.Notes,
				Protected: volume.Protected,
			})
		}
	}

	slices.SortFunc(result.Backups, func(a, b StorageContentSummary) int {
		return strings.Compare(a.VolumeID, b.VolumeID)
	})

	return result, nil
}
