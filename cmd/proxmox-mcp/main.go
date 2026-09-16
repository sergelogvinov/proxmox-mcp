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

// Package main implements the proxmox-mcp command-line tool.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sergelogvinov/proxmox-mcp/internal/config"
	"github.com/sergelogvinov/proxmox-mcp/internal/logger"
	"github.com/spf13/cobra"
)

var (
	version     = "dev"
	commit      = "none"
	bin         = "proxmox-mcp"
	description = "Proxmox MCP server"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	rootCmd := newRootCmd()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		errorString := err.Error()
		fmt.Fprintf(os.Stderr, "Error: %s\n\n", errorString)

		if strings.Contains(errorString, "arg(s)") {
			fmt.Fprintln(os.Stderr, rootCmd.UsageString())
		}

		cancel()
		os.Exit(1) //nolint:gocritic
	}
}

func newLogger(cfg *config.Config) (*slog.Logger, error) {
	return logger.New(logger.Options{
		Level:  logger.Level(cfg.LogLevel),
		Format: logger.Format(cfg.LogFormat),
	})
}

func newRootCmd() *cobra.Command {
	flags := DefaultFlags()

	rootCmd := &cobra.Command{
		Use:           bin,
		Short:         "Proxmox MCP Server - Proxmox tooling",
		Long:          "Proxmox MCP Server provides Proxmox tooling via MCP protocol",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Global (persistent) flags, inherited by every subcommand.
	flags.AddPersistentFlags(rootCmd.PersistentFlags())

	rootCmd.AddCommand(
		newMCPCmd(flags),
		newServerCmd(flags),
		newToolsCmd(flags),
		newVersionCmd(),
	)

	return rootCmd
}

func loggingMiddleware(next mcp.MethodHandler) mcp.MethodHandler {
	return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
		logger := logger.FromContext(ctx)

		if ctr, ok := req.(*mcp.CallToolRequest); ok {
			params := make([]any, 0)

			params = append(params,
				"method", "tools/call",
				"name", ctr.Params.Name,
			)

			if ctr.Extra != nil {
				authHeader := strings.TrimSpace(ctr.Extra.Header.Get("Authorization"))
				if authHeader != "" {
					if rest, ok := strings.CutPrefix(authHeader, "PVEAPIToken="); ok {
						tokenID, _, ok := strings.Cut(rest, "=")
						if ok && tokenID != "" {
							params = append(params, "PVEAPIToken", tokenID)
						}
					}
				}
			}

			if len(ctr.Params.Arguments) > 0 {
				var raw map[string]any
				if err := json.Unmarshal(ctr.Params.Arguments, &raw); err != nil {
					raw = nil
				}

				for k, v := range raw {
					params = append(params, k, fmt.Sprintf("%v", v))
				}
			}

			logger.Info("calling", params...)
		}

		return next(ctx, method, req)
	}
}
