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

