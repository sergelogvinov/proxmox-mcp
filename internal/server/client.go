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

package server

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolInfo describes a tool registered on the MCP server.
type ToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// ToolResult is the SDK-free result of a tool invocation.
type ToolResult struct {
	// Content holds the unstructured text blocks of the result.
	Content []string `json:"content"`
	// Structured holds the structured content of the result, if any.
	Structured any `json:"structured,omitempty"`
	// IsError reports whether the tool call ended in an error.
	IsError bool `json:"isError"`
}

// ListTools returns the tools registered on the server, viewed as an MCP client.
func ListTools(ctx context.Context, srv *mcp.Server) ([]ToolInfo, error) {
	clientSession, serverSession, err := connectClient(ctx, srv)
	if err != nil {
		return nil, err
	}
	defer clientSession.Close() //nolint:errcheck
	defer serverSession.Close() //nolint:errcheck

	result, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		return nil, err
	}

	toolList := make([]ToolInfo, 0, len(result.Tools))
	for _, tool := range result.Tools {
		toolList = append(toolList, ToolInfo{
			Name:        tool.Name,
			Description: tool.Description,
		})
	}

	return toolList, nil
}

// CallTool invokes a tool on the server via an in-memory MCP session.
func CallTool(ctx context.Context, srv *mcp.Server, name string, arguments map[string]any) (*ToolResult, error) {
	clientSession, serverSession, err := connectClient(ctx, srv)
	if err != nil {
		return nil, err
	}
	defer clientSession.Close() //nolint:errcheck
	defer serverSession.Close() //nolint:errcheck

	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      name,
		Arguments: arguments,
	})
	if err != nil {
		return nil, err
	}

	toolResult := &ToolResult{
		Structured: result.StructuredContent,
		IsError:    result.IsError,
	}

	for _, content := range result.Content {
		if text, ok := content.(*mcp.TextContent); ok {
			toolResult.Content = append(toolResult.Content, text.Text)
		}
	}

	return toolResult, nil
}

// connectClient connects an in-memory MCP client to the server.
func connectClient(ctx context.Context, srv *mcp.Server) (*mcp.ClientSession, *mcp.ServerSession, error) {
	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	serverSession, err := srv.Connect(ctx, serverTransport, nil)
	if err != nil {
		return nil, nil, err
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "proxmox-mcp-cli", Version: "dev"}, nil)

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		serverSession.Close() //nolint:errcheck

		return nil, nil, err
	}

	return clientSession, serverSession, nil
}
