package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRule_firewall(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccEnv(t, "PEAKHOUR_TEST_DOMAIN")
	name := fmt.Sprintf("tfacc-%s", acctest.RandString(10))
	nameUpdated := fmt.Sprintf("tfacc-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRuleConfigFirewall(domain, name, true, "deny"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("peakhour_rule.test", "domain", domain),
					resource.TestCheckResourceAttr("peakhour_rule.test", "phase", "firewall"),
					resource.TestCheckResourceAttr("peakhour_rule.test", "name", name),
					resource.TestCheckResourceAttrSet("peakhour_rule.test", "uuid"),
					resource.TestMatchResourceAttr("peakhour_rule.test", "actions_json", regexp.MustCompile(`"type":"firewall"`)),
				),
			},
			{
				Config: testAccRuleConfigFirewall(domain, nameUpdated, false, "challenge"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("peakhour_rule.test", "name", nameUpdated),
					resource.TestCheckResourceAttr("peakhour_rule.test", "enabled", "false"),
					resource.TestMatchResourceAttr("peakhour_rule.test", "actions_json", regexp.MustCompile(`"action":"challenge"`)),
				),
			},
			{
				ResourceName:      "peakhour_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccRule_requestHeaders(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccEnv(t, "PEAKHOUR_TEST_DOMAIN")
	name := fmt.Sprintf("tfacc-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRuleConfigHeaders(domain, "request_headers", name, true, "1"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("peakhour_rule.test", "phase", "request_headers"),
					resource.TestCheckResourceAttrSet("peakhour_rule.test", "uuid"),
					resource.TestMatchResourceAttr("peakhour_rule.test", "actions_json", regexp.MustCompile(`"X-TFACC":"1"`)),
				),
			},
			{
				Config: testAccRuleConfigHeaders(domain, "request_headers", name, true, "2"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("peakhour_rule.test", "actions_json", regexp.MustCompile(`"X-TFACC":"2"`)),
				),
			},
			{
				ResourceName:      "peakhour_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccRule_responseHeaders(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccEnv(t, "PEAKHOUR_TEST_DOMAIN")
	name := fmt.Sprintf("tfacc-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRuleConfigHeaders(domain, "response_headers", name, true, "1"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("peakhour_rule.test", "phase", "response_headers"),
					resource.TestCheckResourceAttrSet("peakhour_rule.test", "uuid"),
					resource.TestMatchResourceAttr("peakhour_rule.test", "actions_json", regexp.MustCompile(`"X-TFACC":"1"`)),
				),
			},
			{
				ResourceName:      "peakhour_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccRule_urlConfig(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccEnv(t, "PEAKHOUR_TEST_DOMAIN")
	name := fmt.Sprintf("tfacc-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRuleConfigVConf(domain, name, true, 60),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("peakhour_rule.test", "phase", "url_config"),
					resource.TestCheckResourceAttrSet("peakhour_rule.test", "uuid"),
					resource.TestMatchResourceAttr("peakhour_rule.test", "actions_json", regexp.MustCompile(`"edge_ttl_sec":60`)),
				),
			},
			{
				Config: testAccRuleConfigVConf(domain, name, true, 120),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("peakhour_rule.test", "actions_json", regexp.MustCompile(`"edge_ttl_sec":120`)),
				),
			},
			{
				ResourceName:      "peakhour_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccRule_requestRewrite(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccEnv(t, "PEAKHOUR_TEST_DOMAIN")
	name := fmt.Sprintf("tfacc-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRuleConfigRequestRewrite(domain, name, true, "/tfacc-rewrite-1"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("peakhour_rule.test", "phase", "request_rewrite"),
					resource.TestCheckResourceAttrSet("peakhour_rule.test", "uuid"),
					resource.TestMatchResourceAttr("peakhour_rule.test", "actions_json", regexp.MustCompile(`"set_uri":"/tfacc-rewrite-1"`)),
				),
			},
			{
				Config: testAccRuleConfigRequestRewrite(domain, name, true, "/tfacc-rewrite-2"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("peakhour_rule.test", "actions_json", regexp.MustCompile(`"set_uri":"/tfacc-rewrite-2"`)),
				),
			},
			{
				ResourceName:      "peakhour_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccRule_loadBalance(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccEnv(t, "PEAKHOUR_TEST_DOMAIN")
	name := fmt.Sprintf("tfacc-%s", acctest.RandString(10))
	poolTag := fmt.Sprintf("tfacc-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRuleConfigOriginSelection(domain, name, true, poolTag),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("peakhour_rule.test", "phase", "load_balance"),
					resource.TestCheckResourceAttrSet("peakhour_rule.test", "uuid"),
					resource.TestMatchResourceAttr("peakhour_rule.test", "actions_json", regexp.MustCompile(regexp.QuoteMeta(fmt.Sprintf(`"set_pool":"%s"`, poolTag)))),
				),
			},
			{
				ResourceName:      "peakhour_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccRule_rateLimitRequest(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccEnv(t, "PEAKHOUR_TEST_DOMAIN")
	name := fmt.Sprintf("tfacc-%s", acctest.RandString(10))
	zoneName := fmt.Sprintf("tfacc-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRuleConfigRateLimitRequest(domain, name, true, zoneName, "block"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("peakhour_rule.test", "phase", "rate_limit_request"),
					resource.TestCheckResourceAttrSet("peakhour_rule.test", "uuid"),
					resource.TestMatchResourceAttr("peakhour_rule.test", "actions_json", regexp.MustCompile(regexp.QuoteMeta(fmt.Sprintf(`"check_zone":"%s"`, zoneName)))),
				),
			},
			{
				Config: testAccRuleConfigRateLimitRequest(domain, name, true, zoneName, "challenge"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("peakhour_rule.test", "actions_json", regexp.MustCompile(`"check_zone_action":"challenge"`)),
				),
			},
			{
				ResourceName:      "peakhour_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccRule_rateLimitRequestLate(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccEnv(t, "PEAKHOUR_TEST_DOMAIN")
	name := fmt.Sprintf("tfacc-%s", acctest.RandString(10))
	zoneName := fmt.Sprintf("tfacc-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRuleConfigRateLimitRequestLate(domain, name, true, zoneName, "block"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("peakhour_rule.test", "phase", "rate_limit_request_late"),
					resource.TestCheckResourceAttrSet("peakhour_rule.test", "uuid"),
					resource.TestMatchResourceAttr("peakhour_rule.test", "actions_json", regexp.MustCompile(regexp.QuoteMeta(fmt.Sprintf(`"check_zone":"%s"`, zoneName)))),
				),
			},
			{
				Config: testAccRuleConfigRateLimitRequestLate(domain, name, true, zoneName, "log"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("peakhour_rule.test", "actions_json", regexp.MustCompile(`"check_zone_action":"log"`)),
				),
			},
			{
				ResourceName:      "peakhour_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccRule_rateLimitResponse(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccEnv(t, "PEAKHOUR_TEST_DOMAIN")
	name := fmt.Sprintf("tfacc-%s", acctest.RandString(10))
	zoneName := fmt.Sprintf("tfacc-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRuleConfigRateLimitResponse(domain, name, true, zoneName, []string{"ip"}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("peakhour_rule.test", "phase", "rate_limit_response"),
					resource.TestCheckResourceAttrSet("peakhour_rule.test", "uuid"),
					resource.TestMatchResourceAttr("peakhour_rule.test", "actions_json", regexp.MustCompile(regexp.QuoteMeta(fmt.Sprintf(`"add_zone":"%s"`, zoneName)))),
				),
			},
			{
				Config: testAccRuleConfigRateLimitResponse(domain, name, true, zoneName, []string{"country"}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("peakhour_rule.test", "actions_json", regexp.MustCompile(`"zone_key":\\[\"country\"\\]`)),
				),
			},
			{
				ResourceName:      "peakhour_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccConfigHeader() string {
	return `
terraform {
  required_providers {
    peakhour = {
      source = "peakhour-io/peakhour"
    }
  }
}

provider "peakhour" {}
`
}

