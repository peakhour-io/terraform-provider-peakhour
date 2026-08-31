package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// JSONNullEquivalentType compares JSON documents exactly after treating null
// object properties as absent. Null array elements remain significant.
type JSONNullEquivalentType struct {
	basetypes.StringType
}

var _ basetypes.StringTypable = JSONNullEquivalentType{}

func (t JSONNullEquivalentType) Equal(other attr.Type) bool {
	otherType, ok := other.(JSONNullEquivalentType)
	return ok && t.StringType.Equal(otherType.StringType)
}

func (t JSONNullEquivalentType) String() string {
	return "JSONNullEquivalentType"
}

func (t JSONNullEquivalentType) ValueFromString(_ context.Context, value basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return JSONNullEquivalentValue{Normalized: jsontypes.Normalized{StringValue: value}}, nil
}

func (t JSONNullEquivalentType) ValueFromTerraform(ctx context.Context, value tftypes.Value) (attr.Value, error) {
	stringAttribute, err := t.StringType.ValueFromTerraform(ctx, value)
	if err != nil {
		return nil, err
	}

	stringValue, ok := stringAttribute.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type of %T", stringAttribute)
	}

	return JSONNullEquivalentValue{Normalized: jsontypes.Normalized{StringValue: stringValue}}, nil
}

func (t JSONNullEquivalentType) ValueType(context.Context) attr.Value {
	return JSONNullEquivalentValue{}
}

// JSONNullEquivalentValue retains jsontypes.Normalized validation while using
// the Peakhour API's null-object-property comparison contract.
type JSONNullEquivalentValue struct {
	jsontypes.Normalized
}

var _ basetypes.StringValuable = JSONNullEquivalentValue{}
var _ basetypes.StringValuableWithSemanticEquals = JSONNullEquivalentValue{}

func (v JSONNullEquivalentValue) Equal(other attr.Value) bool {
	otherValue, ok := other.(JSONNullEquivalentValue)
	return ok && v.StringValue.Equal(otherValue.StringValue)
}

func (v JSONNullEquivalentValue) Type(context.Context) attr.Type {
	return JSONNullEquivalentType{}
}

func (v JSONNullEquivalentValue) StringSemanticEquals(_ context.Context, priorValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	priorValue, ok := priorValuable.(JSONNullEquivalentValue)
	if !ok {
		return false, semanticEqualityTypeDiagnostics(v, priorValuable)
	}

	return compareJSONValues(v.StringValue, priorValue.StringValue, false)
}

func NewJSONNullEquivalentValue(value string) JSONNullEquivalentValue {
	return JSONNullEquivalentValue{
		Normalized: jsontypes.Normalized{StringValue: basetypes.NewStringValue(value)},
	}
}

// JSONSubsetNullEquivalentType ignores API-added object properties while also
// treating null object properties as absent. It is intentionally distinct from
// JSONNormalizedType so this behavior can be enabled per API attribute.
type JSONSubsetNullEquivalentType struct {
	basetypes.StringType
}

var _ basetypes.StringTypable = JSONSubsetNullEquivalentType{}

func (t JSONSubsetNullEquivalentType) Equal(other attr.Type) bool {
	otherType, ok := other.(JSONSubsetNullEquivalentType)
	return ok && t.StringType.Equal(otherType.StringType)
}

func (t JSONSubsetNullEquivalentType) String() string {
	return "JSONSubsetNullEquivalentType"
}

func (t JSONSubsetNullEquivalentType) ValueFromString(_ context.Context, value basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return JSONSubsetNullEquivalentValue{Normalized: jsontypes.Normalized{StringValue: value}}, nil
}

func (t JSONSubsetNullEquivalentType) ValueFromTerraform(ctx context.Context, value tftypes.Value) (attr.Value, error) {
	stringAttribute, err := t.StringType.ValueFromTerraform(ctx, value)
	if err != nil {
		return nil, err
	}

	stringValue, ok := stringAttribute.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type of %T", stringAttribute)
	}

	return JSONSubsetNullEquivalentValue{Normalized: jsontypes.Normalized{StringValue: stringValue}}, nil
}

func (t JSONSubsetNullEquivalentType) ValueType(context.Context) attr.Value {
	return JSONSubsetNullEquivalentValue{}
}

type JSONSubsetNullEquivalentValue struct {
	jsontypes.Normalized
}

var _ basetypes.StringValuable = JSONSubsetNullEquivalentValue{}
var _ basetypes.StringValuableWithSemanticEquals = JSONSubsetNullEquivalentValue{}

func (v JSONSubsetNullEquivalentValue) Equal(other attr.Value) bool {
	otherValue, ok := other.(JSONSubsetNullEquivalentValue)
	return ok && v.StringValue.Equal(otherValue.StringValue)
}

func (v JSONSubsetNullEquivalentValue) Type(context.Context) attr.Type {
	return JSONSubsetNullEquivalentType{}
}

func (v JSONSubsetNullEquivalentValue) StringSemanticEquals(_ context.Context, priorValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	priorValue, ok := priorValuable.(JSONSubsetNullEquivalentValue)
	if !ok {
		return false, semanticEqualityTypeDiagnostics(v, priorValuable)
	}

	return compareJSONValues(v.StringValue, priorValue.StringValue, true)
}

func NewJSONSubsetNullEquivalentValue(value string) JSONSubsetNullEquivalentValue {
	return JSONSubsetNullEquivalentValue{
		Normalized: jsontypes.Normalized{StringValue: basetypes.NewStringValue(value)},
	}
}

func semanticEqualityTypeDiagnostics(expected, actual any) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	diagnostics.AddError(
		"Semantic Equality Check Error",
		fmt.Sprintf("Expected value type %T while comparing JSON, got %T.", expected, actual),
	)
	return diagnostics
}

func compareJSONValues(proposed, prior basetypes.StringValue, subset bool) (bool, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	if proposed.IsNull() || proposed.IsUnknown() || prior.IsNull() || prior.IsUnknown() {
		return false, diagnostics
	}

	proposedJSON, err := decodeJSONDropNullObjectKeys(proposed.ValueString())
	if err != nil {
		diagnostics.AddError("Semantic Equality Check Error", "Could not parse proposed JSON value: "+err.Error())
		return false, diagnostics
	}

	priorJSON, err := decodeJSONDropNullObjectKeys(prior.ValueString())
	if err != nil {
		diagnostics.AddError("Semantic Equality Check Error", "Could not parse prior JSON value: "+err.Error())
		return false, diagnostics
	}

	if subset {
		return isSubset(priorJSON, proposedJSON), diagnostics
	}
	return reflect.DeepEqual(proposedJSON, priorJSON), diagnostics
}

func decodeJSONDropNullObjectKeys(input string) (any, error) {
	if !json.Valid([]byte(input)) {
		return nil, fmt.Errorf("invalid JSON")
	}

	decoder := json.NewDecoder(strings.NewReader(input))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return dropNullObjectKeys(value), nil
}
