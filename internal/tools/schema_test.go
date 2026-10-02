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
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/sergelogvinov/proxmox-mcp/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type schemaFixture struct {
	Tags []string `json:"tags,omitempty" jsonschema:"Tags"`
	Port *int     `json:"port,omitempty" jsonschema:"Port"`
}

func fixtureSchema(t *testing.T, keepNull bool) string {
	t.Helper()

	s, err := jsonschema.For[schemaFixture](nil)
	require.NoError(t, err)

	tools.SingleTypes(s, keepNull)

	data, err := json.Marshal(s.Properties)
	require.NoError(t, err)

	return string(data)
}

func TestSingleTypesOutput(t *testing.T) {
	assert.JSONEq(t, `{
		"tags": {"description": "Tags", "anyOf": [{"type": "null"}, {"type": "array", "items": {"type": "string"}}]},
		"port": {"description": "Port", "anyOf": [{"type": "null"}, {"type": "integer"}]}
	}`, fixtureSchema(t, true))
}

func TestSingleTypesInput(t *testing.T) {
	assert.JSONEq(t, `{
		"tags": {"description": "Tags", "type": "array", "items": {"type": "string"}},
		"port": {"description": "Port", "type": "integer"}
	}`, fixtureSchema(t, false), "an input field is optional by leaving it out, so it has no null branch")
}
