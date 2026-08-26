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

# Enable rate limiting modes for this domain
resource "peakhour_rate_limit_settings" "example" {
  domain = peakhour_domain.example.name
  mode   = ["zone", "global"]

  depends_on = [peakhour_reverse_proxy_service.example]
}

# Example 1: Create rate limit zone for API endpoints
resource "peakhour_rate_limit_zone" "api_zone" {
  domain = peakhour_domain.example.name
  name   = "api_limit"

  # Allow 100 requests per 60 seconds
  requests_max          = 100
  requests_interval_sec = 60

  # Block for 300 seconds when limit exceeded
  block_duration_sec = 300

  depends_on = [peakhour_reverse_proxy_service.example]
}

# Example 2: Create zone for login endpoints (stricter)
resource "peakhour_rate_limit_zone" "login_zone" {
  domain = peakhour_domain.example.name
  name   = "login_limit"

  # Allow only 5 login attempts per 300 seconds
  requests_max          = 5
  requests_interval_sec = 300

  # Block for 1 hour when limit exceeded
  block_duration_sec = 3600

  # Also limit concurrent connections
  connections_max          = 2
  connections_interval_sec = 60

  depends_on = [peakhour_reverse_proxy_service.example]
}

# Example 3: Zone for monitoring response errors
resource "peakhour_rate_limit_zone" "error_monitor" {
  domain = peakhour_domain.example.name
  name   = "error_tracking"

  # Track 50x errors
  response_errors_max          = 100
  response_errors_interval_sec = 300

  block_duration_sec = 600

  depends_on = [peakhour_reverse_proxy_service.example]
}

# Example 4: Global concurrent connections limit
resource "peakhour_rate_limit_global" "global" {
  domain = peakhour_domain.example.name

  # Maximum 1000 concurrent connections
  concurrent_connections = 1000

  depends_on = [peakhour_reverse_proxy_service.example]
}

# Example 5: Use zone in a rule to enforce API rate limiting
resource "peakhour_rule" "api_ratelimit" {
  domain     = peakhour_domain.example.name
  phase      = "rate_limit_request"
  name       = "API Rate Limit"
  filter_str = "http.request.uri.path matches \"^/api/\""
  enabled    = true

  actions_json = jsonencode({
    rate_limit_request = [{
      type                          = "rate_limit_request"
      check_zone                    = "api_limit"
      check_zone_action             = "block"
      check_zone_action_status_code = 429
      zone_key                      = ["ip"]
    }]
  })

  depends_on = [
    peakhour_reverse_proxy_service.example,
    peakhour_rate_limit_zone.api_zone
  ]
}

# Example 6: Login rate limiting with per-user tracking
resource "peakhour_rule" "login_ratelimit" {
  domain     = peakhour_domain.example.name
  phase      = "rate_limit_request"
  name       = "Login Rate Limit"
  filter_str = "http.request.uri.path == \"/api/login\" and http.request.method == \"POST\""
  enabled    = true

  actions_json = jsonencode({
    rate_limit_request = [{
      type                          = "rate_limit_request"
      check_zone                    = "login_limit"
      check_zone_action             = "block"
      check_zone_action_status_code = 429
      zone_key                      = ["ip", "header"]
      zone_key_headers              = ["X-User-Email"]
    }]
  })

  depends_on = [
    peakhour_reverse_proxy_service.example,
    peakhour_rate_limit_zone.login_zone
  ]
}

# Example 7: Rate limit by API key
resource "peakhour_rule" "api_key_limit" {
  domain     = peakhour_domain.example.name
  phase      = "rate_limit_request"
  name       = "API Key Rate Limit"
  filter_str = "http.request.uri.path matches \"^/api/v1/\""
  enabled    = true

  actions_json = jsonencode({
    rate_limit_request = [{
      type              = "rate_limit_request"
      check_zone        = "api_limit"
      check_zone_action = "challenge"
      zone_key          = ["header"]
      zone_key_headers  = ["X-API-Key"]
    }]
  })

  depends_on = [
    peakhour_reverse_proxy_service.example,
    peakhour_rate_limit_zone.api_zone
  ]
}

output "api_zone_name" {
  value       = peakhour_rate_limit_zone.api_zone.name
  description = "API rate limit zone name"
}

output "login_zone_name" {
  value       = peakhour_rate_limit_zone.login_zone.name
  description = "Login rate limit zone name"
}
