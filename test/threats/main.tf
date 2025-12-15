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

resource "peakhour_rp_threat_access_list_rule" "office_ip" {
  domain      = peakhour_domain.example.name
  rule_type   = "whitelist"
  content     = "203.0.113.10"
  description = "Allow office IP"

  depends_on = [peakhour_reverse_proxy_service.example]
}

resource "peakhour_rp_threat_block_list" "example" {
  domain     = peakhour_domain.example.name
  blocklists = ["tor"]

  depends_on = [peakhour_reverse_proxy_service.example]
}

