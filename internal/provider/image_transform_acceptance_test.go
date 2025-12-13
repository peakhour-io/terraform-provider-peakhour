package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccImageTransform_basic(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccEnv(t, "PEAKHOUR_TEST_DOMAIN")
	name := fmt.Sprintf("tfacc-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImageTransformConfig(domain, name, 200),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("peakhour_image_transform.test", "domain", domain),
					resource.TestCheckResourceAttr("peakhour_image_transform.test", "name", name),
					resource.TestCheckResourceAttrSet("peakhour_image_transform.test", "uuid"),
					resource.TestMatchResourceAttr("peakhour_image_transform.test", "config_json", regexp.MustCompile(`"width":200`)),
				),
			},
			{
				Config: testAccImageTransformConfig(domain, name, 300),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("peakhour_image_transform.test", "config_json", regexp.MustCompile(`"width":300`)),
				),
			},
			{
				ResourceName:      "peakhour_image_transform.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccImageTransformConfig(domain, name string, width int) string {
	return fmt.Sprintf(`
terraform {
  required_providers {
    peakhour = {
      source = "peakhour-io/peakhour"
    }
  }
}

provider "peakhour" {}

resource "peakhour_image_transform" "test" {
  domain = %q
  name   = %q

  config_json = jsonencode({
    width   = %d
    height  = 200
    fit     = "cover"
    quality = 80
  })
}
`, domain, name, width)
}
