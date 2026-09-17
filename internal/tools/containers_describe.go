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
	proxmoxlxc "github.com/sergelogvinov/go-proxmox-rest/nodes/lxc"
	proxmoxstorage "github.com/sergelogvinov/go-proxmox-rest/nodes/storage"
	"github.com/sergelogvinov/proxmox-mcp/pkg/formatter"
)

// ContainersDescribeResult is the structured output of the proxmox_containers_describe tool.
type ContainersDescribeResult struct {
	Cluster           string                  `json:"cluster" jsonschema:"Name of the cluster"`
	Node              string                  `json:"node" jsonschema:"Node hosting the container"`
	VMID              int                     `json:"vmid" jsonschema:"Container ID"`
	Name              string                  `json:"name" jsonschema:"Container name"`
	Status            string                  `json:"status" jsonschema:"Container status"`
	Uptime            string                  `json:"uptime,omitempty" jsonschema:"Container uptime"`
	CPUUsage          float64                 `json:"cpu_usage,omitempty" jsonschema:"Current CPU usage as a fraction"`
	CPUs              float64                 `json:"cpus" jsonschema:"Maximum usable CPU count"`
	Memory            int64                   `json:"memory,omitempty" jsonschema:"Current memory usage in bytes"`
	MaxMemory         int64                   `json:"max_memory" jsonschema:"Maximum memory in bytes"`
	MaxSwap           int64                   `json:"max_swap" jsonschema:"Maximum swap in bytes"`
	Disk              int64                   `json:"disk_usage,omitempty" jsonschema:"Current root disk usage in bytes"`
	MaxDisk           int64                   `json:"disk" jsonschema:"Root disk size in bytes"`
	DiskRead          int64                   `json:"disk_read,omitempty" jsonschema:"Bytes read from block devices since start"`
	DiskWrite         int64                   `json:"disk_write,omitempty" jsonschema:"Bytes written to block devices since start"`
	NetworkIn         int64                   `json:"network_in,omitempty" jsonschema:"Network bytes received since start"`
	NetworkOut        int64                   `json:"network_out,omitempty" jsonschema:"Network bytes sent since start"`
	Lock              string                  `json:"lock,omitempty" jsonschema:"Current configuration lock"`
	Tags              []string                `json:"tags,omitempty" jsonschema:"Container tags"`
	Template          bool                    `json:"template" jsonschema:"Container is a template"`
	HA                map[string]any          `json:"ha,omitempty" jsonschema:"High availability service status"`
	NetworkInterfaces []GuestNetworkInterface `json:"network_interfaces,omitempty" jsonschema:"Runtime container network interfaces"`
	Backups           []StorageContentSummary `json:"backups,omitempty" jsonschema:"Backups available for the container"`
}

type containersDescribeInput struct {
	Cluster string `json:"cluster" jsonschema:"Cluster name"`
	Node    string `json:"node,omitempty" jsonschema:"Node hosting the container; discovered automatically when omitted"`
	VMID    int    `json:"vmid" jsonschema:"Container ID"`
}

// RegisterContainersDescribe registers the container describe tool.
func (t *ProxmoxTools) RegisterContainersDescribe(srv *mcp.Server) {
	mcp.AddTool(srv,
		&mcp.Tool{
			Name:        "proxmox_containers_describe",
			Description: "Describe an LXC container, including its current status and runtime network interfaces when running.",
			Annotations: &mcp.ToolAnnotations{
				IdempotentHint: true,
				ReadOnlyHint:   true,
				OpenWorldHint:  new(true),
			},
		},
		t.handlerContainersDescribe,
	)
}

func (t *ProxmoxTools) handlerContainersDescribe(ctx context.Context, req *mcp.CallToolRequest, input containersDescribeInput) (*mcp.CallToolResult, any, error) {
	result, err := t.ContainersDescribe(ctx, input.Cluster, input.Node, input.VMID, authorizationToken(req))
	if err != nil {
		return nil, ContainersDescribeResult{}, err
	}

	return &mcp.CallToolResult{Content: []mcp.Content{
		&mcp.TextContent{Text: formatter.ToText(result)},
	}}, *result, nil
}

// ContainersDescribe returns the current status of an LXC container.
func (t *ProxmoxTools) ContainersDescribe(ctx context.Context, cluster, node string, vmid int, authToken string) (*ContainersDescribeResult, error) {
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
			if resource.Type == "lxc" && resource.VMID == vmid {
				node = resource.Node

				break
			}
		}
		if node == "" {
			return nil, fmt.Errorf("container %d not found in cluster %q", vmid, cluster)
		}
	}

	status, err := px.Nodes(node).LXC().Status(ctx, vmid)
	if err != nil {
		return nil, err
	}

	tags := strings.FieldsFunc(status.Tags, func(r rune) bool { return r == ',' || r == ';' })
	for i := range tags {
		tags[i] = strings.TrimSpace(tags[i])
	}
	slices.Sort(tags)

	result := &ContainersDescribeResult{
		Cluster:    cluster,
		Node:       node,
		VMID:       status.VMID,
		Name:       status.Name,
		Status:     string(status.Status),
		CPUUsage:   status.CPU,
		CPUs:       status.CPUs,
		Memory:     status.Mem,
		MaxMemory:  status.MaxMem,
		MaxSwap:    status.MaxSwap,
		Disk:       status.Disk,
		MaxDisk:    status.MaxDisk,
		DiskRead:   status.DiskRead,
		DiskWrite:  status.DiskWrite,
		NetworkIn:  status.NetIn,
		NetworkOut: status.NetOut,
		Lock:       status.Lock,
		Tags:       tags,
		Template:   status.Template,
		HA:         status.HA,
	}
	if status.Uptime > 0 {
		result.Uptime = formatUptime(status.Uptime)
	}

	if status.Status == proxmoxlxc.StateRunning {
		interfaces, err := px.Nodes(node).LXC().Interfaces(ctx, vmid)
		if err == nil {
			result.NetworkInterfaces = make([]GuestNetworkInterface, 0, len(interfaces))
			for _, networkInterface := range interfaces {
				addresses := make([]string, 0, len(networkInterface.IPAddresses))
				for _, address := range networkInterface.IPAddresses {
					if address.IPAddress != "" {
						addresses = append(addresses, fmt.Sprintf("%s/%d", address.IPAddress, address.Prefix))
					}
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
