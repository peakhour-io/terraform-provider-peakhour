package jsondefaults

import (
	"os/exec"
	"testing"
)

func TestGeneratedBundleIsCurrent(t *testing.T) {
	cmd := exec.Command("go", "run", "../../cmd/generate-json-defaults", "-spec", "../../docs/spec/peakhour-api-v1.json", "-out", "schemas.json", "-check")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("schema regeneration check: %v\n%s", err, output)
	}
}

func TestEqual(t *testing.T) {
	tests := []struct {
		name, schema, left, right string
		want                      bool
	}{
		{"action defaults", "WafAction", `{"action_name":"deny"}`, `{"action_name":"deny","action_arg_val":null,"action_arg_val_param":null,"action_arg_val_param_val":""}`, true},
		{"expression defaults", "WafRules", `[{"variable":"ARGS","operator":"@contains","operator_arg":"bad"}]`, `[{"variable":"ARGS","operator":"@contains","operator_arg":"bad","variable_part":"","variable_negated":false,"variable_counter":false,"variable_quote_type":null,"operator_negated":false,"transforms":[]}]`, true},
		{"logging defaults", "WafLogging", `{}`, `{"message":null,"severity":null,"tags":null}`, true},
		{"explicit null is not empty", "WafAction", `{"action_name":"deny","action_arg_val_param_val":null}`, `{"action_name":"deny"}`, false},
		{"logging null is not array", "WafLogging", `{}`, `{"tags":[]}`, false},
		{"nondefault extra value", "WafRules", `[{"variable":"ARGS","operator":"@contains"}]`, `[{"variable":"ARGS","operator":"@contains","operator_negated":true}]`, false},
		{"unknown null retained", "WafAction", `{"action_name":"deny"}`, `{"action_name":"deny","future":null}`, false},
		{"unknown value retained", "WafAction", `{"action_name":"deny"}`, `{"action_name":"deny","future":false}`, false},
		{"array order", "WafLogging", `{"tags":["a","b"]}`, `{"tags":["b","a"]}`, false},
		{"null array element", "WafLogging", `{"tags":[null]}`, `{"tags":[]}`, false},
		{"array length", "WafRules", `[]`, `[{}]`, false},
		{"formatting", "WafAction", `{"action_name":"deny", "action_arg_val":403}`, `{"action_arg_val":403,"action_name":"deny"}`, true},
		{"numeric equivalence", "WafAction", `{"action_name":"status","action_arg_val":403}`, `{"action_name":"status","action_arg_val":403.0}`, true},
		{"large integers distinct", "WafAction", `{"action_name":"status","action_arg_val":9007199254740992}`, `{"action_name":"status","action_arg_val":9007199254740993}`, false},
		{"string is not number", "WafAction", `{"action_name":"status","action_arg_val":"403"}`, `{"action_name":"status","action_arg_val":403}`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, pair := range [][2]string{{tt.left, tt.right}, {tt.right, tt.left}} {
				got, err := Equal(tt.schema, pair[0], pair[1])
				if err != nil || got != tt.want {
					t.Fatalf("Equal() = %v, %v; want %v", got, err, tt.want)
				}
			}
		})
	}
}

func TestInvalidJSON(t *testing.T) {
	for _, invalid := range []string{`{`, `{} {}`, `NaN`} {
		if _, err := Equal("WafAction", invalid, `{}`); err == nil {
			t.Errorf("accepted %q", invalid)
		}
		if _, err := Equal("WafAction", `{}`, invalid); err == nil {
			t.Errorf("accepted %q", invalid)
		}
	}
}

func TestSchemaTraversal(t *testing.T) {
	// Synthetic schemas prove the mechanism is generic, independent of WAF fields.
	tests := []struct {
		name, schema, left, right string
		want                      bool
	}{
		{"nested default", `{"type":"object","properties":{"child":{"type":"object","properties":{"x":{"default":false}}}}}`, `{"child":{}}`, `{"child":{"x":false}}`, true},
		{"absent parent stays absent", `{"type":"object","properties":{"child":{"type":"object","properties":{"x":{"default":false}}}}}`, `{}`, `{"child":{"x":false}}`, false},
		{"explicit parent default", `{"type":"object","properties":{"child":{"type":"object","default":{},"properties":{"x":{"default":false}}}}}`, `{}`, `{"child":{"x":false}}`, true},
		{"nullable object", `{"type":"object","properties":{"child":{"anyOf":[{"type":"null"},{"type":"object","properties":{"x":{"default":false}}}]}}}`, `{"child":{}}`, `{"child":{"x":false}}`, true},
		{"ambiguous union fails closed", `{"anyOf":[{"type":"object","properties":{"x":{"default":false}}},{"type":"object","properties":{"x":{"default":true}}}]}`, `{}`, `{"x":false}`, false},
		{"conditional fails closed", `{"type":"object","if":{"required":["kind"]},"then":{"properties":{"x":{"default":false}}},"properties":{"x":{"default":false}}}`, `{}`, `{"x":false}`, false},
		{"unknown schema keyword fails closed", `{"type":"object","unevaluatedProperties":false,"properties":{"x":{"default":false}}}`, `{}`, `{"x":false}`, false},
		{"union siblings fail closed", `{"anyOf":[{"type":"object","properties":{"x":{"default":false}}}],"properties":{"x":{"default":true}}}`, `{}`, `{"x":false}`, false},
		{"required default not optional", `{"type":"object","required":["x"],"properties":{"x":{"default":false}}}`, `{}`, `{"x":false}`, false},
		{"missing ref fails closed", `{"$ref":"#/components/schemas/Missing"}`, `{}`, `{"x":false}`, false},
		{"cycle fails closed", `{"$ref":"#/components/schemas/Test"}`, `{}`, `{"x":false}`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := parseBundle([]byte(`{"schemas":{"Test":` + tt.schema + `}}`))
			if err != nil {
				t.Fatal(err)
			}
			got, err := b.equal("Test", tt.left, tt.right)
			if err != nil || got != tt.want {
				t.Fatalf("equal = %v, %v; want %v", got, err, tt.want)
			}
		})
	}
}
