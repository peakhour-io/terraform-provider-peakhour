package provider

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func TestStableJSONNullContractVectors(t *testing.T) {
	t.Parallel()

	contractBytes, err := os.ReadFile("testdata/stable-json-null-contract.json")
	if err != nil {
		t.Fatalf("read stable JSON contract: %v", err)
	}
	var contract struct {
		FormatVersion int `json:"format_version"`
		Vectors       []struct {
			Name       string          `json:"name"`
			Policy     string          `json:"policy"`
			Configured json.RawMessage `json:"configured"`
			API        json.RawMessage `json:"api"`
			Equal      bool            `json:"equal"`
		} `json:"vectors"`
	}
	if err := json.Unmarshal(contractBytes, &contract); err != nil {
		t.Fatalf("parse stable JSON contract: %v", err)
	}
	if contract.FormatVersion != 1 {
		t.Fatalf("contract format version = %d, want 1", contract.FormatVersion)
	}

	for _, vector := range contract.Vectors {
		vector := vector
		t.Run(vector.Name, func(t *testing.T) {
			t.Parallel()

			var got bool
			var hasErrors bool
			switch vector.Policy {
			case "exact_null_object_keys":
				api := NewJSONNullEquivalentValue(string(vector.API))
				configured := NewJSONNullEquivalentValue(string(vector.Configured))
				result, diagnostics := api.StringSemanticEquals(context.Background(), configured)
				got = result
				hasErrors = diagnostics.HasError()
			case "subset_null_object_keys":
				api := NewJSONSubsetNullEquivalentValue(string(vector.API))
				configured := NewJSONSubsetNullEquivalentValue(string(vector.Configured))
				result, diagnostics := api.StringSemanticEquals(context.Background(), configured)
				got = result
				hasErrors = diagnostics.HasError()
			default:
				t.Fatalf("unknown comparison policy %q", vector.Policy)
			}
			if hasErrors {
				t.Fatal("semantic equality returned error diagnostics")
			}
			if got != vector.Equal {
				t.Fatalf("semantic equality = %t, want %t", got, vector.Equal)
			}
		})
	}
}

