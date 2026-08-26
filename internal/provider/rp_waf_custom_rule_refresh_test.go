package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccRPWAFCustomRule_ReconcilesEnabledWithToggleEndpoint(t *testing.T) {
	var reads atomic.Int32
	var toggles atomic.Int32
	var currentEnabled atomic.Bool
	currentEnabled.Store(true)
	const ruleUUID = "11111111-1111-1111-1111-111111111111"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/domains/example.com/services/rp/waf/customrule":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if _, ok := body["name"]; ok {
				t.Error("create request must not send unsupported custom rule name")
			}
			if _, ok := body["enabled"]; ok {
				t.Error("create request must manage enabled state through the toggle endpoint")
			}
			_ = json.NewEncoder(w).Encode(testWAFCustomRuleResponse(ruleUUID, currentEnabled.Load()))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/domains/example.com/services/rp/waf/customrule":
			reads.Add(1)
			_ = json.NewEncoder(w).Encode([]any{testWAFCustomRuleResponse(ruleUUID, currentEnabled.Load())})
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/domains/example.com/services/rp/waf/customrule/"+ruleUUID:
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if _, ok := body["name"]; ok {
				t.Error("update request must not send unsupported custom rule name")
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

func testWAFCustomRuleResponse(uuid string, enabled bool) map[string]any {
	return map[string]any{
		"uuid": uuid, "rule_id": 1, "created": "2026-07-22T00:00:00Z",
		"name": nil, "description": nil, "enabled": enabled,
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
  enabled      = %t
  rules_json   = jsonencode([{ variable = "ARGS", operator = "@contains", operator_arg = "bad" }])
  action_json  = jsonencode({ action_name = "deny" })
  logging_json = jsonencode({ message = "blocked", severity = "INFO", tags = [] })
}
`, enabled)
}
