package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/peakhour-io/terraform-provider-peakhour/internal/jsondefaults"
)

// JSONSchemaType opts one JSON attribute into a pinned API defaults contract.
// Terraform still sees a string; changing comparison does not change state shape.
type JSONSchemaType struct {
	basetypes.StringType
	SchemaName string
}

var _ basetypes.StringTypable = JSONSchemaType{}

func (t JSONSchemaType) Equal(other attr.Type) bool {
	o, ok := other.(JSONSchemaType)
	return ok && t.SchemaName == o.SchemaName && t.StringType.Equal(o.StringType)
}
func (t JSONSchemaType) String() string { return "JSONSchemaType(" + t.SchemaName + ")" }
func (t JSONSchemaType) ValueFromString(_ context.Context, v basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return JSONSchemaValue{Normalized: jsontypes.Normalized{StringValue: v}, SchemaName: t.SchemaName}, nil
}
func (t JSONSchemaType) ValueFromTerraform(ctx context.Context, v tftypes.Value) (attr.Value, error) {
	raw, err := t.StringType.ValueFromTerraform(ctx, v)
	if err != nil {
		return nil, err
	}
	s, ok := raw.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("unexpected string value type %T", raw)
	}
	return JSONSchemaValue{Normalized: jsontypes.Normalized{StringValue: s}, SchemaName: t.SchemaName}, nil
}
func (t JSONSchemaType) ValueType(context.Context) attr.Value {
	return JSONSchemaValue{SchemaName: t.SchemaName}
}

type JSONSchemaValue struct {
	jsontypes.Normalized
	SchemaName string
}

var _ basetypes.StringValuableWithSemanticEquals = JSONSchemaValue{}

func (v JSONSchemaValue) Equal(other attr.Value) bool {
	o, ok := other.(JSONSchemaValue)
	return ok && v.SchemaName == o.SchemaName && v.StringValue.Equal(o.StringValue)
}
func (v JSONSchemaValue) Type(context.Context) attr.Type {
	return JSONSchemaType{SchemaName: v.SchemaName}
}
func (v JSONSchemaValue) StringSemanticEquals(_ context.Context, prior basetypes.StringValuable) (bool, diag.Diagnostics) {
	p, ok := prior.(JSONSchemaValue)
	if !ok || p.SchemaName != v.SchemaName {
		return false, semanticEqualityTypeDiagnostics(v, prior)
	}
	return schemaJSONEqual(v.SchemaName, v.StringValue, p.StringValue)
}
func NewJSONSchemaValue(schemaName, value string) JSONSchemaValue {
	return JSONSchemaValue{Normalized: jsontypes.NewNormalizedValue(value), SchemaName: schemaName}
}

func schemaJSONEqual(schemaName string, a, b basetypes.StringValue) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	if a.IsNull() || a.IsUnknown() || b.IsNull() || b.IsUnknown() {
		return false, diags
	}
	equal, err := jsondefaults.Equal(schemaName, a.ValueString(), b.ValueString())
	if err != nil {
		diags.AddError("Invalid JSON", err.Error())
	}
	return equal, diags
}

type jsonDefaultsPlanModifier struct{ schemaName string }

var _ planmodifier.String = jsonDefaultsPlanModifier{}

func (m jsonDefaultsPlanModifier) Description(context.Context) string {
	return "Preserve prior JSON when configuration differs only by documented API defaults."
}
func (m jsonDefaultsPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}
func (m jsonDefaultsPlanModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// Do not interfere with creation, destruction, or unresolved expressions.
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}
	equal, diags := schemaJSONEqual(m.schemaName, req.ConfigValue, req.StateValue)
	resp.Diagnostics.Append(diags...)
	if equal {
		resp.PlanValue = req.StateValue
	}
}
