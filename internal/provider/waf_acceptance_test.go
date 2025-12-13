package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccRPWAFOptions_basic(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccEnv(t, "PEAKHOUR_TEST_DOMAIN")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRPWAFOptionsConfig(domain),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("peakhour_rp_waf_options.test", "domain", domain),
					resource.TestCheckResourceAttr("peakhour_rp_waf_options.test", "waf_mode", "warn"),
					resource.TestCheckResourceAttrSet("peakhour_rp_waf_options.test", "excluded_rules_json"),
					resource.TestCheckResourceAttrSet("peakhour_rp_waf_options.test", "excluded_files_json"),
					resource.TestCheckResourceAttrSet("peakhour_rp_waf_options.test", "id"),
				),
			},
			{
				ResourceName:      "peakhour_rp_waf_options.test",
				ImportState:       true,
				ImportStateIdFunc: func(*terraform.State) (string, error) { return domain, nil },
				ImportStateVerify: true,
			},
		},
	})
}

func testAccRPWAFOptionsConfig(domain string) string {
	return fmt.Sprintf(`
terraform {
  required_providers {
    peakhour = {
      source = "peakhour-io/peakhour"
    }
  }
}

provider "peakhour" {}

resource "peakhour_rp_waf_options" "test" {
  domain   = %q
  waf_mode = "warn"
}
`, domain)
}

func TestAccRPWAFOWASPSettings_read(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccEnv(t, "PEAKHOUR_TEST_DOMAIN")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRPWAFOWASPSettingsReadConfig(domain),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("peakhour_rp_waf_owasp_settings.test", "domain", domain),
					resource.TestCheckResourceAttrSet("peakhour_rp_waf_owasp_settings.test", "settings_json"),
					resource.TestCheckResourceAttrSet("peakhour_rp_waf_owasp_settings.test", "id"),
				),
			},
			{
				ResourceName:      "peakhour_rp_waf_owasp_settings.test",
				ImportState:       true,
				ImportStateIdFunc: func(*terraform.State) (string, error) { return domain, nil },
				ImportStateVerify: true,
			},
		},
	})
}

func testAccRPWAFOWASPSettingsReadConfig(domain string) string {
	return fmt.Sprintf(`
terraform {
  required_providers {
    peakhour = {
      source = "peakhour-io/peakhour"
    }
  }
}

provider "peakhour" {}

resource "peakhour_rp_waf_owasp_settings" "test" {
  domain = %q
}
`, domain)
}
