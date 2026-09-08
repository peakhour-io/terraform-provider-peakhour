package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestRPWAFCustomRuleBuildBody_ManagesMetadata(t *testing.T) {
	newModel := func() RPWAFCustomRuleResourceModel {
		return RPWAFCustomRuleResourceModel{
			RulesJSON:   NewJSONSubsetNullEquivalentValue(`[]`),
			ActionJSON:  NewJSONNormalizedValue(`{"action_name":"pass"}`),
			LoggingJSON: NewJSONSubsetNullEquivalentValue(`{}`),
		}
	}

	t.Run("sets configured values", func(t *testing.T) {
		model := newModel()
		model.Name = types.StringValue("Block suspicious clients")
		model.Description = types.StringValue("Blocks curl user agents")
		var diags diag.Diagnostics
		body := (&RPWAFCustomRuleResource{}).buildRuleBody(&model, &diags)
		if diags.HasError() {
			t.Fatalf("buildRuleBody diagnostics: %v", diags)
		}
		if body["name"] != "Block suspicious clients" || body["description"] != "Blocks curl user agents" {
			t.Fatalf("unexpected metadata body: %#v", body)
		}
	})

	t.Run("sends nulls to clear removed values", func(t *testing.T) {
		model := newModel()
		model.Name = types.StringNull()
		model.Description = types.StringNull()
		var diags diag.Diagnostics
		body := (&RPWAFCustomRuleResource{}).buildRuleBody(&model, &diags)
		if diags.HasError() {
			t.Fatalf("buildRuleBody diagnostics: %v", diags)
		}
		if name, exists := body["name"]; !exists || name != nil {
			t.Fatalf("name clear was not sent: %#v", body)
		}
		if description, exists := body["description"]; !exists || description != nil {
			t.Fatalf("description clear was not sent: %#v", body)
		}
	})
}

func TestAccRPWAFCustomRule_NullAndMissingPropertiesAreStable(t *testing.T) {
	const ruleUUID = "11111111-1111-1111-1111-111111111111"
	var mutex sync.Mutex
	var createBody map[string]any

	response := testWAFCustomRuleResponse(ruleUUID, true, nil)
	rules := response["rules"].([]any)
	delete(rules[0].(map[string]any), "variable_quote_type")
	response["logging"] = map[string]any{"message": "blocked"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/domains/example.com/services/rp/waf/customrule":
			mutex.Lock()
			defer mutex.Unlock()
			if err := json.NewDecoder(r.Body).Decode(&createBody); err != nil {
				t.Errorf("decode create request: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(response)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/domains/example.com/services/rp/waf/customrule":
			_ = json.NewEncoder(w).Encode([]any{response})
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/domains/example.com/services/rp/waf/customrule/"+ruleUUID:
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "unexpected request "+r.Method+" "+r.URL.Path, http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)

	configureMockAcceptanceTest(t, server.URL)
	config := testUnitWAFCustomRuleNullConfig()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config},
			{Config: config, PlanOnly: true},
		},
	})

	mutex.Lock()
	defer mutex.Unlock()
	if createBody == nil {
		t.Fatal("create request was not captured")
	}
	rule := createBody["rules"].([]any)[0].(map[string]any)
	if value, exists := rule["variable_quote_type"]; !exists || value != nil {
		t.Fatalf("configured null variable_quote_type was not preserved: %#v", rule)
	}
	logging := createBody["logging"].(map[string]any)
	for _, key := range []string{"severity", "tags"} {
		if value, exists := logging[key]; !exists || value != nil {
			t.Fatalf("configured null %s was not preserved: %#v", key, logging)
		}
	}
}

