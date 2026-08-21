package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	phclient "github.com/peakhour-io/terraform-provider-peakhour/internal/client"
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

func TestAccRPWAFCustomRule_basic(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccEnv(t, "PEAKHOUR_TEST_DOMAIN")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRPWAFCustomRuleConfig(domain),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("peakhour_rp_waf_custom_rule.test", "domain", domain),
					resource.TestCheckResourceAttr("peakhour_rp_waf_custom_rule.test", "enabled", "false"),
					resource.TestCheckResourceAttrSet("peakhour_rp_waf_custom_rule.test", "uuid"),
					resource.TestCheckResourceAttrSet("peakhour_rp_waf_custom_rule.test", "rule_id"),
					resource.TestCheckResourceAttrSet("peakhour_rp_waf_custom_rule.test", "created"),
					resource.TestCheckResourceAttrSet("peakhour_rp_waf_custom_rule.test", "id"),
				),
			},
			{
				ResourceName:      "peakhour_rp_waf_custom_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccRPWAFCustomRuleConfig(domain string) string {
	return fmt.Sprintf(`
terraform {
  required_providers {
    peakhour = {
      source = "peakhour-io/peakhour"
    }
  }
}

provider "peakhour" {}

resource "peakhour_rp_waf_custom_rule" "test" {
  domain = %q

  description = "tfacc custom rule"
  enabled     = false

  rules_json = jsonencode([
    {
      variable            = "REQUEST_HEADERS"
      variable_part       = "user-agent"
      variable_negated    = false
      variable_quote_type = null
      variable_counter    = false

      operator         = "@contains"
      operator_arg     = "curl"
      operator_negated = false
      transforms       = []
    }
  ])

  action_json = jsonencode({
    action_name              = "pass"
    action_arg_val           = null
    action_arg_val_param     = null
    action_arg_val_param_val = ""
  })

  logging_json = jsonencode({
    message  = "tfacc"
    severity = "INFO"
    tags     = ["terraform"]
  })
}
`, domain)
}

func TestAccRPWAFRuleGroup_basic(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccEnv(t, "PEAKHOUR_TEST_DOMAIN")
	apiKey := testAccEnv(t, "PEAKHOUR_API_KEY")
	baseURL := os.Getenv("PEAKHOUR_BASE_URL")

	c := phclient.NewClient(apiKey, baseURL)
	groups, err := c.ListRPWAFRuleGroups(domain, "owaspv33")
	if err != nil {
		t.Skipf("ListRPWAFRuleGroups failed (skipping): %v", err)
	}
	if len(groups) == 0 {
		t.Skip("no rule groups returned; skipping")
	}

	fileName := groups[0].FileName

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRPWAFRuleGroupConfig(domain, "owaspv33", fileName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("peakhour_rp_waf_rule_group.test", "domain", domain),
					resource.TestCheckResourceAttr("peakhour_rp_waf_rule_group.test", "ruleset", "owaspv33"),
					resource.TestCheckResourceAttr("peakhour_rp_waf_rule_group.test", "file_name", fileName),
					resource.TestCheckResourceAttr("peakhour_rp_waf_rule_group.test", "enabled", "false"),
					resource.TestCheckResourceAttrSet("peakhour_rp_waf_rule_group.test", "id"),
				),
			},
			{
				ResourceName:      "peakhour_rp_waf_rule_group.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccRPWAFRuleGroupConfig(domain, ruleset, fileName string) string {
	return fmt.Sprintf(`
terraform {
  required_providers {
    peakhour = {
      source = "peakhour-io/peakhour"
    }
  }
}

provider "peakhour" {}

resource "peakhour_rp_waf_rule_group" "test" {
  domain    = %q
  ruleset   = %q
  file_name = %q
  enabled   = false
}
`, domain, ruleset, fileName)
}
