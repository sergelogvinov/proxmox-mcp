# Proxmox MCP Server

> **NOTE:**
> This project is under active development.

## Motivation

Modern infrastructure often uses tools such as Terraform/OpenTofu, Ansible,
and GitOps. These tools deploy and configure Proxmox infrastructure and
Kubernetes clusters.

The Proxmox MCP Server does not replace these tools. It gives AI assistants
and automation agents access to information about your Proxmox clusters.

The default tools are read-only. Optional action tools can be enabled with
`--allow-destructive` for operations such as backups and reboots.

Instead of checking many pages in the Proxmox UI or running several commands,
you can ask an AI assistant to collect and analyze the required information.
Your existing automation tools remain the main source for infrastructure
changes.

## Overview

The server connects an MCP client, such as an AI assistant, to the Proxmox VE API.
It supports multiple Proxmox clusters and returns structured data.

You can use it to:

- inspect clusters, nodes, and storage
- check resource usage and node status
- review recent Proxmox task events
- find differences and possible configuration problems
- collect information before you make infrastructure changes

Keep your AI on the leash.

## Proxmox core tools

The MCP server provides the following read-only tools:

| Tool | Arguments | Description |
| --- | --- | --- |
| `proxmox_clusters_list` | None | List the configured Proxmox clusters. |
| `proxmox_clusters_describe` | `cluster` | Show a cluster's version, nodes, node statuses, and total resources. |
| `proxmox_containers_list` | `cluster` | List LXC containers with their status and resource usage. |
| `proxmox_containers_describe` | `cluster`, `vmid`, `node` (optional) | Show an LXC container's current status and runtime network interfaces when it is running. |
| `proxmox_events_list` | `cluster`, `limit` (optional) | List recent task events across all nodes. The default limit is `10`. |
| `proxmox_nodes_list` | `cluster` | List nodes with their status and resource usage. |
| `proxmox_nodes_describe` | `cluster`, `node` | Show a node's status, resources, uptime, version, kernel, and boot information. |
| `proxmox_storage_list` | `cluster` | List storage resources with their status, type, content, and capacity usage. |
| `proxmox_storage_describe` | `cluster`, `name`, `node` (optional) | Show storage details and, when a node is provided, content visible on that node. |
| `proxmox_vms_list` | `cluster` | List QEMU virtual machines with their status and resource usage. |
| `proxmox_vms_describe` | `cluster`, `vmid`, `node` (optional) | Show a QEMU virtual machine's current status and guest network interfaces when its QEMU guest agent is responding. |

Use `proxmox_clusters_list` first to find the cluster names used by the other
tools. MCP clients can discover the full input and output schemas. You can
also list the tools from the command line:

```sh
proxmox-mcp tools --config /absolute/path/to/cluster-config.yaml
```

When `--allow-destructive` is enabled, `proxmox_vms_backup` and
`proxmox_containers_backup` accept `cluster`, `vmid`, optional `node`,
`storage`, `mode`, and `compress` arguments. They start an asynchronous backup
and return its Proxmox task identifier; task creation does not mean that the
backup has already completed successfully.

## Installation

For a local installation with Homebrew, run:

```sh
brew install sergelogvinov/tap/proxmox-mcp
```

You do not need a local installation when you use an MCP server hosted on
another machine.

## Configure Proxmox clusters

Create a YAML file that contains one or more Proxmox clusters:

```yaml
clusters:
  - region: homelab
    url: https://pve.example.com:8006/api2/json
    token_id: mcp@pve!reader
    token_secret_file: /absolute/path/to/token-secret
    ca_file: /absolute/path/to/proxmox-ca.crt
```

Each cluster supports these fields:

