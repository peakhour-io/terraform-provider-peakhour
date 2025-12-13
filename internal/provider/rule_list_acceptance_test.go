package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRuleList_ip(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccEnv(t, "PEAKHOUR_TEST_DOMAIN")
	listName := fmt.Sprintf("tfacc-%s", acctest.RandString(10))
	listNameUpdated := fmt.Sprintf("tfacc-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRuleListConfigIP(domain, listName, []string{"192.0.2.0/24", "198.51.100.10"}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("peakhour_rule_list.test", "domain", domain),
					resource.TestCheckResourceAttr("peakhour_rule_list.test", "name", listName),
					resource.TestCheckResourceAttr("peakhour_rule_list.test", "type", "ip"),
					resource.TestCheckResourceAttrSet("peakhour_rule_list.test", "uuid"),
					resource.TestCheckResourceAttr("peakhour_rule_list.test", "ips.#", "2"),
				),
			},
			{
				Config: testAccRuleListConfigIP(domain, listNameUpdated, []string{"192.0.2.0/24"}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("peakhour_rule_list.test", "name", listNameUpdated),
					resource.TestCheckResourceAttr("peakhour_rule_list.test", "ips.#", "1"),
				),
			},
			{
				ResourceName:      "peakhour_rule_list.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccRuleListConfigIP(domain, name string, ips []string) string {
	ipsHCL := ""
	for i, ip := range ips {
		if i > 0 {
			ipsHCL += ", "
		}
		ipsHCL += fmt.Sprintf("%q", ip)
	}

	return fmt.Sprintf(`
terraform {
  required_providers {
    peakhour = {
      source = "peakhour-io/peakhour"
    }
  }
}

provider "peakhour" {}

resource "peakhour_rule_list" "test" {
  domain = %q
  name   = %q
  type   = "ip"
  ips    = [%s]
}
`, domain, name, ipsHCL)
}
