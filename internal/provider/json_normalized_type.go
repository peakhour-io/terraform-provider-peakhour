package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// JSONNormalizedType is a custom string type for JSON that implements
// "subset semantic equality". This means user config is considered equal
// to API response if all user-specified fields match, even if the API
// returns additional fields with server-side defaults.
//
// Example:
//   - User config: {"width":100}
//   - API response: {"width":100,"crop":{"centre":true}}
//   - Result: EQUAL (user's fields match, server defaults are ignored)
//
// But:
//   - User config: {"width":100}
//   - API response: {"width":200}
//   - Result: NOT EQUAL (real drift detected)
type JSONNormalizedType struct {
	basetypes.StringType
}

var _ basetypes.StringTypable = JSONNormalizedType{}

func (t JSONNormalizedType) Equal(o attr.Type) bool {
	other, ok := o.(JSONNormalizedType)
	if !ok {
		return false
	}
	return t.StringType.Equal(other.StringType)
}

func (t JSONNormalizedType) String() string {
	return "JSONNormalizedType"
}

func (t JSONNormalizedType) ValueFromString(ctx context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return JSONNormalizedValue{StringValue: in}, nil
}

func (t JSONNormalizedType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrValue, err := t.StringType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}

	stringValue, ok := attrValue.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type of %T", attrValue)
	}

	stringValuable, diags := t.ValueFromString(ctx, stringValue)
	if diags.HasError() {
		return nil, fmt.Errorf("unexpected error converting StringValue to StringValuable: %v", diags)
	}

	return stringValuable, nil
}

func (t JSONNormalizedType) ValueType(ctx context.Context) attr.Value {
	return JSONNormalizedValue{}
}

// JSONNormalizedValue implements subset semantic equality for JSON strings.
type JSONNormalizedValue struct {
	basetypes.StringValue
}

var _ basetypes.StringValuable = JSONNormalizedValue{}
var _ basetypes.StringValuableWithSemanticEquals = JSONNormalizedValue{}

func (v JSONNormalizedValue) ToStringValue(ctx context.Context) (basetypes.StringValue, diag.Diagnostics) {
	return v.StringValue, nil
}

func (v JSONNormalizedValue) Equal(o attr.Value) bool {
	other, ok := o.(JSONNormalizedValue)
	if !ok {
		return false
	}
	return v.StringValue.Equal(other.StringValue)
}

func (v JSONNormalizedValue) Type(ctx context.Context) attr.Type {
	return JSONNormalizedType{}
}

// StringSemanticEquals implements subset equality for JSON.
// Terraform Plugin Framework invokes this method on the proposed new value and
// passes the prior value as the argument. During Read:
//   - v = proposed new value from the API
//   - priorValuable = prior state derived from the user's configuration
//
// We return true if all fields in the prior value exist in the proposed new
// value with the same values (the API value may have additional defaults).
func (v JSONNormalizedValue) StringSemanticEquals(ctx context.Context, priorValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	if v.IsNull() || v.IsUnknown() {
		return false, diags
	}

	priorValue, ok := priorValuable.(JSONNormalizedValue)
	if !ok {
		return false, diags
	}

	if priorValue.IsNull() || priorValue.IsUnknown() {
		return false, diags
	}

	// Parse both JSON values
	var newJSON, priorJSON any
	if err := json.Unmarshal([]byte(v.ValueString()), &newJSON); err != nil {
		diags.AddWarning(
			"Invalid JSON in proposed state",
			fmt.Sprintf("Could not parse proposed JSON value: %s", err),
		)
		return false, diags
	}

	if err := json.Unmarshal([]byte(priorValue.ValueString()), &priorJSON); err != nil {
		diags.AddWarning(
			"Invalid JSON in prior state",
			fmt.Sprintf("Could not parse prior JSON value: %s", err),
		)
		return false, diags
	}

	// Check if prior is a subset of new (all prior fields exist in new with same values)
	return isSubset(priorJSON, newJSON), diags
}

// isSubset checks if 'subset' is a subset of 'superset'.
// Returns true if all fields in subset exist in superset with equal values.
// Superset may have additional fields that are ignored.
func isSubset(subset, superset any) bool {
	switch s := subset.(type) {
	case map[string]any:
		// For objects, check each key in subset exists in superset with matching value
		superMap, ok := superset.(map[string]any)
		if !ok {
			return false
		}
		for key, subVal := range s {
			superVal, exists := superMap[key]
			if !exists {
				// Key in user config doesn't exist in API response - not equal
				return false
			}
			if !isSubset(subVal, superVal) {
				return false
			}
		}
		return true

	case []any:
		// For arrays, require exact match (order and length matter)
		superArr, ok := superset.([]any)
		if !ok {
			return false
		}
		if len(s) != len(superArr) {
			return false
		}
		for i, subVal := range s {
			if !isSubset(subVal, superArr[i]) {
				return false
			}
		}
		return true

	default:
		// For primitives (string, number, bool, null), use direct equality
		// JSON numbers are float64 in Go
		return subset == superset
	}
}

// Helper functions for creating JSONNormalizedValue instances

func NewJSONNormalizedNull() JSONNormalizedValue {
	return JSONNormalizedValue{StringValue: basetypes.NewStringNull()}
}

func NewJSONNormalizedUnknown() JSONNormalizedValue {
	return JSONNormalizedValue{StringValue: basetypes.NewStringUnknown()}
}

func NewJSONNormalizedValue(value string) JSONNormalizedValue {
	return JSONNormalizedValue{StringValue: basetypes.NewStringValue(value)}
}
