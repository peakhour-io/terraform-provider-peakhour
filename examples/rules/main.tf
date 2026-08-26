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

# Setup: resources referenced by rules (zones, pools, bulk redirects)
resource "peakhour_rate_limit_zone" "api_zone" {
  domain = peakhour_domain.example.name
  name   = "api_limit"

  requests_max          = 100
  requests_interval_sec = 60
  block_duration_sec    = 300

  depends_on = [peakhour_reverse_proxy_service.example]
}

resource "peakhour_origin_pool" "api_backend" {
  domain = peakhour_domain.example.name
  tag    = "api-backend"

  address = [
    {
      address = "http://192.0.2.10:8080" # Use URL format: http://host:port or https://host:port
      weight  = 100
    }
  ]

  depends_on = [peakhour_reverse_proxy_service.example]
}

resource "peakhour_bulk_redirect_list" "legacy" {
  domain = peakhour_domain.example.name
  name   = "legacy_redirects"

  depends_on = [peakhour_reverse_proxy_service.example]
}

resource "peakhour_bulk_redirect_entry" "legacy_home" {
  domain             = peakhour_domain.example.name
  bulk_redirect_uuid = peakhour_bulk_redirect_list.legacy.uuid

  source_path = "/old-home"
  target_url  = "https://example.com/new-home"
  status_code = 301
}

# Example 1: Firewall rule - block specific paths
resource "peakhour_rule" "block_admin" {
  domain     = peakhour_domain.example.name
  phase      = "firewall"
  name       = "Block Admin Access"
  filter_str = "http.request.uri.path matches \"^/admin/\""
  enabled    = true

  actions_json = jsonencode({
    firewall = [{
      type   = "firewall"
      action = "deny"
      reason = "Admin access denied"
    }]
  })

  depends_on = [peakhour_reverse_proxy_service.example]
}

# Example 2: Rate limit API endpoints
resource "peakhour_rule" "ratelimit_api" {
  domain     = peakhour_domain.example.name
  phase      = "rate_limit_request"
  name       = "API Rate Limit"
  filter_str = "http.request.uri.path matches \"^/api/\""
  enabled    = true

  actions_json = jsonencode({
    rate_limit_request = [{
      type                          = "rate_limit_request"
      check_zone                    = peakhour_rate_limit_zone.api_zone.name
      check_zone_action             = "block"
      check_zone_action_status_code = 429
      zone_key                      = ["ip"]
    }]
  })

  depends_on = [peakhour_rate_limit_zone.api_zone]
}

# Example 2b: Rate limit (late phase)
resource "peakhour_rule" "ratelimit_api_late" {
  domain     = peakhour_domain.example.name
  phase      = "rate_limit_request_late"
  name       = "API Rate Limit (Late)"
  filter_str = "http.request.uri.path matches \"^/api/\""
  enabled    = true

  actions_json = jsonencode({
    rate_limit_request_late = [{
      type                          = "rate_limit_request_late"
      check_zone                    = peakhour_rate_limit_zone.api_zone.name
      check_zone_action             = "block"
      check_zone_action_status_code = 429
      zone_key                      = ["ip"]
    }]
  })

  depends_on = [peakhour_rate_limit_zone.api_zone]
}

# Example 2c: Rate limit on responses (track 50x errors)
resource "peakhour_rule" "ratelimit_errors" {
  domain     = peakhour_domain.example.name
  phase      = "rate_limit_response"
  name       = "Rate Limit 50x Responses"
  filter_str = "http.response.status_code >= 500"
  enabled    = true

  actions_json = jsonencode({
    rate_limit_response = [{
      type     = "rate_limit_response"
      add_zone = peakhour_rate_limit_zone.api_zone.name
      zone_key = ["ip"]
    }]
  })

  depends_on = [peakhour_rate_limit_zone.api_zone]
}

# Example 3: Custom request headers
resource "peakhour_rule" "add_headers" {
  domain     = peakhour_domain.example.name
  phase      = "request_headers"
  name       = "Add Custom Headers"
  filter_str = "http.request.uri.path matches \"^/api/\""
  enabled    = true

  actions_json = jsonencode({
    header = [{
      type = "header"
      set_headers = {
        "X-Custom-Header" = "MyValue"
        "X-Environment"   = "Production"
      }
      remove_headers = ["X-Powered-By"]
    }]
  })

  depends_on = [peakhour_reverse_proxy_service.example]
}

