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

// Package proxmoxpool provides a pool of Telmate/proxmox-api-go/proxmox clients
package proxmoxpool

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"

	proxmox "github.com/luthermonson/go-proxmox"
	goproxmox "github.com/sergelogvinov/go-proxmox"
)

// ProxmoxCluster defines a Proxmox cluster configuration.
type ProxmoxCluster struct {
	URL             string `yaml:"url"`
	CAFile          string `yaml:"ca_file,omitempty"`
	Insecure        bool   `yaml:"insecure,omitempty"`
	TokenID         string `yaml:"token_id,omitempty"`
	TokenIDFile     string `yaml:"token_id_file,omitempty"`
	TokenSecret     string `yaml:"token_secret,omitempty"`
	TokenSecretFile string `yaml:"token_secret_file,omitempty"`
	Username        string `yaml:"username,omitempty"`
	Password        string `yaml:"password,omitempty"`
	Region          string `yaml:"region,omitempty"`
}

// ProxmoxPool is a Proxmox client pool of proxmox clusters.
type ProxmoxPool struct {
	clients map[string]*goproxmox.APIClient
	configs map[string]*ProxmoxCluster
}

// NewProxmoxPool creates a new Proxmox cluster client pool.
func NewProxmoxPool(config []*ProxmoxCluster, options ...proxmox.Option) (*ProxmoxPool, error) {
	if len(config) == 0 {
		return nil, ErrClustersNotFound
	}

	clients := make(map[string]*goproxmox.APIClient, len(config))
	configs := make(map[string]*ProxmoxCluster, len(config))

	for _, cfg := range config {
		if cfg.TokenID == "" && cfg.TokenIDFile != "" {
			var err error

			cfg.TokenID, err = readValueFromFile(cfg.TokenIDFile)
			if err != nil {
				return nil, err
			}
		}

		if cfg.TokenSecret == "" && cfg.TokenSecretFile != "" {
			var err error

			cfg.TokenSecret, err = readValueFromFile(cfg.TokenSecretFile)
			if err != nil {
				return nil, err
			}
		}

		pxClient, err := newProxmoxClient(cfg, "", options...)
		if err != nil {
			return nil, err
		}

		clients[cfg.Region] = pxClient
		configs[cfg.Region] = cfg
	}

	return &ProxmoxPool{
		clients: clients,
		configs: configs,
	}, nil
}

// newProxmoxClient builds a Proxmox API client for the given cluster config.
// When authHeader is non-empty it is passed through verbatim as the request
// Authorization header, taking precedence over cloud-config credentials.
func newProxmoxClient(cfg *ProxmoxCluster, authHeader string, options ...proxmox.Option) (*goproxmox.APIClient, error) {
	opts := []proxmox.Option{proxmox.WithUserAgent("ProxmoxMCP/1.0")}
	opts = append(opts, options...)

	transport := http.DefaultTransport

	if cfg.Insecure {
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cfg.Insecure,
				MinVersion:         tls.VersionTLS12,
			},
		}
	}

	if authHeader != "" {
		transport = &authHeaderTransport{base: transport, header: authHeader}
	}

	opts = append(opts, proxmox.WithHTTPClient(&http.Client{Transport: transport}))

	// Only configure cloud-config credentials when no per-request auth header
	// was provided. This lets clusters be configured without credentials and
	// still be usable when the client supplies its own token.
	if authHeader == "" {
		switch {
		case cfg.Username != "" && cfg.Password != "":
			opts = append(opts, proxmox.WithCredentials(&proxmox.Credentials{
				Username: cfg.Username,
				Password: cfg.Password,
			}))
		case cfg.TokenID != "" && cfg.TokenSecret != "":
			opts = append(opts, proxmox.WithAPIToken(cfg.TokenID, cfg.TokenSecret))
		}
	}

	return goproxmox.NewAPIClient(cfg.URL, opts...)
}

// authHeaderTransport injects a fixed Authorization header into every request
// before delegating to the underlying RoundTripper. It is used to pass a
// client-supplied Proxmox API token through to the Proxmox API unchanged.
type authHeaderTransport struct {
	base   http.RoundTripper
	header string
}

// RoundTrip implements http.RoundTripper.
func (t *authHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", t.header)

	return t.base.RoundTrip(req)
}

// GetRegions returns supported regions.
func (c *ProxmoxPool) GetRegions() []string {
	regions := make([]string, 0, len(c.clients))

	for region := range c.clients {
		regions = append(regions, region)
	}

	return regions
}

// GetProxmoxCluster returns a Proxmox cluster client in a given region.
func (c *ProxmoxPool) GetProxmoxCluster(region string) (*goproxmox.APIClient, error) {
	if c.clients[region] != nil {
		return c.clients[region], nil
	}

	return nil, ErrRegionNotFound
}

// GetProxmoxClusterWithToken returns a Proxmox cluster client for the given
// region. When authHeader is non-empty, a client is built that passes the
// header through to the Proxmox API verbatim; otherwise the pre-configured
// cloud-config client is returned.
func (c *ProxmoxPool) GetProxmoxClusterWithToken(region, authHeader string) (*goproxmox.APIClient, error) {
	if authHeader == "" {
		return c.GetProxmoxCluster(region)
	}

	cfg, ok := c.configs[region]
	if !ok {
		return nil, ErrRegionNotFound
	}

	return newProxmoxClient(cfg, authHeader)
}

// GetClusterVersion returns the Proxmox version of the cluster in the given region.
// The optional authHeader is passed through to the Proxmox API when non-empty.
func (c *ProxmoxPool) GetClusterVersion(ctx context.Context, region, authHeader string) (string, error) {
	client, err := c.GetProxmoxClusterWithToken(region, authHeader)
	if err != nil {
		return "", err
	}

	v, err := client.Version(ctx)
	if err != nil {
		return "", err
	}

	return v.Version, nil
}

func readValueFromFile(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file '%s': %w", path, err)
	}

	return strings.TrimSpace(string(content)), nil
}
