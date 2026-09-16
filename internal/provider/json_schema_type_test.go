package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestJSONSchemaPlanModifier(t *testing.T) {
	compact := basetypes.NewStringValue(`{"action_name":"deny"}`)
	expanded := basetypes.NewStringValue(`{"action_name":"deny","action_arg_val_param_val":""}`)
	nondefault := basetypes.NewStringValue(`{"action_name":"deny","action_arg_val_param_val":"password"}`)
	unknown, null := basetypes.NewStringUnknown(), basetypes.NewStringNull()
	for _, tt := range []struct {
		name                      string
		config, state, plan, want basetypes.StringValue
	}{
		{"defaults", compact, expanded, compact, expanded},
		{"reset nondefault", compact, nondefault, compact, compact},
		{"unknown config", unknown, expanded, unknown, unknown},
		{"unknown prior", compact, unknown, compact, compact},
		{"create", compact, null, compact, compact},
		{"destroy", null, expanded, null, null},
	} {
		t.Run(tt.name, func(t *testing.T) {
			request := planmodifier.StringRequest{ConfigValue: tt.config, StateValue: tt.state, PlanValue: tt.plan}
			response := planmodifier.StringResponse{PlanValue: tt.plan}
			jsonDefaultsPlanModifier{schemaName: "WafAction"}.PlanModifyString(context.Background(), request, &response)
			if response.Diagnostics.HasError() || !response.PlanValue.Equal(tt.want) {
				t.Fatalf("got %v (%v), want %v", response.PlanValue, response.Diagnostics, tt.want)
			}
		})
	}
}

func TestJSONSchemaTypeRoundTrip(t *testing.T) {
	ctx := context.Background()
	typ := JSONSchemaType{SchemaName: "WafAction"}
	for _, raw := range []any{nil, tftypes.UnknownValue, `{"action_name":"deny"}`} {
		value, err := typ.ValueFromTerraform(ctx, tftypes.NewValue(tftypes.String, raw))
		if err != nil {
			t.Fatal(err)
		}
		if !value.Type(ctx).Equal(typ) {
			t.Fatalf("schema identity lost: %v", value.Type(ctx))
		}
		roundTrip, err := value.ToTerraformValue(ctx)
		if err != nil || !roundTrip.Equal(tftypes.NewValue(tftypes.String, raw)) {
			t.Fatalf("round trip = %v, %v", roundTrip, err)
		}
	}
	if typ.Equal(JSONSchemaType{SchemaName: "WafLogging"}) {
		t.Fatal("distinct schema types compare equal")
	}
	a := NewJSONSchemaValue("WafAction", `{"action_name":"deny"}`)
	b := NewJSONSchemaValue("WafAction", `{"action_name":"deny","action_arg_val_param_val":""}`)
	if a.Equal(b) {
		t.Fatal("structural equality must remain exact")
	}
	if equal, diags := a.StringSemanticEquals(ctx, b); !equal || diags.HasError() {
		t.Fatalf("semantic equality = %v, %v", equal, diags)
	}
	if _, diags := a.StringSemanticEquals(ctx, NewJSONSchemaValue("WafLogging", `{}`)); !diags.HasError() {
		t.Fatal("schema mismatch was accepted")
	}
	if _, diags := a.StringSemanticEquals(ctx, NewJSONSchemaValue("WafAction", `{`)); !diags.HasError() {
		t.Fatal("invalid JSON was accepted")
	}
}