func TestAccRPWAFCustomRule_ReconcilesEnabledWithToggleEndpoint(t *testing.T) {
	var reads atomic.Int32
	var toggles atomic.Int32
	var currentEnabled atomic.Bool
	currentEnabled.Store(true)
	const ruleUUID = "11111111-1111-1111-1111-111111111111"
	ruleResponse := func() map[string]any {
		name := "Terraform custom rule"
		response := testWAFCustomRuleResponse(ruleUUID, currentEnabled.Load(), &name)
		response["description"] = "Terraform managed rule"
		return response
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/domains/example.com/services/rp/waf/customrule":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["name"] != "Terraform custom rule" {
				t.Errorf("create request name = %#v, want Terraform custom rule", body["name"])
			}
			if body["description"] != "Terraform managed rule" {
				t.Errorf("create request description = %#v, want Terraform managed rule", body["description"])
			}
			if _, ok := body["enabled"]; ok {
				t.Error("create request must manage enabled state through the toggle endpoint")
			}
			_ = json.NewEncoder(w).Encode(ruleResponse())
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/domains/example.com/services/rp/waf/customrule":
			reads.Add(1)
			_ = json.NewEncoder(w).Encode([]any{ruleResponse()})
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/domains/example.com/services/rp/waf/customrule/"+ruleUUID:
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["name"] != "Terraform custom rule" {
				t.Errorf("update request name = %#v, want Terraform custom rule", body["name"])
			}
			if body["description"] != "Terraform managed rule" {
				t.Errorf("update request description = %#v, want Terraform managed rule", body["description"])
			}
			if _, ok := body["enabled"]; ok {
				t.Error("update request must manage enabled state through the toggle endpoint")
			}
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/domains/example.com/services/rp/waf/customrule/"+ruleUUID+"/enable":
			currentEnabled.Store(!currentEnabled.Load())
			toggles.Add(1)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/domains/example.com/services/rp/waf/customrule/"+ruleUUID:
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "unexpected request "+r.Method+" "+r.URL.Path, http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)

	t.Setenv("TF_ACC", "1")
	t.Setenv("PEAKHOUR_API_KEY", "test-key")
	t.Setenv("PEAKHOUR_BASE_URL", server.URL)
	t.Setenv("TF_ACC_PROVIDER_NAMESPACE", "peakhour-io")
	if terraformPath, ok := terraformBinPath(t); ok {
		t.Setenv("TF_ACC_TERRAFORM_PATH", terraformPath)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: testUnitWAFCustomRuleConfig(false),
			Check: func(_ *terraform.State) error {
				if reads.Load() == 0 {
					return fmt.Errorf("expected a GET after create")
				}
				if toggles.Load() != 1 {
					return fmt.Errorf("toggle calls after create = %d, want 1", toggles.Load())
				}
				return nil
			},
		}, {
			Config: testUnitWAFCustomRuleConfig(true),
			Check: func(_ *terraform.State) error {
				if reads.Load() < 2 {
					return fmt.Errorf("expected a GET after update")
				}
				if toggles.Load() != 2 {
					return fmt.Errorf("toggle calls after update = %d, want 2", toggles.Load())
				}
				return nil
			},
		}},
	})
}

func testWAFCustomRuleResponse(uuid string, enabled bool, name *string) map[string]any {
	return map[string]any{
		"uuid": uuid, "rule_id": 1, "created": "2026-07-22T00:00:00Z",
		"name": name, "description": nil, "enabled": enabled,
		"rules": []any{map[string]any{
			"variable": "ARGS", "variable_part": nil, "variable_negated": false,
			"variable_quote_type": nil, "variable_counter": false,
			"operator": "@contains", "operator_arg": "bad", "operator_negated": false,
			"transforms": []any{},
		}},
		"action": map[string]any{
			"action_name": "deny", "action_arg_val": nil,
			"action_arg_val_param": nil, "action_arg_val_param_val": "",
		},
		"logging": map[string]any{"message": "blocked", "severity": "INFO", "tags": []any{}},
	}
}

func testUnitWAFCustomRuleConfig(enabled bool) string {
	return fmt.Sprintf(`
terraform {
  required_providers {
    peakhour = { source = "peakhour-io/peakhour" }
  }
}
provider "peakhour" {}
resource "peakhour_rp_waf_custom_rule" "test" {
  domain       = "example.com"
  name         = "Terraform custom rule"
  description  = "Terraform managed rule"
  enabled      = %t
  rules_json   = jsonencode([{ variable = "ARGS", operator = "@contains", operator_arg = "bad" }])
  action_json  = jsonencode({ action_name = "deny" })
  logging_json = jsonencode({ message = "blocked", severity = "INFO", tags = [] })
}
`, enabled)
}

func testUnitWAFCustomRuleNullConfig() string {
	return `
terraform {
  required_providers {
    peakhour = { source = "peakhour-io/peakhour" }
  }
}
provider "peakhour" {}
resource "peakhour_rp_waf_custom_rule" "test" {
  domain = "example.com"
  rules_json = jsonencode([{
    variable            = "ARGS"
    variable_quote_type = null
    operator            = "@contains"
    operator_arg        = "bad"
  }])
  action_json = jsonencode({ action_name = "deny" })
  logging_json = jsonencode({
    message  = "blocked"
    severity = null
    tags     = null
  })
}
`
}