func testAccRuleConfigFirewall(domain, name string, enabled bool, action string) string {
	filter := `http.request.uri.path matches "^/tfacc/"`

	return fmt.Sprintf(`
%s

resource "peakhour_rule" "test" {
  domain     = %q
  phase      = "firewall"
  name       = %q
  filter_str = %q
  enabled    = %t

  actions_json = jsonencode({
    firewall = [{
      type   = "firewall"
      action = %q
      reason = "tfacc"
    }]
  })
}
`, testAccConfigHeader(), domain, name, filter, enabled, action)
}

func testAccRuleConfigHeaders(domain, phase, name string, enabled bool, headerValue string) string {
	filter := `http.request.uri.path matches "^/tfacc/headers"`

	return fmt.Sprintf(`
%s

resource "peakhour_rule" "test" {
  domain     = %q
  phase      = %q
  name       = %q
  filter_str = %q
  enabled    = %t

  actions_json = jsonencode({
    header = [{
      type = "header"
      set_headers = {
        "X-TFACC" = %q
      }
    }]
  })
}
`, testAccConfigHeader(), domain, phase, name, filter, enabled, headerValue)
}

func testAccRuleConfigVConf(domain, name string, enabled bool, edgeTTL int) string {
	filter := `http.request.uri.path matches "^/tfacc/cache"`

	return fmt.Sprintf(`
%s

resource "peakhour_rule" "test" {
  domain     = %q
  phase      = "url_config"
  name       = %q
  filter_str = %q
  enabled    = %t

  actions_json = jsonencode({
    vconf = [{
      type            = "vconf"
      cache_enabled   = true
      force_cache     = true
      edge_ttl_sec    = %d
      browser_ttl_sec = 60
    }]
  })
}
`, testAccConfigHeader(), domain, name, filter, enabled, edgeTTL)
}

