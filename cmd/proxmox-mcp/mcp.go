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

package main

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sergelogvinov/proxmox-mcp/internal/config"
	"github.com/sergelogvinov/proxmox-mcp/internal/logger"
	"github.com/sergelogvinov/proxmox-mcp/internal/proxmoxpool"
	"github.com/sergelogvinov/proxmox-mcp/internal/tools"
	"github.com/spf13/cobra"
)

// newMCPCmd creates the `mcp` subcommand that runs the MCP server over stdio.
func newMCPCmd(flags *Flags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Run MCP server over stdio",
		Long:  "Run MCP server over stdio",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runMCP(cmd.Context(), flags)
		},
	}

	return cmd
}

// runMCP loads the clusters config, creates the Proxmox pool, and serves MCP over stdio.
func runMCP(ctx context.Context, f *Flags) error {
	cfg, err := f.Config()
	if err != nil {
		return err
	}

	log, err := newLogger(cfg)
	if err != nil {
		return err
	}

	clusters, err := config.ReadCloudConfigFromFile(f.ConfigFile)
	if err != nil {
		return err
	}

	pool, err := proxmoxpool.NewProxmoxPool(clusters.Clusters)
	if err != nil {
		return err
	}

	log.Info("server config",
		"allowDestructive", cfg.AllowDestructive,
		"extensions", cfg.Extensions,
	)

	srv := mcp.NewServer(&mcp.Implementation{
		Name:        bin,
		Title:       description,
		Description: description,
		Version:     version,
	}, &mcp.ServerOptions{
		Logger: log,
	})
	srv.AddReceivingMiddleware(loggingMiddleware)

	tools.NewProxmoxTools(pool).RegisterTools(srv)

	return srv.Run(logger.Inject(ctx, log), &mcp.StdioTransport{})
}