# Example 3b: Custom response headers
resource "peakhour_rule" "add_response_headers" {
  domain     = peakhour_domain.example.name
  phase      = "response_headers"
  name       = "Add Response Headers"
  filter_str = "http.request.uri.path matches \"^/api/\""
  enabled    = true

  actions_json = jsonencode({
    header = [{
      type = "header"
      set_headers = {
        "X-Edge-Provider" = "Peakhour"
      }
    }]
  })

  depends_on = [peakhour_reverse_proxy_service.example]
}

# Example 4: URL/Config override - force caching for specific paths
resource "peakhour_rule" "cache_images" {
  domain     = peakhour_domain.example.name
  phase      = "url_config"
  name       = "Cache Images"
  filter_str = "http.request.uri.path matches \"\\.(jpg|png|gif|webp)$\""
  enabled    = true

  actions_json = jsonencode({
    vconf = [{
      type              = "vconf"
      force_cache       = true
      cache_enabled     = true
      edge_ttl_sec      = 86400
      browser_ttl_sec   = 3600
      continue_on_match = false
    }]
  })

  depends_on = [peakhour_reverse_proxy_service.example]
}

# Example 5: Origin selection - route to different backend
resource "peakhour_rule" "route_api" {
  domain     = peakhour_domain.example.name
  phase      = "load_balance"
  name       = "Route API to API Backend"
  filter_str = "http.request.uri.path matches \"^/api/\""
  enabled    = true

  actions_json = jsonencode({
    origin_selection = [{
      type     = "origin_selection"
      set_pool = peakhour_origin_pool.api_backend.tag
    }]
  })

  depends_on = [peakhour_origin_pool.api_backend]
}

# Example 6: Request rewrite - modify URL before sending to origin
resource "peakhour_rule" "rewrite_legacy" {
  domain     = peakhour_domain.example.name
  phase      = "request_rewrite"
  name       = "Rewrite Legacy URLs"
  filter_str = "http.request.uri.path matches \"^/old/\""
  enabled    = true

  actions_json = jsonencode({
    request_rewrite = [{
      type    = "request_rewrite"
      set_uri = "/new/"
    }]
  })

  depends_on = [peakhour_reverse_proxy_service.example]
}

# Example 6b: Bulk redirects (use a redirect list)
resource "peakhour_rule" "bulk_redirects" {
  domain     = peakhour_domain.example.name
  phase      = "bulk_redirect"
  name       = "Bulk Redirects"
  filter_str = "http.request.uri.path matches \"^/old-\""
  enabled    = true

  actions_json = jsonencode({
    redirect = [{
      type      = "redirect"
      from_list = peakhour_bulk_redirect_list.legacy.name
    }]
  })

  depends_on = [peakhour_bulk_redirect_entry.legacy_home]
}

# Example 7: Challenge bot traffic
resource "peakhour_rule" "challenge_bots" {
  domain     = peakhour_domain.example.name
  phase      = "firewall"
  name       = "Challenge Suspicious Bots"
  filter_str = "http.user_agent matches \"(?i)(bot|crawler|spider)\" and not cf.bot_management.verified_bot"
  enabled    = true

  actions_json = jsonencode({
    firewall = [{
      type   = "firewall"
      action = "challenge"
      reason = "Bot verification required"
    }]
  })

  depends_on = [peakhour_reverse_proxy_service.example]
}

# Example 8: Cache control - add cache tags
resource "peakhour_rule" "cache_tags" {
  domain     = peakhour_domain.example.name
  phase      = "url_config"
  name       = "Add Cache Tags"
  filter_str = "http.request.uri.path matches \"^/products/\""
  enabled    = true

  actions_json = jsonencode({
    cache = [{
      type     = "cache"
      add_tags = ["products", "catalog"]
    }]
  })

  depends_on = [peakhour_reverse_proxy_service.example]
}

# Example 9: Order rules within a phase
#
# By default, peakhour_rule_phase_order expects the full order for the phase
# (it will detect out-of-band rule additions/removals as drift).
# To manage only the relative order of the listed rules, set include_all_rules=false.
resource "peakhour_rule_phase_order" "firewall" {
  domain = peakhour_domain.example.name
  phase  = "firewall"

  order = [
    peakhour_rule.block_admin.uuid,
    peakhour_rule.challenge_bots.uuid,
  ]
}

resource "peakhour_rule_phase_order" "url_config" {
  domain = peakhour_domain.example.name
  phase  = "url_config"

  order = [
    peakhour_rule.cache_images.uuid,
    peakhour_rule.cache_tags.uuid,
  ]
}

output "firewall_rule_uuid" {
  value       = peakhour_rule.block_admin.uuid
  description = "UUID of the firewall rule"
}
