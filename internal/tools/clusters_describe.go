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
	proxmox "github.com/sergelogvinov/go-proxmox-rest"
	proxmoxcluster "github.com/sergelogvinov/go-proxmox-rest/cluster"
	"github.com/sergelogvinov/proxmox-mcp/pkg/formatter"
)

// ClustersDescribeResult is the structured output of the proxmox_clusters_describe tool.
type ClustersDescribeResult struct {
	Cluster       string   `json:"cluster" jsonschema:"Name of the cluster"`
	Version       string   `json:"version" jsonschema:"Proxmox version"`
	HAStatus      string   `json:"ha_status,omitempty" jsonschema:"High availability status of the cluster"`
	HAState       []string `json:"ha_state,omitempty" jsonschema:"Local Resource Manager state"`
	CephStatus    string   `json:"ceph_status,omitempty" jsonschema:"Ceph cluster health status"`
	NodeStatus    string   `json:"node_status" jsonschema:"Node counts in Ready/NotReady/Unknown"`
	NodeResources string   `json:"node_resources" jsonschema:"Total resources in the cluster (CPU, Memory, Storage)"`
	Nodes         []string `json:"nodes" jsonschema:"List of nodes"`
}

// clustersDescribeInput is the input of the proxmox_clusters_describe tool.
type clustersDescribeInput struct {
	Cluster string `json:"cluster" jsonschema:"Cluster name"`
}

// RegisterClustersDescribe registers the clusters describe tool.
func (t *ProxmoxTools) RegisterClustersDescribe(srv *mcp.Server) {
	mcp.AddTool(srv,
		&mcp.Tool{
			Name:        "proxmox_clusters_describe",
			Description: "Describe a Proxmox cluster: returns its version, nodes, node statuses, and total node resources.",
			Annotations: &mcp.ToolAnnotations{
				IdempotentHint: true,
				ReadOnlyHint:   true,
				OpenWorldHint:  new(true),
			},
		},
		t.handlerClustersDescribe,
	)
}

func (t *ProxmoxTools) handlerClustersDescribe(ctx context.Context, req *mcp.CallToolRequest, input clustersDescribeInput) (*mcp.CallToolResult, any, error) {
	result, err := t.ClustersDescribe(ctx, input.Cluster, authorizationToken(req))
	if err != nil {
		return nil, ClustersDescribeResult{}, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: formatter.ToText(result)},
		},
	}, *result, nil
}

// ClustersDescribe returns details of the Proxmox cluster in the given cluster.
func (t *ProxmoxTools) ClustersDescribe(ctx context.Context, cluster, authToken string) (*ClustersDescribeResult, error) {
	px, err := t.pool.GetProxmoxClusterWithToken(cluster, authToken)
	if err != nil {
		return nil, err
	}

	version, err := px.Version(ctx)
	if err != nil {
		return nil, err
	}

	resources, err := px.Cluster().Resources().List(ctx, proxmoxcluster.ListFilter{Type: proxmoxcluster.ResourceTypeNode})
	if err != nil {
		return nil, err
	}

	nodes := make([]string, 0, len(resources))

	var (
		ready, notReady, unknown int
		cpu                      int
		memory, storage          int64
		usedCPU                  float64
		usedMemory, usedStorage  int64
	)

	for _, resource := range resources {
		node := resource.Node
		if node == "" {
			node = resource.Name
		}

		if node != "" {
			nodes = append(nodes, node)
		}

		cpu += resource.MaxCPU
		usedCPU += resource.CPU
		memory += resource.MaxMem / (1024 * 1024 * 1024)
		usedMemory += resource.Mem / (1024 * 1024 * 1024)
		storage += resource.MaxDisk / (1024 * 1024 * 1024)
		usedStorage += resource.Disk / (1024 * 1024 * 1024)

		switch resource.Status {
		case "online":
			ready++
		case "stopped":
			notReady++
		default:
			unknown++
		}
	}

	slices.Sort(nodes)

	nodeStatus := fmt.Sprintf("%d/%d/%d", ready, notReady, unknown)
	nodeResources := fmt.Sprintf("cpu=%d (used=%.0f%%), memory=%dGiB (used=%dGiB), system storage=%dGiB (used=%dGiB)", cpu, (usedCPU/float64(ready))*100, memory, usedMemory, storage, usedStorage)

	cephStatus := ""
	ceph, err := px.Cluster().Ceph().Status(ctx)
	if err != nil && !proxmox.IsNotFound(err) {
		return nil, err
	}

	if ceph != nil && ceph.Health != nil {
		cephStatus = strings.TrimPrefix(strings.ToLower(ceph.Health.Status), "health_")
	}

	haStatus := ""
	haState := []string{}
	if managerStatus, err := px.Cluster().HA().Status().ManagerStatus(ctx); err == nil && managerStatus != nil {
		if managerStatus.Quorum != nil && managerStatus.Quorum.Quorate == 1 {
			haStatus = "quorate"
		} else {
			haStatus = "not quorate"
		}

		for node, lrm := range managerStatus.LRM {
			mode := lrm.Mode
			if mode == "" {
				mode = lrm.State
			}

			haState = append(haState, fmt.Sprintf("%s=%s", node, mode))
		}

		slices.Sort(haState)
	}

	return &ClustersDescribeResult{
		Cluster:       cluster,
		Version:       version.Version,
		HAStatus:      haStatus,
		HAState:       haState,
		CephStatus:    cephStatus,
		NodeStatus:    nodeStatus,
		NodeResources: nodeResources,
		Nodes:         nodes,
	}, nil
}
