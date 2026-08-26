package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRPWAFOWASPSettings_PreservesConfiguredSubsetOnRefresh(t *testing.T) {
	var reads atomic.Int32
	var configuredPatches atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/v1/domains/example.com/services/rp/waf/owasp" {
			http.Error(w, "unexpected path "+r.URL.Path, http.StatusNotFound)
			return
		}

		switch r.Method {
		case http.MethodGet:
			reads.Add(1)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"methods": map[string]any{
					"allowed_methods": []any{"GET", "HEAD", "POST", "OPTIONS"},
				},
				"protocol": map[string]any{
					"max_num_args":          100,
					"allowed_http_versions": []any{"HTTP/1.0", "HTTP/1.1", "HTTP/2"},
					"restricted_headers_basic": []any{
						"content-encoding",
						"proxy",
					},
				},
				"initialization": map[string]any{
					"blocking_paranoia_level":         1,
					"inbound_anomaly_score_threshold": 5,
				},
				"peakhour_settings": nil,
			})
		case http.MethodPatch:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode PATCH body: %v", err)
				http.Error(w, "invalid JSON", http.StatusBadRequest)
				return
			}
			if len(body) == 2 && body["methods"] != nil && body["protocol"] != nil {
				configuredPatches.Add(1)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "unexpected method "+r.Method, http.StatusMethodNotAllowed)
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
			Config: testUnitWAFOWASPSettingsConfig,
		}},
	})

	if reads.Load() == 0 {
		t.Error("expected the resource to refresh from the API")
	}
	if configuredPatches.Load() != 1 {
		t.Errorf("configured PATCH requests = %d, want 1", configuredPatches.Load())
	}
}

const testUnitWAFOWASPSettingsConfig = `
terraform {
  required_providers {
    peakhour = { source = "peakhour-io/peakhour" }
  }
}

provider "peakhour" {}

resource "peakhour_rp_waf_owasp_settings" "test" {
  domain = "example.com"

  settings_json = jsonencode({
    methods = {
      allowed_methods = ["GET", "HEAD", "POST", "OPTIONS"]
    }
    protocol = {
      max_num_args = 100
    }
  })
}
`
