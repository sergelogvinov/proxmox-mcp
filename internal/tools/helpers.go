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

	proxmoxcluster "github.com/sergelogvinov/go-proxmox-rest/cluster"
)

func (t *ProxmoxTools) listGuests(ctx context.Context, cluster, authToken, guestType string) ([]GuestSummary, error) {
	px, err := t.pool.GetProxmoxClusterWithToken(cluster, authToken)
	if err != nil {
		return nil, err
	}

	resources, err := px.Cluster().Resources().List(ctx, proxmoxcluster.ListFilter{Type: proxmoxcluster.ResourceTypeVM})
	if err != nil {
		return nil, err
	}

	guests := make([]GuestSummary, 0, len(resources))
	for _, resource := range resources {
		if resource.Type != guestType {
			continue
		}

		if resource.Name == "node-capacity" && resource.Tags == "karpenter" {
			continue
		}

		// Resources usage in hugepages is always close to zero
		if resource.Type == "qemu" && resource.Status == "running" {
			if resource.Mem < 512*1024*1024 {
				resource.Mem = resource.MaxMem
			}
		}

		guests = append(guests, newGuestSummary(resource))
	}
	slices.SortFunc(guests, func(a, b GuestSummary) int { return a.VMID - b.VMID })

	return guests, nil
}

func newGuestSummary(resource proxmoxcluster.Resource) GuestSummary {
	tags := strings.FieldsFunc(resource.Tags, func(r rune) bool { return r == ',' || r == ';' })
	for i := range tags {
		tags[i] = strings.TrimSpace(tags[i])
	}
	slices.Sort(tags)

	uptime := ""
	if resource.Uptime > 0 {
		uptime = formatUptime(int64(resource.Uptime))
	}

	return GuestSummary{
		VMID:     resource.VMID,
		Name:     resource.Name,
		Node:     resource.Node,
		Status:   resource.Status,
		Template: resource.Template == 1,
		Tags:     tags,
		Uptime:   uptime,
		Resources: fmt.Sprintf(
			"cpu=%d (used=%.0f%%), memory=%dGiB (used=%dGiB), disk=%dGiB",
			resource.MaxCPU,
			resource.CPU*100,
			resource.MaxMem/(1024*1024*1024),
			resource.Mem/(1024*1024*1024),
			resource.MaxDisk/(1024*1024*1024),
		),
	}
}
