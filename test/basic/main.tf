terraform {
  required_providers {
    peakhour = {
      source = "peakhour-io/peakhour"
    }
  }
}

provider "peakhour" {
  # API key can be set via PEAKHOUR_API_KEY environment variable
  # or explicitly here:
  # api_key = "your-api-key"
}

# Create a domain
resource "peakhour_domain" "example" {
  name = "example.com"
}

# Assign a plan to the domain
resource "peakhour_domain_plan" "example" {
  domain = peakhour_domain.example.name
  code   = "basic"
}

# Enable reverse proxy service
resource "peakhour_reverse_proxy_service" "example" {
  domain     = peakhour_domain.example.name
  depends_on = [peakhour_domain_plan.example]
}

# Add origin pool
resource "peakhour_origin_pool" "backend" {
  domain = peakhour_domain.example.name
  tag    = "production"

  address = [
    {
      address = "192.0.2.1:8080"
      weight  = 100
    },
    {
      address = "192.0.2.2:8080"
      weight  = 100
    },
  ]

  shield_name         = "sydney"
  load_balancing_mode = "round_robin"

  depends_on = [peakhour_reverse_proxy_service.example]
}
