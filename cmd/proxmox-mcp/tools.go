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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sergelogvinov/proxmox-mcp/internal/config"
	"github.com/sergelogvinov/proxmox-mcp/internal/logger"
	"github.com/sergelogvinov/proxmox-mcp/internal/proxmoxpool"
	"github.com/sergelogvinov/proxmox-mcp/internal/server"
	"github.com/sergelogvinov/proxmox-mcp/internal/tools"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

// OutputFormat specifies the output format for tool results.
type OutputFormat string

const (
	OutputText OutputFormat = "text"
	OutputJSON OutputFormat = "json"
	OutputYAML OutputFormat = "yaml"
)

// newToolsCmd creates the `tools` subcommand that lets users invoke MCP tools
// directly from the CLI without going through an MCP client.
func newToolsCmd(flags *Flags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tools [tool-name] [key=value ...]",
		Short: "Invoke MCP tools from the CLI",
		Long:  "Invoke MCP tools from the CLI. If no tool-name is provided, lists available tools.",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTools(cmd.Context(), flags, args)
		},
	}

	flags.AddToolFlags(cmd.Flags())

	return cmd
}

// runTools executes the tools command.
func runTools(ctx context.Context, f *Flags, args []string) error {
	switch OutputFormat(f.Output) {
	case OutputText, OutputJSON, OutputYAML:
	default:
		return fmt.Errorf("invalid output format %q: must be one of text, json, yaml", f.Output)
	}

	cfg, err := f.Config()
	if err != nil {
		return err
	}

	log, err := newLogger(cfg)
	if err != nil {
		return err
	}

	ctx = logger.Inject(ctx, log)

	clusters, err := config.ReadCloudConfigFromFile(f.ConfigFile)
	if err != nil {
		return err
	}

	pool, err := proxmoxpool.NewProxmoxPool(clusters.Clusters)
	if err != nil {
		return err
	}

	srv := mcp.NewServer(&mcp.Implementation{
		Name:    "proxmox-mcp",
		Version: version,
	}, nil)

	tools.NewProxmoxTools(pool).RegisterTools(srv)

	if len(args) == 0 {
		return listTools(ctx, srv, OutputFormat(f.Output))
	}

	toolName := args[0]

	toolArgs, err := parseArguments(args[1:])
	if err != nil {
		return err
	}

	return callTool(ctx, srv, toolName, toolArgs, OutputFormat(f.Output))
}

// listTools prints the tools registered on the server.
func listTools(ctx context.Context, srv *mcp.Server, format OutputFormat) error {
	toolList, err := server.ListTools(ctx, srv)
	if err != nil {
		return err
	}

	if format == OutputText {
		for _, tool := range toolList {
			fmt.Printf("%s\t%s\n", tool.Name, tool.Description)
		}

		return nil
	}

	return printEncoded(toolList, format)
}

// callTool invokes a single tool and prints its result.
func callTool(ctx context.Context, svc *mcp.Server, name string, args map[string]any, format OutputFormat) error {
	result, err := server.CallTool(ctx, svc, name, args)
	if err != nil {
		return err
	}

	if result.IsError {
		return fmt.Errorf("tool %s failed: %s", name, strings.Join(result.Content, " "))
	}

	if format == OutputText {
		for _, content := range result.Content {
			fmt.Println(content)
		}

		return nil
	}

	value := result.Structured
	if value == nil {
		value = result.Content
	}

	return printEncoded(value, format)
}

// printEncoded prints a value as JSON or YAML.
func printEncoded(value any, format OutputFormat) error {
	var (
		data []byte
		err  error
	)

	switch format { //nolint:exhaustive
	case OutputJSON:
		data, err = json.MarshalIndent(value, "", "  ")
	case OutputYAML:
		data, err = yaml.Marshal(value)
	default:
		return fmt.Errorf("unsupported output format %q", format)
	}

	if err != nil {
		return err
	}

	fmt.Println(string(data))

	return nil
}

// parseArguments parses key=value pairs from command line arguments.
// Values may contain '='; we split on the first '=' only.
// A bare token without '=' is an error (suggest quoting).
// key= means empty string value.
func parseArguments(args []string) (map[string]any, error) {
	result := make(map[string]any)

	for _, arg := range args {
		// Split on first '='
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("argument %q must be in key=value format", arg)
		}

		key := parts[0]
		value := parts[1]

		if key == "" {
			return nil, fmt.Errorf("argument %q has empty key", arg)
		}

		result[key] = value
	}

	return result, nil
}
