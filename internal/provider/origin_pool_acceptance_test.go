package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccOriginPool_basic(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccEnv(t, "PEAKHOUR_TEST_DOMAIN")
	tag := fmt.Sprintf("tfacc-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOriginPoolConfig(domain, tag, "192.0.2.10:8080"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("peakhour_origin_pool.test", "domain", domain),
					resource.TestCheckResourceAttr("peakhour_origin_pool.test", "tag", tag),
					resource.TestCheckResourceAttr("peakhour_origin_pool.test", "address.#", "1"),
					resource.TestCheckResourceAttrSet("peakhour_origin_pool.test", "id"),
				),
			},
			{
				ResourceName:      "peakhour_origin_pool.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccOriginPoolConfig(domain, tag, addr string) string {
	return fmt.Sprintf(`
terraform {
  required_providers {
    peakhour = {
      source = "peakhour-io/peakhour"
    }
  }
}

provider "peakhour" {}

resource "peakhour_origin_pool" "test" {
  domain = %q
  tag    = %q

  address = [{
    address = %q
    weight  = 100
  }]
}
`, domain, tag, addr)
}
