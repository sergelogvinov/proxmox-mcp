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
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/sergelogvinov/proxmox-mcp/internal/config"
	"github.com/sergelogvinov/proxmox-mcp/internal/logger"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
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
		Use:           "proxmox-mcp",
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
