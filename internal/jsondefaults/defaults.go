// Package jsondefaults compares JSON using explicitly declared API defaults.
// It does not validate requests or change the JSON sent to the API.
package jsondefaults

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
)

//go:generate go run ../../cmd/generate-json-defaults -spec ../../docs/spec/peakhour-api-v1.json -out schemas.json

//go:embed schemas.json
var schemaJSON []byte

type bundle struct {
	Schemas map[string]map[string]any `json:"schemas"`
}

func parseBundle(raw []byte) (bundle, error) {
	var b bundle
	d := json.NewDecoder(strings.NewReader(string(raw)))
	d.UseNumber()
	err := d.Decode(&b)
	return b, err
}

var schemas = func() bundle {
	b, err := parseBundle(schemaJSON)
	if err != nil {
		panic("invalid embedded JSON defaults schema: " + err.Error())
	}
	return b
}()

// Equal is symmetric: omission means the schema's explicit default. Undeclared
// keys remain significant. Unsupported schemas conservatively retain the diff.
func Equal(schemaName, left, right string) (bool, error) {
	return schemas.equal(schemaName, left, right)
}

func decode(raw string) (any, error) {
	if !json.Valid([]byte(raw)) {
		return nil, fmt.Errorf("invalid JSON")
	}
	d := json.NewDecoder(strings.NewReader(raw))
	d.UseNumber()
	var value any
	err := d.Decode(&value)
	return value, err
}

func (b bundle) equal(name, left, right string) (bool, error) {
	l, err := decode(left)
	if err != nil {
		return false, err
	}
	r, err := decode(right)
	if err != nil {
		return false, err
	}
	if same(l, r) {
		return true, nil
	}
	schema, ok := b.Schemas[name]
	if !ok {
		return false, nil
	}
	l, err = b.normalize(schema, l, 0)
	if err != nil {
		return false, nil
	}
	r, err = b.normalize(schema, r, 0)
	if err != nil {
		return false, nil
	}
	return same(l, r), nil
}

// resolve accepts references without semantic siblings. The generator retains
// the entire schema, so future unsupported keywords cannot silently disappear.
func (b bundle) resolve(s map[string]any, depth int) (map[string]any, error) {
	if depth > 64 {
		return nil, fmt.Errorf("schema recursion limit")
	}
	for key := range s {
		switch key {
		case "$ref", "type", "properties", "items", "default", "required", "enum", "anyOf", "title", "description", "examples", "deprecated", "format":
		default:
			return nil, fmt.Errorf("unsupported schema keyword %s", key)
		}
	}
	if ref, ok := s["$ref"]; ok {
		for key := range s {
			if key != "$ref" && key != "title" && key != "description" {
				return nil, fmt.Errorf("reference with siblings")
			}
		}
		name, ok := ref.(string)
		if !ok || !strings.HasPrefix(name, "#/components/schemas/") {
			return nil, fmt.Errorf("unsupported reference")
		}
		target, ok := b.Schemas[strings.TrimPrefix(name, "#/components/schemas/")]
		if !ok {
			return nil, fmt.Errorf("missing reference")
		}
		return b.resolve(target, depth+1)
	}
	return s, nil
}

func (b bundle) normalize(schema map[string]any, value any, depth int) (any, error) {
	if depth > 64 {
		return nil, fmt.Errorf("JSON recursion limit")
	}
	s, err := b.resolve(schema, depth)
	if err != nil {
		return nil, err
	}
	if branches, ok := s["anyOf"].([]any); ok {
		for key := range s {
			switch key {
			case "anyOf", "default", "title", "description", "examples", "deprecated":
			default:
				return nil, fmt.Errorf("union with unhandled sibling %s", key)
			}
		}
		// Only unions distinguishable by JSON type are supported. Never guess
		// between object variants whose conditional defaults could differ.
		var selected map[string]any
		for _, raw := range branches {
			branch, ok := raw.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("invalid union")
			}
			branch, err = b.resolve(branch, depth+1)
			if err != nil {
				return nil, err
			}
			if matchesType(branch["type"], value) {
				if selected != nil {
					return nil, fmt.Errorf("ambiguous union")
				}
				selected = branch
			}
		}
		if selected == nil {
			return nil, fmt.Errorf("unmatched union")
		}
		return b.normalize(selected, value, depth+1)
	}
	if typ, ok := s["type"]; ok && !matchesType(typ, value) {
		return nil, fmt.Errorf("unexpected JSON type")
	}
	switch v := value.(type) {
	case map[string]any:
		if required, ok := s["required"].([]any); ok {
			for _, raw := range required {
				key, ok := raw.(string)
				if !ok {
					return nil, fmt.Errorf("invalid required property")
				}
				if _, exists := v[key]; !exists {
					return nil, fmt.Errorf("missing required property %s", key)
				}
			}
		}
		result := make(map[string]any, len(v))
		for key, item := range v {
			result[key] = item
		}
		properties, _ := s["properties"].(map[string]any)
		for key, raw := range properties {
			child, ok := raw.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("invalid property schema")
			}
			child, err = b.resolve(child, depth+1)
			if err != nil {
				return nil, err
			}
			item, exists := result[key]
			if !exists {
				item, exists = child["default"]
			}
			if !exists {
				continue
			}
			result[key], err = b.normalize(child, item, depth+1)
			if err != nil {
				return nil, err
			}
		}
		return result, nil
	case []any:
		child, ok := s["items"].(map[string]any)
		if !ok {
			return value, nil
		}
		result := make([]any, len(v))
		for i, item := range v {
			result[i], err = b.normalize(child, item, depth+1)
			if err != nil {
				return nil, err
			}
		}
		return result, nil
	default:
		return value, nil
	}
}

func matchesType(typ, value any) bool {
	switch typ {
	case "null":
		return value == nil
	case "object":
		_, ok := value.(map[string]any)
		return ok
	case "array":
		_, ok := value.([]any)
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "number", "integer":
		_, ok := value.(json.Number)
		return ok
	default:
		return false
	}
}

func same(left, right any) bool {
	switch l := left.(type) {
	case map[string]any:
		r, ok := right.(map[string]any)
		if !ok || len(l) != len(r) {
			return false
		}
		for key, value := range l {
			other, exists := r[key]
			if !exists || !same(value, other) {
				return false
			}
		}
		return true
	case []any:
		r, ok := right.([]any)
		if !ok || len(l) != len(r) {
			return false
		}
		for i, value := range l {
			if !same(value, r[i]) {
				return false
			}
		}
		return true
	case json.Number:
		r, ok := right.(json.Number)
		if !ok {
			return false
		}
		if l == r {
			return true
		}
		// Exact rational comparison avoids silently rounding large integers.
		ln, lok := new(big.Rat).SetString(string(l))
		rn, rok := new(big.Rat).SetString(string(r))
		return lok && rok && ln.Cmp(rn) == 0
	default:
		return left == right
	}
}
