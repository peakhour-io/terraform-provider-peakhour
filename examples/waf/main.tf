terraform {
  required_providers {
    peakhour = {
      source = "peakhour-io/peakhour"
    }
  }
}

provider "peakhour" {
  # API key can be set via PEAKHOUR_API_KEY environment variable
}

resource "peakhour_domain" "example" {
  name = "example.com"
}

resource "peakhour_reverse_proxy_service" "example" {
  domain = peakhour_domain.example.name
}

resource "peakhour_rp_waf_options" "example" {
  domain = peakhour_domain.example.name

  waf_mode                        = "enabled"
  waf_ruleset                     = "owaspv33"
  waf_set_exposed_password_header = true

  depends_on = [peakhour_reverse_proxy_service.example]
}

resource "peakhour_rp_waf_owasp_settings" "example" {
  domain = peakhour_domain.example.name

  # The OWASP settings schema is represented as JSON.
  # Omitted keys are left unchanged; explicit nulls clear values.
  settings_json = jsonencode({
    methods = {
      allowed_methods = ["GET", "HEAD", "POST", "OPTIONS"]
    }
    protocol = {
      max_num_args = 100
    }
  })

  depends_on = [peakhour_reverse_proxy_service.example]
}

resource "peakhour_rp_waf_custom_rule" "example" {
  domain = peakhour_domain.example.name

  description = "Example custom rule (pass action)"
  enabled     = true

  rules_json = jsonencode([
    {
      variable      = "REQUEST_HEADERS"
      variable_part = "user-agent"
      operator      = "@contains"
      operator_arg  = "curl"
    }
  ])

  action_json = jsonencode({
    action_name = "pass"
  })

  logging_json = jsonencode({
    message  = "matched tf custom rule"
    severity = "INFO"
    tags     = ["terraform"]
  })

  depends_on = [peakhour_reverse_proxy_service.example]
}

# Ordering is managed separately so each rule keeps its stable UUID identity.
# Subset mode permits rules to be added or removed in the same apply while the
# provider still sends the complete order required by the Peakhour API.
resource "peakhour_rp_waf_custom_rule_order" "example" {
  domain            = peakhour_domain.example.name
  include_all_rules = false
  order             = [peakhour_rp_waf_custom_rule.example.uuid]
}

# Disable a WAF rule group. Uncomment only after replacing file_name with a
# group returned for the selected ruleset; the placeholder is not universally
# available and must not make the runnable example fail.
# resource "peakhour_rp_waf_rule_group" "example" {
#   domain = peakhour_domain.example.name
#
#   ruleset   = "peakhour"
#   file_name = "replace-with-an-existing-rule-group"
#   enabled   = false
#
#   depends_on = [peakhour_reverse_proxy_service.example]
# }
