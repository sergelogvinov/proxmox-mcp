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

package tools

import (
	"fmt"
	"reflect"
	"slices"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// nullType is the JSON Schema type of null.
const nullType = "null"

// addTool registers a tool like mcp.AddTool, but first infers any schema the
// tool does not set and rewrites both schemas with singleTypes. Output keeps
// a null branch, because a nil slice is serialized as null. Input does not:
// a client leaves an optional field out instead of sending null.
//
// Like mcp.AddTool, an `any` input is an empty object, and an `any` output
// has no output schema.
func addTool[In, Out any](srv *mcp.Server, tool *mcp.Tool, h mcp.ToolHandlerFor[In, Out]) {
	if tool.InputSchema == nil {
		if reflect.TypeFor[In]() == reflect.TypeFor[any]() {
			tool.InputSchema = &jsonschema.Schema{Type: "object"}
		} else {
			tool.InputSchema = inferSchema[In](tool.Name, "input")
		}
	}

	if tool.OutputSchema == nil && reflect.TypeFor[Out]() != reflect.TypeFor[any]() {
		tool.OutputSchema = inferSchema[Out](tool.Name, "output")
	}

	if schema, ok := tool.InputSchema.(*jsonschema.Schema); ok {
		singleTypes(schema, false)
	}

	if schema, ok := tool.OutputSchema.(*jsonschema.Schema); ok {
		singleTypes(schema, true)
	}

	mcp.AddTool(srv, tool, h)
}

// inferSchema builds the schema of T the way mcp.AddTool does in go-sdk
// v1.8.0 (setSchema in mcp/server.go): a pointer type is unwrapped, then the
// schema is inferred with the default options. Check it still matches when
// upgrading go-sdk. TestToolSchemas covers the result, not the match.
func inferSchema[T any](tool, kind string) *jsonschema.Schema {
	rt := reflect.TypeFor[T]()
	if rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}

	schema, err := jsonschema.ForType(rt, &jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("%s: %s schema: %v", tool, kind, err))
	}

	return schema
}

// singleTypes rewrites, in s and every schema below it, a `type` list such as
// ["null","array"] (what jsonschema-go infers for slices and pointers) into
// anyOf branches with one `type` each. Clients that map tool schemas onto a
// single-type dialect, such as Gemini function declarations, reject the
// list form or drop the constraint.
//
// The annotations stay on the outer schema. Each non-null branch keeps the
// type-specific keywords, such as items or properties. Without keepNull the
// null type is dropped, and a single remaining type is set directly, with no
// anyOf.
func singleTypes(s *jsonschema.Schema, keepNull bool) {
	if s == nil {
		return
	}

	forEachChild(s, func(c *jsonschema.Schema) { singleTypes(c, keepNull) })

	if len(s.Types) == 0 {
		return
	}

	if !keepNull {
		s.Types = slices.DeleteFunc(slices.Clone(s.Types), func(t string) bool { return t == nullType })

		switch len(s.Types) {
		case 0:
			s.Types = nil

			return
		case 1:
			s.Type, s.Types = s.Types[0], nil

			return
		}
	}

	outer := &jsonschema.Schema{
		Title:       s.Title,
		Description: s.Description,
		Default:     s.Default,
		Examples:    s.Examples,
		Deprecated:  s.Deprecated,
		ReadOnly:    s.ReadOnly,
		WriteOnly:   s.WriteOnly,
		Comment:     s.Comment,
	}

	for _, typ := range s.Types {
		if typ == nullType {
			outer.AnyOf = append(outer.AnyOf, &jsonschema.Schema{Type: nullType})

			continue
		}

		branch := *s
		branch.Types = nil
		branch.Type = typ
		branch.Title, branch.Description, branch.Comment = "", "", ""
		branch.Default, branch.Examples = nil, nil
		branch.Deprecated, branch.ReadOnly, branch.WriteOnly = false, false, false

		outer.AnyOf = append(outer.AnyOf, &branch)
	}

	*s = *outer
}

// forEachChild calls f on each schema directly below s: every *Schema,
// []*Schema and map[string]*Schema field, such as properties, items or $defs.
func forEachChild(s *jsonschema.Schema, f func(*jsonschema.Schema)) {
	for _, v := range reflect.ValueOf(s).Elem().Fields() {
		switch field := v.Interface().(type) {
		case *jsonschema.Schema:
			f(field)
		case []*jsonschema.Schema:
			for _, c := range field {
				f(c)
			}
		case map[string]*jsonschema.Schema:
			for _, c := range field {
				f(c)
			}
		}
	}
}
