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

package tools_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sergelogvinov/proxmox-mcp/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolSchemas(t *testing.T) {
	srv := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "dev"}, nil)
	tools.NewProxmoxTools(nil, true).RegisterTools(srv)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	serverSession, err := srv.Connect(t.Context(), serverTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { serverSession.Close() }) //nolint:errcheck

	session, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "dev"}, nil).Connect(t.Context(), clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { session.Close() }) //nolint:errcheck

	res, err := session.ListTools(t.Context(), nil)
	require.NoError(t, err)
	require.NotEmpty(t, res.Tools)

	for _, tool := range res.Tools {
		t.Run(tool.Name, func(t *testing.T) {
			assert.NotEmpty(t, tool.Description)
			require.NotNil(t, tool.Annotations, "every tool declares annotations")

			require.NotNil(t, tool.InputSchema, "input schema")

			for kind, schema := range map[string]any{"input": tool.InputSchema, "output": tool.OutputSchema} {
				// A tool with an `any` output has no output schema.
				if schema == nil {
					continue
				}

				data, err := json.Marshal(schema)
				require.NoError(t, err)

				var raw any
				require.NoError(t, json.Unmarshal(data, &raw))
				assert.Empty(t, typeLists(raw, kind), "a nullable field uses anyOf branches, not a `type` list")

				var s jsonschema.Schema
				require.NoError(t, json.Unmarshal(data, &s), "%s schema", kind)
				assert.Equal(t, "object", s.Type, "%s schema", kind)

				_, err = s.Resolve(nil)
				require.NoError(t, err, "%s schema resolves", kind)
			}
		})
	}
}

// typeLists returns the paths of the schemas in node whose `type` is a list,
// such as ["null","array"].
func typeLists(node any, path string) []string {
	var found []string

	switch n := node.(type) {
	case map[string]any:
		if _, ok := n["type"].([]any); ok {
			found = append(found, path)
		}

		for k, v := range n {
			found = append(found, typeLists(v, path+"/"+k)...)
		}
	case []any:
		for i, v := range n {
			found = append(found, typeLists(v, fmt.Sprintf("%s/%d", path, i))...)
		}
	}

	return found
}
