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
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sergelogvinov/proxmox-mcp/internal/proxmoxpool"
)

// ProxmoxTools provides tool handlers with access to the Proxmox cluster pool.
type ProxmoxTools struct {
	pool *proxmoxpool.ProxmoxPool
}

// NewProxmoxTools creates tool handlers backed by the given Proxmox cluster pool.
func NewProxmoxTools(pool *proxmoxpool.ProxmoxPool) *ProxmoxTools {
	return &ProxmoxTools{pool: pool}
}

// RegisterTools registers all proxmox-mcp tools on the MCP server.
func (t *ProxmoxTools) RegisterTools(srv *mcp.Server) {
	t.RegisterClustersDescribe(srv)
	t.RegisterClustersList(srv)
	t.RegisterContainersList(srv)
	t.RegisterEventsList(srv)
	t.RegisterNodesDescribe(srv)
	t.RegisterNodesList(srv)
	t.RegisterStorageDescribe(srv)
	t.RegisterStorageList(srv)
	t.RegisterVMsList(srv)
}

// authorizationToken returns the Authorization token value from an MCP
// tool request, or an empty string if there is none. It specifically looks
// for the "PVEAPIToken" prefix in the Authorization header.
func authorizationToken(req *mcp.CallToolRequest) string {
	if req == nil || req.Extra == nil {
		return ""
	}

	authToken, _ := strings.CutPrefix(req.Extra.Header.Get("Authorization"), "PVEAPIToken")
	authToken = strings.TrimPrefix(authToken, "=")
	return strings.TrimSpace(authToken)
}
