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
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sergelogvinov/proxmox-mcp/internal/config"
	"github.com/sergelogvinov/proxmox-mcp/internal/proxmoxpool"
	"github.com/sergelogvinov/proxmox-mcp/internal/tools"
	"github.com/spf13/cobra"
)

// shutdownTimeout bounds the graceful shutdown of the HTTP server.
const shutdownTimeout = 1 * time.Second

// newServerCmd creates the `server` subcommand that runs the MCP server over HTTP/SSE.
func newServerCmd(flags *Flags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Run MCP server over HTTP/SSE",
		Long:  "Run MCP server over HTTP/SSE. Streamable HTTP is served on /mcp, legacy SSE on /sse.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runServer(cmd.Context(), flags)
		},
	}

	flags.AddServerFlags(cmd.Flags())

	return cmd
}

// runServer loads the clusters config, creates the Proxmox pool, and serves MCP over HTTP/SSE.
func runServer(ctx context.Context, f *Flags) error {
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

	log.Info("starting MCP server",
		"address", fmt.Sprintf(":%d", cfg.Port),
		"allowDestructive", cfg.AllowDestructive,
		"extensions", cfg.Extensions,
	)

	srv := mcp.NewServer(&mcp.Implementation{
		Name:    "proxmox-mcp",
		Version: version,
	}, &mcp.ServerOptions{
		Logger: log,
	})

	tools.NewProxmoxTools(pool).RegisterTools(srv)

	mux := http.NewServeMux()
	mux.Handle("/mcp", mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return srv }, nil))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok")) //nolint:errcheck
	})

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)

	go func() {
		errCh <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("failed to shutdown server: %w", err)
		}
	}

	return nil
}
