package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/peakhour-io/terraform-provider-peakhour/internal/client"
)

func TestRPWAFCustomRuleOrderSchema_ManagesCompleteOrder(t *testing.T) {
	r := NewRPWAFCustomRuleOrderResource()

	var metadata resource.MetadataResponse
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "peakhour"}, &metadata)
	if metadata.TypeName != "peakhour_rp_waf_custom_rule_order" {
		t.Fatalf("type name = %q", metadata.TypeName)
	}

	var schemaResponse resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResponse)
	for _, name := range []string{"id", "domain", "include_all_rules", "order"} {
		if _, ok := schemaResponse.Schema.Attributes[name]; !ok {
			t.Fatalf("schema missing %q", name)
		}
	}
}

func TestRPWAFCustomRuleOrder_applyOrderAllowsManagedSubsetDuringRemoval(t *testing.T) {
	var patched client.WAFCustomRuleReorder
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode([]client.WAFCustomRule{
				{UUID: "rule-a", RuleID: 1},
				{UUID: "rule-b", RuleID: 2},
			})
		case http.MethodPatch:
			if err := json.NewDecoder(r.Body).Decode(&patched); err != nil {
				t.Fatalf("decode reorder request: %v", err)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "unexpected request", http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)

	r := &RPWAFCustomRuleOrderResource{client: client.NewClient("test", server.URL)}
	model := RPWAFCustomRuleOrderResourceModel{
		Domain:          types.StringValue("example.com"),
		IncludeAllRules: types.BoolValue(false),
		Order: types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("rule-a"),
		}),
	}
	var diags diag.Diagnostics
	if err := r.applyOrder(context.Background(), &model, &diags); err != nil {
		t.Fatalf("managed subset should tolerate a rule pending deletion: %v", err)
	}

	if len(patched.Order) != 2 || patched.Order[0] != "rule-a" || patched.Order[1] != "rule-b" {
		t.Fatalf("transitional full order = %v", patched.Order)
	}
}

func TestRPWAFCustomRuleOrder_readOrderKeepsManagedSubsetDuringRemoval(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]client.WAFCustomRule{
			{UUID: "rule-a", RuleID: 1},
			{UUID: "rule-b", RuleID: 2},
		})
	}))
	t.Cleanup(server.Close)

	r := &RPWAFCustomRuleOrderResource{client: client.NewClient("test", server.URL)}
	model := RPWAFCustomRuleOrderResourceModel{
		Domain:          types.StringValue("example.com"),
		IncludeAllRules: types.BoolValue(false),
		Order: types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("rule-a"),
		}),
	}
	var diags diag.Diagnostics
	if err := r.readOrder(context.Background(), &model, &diags); err != nil {
		t.Fatal(err)
	}

	var got []string
	diags.Append(model.Order.ElementsAs(context.Background(), &got, false)...)
	if diags.HasError() {
		t.Fatalf("decode managed order: %v", diags)
	}
	if len(got) != 1 || got[0] != "rule-a" {
		t.Fatalf("managed order = %v", got)
	}
}

func TestRPWAFCustomRuleOrder_currentOrderSortsByRuleID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/domains/example.com/services/rp/waf/customrule" {
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode([]client.WAFCustomRule{
			{UUID: "rule-b", RuleID: 2},
			{UUID: "rule-a", RuleID: 1},
		})
	}))
	t.Cleanup(server.Close)

	r := &RPWAFCustomRuleOrderResource{client: client.NewClient("test", server.URL)}
	got, err := r.currentOrder(context.Background(), "example.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "rule-a" || got[1] != "rule-b" {
		t.Fatalf("current order = %v", got)
	}
}

func TestRPWAFCustomRuleOrder_applyOrderRequiresCompleteSet(t *testing.T) {
	patchCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode([]client.WAFCustomRule{
				{UUID: "rule-a", RuleID: 1},
				{UUID: "rule-b", RuleID: 2},
			})
		case http.MethodPatch:
			patchCalled = true
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "unexpected request", http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)

	r := &RPWAFCustomRuleOrderResource{client: client.NewClient("test", server.URL)}
	model := RPWAFCustomRuleOrderResourceModel{
		Domain: types.StringValue("example.com"),
		Order: types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("rule-a"),
		}),
	}
	var diags diag.Diagnostics
	if err := r.applyOrder(context.Background(), &model, &diags); err == nil {
		t.Fatal("expected incomplete order to fail")
	}
	if patchCalled {
		t.Fatal("invalid order must not call reorder endpoint")
	}
}
