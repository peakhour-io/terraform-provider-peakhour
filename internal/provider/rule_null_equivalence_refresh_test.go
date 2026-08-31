package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRule_NullAndMissingActionPropertiesAreStable(t *testing.T) {
	tests := map[string]struct {
		configReason      string
		responseHasReason bool
	}{
		"configured null and API omission": {
			configReason:      "reason = null",
			responseHasReason: false,
		},
		"configured omission and API null": {
			configReason:      "",
			responseHasReason: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var mutex sync.Mutex
			var createBody map[string]any
			server := newRuleNullContractServer(t, test.responseHasReason, func(body map[string]any) {
				mutex.Lock()
				defer mutex.Unlock()
				createBody = body
			})
			t.Cleanup(server.Close)

			configureMockAcceptanceTest(t, server.URL)
			config := testUnitRuleNullConfig(test.configReason)
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
			if test.configReason != "" {
				firewall := createBody["actions"].(map[string]any)["firewall"].([]any)[0].(map[string]any)
				reason, exists := firewall["reason"]
				if !exists || reason != nil {
					t.Fatalf("configured null reason was not preserved in request: %#v", firewall)
				}
			}
		})
	}
}

func newRuleNullContractServer(t *testing.T, responseHasReason bool, captureCreate func(map[string]any)) *httptest.Server {
	t.Helper()
	const rulePath = "/api/v1/domains/example.com/services/rp/rules/phases/firewall/rule/11111111-1111-1111-1111-111111111111"

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/api/v1/domains/example.com/services/rp/rules/phases/firewall":
			var body map[string]any
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Errorf("decode create request: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			captureCreate(body)
			_ = json.NewEncoder(w).Encode(map[string]any{"uuid": "11111111-1111-1111-1111-111111111111"})
		case request.Method == http.MethodGet && request.URL.Path == rulePath:
			action := map[string]any{"type": "firewall", "action": "allow"}
			if responseHasReason {
				action["reason"] = nil
			}
			enabled := true
			_ = json.NewEncoder(w).Encode(map[string]any{
				"uuid": "11111111-1111-1111-1111-111111111111", "pos": 0,
				"enabled": enabled, "name": "null-contract", "filter_str": "true",
				"phase": "firewall", "actions": map[string]any{"firewall": []any{action}},
			})
		case request.Method == http.MethodDelete && request.URL.Path == rulePath:
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "unexpected request "+request.Method+" "+request.URL.Path, http.StatusBadRequest)
		}
	}))
}

func configureMockAcceptanceTest(t *testing.T, baseURL string) {
	t.Helper()
	t.Setenv("TF_ACC", "1")
	t.Setenv("PEAKHOUR_API_KEY", "test-key")
	t.Setenv("PEAKHOUR_BASE_URL", baseURL)
	t.Setenv("TF_ACC_PROVIDER_NAMESPACE", "peakhour-io")
	if terraformPath, ok := terraformBinPath(t); ok {
		t.Setenv("TF_ACC_TERRAFORM_PATH", terraformPath)
	}
}

func testUnitRuleNullConfig(reasonLine string) string {
	return fmt.Sprintf(`
terraform {
  required_providers {
    peakhour = { source = "peakhour-io/peakhour" }
  }
}
provider "peakhour" {}
resource "peakhour_rule" "test" {
  domain     = "example.com"
  phase      = "firewall"
  name       = "null-contract"
  filter_str = "true"
  actions_json = jsonencode({
    firewall = [{
      type   = "firewall"
      action = "allow"
      %s
    }]
  })
}
`, reasonLine)
}
