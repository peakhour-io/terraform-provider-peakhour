package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccBulkRedirect_basic(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccEnv(t, "PEAKHOUR_TEST_DOMAIN")
	listName := fmt.Sprintf("tfacc-%s", acctest.RandString(10))
	entryPath := fmt.Sprintf("/tfacc-%s", acctest.RandString(10))
	ruleName := fmt.Sprintf("tfacc-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBulkRedirectConfig(domain, listName, entryPath, ruleName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("peakhour_bulk_redirect_list.test", "domain", domain),
					resource.TestCheckResourceAttr("peakhour_bulk_redirect_list.test", "name", listName),
					resource.TestCheckResourceAttrSet("peakhour_bulk_redirect_list.test", "uuid"),
					resource.TestCheckResourceAttrSet("peakhour_bulk_redirect_entry.test", "entry_id"),
					resource.TestCheckResourceAttrSet("peakhour_rule.test", "uuid"),
					resource.TestMatchResourceAttr("peakhour_rule.test", "actions_json", regexp.MustCompile(regexp.QuoteMeta(fmt.Sprintf(`"from_list":"%s"`, listName)))),
				),
			},
			{
				ResourceName:      "peakhour_bulk_redirect_list.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "peakhour_bulk_redirect_entry.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "peakhour_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccBulkRedirectConfig(domain, listName, entryPath, ruleName string) string {
	filter := `http.request.uri.path matches "^/tfacc/"`

	return fmt.Sprintf(`
terraform {
  required_providers {
    peakhour = {
      source = "peakhour-io/peakhour"
    }
  }
}

provider "peakhour" {}

resource "peakhour_bulk_redirect_list" "test" {
  domain = %q
  name   = %q
}

resource "peakhour_bulk_redirect_entry" "test" {
  domain             = %q
  bulk_redirect_uuid = peakhour_bulk_redirect_list.test.uuid

  source_path = %q
  target_url  = "https://example.com/new"
  status_code = 301
}

resource "peakhour_rule" "test" {
  domain     = %q
  phase      = "bulk_redirect"
  name       = %q
  filter_str = %q
  enabled    = true

  actions_json = jsonencode({
    redirect = [{
      type      = "redirect"
      from_list = peakhour_bulk_redirect_list.test.name
    }]
  })

  depends_on = [peakhour_bulk_redirect_entry.test]
}
`, domain, listName, domain, entryPath, domain, ruleName, filter)
}
