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

// Package formatter renders Go struct values as human-readable markdown text,
// used as a fallback for agents that cannot consume JSON output.
package formatter

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// ToText converts a struct value into a markdown string describing its
// data, following docs/fallbackText.md.
//
// Only fields carrying a `jsonschema` tag are rendered, and the tag value is
// used as the field label. Embedded structs are flattened. Flat fields
// (scalars, maps, slices of scalars) are rendered first as "Label: value"
// lines, followed by the block fields: nested structs become heading blocks
// and slices of structs become markdown tables. Maps become sorted key=value
// lists. Fields tagged `,omitempty` are omitted when zero-valued, and empty
// slices/maps are always omitted.
func ToText(v any) string {
	sv := indirect(reflect.ValueOf(v))
	if !sv.IsValid() || sv.Kind() != reflect.Struct {
		return ""
	}

	var sb strings.Builder

	renderStruct(&sb, sv, 0)

	return strings.TrimRight(sb.String(), "\n")
}

// renderStruct writes the printable fields of sv: flat fields first as
// "Label: value" lines, then block fields (nested structs and tables) as
// headings. depth is the nesting level of sv (0 for the top-level struct) and
// drives heading levels of nested blocks.
func renderStruct(sb *strings.Builder, sv reflect.Value, depth int) {
	flat, blocks := collectItems(sv)

	for _, it := range flat {
		writeField(sb, it.label, formatInline(it.fv))
	}

	for _, it := range blocks {
		renderBlock(sb, it.label, it.fv, depth)
	}
}

// item is one printable field of a struct, classified by rendering style.
type item struct {
	label string
	fv    reflect.Value
}

// collectItems returns the printable fields of sv split into flat items
// (scalars, maps, slices of scalars — rendered as "Label: value" lines) and
// block items (nested structs and slices of structs — rendered as headings).
// Declaration order is preserved within each group. Embedded structs without
// their own label contribute to both groups, mirroring how encoding/json
// promotes embedded fields.
func collectItems(sv reflect.Value) (flat, blocks []item) {
	t := sv.Type()

	for i := range t.NumField() {
		f := t.Field(i)
		if f.PkgPath != "" {
			continue // unexported
		}

		label := f.Tag.Get("jsonschema")
		fv := indirect(sv.Field(i))

		// Embedded structs without their own label are flattened into the
		// parent, mirroring how encoding/json promotes embedded fields.
		if f.Anonymous && label == "" && fv.Kind() == reflect.Struct {
			subFlat, subBlocks := collectItems(fv)
			flat = append(flat, subFlat...)
			blocks = append(blocks, subBlocks...)

			continue
		}

		if label == "" || !fv.IsValid() {
			continue
		}

		if omitted(f.Tag.Get("json"), fv) {
			continue
		}

		if isBlock(fv) {
			blocks = append(blocks, item{label: label, fv: fv})
		} else {
			flat = append(flat, item{label: label, fv: fv})
		}
	}

	return flat, blocks
}

// isBlock reports whether fv renders as a heading block: a nested struct or a
// slice/array of structs. Everything else renders inline.
func isBlock(fv reflect.Value) bool {
	//nolint:exhaustive
	switch fv.Kind() {
	case reflect.Struct:
		return true
	case reflect.Slice, reflect.Array:
		et := fv.Type().Elem()
		for et.Kind() == reflect.Pointer {
			et = et.Elem()
		}

		return et.Kind() == reflect.Struct
	}

	return false
}

// renderBlock writes a block item: a nested struct as a heading followed by
// its fields, or a slice of structs as a markdown table under a heading.
func renderBlock(sb *strings.Builder, label string, fv reflect.Value, depth int) {
	if fv.Kind() == reflect.Struct {
		writeHeading(sb, depth, label)
		renderStruct(sb, fv, depth+1)

		return
	}

	renderTable(sb, label, fv, depth)
}

// renderTable writes a slice of structs as a markdown table under a heading.
func renderTable(sb *strings.Builder, label string, fv reflect.Value, depth int) {
	et := fv.Type().Elem()
	for et.Kind() == reflect.Pointer {
		et = et.Elem()
	}

	cols := tableColumns(et)
	if len(cols) == 0 {
		return
	}

	writeHeading(sb, depth, label)

	headers := make([]string, len(cols))
	for i, c := range cols {
		headers[i] = c.label
	}

	seps := make([]string, len(cols))
	for i := range seps {
		seps[i] = "---"
	}

	writeLine(sb, "| "+strings.Join(headers, " | ")+" |")
	writeLine(sb, "| "+strings.Join(seps, " | ")+" |")

	for i := range fv.Len() {
		ev := indirect(fv.Index(i))
		if !ev.IsValid() || ev.Kind() != reflect.Struct {
			continue
		}

		cells := make([]string, len(cols))
		for j, c := range cols {
			cells[j] = formatCell(fieldByIndexPath(ev, c.index))
		}

		writeLine(sb, "| "+strings.Join(cells, " | ")+" |")
	}

	// Blank line so content following the table is not swallowed by it.
	sb.WriteString("\n")
}

// fieldByIndexPath traverses a multi-level field index, dereferencing
// intermediate pointers. Unlike reflect.Value.FieldByIndex it never panics:
// traversal through a nil embedded pointer yields the zero Value, which
// callers render as an empty cell.
func fieldByIndexPath(v reflect.Value, index []int) reflect.Value {
	for _, i := range index {
		if v.Kind() == reflect.Pointer {
			if v.IsNil() {
				return reflect.Value{}
			}

			v = v.Elem()
		}

		v = v.Field(i)
	}

	return v
}

