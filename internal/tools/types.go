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

// NodeVersion contains the Proxmox VE version installed on a node.
type NodeVersion struct {
	Release string `json:"release" jsonschema:"Proxmox VE release"`
	Version string `json:"version" jsonschema:"Installed pve-manager version"`
}

// NodeCurrentKernel contains details about the kernel running on a node.
type NodeCurrentKernel struct {
	Sysname string `json:"sysname" jsonschema:"Operating system name"`
	Release string `json:"release" jsonschema:"Kernel release"`
	Version string `json:"version" jsonschema:"Kernel build version"`
	Machine string `json:"machine" jsonschema:"Machine architecture"`
}

// NodeBootInfo contains details about how a node booted.
type NodeBootInfo struct {
	Mode       string `json:"mode" jsonschema:"Firmware boot mode"`
	SecureBoot bool   `json:"secure_boot" jsonschema:"Whether EFI Secure Boot is enabled"`
}

// NodeSummary contains the status and resource usage of a Proxmox node.
type NodeSummary struct {
	Name      string `json:"name" jsonschema:"Node name"`
	Status    string `json:"status" jsonschema:"Node status"`
	Resources string `json:"resources" jsonschema:"Node resources (CPU, Memory, Storage)"`
}

// GuestSummary contains the status and resource usage of a virtual machine or container.
type GuestSummary struct {
	VMID      int      `json:"vmid" jsonschema:"ID"`
	Name      string   `json:"name" jsonschema:"Name"`
	Node      string   `json:"node" jsonschema:"Node"`
	Status    string   `json:"status" jsonschema:"Status"`
	Template  bool     `json:"template" jsonschema:"Template"`
	Tags      []string `json:"tags,omitempty" jsonschema:"Tags"`
	Uptime    string   `json:"uptime,omitempty" jsonschema:"Uptime"`
	Resources string   `json:"resources" jsonschema:"Resources (CPU, Memory, Disk)"`
}

// GuestNetworkInterface contains runtime network information for a guest.
type GuestNetworkInterface struct {
	Name            string   `json:"name" jsonschema:"Interface name"`
	HardwareAddress string   `json:"hardware_address,omitempty" jsonschema:"Hardware address"`
	IPAddresses     []string `json:"ip_addresses,omitempty" jsonschema:"IP addresses in CIDR notation"`
}

// EventSummary contains details of a Proxmox task event.
type EventSummary struct {
	Node      string `json:"node" jsonschema:"Node name"`
	Type      string `json:"type" jsonschema:"Task type"`
	User      string `json:"user" jsonschema:"User that started the task"`
	StartTime string `json:"start_time" jsonschema:"Task start time"`
	Duration  string `json:"duration,omitempty" jsonschema:"Task duration"`
	Status    string `json:"status" jsonschema:"Task status"`
}

// StorageSummary contains the status and capacity of a Proxmox storage.
type StorageSummary struct {
	Name      string              `json:"name" jsonschema:"Storage name"`
	Available []string            `json:"available,omitempty" jsonschema:"Available on nodes"`
	Other     map[string][]string `json:"other,omitempty" jsonschema:"Other statuses and their nodes"`
	Type      string              `json:"type" jsonschema:"Storage plugin type"`
	Content   string              `json:"content" jsonschema:"Supported storage content types"`
	Shared    bool                `json:"shared" jsonschema:"Whether the storage is shared"`
	Resources string              `json:"resources" jsonschema:"Storage capacity and usage"`
}

// StorageContentSummary contains a volume stored on a Proxmox storage.
type StorageContentSummary struct {
	VolumeID  string `json:"volume_id" jsonschema:"Volume identifier"`
	VMID      int    `json:"vmid,omitempty" jsonschema:"Owning guest ID"`
	Format    string `json:"format,omitempty" jsonschema:"Volume format"`
	Size      int64  `json:"size" jsonschema:"Volume size in bytes"`
	Used      int64  `json:"used,omitempty" jsonschema:"Used space in bytes"`
	CreatedAt string `json:"created_at,omitempty" jsonschema:"Volume creation time"`
	Notes     string `json:"notes,omitempty" jsonschema:"Volume notes"`
	Protected bool   `json:"protected" jsonschema:"Whether the volume is protected"`
}
