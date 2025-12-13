package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccRPThreatAccessListRule_basic(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccEnv(t, "PEAKHOUR_TEST_DOMAIN")
	ip := fmt.Sprintf("203.0.113.%d", acctest.RandIntRange(1, 254))
	desc1 := fmt.Sprintf("tfacc allow %s", acctest.RandString(8))
	desc2 := desc1 + " updated"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRPThreatAccessListRuleConfig(domain, ip, desc1),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("peakhour_rp_threat_access_list_rule.test", "domain", domain),
					resource.TestCheckResourceAttr("peakhour_rp_threat_access_list_rule.test", "rule_type", "whitelist"),
					resource.TestCheckResourceAttr("peakhour_rp_threat_access_list_rule.test", "content", ip),
					resource.TestCheckResourceAttr("peakhour_rp_threat_access_list_rule.test", "description", desc1),
					resource.TestCheckResourceAttrSet("peakhour_rp_threat_access_list_rule.test", "uuid"),
					resource.TestCheckResourceAttrSet("peakhour_rp_threat_access_list_rule.test", "id"),
				),
			},
			{
				Config: testAccRPThreatAccessListRuleConfig(domain, ip, desc2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("peakhour_rp_threat_access_list_rule.test", "description", desc2),
				),
			},
			{
				ResourceName:      "peakhour_rp_threat_access_list_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccRPThreatAccessListRuleConfig(domain, ip, description string) string {
	return fmt.Sprintf(`
terraform {
  required_providers {
    peakhour = {
      source = "peakhour-io/peakhour"
    }
  }
}

provider "peakhour" {}

resource "peakhour_rp_threat_access_list_rule" "test" {
  domain      = %q
  rule_type   = "whitelist"
  content     = %q
  description = %q
}
`, domain, ip, description)
}

func TestAccRPThreatBlockList_basic(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccEnv(t, "PEAKHOUR_TEST_DOMAIN")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRPThreatBlockListConfig(domain),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("peakhour_rp_threat_block_list.test", "domain", domain),
					resource.TestCheckResourceAttr("peakhour_rp_threat_block_list.test", "blocklists.#", "1"),
					resource.TestCheckResourceAttr("peakhour_rp_threat_block_list.test", "blocklists.0", "tor"),
					resource.TestCheckResourceAttrSet("peakhour_rp_threat_block_list.test", "id"),
				),
			},
			{
				ResourceName:      "peakhour_rp_threat_block_list.test",
				ImportState:       true,
				ImportStateIdFunc: func(*terraform.State) (string, error) { return domain, nil },
				ImportStateVerify: true,
			},
		},
	})
}

func testAccRPThreatBlockListConfig(domain string) string {
	return fmt.Sprintf(`
terraform {
  required_providers {
    peakhour = {
      source = "peakhour-io/peakhour"
    }
  }
}

provider "peakhour" {}

resource "peakhour_rp_threat_block_list" "test" {
  domain     = %q
  blocklists = ["tor"]
}
`, domain)
}
