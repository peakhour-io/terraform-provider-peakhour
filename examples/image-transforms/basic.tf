# Create a domain
resource "peakhour_domain" "example" {
  name = "example.com"
}

# Assign a plan to the domain
resource "peakhour_domain_plan" "example" {
  domain = peakhour_domain.example.name
  code   = "business2"
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
      address = "http://192.0.2.1:8080"  # Use URL format for addresses with ports
      weight  = 100
    },
    {
      address = "http://192.0.2.2:8080"
      weight  = 100
    },
  ]

  # shield_name = "sydney"  # Must be a valid shield location from /api/v1/shields
  load_balancing_mode = "round_robin"  # Valid: none, round_robin, map, consistent

  depends_on = [peakhour_reverse_proxy_service.example]
}
