terraform {
  required_providers {
    peakhour = {
      source = "peakhour-io/peakhour"
    }
  }
}

provider "peakhour" {
  # API key from environment: PEAKHOUR_API_KEY
}

# Setup: domain and service
resource "peakhour_domain" "example" {
  name = "example.com"
}

resource "peakhour_reverse_proxy_service" "example" {
  domain = peakhour_domain.example.name
}

# Configure SSL/TLS cipher profile
resource "peakhour_rp_ssl_config" "example" {
  domain  = peakhour_domain.example.name
  ciphers = "intermediate"

  depends_on = [peakhour_reverse_proxy_service.example]
}

# ACME settings (hostnames/SANs)
resource "peakhour_acme_settings" "example" {
  domain = peakhour_domain.example.name

  domain_names = [
    peakhour_domain.example.name,
    "www.example.com",
  ]

  depends_on = [peakhour_reverse_proxy_service.example]
}

# ACME certificate status (and optional issuance trigger)
resource "peakhour_acme_certificate" "example" {
  domain = peakhour_domain.example.name

  # Optional: toggle to trigger issuance (async)
  # issue = true

  depends_on = [peakhour_acme_settings.example]
}

# Optional: upload a custom certificate/private key instead of ACME.
# WARNING: private key material is stored in Terraform state (sensitive) and
# the API does not return the private key (or certificate PEM), so drift cannot
# be automatically verified for those values.
#
# resource "peakhour_rp_ssl_certificate" "example" {
#   domain = peakhour_domain.example.name
#
#   certificate_pem = file("cert.pem")
#   private_key_pem = file("key.pem")
# }