// column is one printable column of a markdown table.
type column struct {
	label string
	index []int
}

// tableColumns returns the printable fields of a table element struct. Fields
// with a `jsonschema` tag are preferred; if the struct has none at all (e.g.
// OwnerReference), fields fall back to their JSON names so the table is still
// renderable.
func tableColumns(t reflect.Type) []column {
	if cols := collectColumns(t, true); len(cols) > 0 {
		return cols
	}

	return collectColumns(t, false)
}

func collectColumns(t reflect.Type, strict bool) []column {
	var cols []column

	for i := range t.NumField() {
		f := t.Field(i)
		if f.PkgPath != "" {
			continue // unexported
		}

		if f.Anonymous {
			ft := f.Type
			if ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}

			if ft.Kind() == reflect.Struct {
				for _, c := range collectColumns(ft, strict) {
					cols = append(cols, column{label: c.label, index: append([]int{i}, c.index...)})
				}
			}

			continue
		}

		label := f.Tag.Get("jsonschema")
		if label == "" {
			if strict {
				continue
			}

			label = jsonName(f)
		}

		if label == "" {
			continue
		}

		cols = append(cols, column{label: label, index: f.Index})
	}

	return cols
}

// jsonName returns the JSON field name of f, falling back to the Go name.
func jsonName(f reflect.StructField) string {
	name := strings.Split(f.Tag.Get("json"), ",")[0]
	if name == "" || name == "-" {
		return f.Name
	}

	return name
}

// omitted reports whether a field value must be skipped: empty slices/maps are
// always omitted, and `,omitempty` fields are omitted when zero-valued.
func omitted(jsonTag string, fv reflect.Value) bool {
	//nolint:exhaustive
	switch fv.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map:
		if fv.Len() == 0 {
			return true
		}
	}

	if !strings.Contains(jsonTag, ",omitempty") {
		return false
	}

	return fv.IsZero()
}

// formatCell renders a table cell with the inline rules.
func formatCell(v reflect.Value) string {
	v = indirect(v)
	if !v.IsValid() {
		return ""
	}

	return formatInline(v)
}

// formatInline renders a value on a single line: scalars as-is, slices
// comma-joined, maps as key=value pairs, structs as "Label: value" pairs.
func formatInline(v reflect.Value) string {
	//nolint:exhaustive
	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		parts := make([]string, 0, v.Len())

		for i := range v.Len() {
			if e := indirect(v.Index(i)); e.IsValid() {
				parts = append(parts, formatInline(e))
			}
		}

		return strings.Join(parts, ", ")
	case reflect.Map:
		return formatMap(v)
	case reflect.Struct:
		var sb strings.Builder

		t := v.Type()

		for i := range t.NumField() {
			f := t.Field(i)
			if f.PkgPath != "" {
				continue // unexported
			}

			label := f.Tag.Get("jsonschema")
			fv := indirect(v.Field(i))
			if label == "" || !fv.IsValid() || omitted(f.Tag.Get("json"), fv) {
				continue
			}

			if sb.Len() > 0 {
				sb.WriteString("; ")
			}

			sb.WriteString(label)
			sb.WriteString(": ")
			sb.WriteString(formatInline(fv))
		}

		return sb.String()
	default:
		return formatScalar(v)
	}
}

// formatMap renders a map as sorted, comma-separated key=value pairs.
func formatMap(v reflect.Value) string {
	keys := make([]string, 0, v.Len())
	byKey := make(map[string]reflect.Value, v.Len())

	iter := v.MapRange()
	for iter.Next() {
		key := indirect(iter.Key())
		k := "<nil>"
		if key.IsValid() {
			k = formatScalar(key)
		}

		keys = append(keys, k)
		byKey[k] = iter.Value()
	}

	slices.Sort(keys)

	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, k+"="+formatCell(byKey[k]))
	}

	return strings.Join(pairs, ", ")
}

// formatScalar renders a scalar value as a string.
func formatScalar(v reflect.Value) string {
	//nolint:exhaustive
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64)
	default:
		return fmt.Sprint(v.Interface())
	}
}

// indirect dereferences pointers and interfaces, returning the zero Value for
// nil references.
func indirect(v reflect.Value) reflect.Value {
	for v.IsValid() && (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) {
		if v.IsNil() {
			return reflect.Value{}
		}

		v = v.Elem()
	}

	return v
}

// writeHeading writes a markdown heading, introducing a blank line first. A
// field of the top-level struct (depth 0) renders as "###", each deeper
// nesting level adds one "#".
func writeHeading(sb *strings.Builder, depth int, label string) {
	if s := sb.String(); s != "" && !strings.HasSuffix(s, "\n\n") {
		sb.WriteString("\n")
	}

	sb.WriteString(strings.Repeat("#", depth+3))
	sb.WriteString(" ")
	writeLine(sb, label)
	sb.WriteString("\n")
}

func writeLine(sb *strings.Builder, s string) {
	sb.WriteString(s)
	sb.WriteString("\n")
}

// writeField writes a "Label: value" line, trimming the dangling space left
// by empty values.
func writeField(sb *strings.Builder, label, value string) {
	writeLine(sb, strings.TrimRight(label+": "+value, " "))
}
