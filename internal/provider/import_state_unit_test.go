package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccImportState_RateLimitZone_ParsesDomainAndName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/domains/example.com/services/rp/rate_limit/zones/my-zone":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"name": "my-zone",
			})
			return
		default:
			http.Error(w, "unexpected request "+r.Method+" "+r.URL.Path, http.StatusBadRequest)
			return
		}
	}))
	t.Cleanup(server.Close)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			t.Setenv("PEAKHOUR_BASE_URL", server.URL)
			t.Setenv("TF_ACC_PROVIDER_NAMESPACE", "peakhour-io")
			if terraformPath, ok := terraformBinPath(t); ok {
				t.Setenv("TF_ACC_TERRAFORM_PATH", terraformPath)
			}
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitRateLimitZoneConfig(),
				ResourceName: "peakhour_rate_limit_zone.test",
				ImportState:  true,
				ImportStateId: "example.com/my-zone",
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					for _, st := range states {
						if st.Ephemeral.Type != "peakhour_rate_limit_zone" {
							continue
						}

						if got := st.Attributes["domain"]; got != "example.com" {
							return fmt.Errorf("domain: got=%q want=%q", got, "example.com")
						}
						if got := st.Attributes["name"]; got != "my-zone" {
							return fmt.Errorf("name: got=%q want=%q", got, "my-zone")
						}
						if got := st.Attributes["id"]; got != "example.com/my-zone" {
							return fmt.Errorf("id: got=%q want=%q", got, "example.com/my-zone")
						}
						return nil
					}
					return fmt.Errorf("expected resource type %q in import state", "peakhour_rate_limit_zone")
				},
			},
		},
	})
}

func testUnitRateLimitZoneConfig() string {
	return `
terraform {
  required_providers {
    peakhour = {
      source = "peakhour-io/peakhour"
    }
  }
}

provider "peakhour" {}

resource "peakhour_rate_limit_zone" "test" {
  domain = "example.com"
  name   = "my-zone"
}
`
}

func terraformBinPath(t *testing.T) (string, bool) {
	t.Helper()

	if v := os.Getenv("TF_ACC_TERRAFORM_PATH"); v != "" {
		if _, err := os.Stat(v); err == nil {
			return v, true
		}
		t.Logf("TF_ACC_TERRAFORM_PATH=%q not usable", v)
	}

	if p, err := exec.LookPath("terraform"); err == nil {
		return p, true
	}

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	path := filepath.Join(repoRoot, ".toolchains", "bin", "terraform")

	if _, err := os.Stat(path); err != nil {
		t.Skipf("terraform binary not found at %q: %v", path, err)
	}
	return path, true
}
