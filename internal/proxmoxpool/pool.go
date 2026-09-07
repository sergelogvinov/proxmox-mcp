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

// Package proxmoxpool provides a pool of Proxmox clients for multiple clusters.
package proxmoxpool

import (
	"fmt"
	"os"
	"strings"

	"github.com/sergelogvinov/go-proxmox-rest"
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
	clients map[string]*proxmox.Client
}

// NewProxmoxPool creates a new Proxmox cluster client pool.
func NewProxmoxPool(config []*ProxmoxCluster, options ...proxmox.Option) (*ProxmoxPool, error) {
	if len(config) == 0 {
		return nil, ErrClusterNotFound
	}

	clients := make(map[string]*proxmox.Client, len(config))

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

		client := proxmox.ClientConfig{
			CACert:      cfg.CAFile,
			Insecure:    cfg.Insecure,
			Username:    cfg.Username,
			Password:    cfg.Password,
			Token:       cfg.TokenID,
			TokenSecret: cfg.TokenSecret,
			UserAgent:   "ProxmoxMCP/1.0",
		}

		options = append(options, proxmox.WithURL(cfg.URL))

		pxClient, err := proxmox.New(client, options...)
		if err != nil {
			return nil, err
		}

		clients[cfg.Region] = pxClient
	}

	return &ProxmoxPool{
		clients: clients,
	}, nil
}

// GetClusters returns supported clusters.
func (c *ProxmoxPool) GetClusters() []string {
	clusters := make([]string, 0, len(c.clients))

	for cluster := range c.clients {
		clusters = append(clusters, cluster)
	}

	return clusters
}

// GetProxmoxCluster returns a Proxmox cluster client in a given cluster.
func (c *ProxmoxPool) GetProxmoxCluster(cluster string) (*proxmox.Client, error) {
	if c.clients[cluster] != nil {
		return c.clients[cluster], nil
	}
	return nil, ErrClusterNotFound
}

// GetProxmoxClusterWithToken returns a Proxmox cluster client for the given
// cluster. When authHeader is non-empty, a client is built that passes the
// header through to the Proxmox API verbatim; otherwise the pre-configured
// cloud-config client is returned.
func (c *ProxmoxPool) GetProxmoxClusterWithToken(cluster, authToken string) (*proxmox.Client, error) {
	client, err := c.GetProxmoxCluster(cluster)
	if err != nil {
		return nil, err
	}

	if authToken == "" {
		return client, nil
	}

	// Clean the existing authentication fields to ensure only the token is used.
	cfg := client.ToRESTConfig()
	cfg.Username = ""
	cfg.Password = ""
	cfg.Token, cfg.TokenSecret, _ = strings.Cut(authToken, "=")

	return proxmox.New(cfg)
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
