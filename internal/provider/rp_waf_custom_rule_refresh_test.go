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

func TestAccRPWAFCustomRule_ReadsAfterCreate(t *testing.T) {
	var reads atomic.Int32
	var currentName atomic.Value
	currentName.Store("Terraform rule")
	const ruleUUID = "11111111-1111-1111-1111-111111111111"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/domains/example.com/services/rp/waf/customrule":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			currentName.Store(body["name"].(string))
			_ = json.NewEncoder(w).Encode(testWAFCustomRuleResponse(ruleUUID, currentName.Load().(string)))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/domains/example.com/services/rp/waf/customrule":
			reads.Add(1)
			_ = json.NewEncoder(w).Encode([]any{testWAFCustomRuleResponse(ruleUUID, currentName.Load().(string))})
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/domains/example.com/services/rp/waf/customrule/"+ruleUUID:
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			currentName.Store(body["name"].(string))
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
			Config: testUnitWAFCustomRuleConfig("Terraform rule"),
			Check: func(_ *terraform.State) error {
				if reads.Load() == 0 {
					return fmt.Errorf("expected a GET after create")
				}
				return nil
			},
		}, {
			Config: testUnitWAFCustomRuleConfig("Updated Terraform rule"),
			Check: func(_ *terraform.State) error {
				if reads.Load() < 2 {
					return fmt.Errorf("expected a GET after update")
				}
				return nil
			},
		}},
	})
}

func testWAFCustomRuleResponse(uuid, name string) map[string]any {
	return map[string]any{
		"uuid": uuid, "rule_id": 1, "created": "2026-07-22T00:00:00Z",
		"name": name, "description": nil, "enabled": true,
		"rules":   []any{map[string]any{"variable": "ARGS", "operator": "@contains", "operator_arg": "bad"}},
		"action":  map[string]any{"action_name": "deny"},
		"logging": map[string]any{"message": "blocked", "severity": "INFO", "tags": []any{}},
	}
}

func testUnitWAFCustomRuleConfig(name string) string {
	return fmt.Sprintf(`
terraform {
  required_providers {
    peakhour = { source = "peakhour-io/peakhour" }
  }
}
provider "peakhour" {}
resource "peakhour_rp_waf_custom_rule" "test" {
  domain       = "example.com"
  name         = %q
  enabled      = true
  rules_json   = jsonencode([{ variable = "ARGS", operator = "@contains", operator_arg = "bad" }])
  action_json  = jsonencode({ action_name = "deny" })
  logging_json = jsonencode({ message = "blocked", severity = "INFO", tags = [] })
}
`, name)
}
