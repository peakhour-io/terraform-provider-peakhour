// generate-json-defaults snapshots the schema closure used for JSON comparison.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	spec := flag.String("spec", "docs/spec/peakhour-api-v1.json", "pinned OpenAPI input")
	out := flag.String("out", "internal/jsondefaults/schemas.json", "generated bundle")
	check := flag.Bool("check", false, "fail if regeneration would change the bundle")
	flag.Parse()
	if err := run(*spec, *out, *check); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(specPath, outPath string, check bool) error {
	source, err := os.ReadFile(specPath)
	if err != nil {
		return err
	}
	var spec struct {
		Components struct {
			Schemas map[string]json.RawMessage `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(source, &spec); err != nil {
		return err
	}
	schemas := map[string]json.RawMessage{}
	var add func(string) error
	var refs func(any) error
	refs = func(value any) error {
		switch v := value.(type) {
		case map[string]any:
			for key, child := range v {
				if key == "$ref" {
					ref, ok := child.(string)
					if !ok || !strings.HasPrefix(ref, "#/components/schemas/") {
						return fmt.Errorf("unsupported schema reference %v", child)
					}
					if err := add(strings.TrimPrefix(ref, "#/components/schemas/")); err != nil {
						return err
					}
				} else if err := refs(child); err != nil {
					return err
				}
			}
		case []any:
			for _, child := range v {
				if err := refs(child); err != nil {
					return err
				}
			}
		}
		return nil
	}
	add = func(name string) error {
		if _, exists := schemas[name]; exists {
			return nil
		}
		raw, exists := spec.Components.Schemas[name]
		if !exists {
			return fmt.Errorf("missing schema %s", name)
		}
		schemas[name] = raw
		var value any
		if err := json.Unmarshal(raw, &value); err != nil {
			return err
		}
		return refs(value)
	}
	for _, name := range []string{"WafCustomRuleExpression", "WafAction", "WafLogging"} {
		if err := add(name); err != nil {
			return err
		}
	}
	schemas["WafRules"] = json.RawMessage(`{"type":"array","items":{"$ref":"#/components/schemas/WafCustomRuleExpression"}}`)
	bundle := struct {
		Source  string                     `json:"source"`
		SHA256  string                     `json:"source_sha256"`
		Schemas map[string]json.RawMessage `json:"schemas"`
	}{"docs/spec/peakhour-api-v1.json", fmt.Sprintf("%x", sha256.Sum256(source)), schemas}
	generated, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return err
	}
	generated = append(generated, '\n')
	if check {
		existing, err := os.ReadFile(outPath)
		if err != nil {
			return err
		}
		if !bytes.Equal(existing, generated) {
			return fmt.Errorf("schema bundle is stale; run go generate ./internal/jsondefaults")
		}
		return nil
	}
	return os.WriteFile(outPath, generated, 0644)
}
