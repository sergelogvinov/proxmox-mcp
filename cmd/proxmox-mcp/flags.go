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
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/sergelogvinov/proxmox-mcp/internal/config"
	"github.com/spf13/pflag"
)

const (
	flagExtensions       = "extensions"
	flagAllowDestructive = "allow-destructive"
	flagConfigFile       = "config"
	flagLogLevel         = "log-level"
	flagLogFormat        = "log-format"
	flagListenAddress    = "listen-address"

	envExtensions       = "EXTENSIONS"
	envAllowDestructive = "ALLOW_DESTRUCTIVE"
	envConfigFile       = "CONFIG_FILE"
	envLogLevel         = "LOG_LEVEL"
	envLogFormat        = "LOG_FORMAT"
	envListenAddress    = "LISTEN_ADDRESS"
)

const (
	defaultExtensions       = "all"
	defaultAllowDestructive = false
	defaultLogLevel         = "info"
	defaultLogFormat        = "text"
	defaultListenAddress    = "127.0.0.1:8080"
	defaultOutputFormat     = "text"
)

// Flags wraps genericclioptions.ConfigFlags and adds application-specific flags.
type Flags struct {
	Extensions       string
	AllowDestructive bool
	ConfigFile       string
	LogLevel         string
	LogFormat        string
	ListenAddress    string
	Output           string
}

// DefaultFlags returns the default flags for the command,
// populated from environment variables where applicable.
func DefaultFlags() *Flags {
	return &Flags{
		Extensions:       withDefaultEnv(envExtensions, defaultExtensions),
		AllowDestructive: withDefaultEnvBool(envAllowDestructive, defaultAllowDestructive),
		ConfigFile:       withDefaultEnv(envConfigFile, ""),
		LogLevel:         withDefaultEnv(envLogLevel, defaultLogLevel),
		LogFormat:        withDefaultEnv(envLogFormat, defaultLogFormat),
		ListenAddress:    withDefaultEnv(envListenAddress, defaultListenAddress),
		Output:           defaultOutputFormat,
	}
}

// AddPersistentFlags adds the global flags shared by every subcommand.
func (f *Flags) AddPersistentFlags(flags *pflag.FlagSet) {
	// Add application-specific flags
	flags.StringVarP(&f.Extensions, flagExtensions, "", f.Extensions, "comma-separated list of extensions to enable, or 'all' (default: all)")
	flags.BoolVarP(&f.AllowDestructive, flagAllowDestructive, "", f.AllowDestructive, "allow destructive operations (default: false)")
	flags.StringVarP(&f.LogLevel, flagLogLevel, "", f.LogLevel, "log level: debug, info, warn, error (default: info)")
	flags.StringVarP(&f.LogFormat, flagLogFormat, "", f.LogFormat, "log output format: text, json (default: text)")
	flags.StringVarP(&f.ConfigFile, flagConfigFile, "", f.ConfigFile, "path to the clusters YAML configuration file (required)")
}

// AddServerFlags adds the flags for the "server" subcommand.
func (f *Flags) AddServerFlags(flags *pflag.FlagSet) {
	flags.StringVarP(&f.ListenAddress, flagListenAddress, "", f.ListenAddress,
		"http listen address as host:port; use :8080 to accept connections on all IPv4 and IPv6 addresses (default: 127.0.0.1:8080)")
}

// AddToolFlags adds the flags for the "tool" subcommand.
func (f *Flags) AddToolFlags(flags *pflag.FlagSet) {
	flags.StringVarP(&f.Output, "output", "o", defaultOutputFormat, "output format: text, json, yaml")
}

// Config returns the internal config populated from the parsed flags.
func (f *Flags) Config() (*config.Config, error) {
	return &config.Config{
		Extensions:       f.Extensions,
		AllowDestructive: f.AllowDestructive,
		LogLevel:         f.LogLevel,
		LogFormat:        f.LogFormat,
	}, nil
}

// listenAddress validates ListenAddress and returns it. An empty host
// listens on every IPv4 and IPv6 address.
func (f *Flags) listenAddress() (string, error) {
	_, port, err := net.SplitHostPort(f.ListenAddress)
	if err != nil {
		if strings.Count(f.ListenAddress, ":") > 1 && !strings.HasPrefix(f.ListenAddress, "[") {
			return "", fmt.Errorf("invalid listen address %q: write an IPv6 address in brackets, like [::1]:8080", f.ListenAddress)
		}

		return "", fmt.Errorf("invalid listen address %q: must be host:port, like 127.0.0.1:8080 or :8080", f.ListenAddress)
	}

	if n, err := strconv.Atoi(port); err != nil || n < 1 || n > 65535 {
		return "", fmt.Errorf("invalid listen port %q: must be a number between 1 and 65535", port)
	}

	return f.ListenAddress, nil
}

func withDefaultEnv(key string, def string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return def
}

func withDefaultEnvBool(key string, def bool) bool {
	if val, ok := os.LookupEnv(key); ok {
		switch val {
		case "true", "1", "yes", "on":
			return true
		case "false", "0", "no", "off":
			return false
		}
	}
	return def
}
