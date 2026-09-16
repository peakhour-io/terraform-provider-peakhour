package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateAndCheck(t *testing.T) {
	dir := t.TempDir()
	spec, out := filepath.Join(dir, "spec.json"), filepath.Join(dir, "bundle.json")
	writeSpec := func(defaultValue bool) {
		t.Helper()
		data, _ := json.Marshal(map[string]any{"components": map[string]any{"schemas": map[string]any{
			"WafAction":               map[string]any{"$ref": "#/components/schemas/Referenced"},
			"Referenced":              map[string]any{"type": "object", "properties": map[string]any{"flag": map[string]any{"default": defaultValue}}},
			"WafLogging":              map[string]any{"type": "object"},
			"WafCustomRuleExpression": map[string]any{"type": "object"},
		}}})
		if err := os.WriteFile(spec, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	writeSpec(false)
	if err := run(spec, out, false); err != nil {
		t.Fatal(err)
	}
	if err := run(spec, out, true); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(out)
	var bundle struct{ Schemas map[string]json.RawMessage }
	if err := json.Unmarshal(data, &bundle); err != nil {
		t.Fatal(err)
	}
	if len(bundle.Schemas) != 5 || bundle.Schemas["Referenced"] == nil || bundle.Schemas["WafRules"] == nil {
		t.Fatal("reference closure or array wrapper missing")
	}
	writeSpec(true)
	if err := run(spec, out, true); err == nil {
		t.Fatal("failed to detect stale defaults")
	}
	if err := run(spec, out, false); err != nil {
		t.Fatal(err)
	}
	if err := run(spec, out, true); err != nil {
		t.Fatal(err)
	}
}

func TestGenerateRejectsMissingSchemas(t *testing.T) {
	dir := t.TempDir()
	spec, out := filepath.Join(dir, "spec.json"), filepath.Join(dir, "out.json")
	for _, input := range []string{`{}`, `{"components":{"schemas":{"WafAction":{"$ref":"https://example.com/schema"}}}}`, `{"components":{"schemas":{"WafCustomRuleExpression":{"$ref":"#/components/schemas/Missing"}}}}`} {
		if err := os.WriteFile(spec, []byte(input), 0600); err != nil {
			t.Fatal(err)
		}
		if err := run(spec, out, false); err == nil {
			t.Fatalf("accepted incomplete schema %s", input)
		}
		if _, err := os.Stat(out); !os.IsNotExist(err) {
			t.Fatal("wrote an incomplete bundle")
		}
	}
}
