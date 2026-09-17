# Proxmox MCP Server — Architecture

## Go-based Local Daemon with stdio + SSE, Typed Structured Output for Every Tool

MCP server will implement basic tools to make simple interactions with Proxmox easier and more structured.
Most of the tools are safe to use and to not cause any unintended side effects.

The tools does not repeat the Proxmox API, those accumulate the functions to provide rich output for AI agents.

### Config

MCP server supports multi cluster management.

```yaml
clusters:
  # List of Proxmox clusters
  - url: https://cluster-api-1.exmple.com:8006/api2/json
    # Optional, path to the CA certificate
    ca_file: "/etc/proxmox/ca.crt"
    # Skip the certificate verification, if needed
    insecure: false
    # Proxmox api token
    token_id: "kubernetes-csi@pve!csi"
    token_id_file: "/etc/proxmox/token_id"          # Optional, alternative to token_id
    token_secret: "secret"
    token_secret_file: "/etc/proxmox/token_secret"  # Optional, alternative to token_secret
    # Region name, which is cluster name
    region: Region-1

  # Add more clusters if needed
  - url: https://cluster-api-2.exmple.com:8006/api2/json
    insecure: false
    token_id: "kubernetes-csi@pve!csi"
    token_secret: "secret"
    region: Region-2
```

### Authorization

By default, the MCP server uses the provided Proxmox API tokens for authorization.
In SSE (server) mode, the MCP server will passthrough the oidc token to delegate authorization to the Proxmox API.

## CLI interface

The binary exposes cobra subcommands that select the transport: `mcp` runs stdio, `server` runs SSE, `version` prints the version, `tools` to run various Proxmox management tasks are available as subcommands as well.

## Tools

Cluster Management:
- proxmox_clusters_describe
- proxmox_clusters_list
- proxmox_nodes_describe
- proxmox_nodes_list
- proxmox_events_list

Virtual Machine Management:
- proxmox_vms_list
- proxmox_vms_describe
- proxmox_vms_backup
- proxmox_vms_reboot

Container Management:
- proxmox_containers_list
- proxmox_containers_describe
- proxmox_containers_backup
- proxmox_containers_reboot

Storage Management:
- proxmox_storage_list
- proxmox_storage_describe

## Architecture overview

```
cmd/proxmox-mcp (cobra root)
   |-- mcp      -> stdio transport
   |-- server   -> SSE transport  (--port, --allow-destructive, --extensions)
   |-- version
   `-- tools    -> ad-hoc Proxmox management tasks
          |
          v
internal/server   MCP server bootstrap, tool registration, transport wiring
          |
          v
internal/tools    tool handlers, grouped by domain:
                  cluster / vm / container / storage / backup
          |
          v
internal/proxmoxpool    Proxmox cluster pool (multi-cluster client pool)
          |
          v
internal/config      YAML loading -> Config / Cluster
          |
          v
```

### Package layout

```
cmd/proxmox-mcp/
    main.go        entrypoint, version/commit ldflags
    root.go        cobra root command
    mcp.go         `mcp` subcommand (stdio)
    server.go      `server` subcommand (SSE)
    version.go     `version` subcommand
    tools.go       `tools` subcommands

internal/
    config/        Config, Cluster, YAML loading & validation
    proxmoxpool/   Proxmox cluster pool (multi-cluster client pool)
    server/        MCP server setup, tool registration, transports
    tools/         tool handlers (cluster, vm, container, storage, backup)
```

### Layering rules

- `internal/tools` depends only on `internal/proxmoxpool`, `internal/config`
  and the MCP SDK (tool registration); it never constructs Proxmox clients directly.
- `internal/proxmoxpool` depends on `internal/config` and `go-proxmox`.
- `internal/server` wires transports to tools; it is the only layer that knows
  about MCP transports and OIDC passthrough.
- `cmd/proxmox-mcp` only parses flags and delegates; no business logic.
- Destructive tools (`*_backup`, `*_restore`) are gated by `--allow-destructive`.

## Core Dependencies

```go
module github.com/sergelogvinov/proxmox-mcp

go 1.26

require (
    modelcontextprotocol/go-sdk v1.7.0      // MCP SDK
    github.com/spf13/pflag v1.0.10          // Flags
    github.com/spf13/cobra v1.10.2          // CLI command dispatch
)
```
