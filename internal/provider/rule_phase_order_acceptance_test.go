package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRulePhaseOrder_firewall(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccEnv(t, "PEAKHOUR_TEST_DOMAIN")
	nameA := fmt.Sprintf("tfacc-%s", acctest.RandString(10))
	nameB := fmt.Sprintf("tfacc-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRulePhaseOrderConfig(domain, nameA, nameB, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("peakhour_rule_phase_order.test", "domain", domain),
					resource.TestCheckResourceAttr("peakhour_rule_phase_order.test", "phase", "firewall"),
					resource.TestCheckResourceAttr("peakhour_rule_phase_order.test", "order.#", "2"),
					resource.TestCheckResourceAttrPair("peakhour_rule_phase_order.test", "order.0", "peakhour_rule.a", "uuid"),
					resource.TestCheckResourceAttrPair("peakhour_rule_phase_order.test", "order.1", "peakhour_rule.b", "uuid"),
				),
			},
			{
				Config: testAccRulePhaseOrderConfig(domain, nameA, nameB, true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("peakhour_rule_phase_order.test", "order.#", "2"),
					resource.TestCheckResourceAttrPair("peakhour_rule_phase_order.test", "order.0", "peakhour_rule.b", "uuid"),
					resource.TestCheckResourceAttrPair("peakhour_rule_phase_order.test", "order.1", "peakhour_rule.a", "uuid"),
				),
			},
			{
				ResourceName:      "peakhour_rule_phase_order.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccRulePhaseOrderConfig(domain, nameA, nameB string, reversed bool) string {
	filterA := `http.request.uri.path matches "^/tfacc/order/a"`
	filterB := `http.request.uri.path matches "^/tfacc/order/b"`

	var orderExpr string
	if reversed {
		orderExpr = "[peakhour_rule.b.uuid, peakhour_rule.a.uuid]"
	} else {
		orderExpr = "[peakhour_rule.a.uuid, peakhour_rule.b.uuid]"
	}

	return fmt.Sprintf(`
%s

resource "peakhour_rule" "a" {
  domain     = %q
  phase      = "firewall"
  name       = %q
  filter_str = %q

  actions_json = jsonencode({
    firewall = [{
      type   = "firewall"
      action = "deny"
      reason = "tfacc"
    }]
  })
}

resource "peakhour_rule" "b" {
  domain     = %q
  phase      = "firewall"
  name       = %q
  filter_str = %q

  actions_json = jsonencode({
    firewall = [{
      type   = "firewall"
      action = "deny"
      reason = "tfacc"
    }]
  })
}

resource "peakhour_rule_phase_order" "test" {
  domain = %q
  phase  = "firewall"

  order = %s
}
`, testAccConfigHeader(), domain, nameA, filterA, domain, nameB, filterB, domain, orderExpr)
}