| Field | Required | Description |
| --- | --- | --- |
| `region` | Yes | Unique name used to select the cluster in tool calls. |
| `url` | Yes | Full Proxmox API URL, including `/api2/json`. |
| `token_id` | For token login | Proxmox API token ID, for example `mcp@pve!reader`. |
| `token_id_file` | For token login | File that contains the token ID. Use this instead of `token_id`. |
| `token_secret` | For token login | Proxmox API token secret. |
| `token_secret_file` | For token login | File that contains the token secret. Use this instead of `token_secret`. |
| `username` | For password login | Proxmox user name. Must be used with `password`. |
| `password` | For password login | Proxmox password. Must be used with `username`. |
| `ca_file` | No | Path to the CA certificate used to check the Proxmox TLS certificate. |
| `insecure` | No | Skip TLS certificate checks. Use only for local testing. Default: `false`. |

Use either token login or password login for a cluster, not both. Files are
recommended for secrets because they keep secret values out of the YAML file.
The Proxmox account should have only the permissions required by the tools.

Set the configuration path with `--config` or `CONFIG_FILE`.

## Configure an MCP client

### Local stdio server

For clients that use a JSON MCP configuration, add an entry like this:

```json
{
  "mcpServers": {
    "proxmox": {
      "command": "proxmox-mcp",
      "args": ["mcp"],
      "env": {
        "CONFIG_FILE": "/absolute/path/to/cluster-config.yaml"
      }
    }
  }
}
```

### Remote HTTP server

Start the streamable HTTP server with:

```sh
proxmox-mcp server \
  --config /absolute/path/to/cluster-config.yaml \
  --port 8080
```

The MCP endpoint is `http://host:8080/mcp`

For a remote client, use an HTTPS URL that ends with `/mcp`:

```json
{
  "mcpServers": {
    "proxmox": {
      "type": "http",
      "url": "https://proxmox-mcp.example.com/mcp",
      "headers": {
        "Authorization": "Bearer PVEAPIToken=mcp@pve!reader=TOKEN_SECRET"
      }
    }
  }
}
```

The server does not provide user authentication for the HTTP endpoint.
It relies on Proxmox authentication and API tokens for access control.
Provide the token in mcp header like:

```text
Authorization: PVEAPIToken=mcp@pve!reader=TOKEN_SECRET
```

When this header is present, its token is used for the tool call instead of
the credentials from the cluster configuration. The configured cluster must
still include `region` and `url`.

Restart or reload the MCP client after you save its configuration.

## Running

### Common flags

The `mcp`, `server`, and `tools` commands use the following flags. Each flag
can also be set with an environment variable. A command-line flag has higher
priority than an environment variable.

| Flag | Environment variable | Default | Description |
| --- | --- | --- | --- |
| `--config <path>` | `CONFIG_FILE` | None | Path to the required clusters YAML configuration file. |
| `--extensions <list>` | `EXTENSIONS` | `all` | Comma-separated list of extensions to enable, or `all`. |
| `--allow-destructive` | `ALLOW_DESTRUCTIVE` | `false` | Allow destructive tools when they are available. Current core tools are read-only. |
| `--log-level <level>` | `LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, or `error`. |
| `--log-format <format>` | `LOG_FORMAT` | `text` | Log output format: `text` or `json`. |

The `server` command also accepts `--port` or `PORT`. The default port is
`8080`. The `tools` command accepts `--output` (`-o`) with `text`, `json`, or
`yaml`. Its default is `text`.

For example, run the stdio server with JSON logs:

```sh
proxmox-mcp mcp \
  --config /absolute/path/to/cluster-config.yaml \
  --log-format json
```

## Test the configuration

List all available tools:

```sh
export CONFIG_FILE="/absolute/path/to/cluster-config.yaml"
proxmox-mcp tools
```

Call a tool directly:

```sh
proxmox-mcp tools proxmox_clusters_list
proxmox-mcp tools proxmox_nodes_list cluster=homelab
proxmox-mcp tools -o json proxmox_events_list cluster=homelab limit=5
```

## License

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

---

`Proxmox®` is a registered trademark of [Proxmox Server Solutions GmbH](https://www.proxmox.com/en/about/company).