func TestJSONNullEquivalentValueStringSemanticEquals(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		proposed string
		prior    string
		want     bool
	}{
		"missing and null object keys are equal": {
			proposed: `{"firewall":[{"type":"firewall","action":"allow"}]}`,
			prior:    `{"firewall":[{"type":"firewall","action":"allow","reason":null}]}`,
			want:     true,
		},
		"null and missing object keys are equal in either direction": {
			proposed: `{"firewall":[{"type":"firewall","action":"allow","reason":null}]}`,
			prior:    `{"firewall":[{"type":"firewall","action":"allow"}]}`,
			want:     true,
		},
		"nested null object keys are equal": {
			proposed: `{"outer":{"kept":1}}`,
			prior:    `{"outer":{"removed":null,"kept":1}}`,
			want:     true,
		},
		"configured non-null value missing remotely is drift": {
			proposed: `{"firewall":[{"type":"firewall","action":"allow"}]}`,
			prior:    `{"firewall":[{"type":"firewall","action":"allow","reason":"openai"}]}`,
			want:     false,
		},
		"remote non-null value differs from configured null": {
			proposed: `{"firewall":[{"type":"firewall","action":"allow","reason":"openai"}]}`,
			prior:    `{"firewall":[{"type":"firewall","action":"allow","reason":null}]}`,
			want:     false,
		},
		"non-null server field is not ignored": {
			proposed: `{"firewall":[{"type":"firewall","action":"allow","server_default":true}]}`,
			prior:    `{"firewall":[{"type":"firewall","action":"allow"}]}`,
			want:     false,
		},
		"null array elements remain significant": {
			proposed: `{"values":[null,1]}`,
			prior:    `{"values":[1]}`,
			want:     false,
		},
		"large numbers retain precision": {
			proposed: `{"value":9007199254740993}`,
			prior:    `{"value":9007199254740993,"ignored":null}`,
			want:     true,
		},
		"different large numbers are drift": {
			proposed: `{"value":9007199254740992}`,
			prior:    `{"value":9007199254740993}`,
			want:     false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			proposed := NewJSONNullEquivalentValue(test.proposed)
			prior := NewJSONNullEquivalentValue(test.prior)
			got, diags := proposed.StringSemanticEquals(context.Background(), prior)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
			if got != test.want {
				t.Fatalf("StringSemanticEquals() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestJSONSubsetNullEquivalentValueStringSemanticEquals(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		proposed string
		prior    string
		want     bool
	}{
		"configured null may be omitted by API": {
			proposed: `[{"variable":"REQUEST_URI","operator":"RX"}]`,
			prior:    `[{"variable":"REQUEST_URI","variable_quote_type":null,"operator":"RX"}]`,
			want:     true,
		},
		"API null may be omitted from configuration": {
			proposed: `{"message":null,"severity":null,"tags":null}`,
			prior:    `{}`,
			want:     true,
		},
		"non-null server defaults remain ignored": {
			proposed: `[{"variable":"REQUEST_URI","variable_part":"","variable_negated":false,"operator":"RX"}]`,
			prior:    `[{"variable":"REQUEST_URI","operator":"RX"}]`,
			want:     true,
		},
		"configured non-null value missing remotely is drift": {
			proposed: `{"severity":null}`,
			prior:    `{"severity":"ERROR"}`,
			want:     false,
		},
		"configured non-null value changed remotely is drift": {
			proposed: `{"severity":"WARNING"}`,
			prior:    `{"severity":"ERROR"}`,
			want:     false,
		},
		"null array elements remain significant": {
			proposed: `{"tags":["one"]}`,
			prior:    `{"tags":[null,"one"]}`,
			want:     false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			proposed := NewJSONSubsetNullEquivalentValue(test.proposed)
			prior := NewJSONSubsetNullEquivalentValue(test.prior)
			got, diags := proposed.StringSemanticEquals(context.Background(), prior)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
			if got != test.want {
				t.Fatalf("StringSemanticEquals() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestNullEquivalentValuesHandleInvalidAndTerraformSpecialValues(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	invalid := NewJSONNullEquivalentValue(`{"invalid"`)
	valid := NewJSONNullEquivalentValue(`{}`)
	if equal, diagnostics := invalid.StringSemanticEquals(ctx, valid); equal || !diagnostics.HasError() {
		t.Fatalf("invalid JSON result = (%t, %v), want false with error", equal, diagnostics)
	}

	nullValuable, diagnostics := (JSONNullEquivalentType{}).ValueFromString(ctx, basetypes.NewStringNull())
	if diagnostics.HasError() {
		t.Fatalf("construct null value: %v", diagnostics)
	}
	if equal, diagnostics := valid.StringSemanticEquals(ctx, nullValuable); equal || diagnostics.HasError() {
		t.Fatalf("null comparison result = (%t, %v), want false without error", equal, diagnostics)
	}

	unknownValuable, diagnostics := (JSONNullEquivalentType{}).ValueFromString(ctx, basetypes.NewStringUnknown())
	if diagnostics.HasError() {
		t.Fatalf("construct unknown value: %v", diagnostics)
	}
	if equal, diagnostics := valid.StringSemanticEquals(ctx, unknownValuable); equal || diagnostics.HasError() {
		t.Fatalf("unknown comparison result = (%t, %v), want false without error", equal, diagnostics)
	}

	if equal, diagnostics := valid.StringSemanticEquals(ctx, jsontypes.NewNormalizedValue(`{}`)); equal || !diagnostics.HasError() {
		t.Fatalf("type mismatch result = (%t, %v), want false with error", equal, diagnostics)
	}

	subset := NewJSONSubsetNullEquivalentValue(`{}`)
	if equal, diagnostics := subset.StringSemanticEquals(ctx, valid); equal || !diagnostics.HasError() {
		t.Fatalf("subset type mismatch result = (%t, %v), want false with error", equal, diagnostics)
	}
}

func TestNullEquivalentJSONTypesAreScopedToApprovedAttributes(t *testing.T) {
	t.Parallel()

	var ruleSchema resource.SchemaResponse
	NewRuleResource().Schema(context.Background(), resource.SchemaRequest{}, &ruleSchema)
	assertStringCustomType[JSONNullEquivalentType](t, ruleSchema, "actions_json")

	var customRuleSchema resource.SchemaResponse
	NewRPWAFCustomRuleResource().Schema(context.Background(), resource.SchemaRequest{}, &customRuleSchema)
	assertStringCustomType[JSONSubsetNullEquivalentType](t, customRuleSchema, "rules_json")
	assertStringCustomType[JSONNormalizedType](t, customRuleSchema, "action_json")
	assertStringCustomType[JSONSubsetNullEquivalentType](t, customRuleSchema, "logging_json")

	var owaspSchema resource.SchemaResponse
	NewRPWAFOWASPSettingsResource().Schema(context.Background(), resource.SchemaRequest{}, &owaspSchema)
	assertStringCustomType[JSONNormalizedType](t, owaspSchema, "settings_json")

	var imageSchema resource.SchemaResponse
	NewImageTransformResource().Schema(context.Background(), resource.SchemaRequest{}, &imageSchema)
	assertStringCustomType[JSONNormalizedType](t, imageSchema, "config_json")

	var optionsSchema resource.SchemaResponse
	NewRPWAFOptionsResource().Schema(context.Background(), resource.SchemaRequest{}, &optionsSchema)
	assertStringCustomType[jsontypes.NormalizedType](t, optionsSchema, "excluded_rules_json")
	assertStringCustomType[jsontypes.NormalizedType](t, optionsSchema, "excluded_files_json")
}

func assertStringCustomType[T any](t *testing.T, response resource.SchemaResponse, attributeName string) {
	t.Helper()

	attribute, ok := response.Schema.Attributes[attributeName].(schema.StringAttribute)
	if !ok {
		t.Fatalf("attribute %q is not a schema.StringAttribute", attributeName)
	}
	if _, ok := attribute.CustomType.(T); !ok {
		t.Fatalf("attribute %q custom type = %T, want %T", attributeName, attribute.CustomType, *new(T))
	}
}
