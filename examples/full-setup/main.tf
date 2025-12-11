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

# Create domain
resource "peakhour_domain" "mysite" {
  name = "mysite.com"
}

# Enable reverse proxy (CDN) service
resource "peakhour_reverse_proxy_service" "mysite" {
  domain = peakhour_domain.mysite.name
}

# Configure reverse proxy
# Note: This resource supports partial updates. Fields not specified here will retain their
# existing values on the server. To reset a field, it must be explicitly defined.
resource "peakhour_reverse_proxy_config" "mysite" {
  domain = peakhour_domain.mysite.name

  # Compression
  gzip   = true
  # brotli = true # Commented out to demonstrate partial update - will persist previous value

  # WebSocket support
  websocket = true

  # Domain aliases

  aliases = [
    "www.mysite.com",
    "cdn.mysite.com"
  ]

  # Optional: Redirect configuration
  # redirect_mode        = "all"
  # redirect_location    = "https://newsite.com"
  # redirect_status_code = 301

  depends_on = [peakhour_reverse_proxy_service.mysite]
}

# Example Firewall Rule
resource "peakhour_rule" "block_admin" {
  domain     = peakhour_domain.mysite.name
  name       = "Block Admin Access"
  phase      = "firewall"
  filter_str = "http.request.uri.path matches \"^/admin/\""
  enabled    = true

  actions_json = jsonencode({
    firewall = [{
      type   = "firewall"
      action = "deny"
      reason = "Admin access restricted"
    }]
  })
}

# Add origin pool with load balancing
resource "peakhour_origin_pool" "backend" {
  domain = peakhour_domain.mysite.name
  tag    = "production"

  # Multiple backend servers
  address {
    address = "backend1.internal:8080"
    weight  = 100
  }

  address {
    address = "backend2.internal:8080"
    weight  = 100
  }

  address {
    address = "backend3.internal:8080"
    weight  = 50
  }

  # Shield configuration
  shield_name = "sydney"

  # Load balancing settings
  load_balancing_mode             = "round_robin"
  load_balancing_overload_percent = 150

  depends_on = [peakhour_reverse_proxy_service.mysite]
}

# Configure transforms (image optimization, HTML processing)
resource "peakhour_transform_settings" "mysite" {
  domain = peakhour_domain.mysite.name

  # HTML transforms
  transform_html         = true
  transform_lazy_sizes   = true
  transform_mixed_content = true

  # Image optimization
  transform_image_api      = true
  transform_image_optimise = true
  transform_image_quality  = 85
  transform_image_format   = true

  # Advanced features
  transform_esi = false

  depends_on = [peakhour_reverse_proxy_service.mysite]
}

# Data source example: look up existing domain
data "peakhour_domain" "existing" {
  name = "existing-domain.com"
}

output "domain_id" {
  value       = peakhour_domain.mysite.id
  description = "The domain identifier"
}

output "existing_domain_id" {
  value       = data.peakhour_domain.existing.id
  description = "Existing domain identifier"
}