func testAccRuleConfigRequestRewrite(domain, name string, enabled bool, uri string) string {
	filter := `http.request.uri.path matches "^/tfacc/rewrite"`

	return fmt.Sprintf(`
%s

resource "peakhour_rule" "test" {
  domain     = %q
  phase      = "request_rewrite"
  name       = %q
  filter_str = %q
  enabled    = %t

  actions_json = jsonencode({
    request_rewrite = [{
      type    = "request_rewrite"
      set_uri = %q
    }]
  })
}
`, testAccConfigHeader(), domain, name, filter, enabled, uri)
}

func testAccRuleConfigOriginSelection(domain, name string, enabled bool, poolTag string) string {
	filter := `http.request.uri.path matches "^/tfacc/origin"`

	return fmt.Sprintf(`
%s

resource "peakhour_origin_pool" "pool" {
  domain = %q
  tag    = %q

  address = [{
    address = "192.0.2.10:8080"
    weight  = 100
  }]
}

resource "peakhour_rule" "test" {
  domain     = %q
  phase      = "load_balance"
  name       = %q
  filter_str = %q
  enabled    = %t

  actions_json = jsonencode({
    origin_selection = [{
      type     = "origin_selection"
      set_pool = peakhour_origin_pool.pool.tag
    }]
  })
}
`, testAccConfigHeader(), domain, poolTag, domain, name, filter, enabled)
}

func testAccRuleConfigRateLimitRequest(domain, name string, enabled bool, zoneName, action string) string {
	filter := `http.request.uri.path matches "^/tfacc/rl/request"`

	return fmt.Sprintf(`
%s

resource "peakhour_rate_limit_zone" "zone" {
  domain = %q
  name   = %q

  requests_max          = 10
  requests_interval_sec = 60
  block_duration_sec    = 60
}

resource "peakhour_rule" "test" {
  domain     = %q
  phase      = "rate_limit_request"
  name       = %q
  filter_str = %q
  enabled    = %t

  actions_json = jsonencode({
    rate_limit_request = [{
      type                          = "rate_limit_request"
      check_zone                    = peakhour_rate_limit_zone.zone.name
      check_zone_action             = %q
      check_zone_action_status_code = 429
      zone_key                      = ["ip"]
    }]
  })
}
`, testAccConfigHeader(), domain, zoneName, domain, name, filter, enabled, action)
}

func testAccRuleConfigRateLimitRequestLate(domain, name string, enabled bool, zoneName, action string) string {
	filter := `http.request.uri.path matches "^/tfacc/rl/late"`

	return fmt.Sprintf(`
%s

resource "peakhour_rate_limit_zone" "zone" {
  domain = %q
  name   = %q

  requests_max          = 10
  requests_interval_sec = 60
  block_duration_sec    = 60
}

resource "peakhour_rule" "test" {
  domain     = %q
  phase      = "rate_limit_request_late"
  name       = %q
  filter_str = %q
  enabled    = %t

  actions_json = jsonencode({
    rate_limit_request_late = [{
      type                          = "rate_limit_request_late"
      check_zone                    = peakhour_rate_limit_zone.zone.name
      check_zone_action             = %q
      check_zone_action_status_code = 429
      zone_key                      = ["ip"]
    }]
  })
}
`, testAccConfigHeader(), domain, zoneName, domain, name, filter, enabled, action)
}

func testAccRuleConfigRateLimitResponse(domain, name string, enabled bool, zoneName string, zoneKeys []string) string {
	filter := `http.request.uri.path matches "^/tfacc/rl/response"`

	zoneKeysHCL := ""
	for i, key := range zoneKeys {
		if i > 0 {
			zoneKeysHCL += ", "
		}
		zoneKeysHCL += fmt.Sprintf("%q", key)
	}

	return fmt.Sprintf(`
%s

resource "peakhour_rate_limit_zone" "zone" {
  domain = %q
  name   = %q

  response_errors_max          = 10
  response_errors_interval_sec = 60
  block_duration_sec           = 60
}

resource "peakhour_rule" "test" {
  domain     = %q
  phase      = "rate_limit_response"
  name       = %q
  filter_str = %q
  enabled    = %t

  actions_json = jsonencode({
    rate_limit_response = [{
      type     = "rate_limit_response"
      add_zone = peakhour_rate_limit_zone.zone.name
      zone_key = [%s]
    }]
  })
}
`, testAccConfigHeader(), domain, zoneName, domain, name, filter, enabled, zoneKeysHCL)
}
