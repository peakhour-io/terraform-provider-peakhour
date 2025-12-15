# Complete Peakhour Terraform Configuration Example
#
# This example demonstrates all available resources for managing a domain
# through the Peakhour CDN/WAF platform.
#
# Usage:
#   export PEAKHOUR_API_KEY="your-api-key"
#   terraform init
#   terraform plan
#   terraform apply

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

# =============================================================================
# DOMAIN SETUP
# =============================================================================

# Create the domain
resource "peakhour_domain" "main" {
  name = "myapp.example.com"
}

# Assign a subscription plan (required before enabling services)
resource "peakhour_domain_plan" "main" {
  domain = peakhour_domain.main.name
  code   = "business2"
}

# Enable reverse proxy (CDN) service
resource "peakhour_reverse_proxy_service" "main" {
  domain     = peakhour_domain.main.name
  depends_on = [peakhour_domain_plan.main]
}

# =============================================================================
# REVERSE PROXY CONFIGURATION
# =============================================================================

# Main reverse proxy settings
resource "peakhour_reverse_proxy_config" "main" {
  domain = peakhour_domain.main.name

  # Compression
  gzip   = true
  brotli = true

  # WebSocket support
  websocket = true

  # Domain aliases
  aliases = [
    "www.myapp.example.com",
    "cdn.myapp.example.com"
  ]

  depends_on = [peakhour_reverse_proxy_service.main]
}

# RP service settings (notifications, quickstart)
resource "peakhour_rp_settings" "main" {
  domain = peakhour_domain.main.name

  notification_emails = ["ops@example.com", "security@example.com"]
  quickstart          = true

  depends_on = [peakhour_reverse_proxy_service.main]
}

# =============================================================================
# SSL/TLS CONFIGURATION
# =============================================================================

# SSL/TLS cipher profile
resource "peakhour_rp_ssl_config" "main" {
  domain  = peakhour_domain.main.name
  ciphers = "intermediate"

  depends_on = [peakhour_reverse_proxy_service.main]
}

# ACME settings (for managed certificates)
resource "peakhour_acme_settings" "main" {
  domain = peakhour_domain.main.name

  domain_names = [
    peakhour_domain.main.name,
    "www.myapp.example.com",
  ]

  depends_on = [peakhour_reverse_proxy_service.main]
}

# ACME certificate (triggers issuance)
resource "peakhour_acme_certificate" "main" {
  domain = peakhour_domain.main.name
  issue  = true

  depends_on = [peakhour_acme_settings.main]
}

# =============================================================================
# ORIGIN CONFIGURATION
# =============================================================================

# Origin behavior settings
resource "peakhour_rp_origin_config" "main" {
  domain = peakhour_domain.main.name

  ssl_mode = "https"

  origin_request_headers = {
    geoip = true
  }

  depends_on = [peakhour_reverse_proxy_service.main]
}

# Primary origin pool
resource "peakhour_origin_pool" "primary" {
  domain = peakhour_domain.main.name
  tag    = "primary"

  address = [
    {
      address = "http://origin1.example.com:8080"
      weight  = 100
    },
    {
      address = "http://origin2.example.com:8080"
      weight  = 100
    },
  ]

  shield_name                     = "sydney"
  load_balancing_mode             = "round_robin"
  load_balancing_overload_percent = 150

  depends_on = [peakhour_reverse_proxy_service.main]
}

# API backend pool
resource "peakhour_origin_pool" "api" {
  domain = peakhour_domain.main.name
  tag    = "api-backend"

  address = [
    {
      address = "https://api1.example.com:8443"
      weight  = 100
    },
    {
      address = "https://api2.example.com:8443"
      weight  = 100
    },
  ]

  load_balancing_mode = "round_robin"

  depends_on = [peakhour_reverse_proxy_service.main]
}

# =============================================================================
# CDN & CACHING
# =============================================================================

# CDN cache settings
resource "peakhour_rp_cdn_cache" "main" {
  domain = peakhour_domain.main.name

  cache_enabled  = true
  cdn_query_mode = "full"

  cdn_remove_query_args = ["utm_source", "utm_medium", "utm_campaign", "fbclid"]

  depends_on = [peakhour_reverse_proxy_service.main]
}

# Transform settings (image optimization, HTML processing)
# Note: Backend currently returns 500 error - needs backend fix
# resource "peakhour_transform_settings" "main" {
#   domain = peakhour_domain.main.name
#
#   # HTML transforms
#   transform_html          = true
#   transform_lazy_sizes    = true
#   transform_mixed_content = true
#
#   # Image optimization
#   transform_image_api      = true
#   transform_image_optimise = true
#   transform_image_quality  = 85
#   transform_image_format   = true
#
#   depends_on = [peakhour_reverse_proxy_service.main]
# }

# =============================================================================
# IMAGE TRANSFORMS
# =============================================================================

resource "peakhour_image_transform" "thumbnail" {
  domain = peakhour_domain.main.name
  name   = "thumbnail"
  config_json = jsonencode({
    size = {
      w   = 200
      h   = 200
      fit = "crop"
    }
    format = {
      q = 80
    }
  })

  depends_on = [peakhour_reverse_proxy_service.main]
}

resource "peakhour_image_transform" "hero" {
  domain = peakhour_domain.main.name
  name   = "hero"
  config_json = jsonencode({
    size = {
      w = 1200
    }
    format = {
      fm = "WEBP"
      q  = "auto"
    }
  })

  depends_on = [peakhour_reverse_proxy_service.main]
}

# Commit image transform changes
resource "peakhour_image_transform_commit" "main" {
  domain = peakhour_domain.main.name

  triggers = {
    thumbnail = peakhour_image_transform.thumbnail.id
    hero      = peakhour_image_transform.hero.id
  }
}

# =============================================================================
# FIREWALL & SECURITY
# =============================================================================

# Firewall settings
resource "peakhour_rp_firewall_settings" "main" {
  domain = peakhour_domain.main.name

  challenge_cookie_key = ["fingerprint_tls", "ip"]

  depends_on = [peakhour_reverse_proxy_service.main]
}

# Custom firewall error page
resource "peakhour_rp_firewall_error_page" "main" {
  domain = peakhour_domain.main.name

  content = <<-EOT
  <!DOCTYPE html>
  <html>
    <head>
      <title>Access Denied</title>
      <style>
        body { font-family: sans-serif; text-align: center; padding: 50px; }
        h1 { color: #e74c3c; }
      </style>
    </head>
    <body>
      <h1>Access Denied</h1>
      <p>Your request has been blocked by our security system.</p>
      <p>If you believe this is an error, please contact support.</p>
    </body>
  </html>
  EOT

  depends_on = [peakhour_reverse_proxy_service.main]
}

# Bot verification settings
resource "peakhour_rp_bots" "main" {
  domain = peakhour_domain.main.name

  bots_verify_list = ["google", "bing", "facebook"]
  bots_verify_rdns = true

  depends_on = [peakhour_reverse_proxy_service.main]
}

# Threat block lists
resource "peakhour_rp_threat_block_list" "main" {
  domain     = peakhour_domain.main.name
  blocklists = ["tor", "datacenter"]

  depends_on = [peakhour_reverse_proxy_service.main]
}

# Whitelist office IP
resource "peakhour_rp_threat_access_list_rule" "office" {
  domain      = peakhour_domain.main.name
  rule_type   = "whitelist"
  content     = "203.0.113.0/24"
  description = "Office network"

  depends_on = [peakhour_reverse_proxy_service.main]
}

# =============================================================================
# WAF (Web Application Firewall)
# =============================================================================

# WAF options
resource "peakhour_rp_waf_options" "main" {
  domain = peakhour_domain.main.name

  waf_mode                        = "enabled"
  waf_ruleset                     = "owaspv33"
  waf_set_exposed_password_header = true

  depends_on = [peakhour_reverse_proxy_service.main]
}

# WAF OWASP settings
resource "peakhour_rp_waf_owasp_settings" "main" {
  domain = peakhour_domain.main.name

  settings_json = jsonencode({
    initialization = {
      inbound_anomaly_score_threshold = 5
      blocking_paranoia_level         = 1
    }
    methods = {
      allowed_methods = ["GET", "HEAD", "POST", "PUT", "DELETE", "OPTIONS"]
    }
    protocol = {
      max_num_args = 100
    }
  })

  depends_on = [peakhour_reverse_proxy_service.main]
}

# Custom WAF rule - block SQL injection attempts in query params
resource "peakhour_rp_waf_custom_rule" "sql_injection" {
  domain = peakhour_domain.main.name

  name        = "Custom SQL Injection Detection"
  description = "Block common SQL injection patterns"
  enabled     = true

  rules_json = jsonencode([
    {
      variable     = "ARGS"
      operator     = "@detectSQLi"
      operator_arg = ""
    }
  ])

  action_json = jsonencode({
    action_name = "deny"
  })

  logging_json = jsonencode({
    message  = "SQL injection attempt blocked"
    severity = "CRITICAL"
    tags     = ["sqli", "attack"]
  })

  depends_on = [peakhour_reverse_proxy_service.main]
}

# Custom WAF rule - protect login endpoint
resource "peakhour_rp_waf_custom_rule" "protect_login" {
  domain = peakhour_domain.main.name

  name        = "Login Protection"
  description = "Extra protection for login endpoint"
  enabled     = true

  rules_json = jsonencode([
    {
      variable     = "REQUEST_URI"
      operator     = "@beginsWith"
      operator_arg = "/api/login"
    },
    {
      variable     = "REQUEST_BODY"
      operator     = "@rx"
      operator_arg = "(?i)(union|select|drop|insert|delete|update|exec|script)"
    }
  ])

  action_json = jsonencode({
    action_name = "deny"
  })

  logging_json = jsonencode({
    message  = "Suspicious login attempt"
    severity = "WARNING"
    tags     = ["login", "security"]
  })

  depends_on = [peakhour_reverse_proxy_service.main]
}

# Disable a peakhour rule group
resource "peakhour_rp_waf_rule_group" "disable_example" {
  domain = peakhour_domain.main.name

  ruleset   = "peakhour"
  file_name = "peakhour-waf-rules"
  enabled   = false

  depends_on = [peakhour_reverse_proxy_service.main]
}

# =============================================================================
# RATE LIMITING
# =============================================================================

# Enable rate limiting
resource "peakhour_rate_limit_settings" "main" {
  domain = peakhour_domain.main.name
  mode   = ["zone", "global"]

  depends_on = [peakhour_reverse_proxy_service.main]
}

# Global connection limit
resource "peakhour_rate_limit_global" "main" {
  domain = peakhour_domain.main.name

  concurrent_connections = 1000

  depends_on = [peakhour_reverse_proxy_service.main]
}

# API rate limit zone
resource "peakhour_rate_limit_zone" "api" {
  domain = peakhour_domain.main.name
  name   = "api_limit"

  requests_max          = 100
  requests_interval_sec = 60
  block_duration_sec    = 300

  depends_on = [peakhour_reverse_proxy_service.main]
}

# Login rate limit zone (stricter)
resource "peakhour_rate_limit_zone" "login" {
  domain = peakhour_domain.main.name
  name   = "login_limit"

  requests_max          = 5
  requests_interval_sec = 300
  block_duration_sec    = 3600

  connections_max          = 2
  connections_interval_sec = 60

  depends_on = [peakhour_reverse_proxy_service.main]
}

# =============================================================================
# RULE LISTS
# =============================================================================

# IP blocklist
resource "peakhour_rule_list" "blocked_ips" {
  domain = peakhour_domain.main.name
  name   = "blocked_ips"
  type   = "ip"

  ips = [
    "192.0.2.0/24",
    "198.51.100.0/24",
  ]

  depends_on = [peakhour_reverse_proxy_service.main]
}

# Admin IP allowlist
resource "peakhour_rule_list" "admin_ips" {
  domain = peakhour_domain.main.name
  name   = "admin_allowed_ips"
  type   = "ip"

  ips = [
    "203.0.113.0/24",
  ]

  depends_on = [peakhour_reverse_proxy_service.main]
}

# Bad user agents
resource "peakhour_rule_list" "bad_agents" {
  domain = peakhour_domain.main.name
  name   = "blocked_user_agents"
  type   = "string"

  strs = [
    "BadBot",
    "EvilScraper",
    "MaliciousCrawler",
  ]

  depends_on = [peakhour_reverse_proxy_service.main]
}

# =============================================================================
# BULK REDIRECTS
# =============================================================================

resource "peakhour_bulk_redirect_list" "legacy" {
  domain = peakhour_domain.main.name
  name   = "legacy_redirects"

  depends_on = [peakhour_reverse_proxy_service.main]
}

resource "peakhour_bulk_redirect_entry" "old_home" {
  domain             = peakhour_domain.main.name
  bulk_redirect_uuid = peakhour_bulk_redirect_list.legacy.uuid

  source_path = "/old-home"
  target_url  = "https://myapp.example.com/"
  status_code = 301
}

resource "peakhour_bulk_redirect_entry" "old_about" {
  domain             = peakhour_domain.main.name
  bulk_redirect_uuid = peakhour_bulk_redirect_list.legacy.uuid

  source_path = "/old-about"
  target_url  = "https://myapp.example.com/about"
  status_code = 301
}

# =============================================================================
# RULES
# =============================================================================

# Firewall: Block blocked IPs
resource "peakhour_rule" "block_ips" {
  domain     = peakhour_domain.main.name
  phase      = "firewall"
  name       = "Block Listed IPs"
  filter_str = "ip.src in $blocked_ips"
  enabled    = true

  actions_json = jsonencode({
    firewall = [{
      type   = "firewall"
      action = "deny"
      reason = "IP blocklisted"
    }]
  })

  depends_on = [peakhour_rule_list.blocked_ips]
}

# Firewall: Admin IP whitelist
resource "peakhour_rule" "admin_whitelist" {
  domain     = peakhour_domain.main.name
  phase      = "firewall"
  name       = "Admin IP Whitelist"
  filter_str = "http.request.uri.path matches \"^/admin/\" and not ip.src in $admin_allowed_ips"
  enabled    = true

  actions_json = jsonencode({
    firewall = [{
      type   = "firewall"
      action = "deny"
      reason = "Admin access restricted"
    }]
  })

  depends_on = [peakhour_rule_list.admin_ips]
}

# Firewall: Block bad user agents
resource "peakhour_rule" "block_agents" {
  domain     = peakhour_domain.main.name
  phase      = "firewall"
  name       = "Block Bad User Agents"
  filter_str = "http.user_agent in $blocked_user_agents"
  enabled    = true

  actions_json = jsonencode({
    firewall = [{
      type   = "firewall"
      action = "deny"
      reason = "User agent blocklisted"
    }]
  })

  depends_on = [peakhour_rule_list.bad_agents]
}

# Rate limit: API endpoints
resource "peakhour_rule" "api_ratelimit" {
  domain     = peakhour_domain.main.name
  phase      = "rate_limit_request"
  name       = "API Rate Limit"
  filter_str = "http.request.uri.path matches \"^/api/\""
  enabled    = true

  actions_json = jsonencode({
    rate_limit_request = [{
      type                          = "rate_limit_request"
      check_zone                    = peakhour_rate_limit_zone.api.name
      check_zone_action             = "block"
      check_zone_action_status_code = 429
      zone_key                      = ["ip"]
    }]
  })

  depends_on = [peakhour_rate_limit_zone.api]
}

# Rate limit: Login endpoint
resource "peakhour_rule" "login_ratelimit" {
  domain     = peakhour_domain.main.name
  phase      = "rate_limit_request"
  name       = "Login Rate Limit"
  filter_str = "http.request.uri.path == \"/api/login\" and http.request.method == \"POST\""
  enabled    = true

  actions_json = jsonencode({
    rate_limit_request = [{
      type                          = "rate_limit_request"
      check_zone                    = peakhour_rate_limit_zone.login.name
      check_zone_action             = "block"
      check_zone_action_status_code = 429
      zone_key                      = ["ip"]
    }]
  })

  depends_on = [peakhour_rate_limit_zone.login]
}

# Request headers: Add custom headers
resource "peakhour_rule" "add_headers" {
  domain     = peakhour_domain.main.name
  phase      = "request_headers"
  name       = "Add Custom Headers"
  filter_str = "http.request.uri.path matches \"^/api/\""
  enabled    = true

  actions_json = jsonencode({
    header = [{
      type = "header"
      set_headers = {
        "X-Forwarded-Proto" = "https"
        "X-Real-IP"         = "{{ip.src}}"
      }
      remove_headers = ["X-Powered-By"]
    }]
  })

  depends_on = [peakhour_reverse_proxy_service.main]
}

# URL config: Cache static assets
resource "peakhour_rule" "cache_static" {
  domain     = peakhour_domain.main.name
  phase      = "url_config"
  name       = "Cache Static Assets"
  filter_str = "http.request.uri.path matches \"\\.(js|css|jpg|png|gif|webp|woff2|svg)$\""
  enabled    = true

  actions_json = jsonencode({
    vconf = [{
      type            = "vconf"
      force_cache     = true
      cache_enabled   = true
      edge_ttl_sec    = 86400
      browser_ttl_sec = 3600
    }]
  })

  depends_on = [peakhour_reverse_proxy_service.main]
}

# URL config: Disable cache for API
resource "peakhour_rule" "nocache_api" {
  domain     = peakhour_domain.main.name
  phase      = "url_config"
  name       = "No Cache for API"
  filter_str = "http.request.uri.path matches \"^/api/\""
  enabled    = true

  actions_json = jsonencode({
    vconf = [{
      type          = "vconf"
      cache_enabled = false
    }]
  })

  depends_on = [peakhour_reverse_proxy_service.main]
}

# Load balance: Route API to API backend
resource "peakhour_rule" "route_api" {
  domain     = peakhour_domain.main.name
  phase      = "load_balance"
  name       = "Route API to API Backend"
  filter_str = "http.request.uri.path matches \"^/api/\""
  enabled    = true

  actions_json = jsonencode({
    origin_selection = [{
      type     = "origin_selection"
      set_pool = peakhour_origin_pool.api.tag
    }]
  })

  depends_on = [peakhour_origin_pool.api]
}

# Bulk redirect: Legacy URLs
resource "peakhour_rule" "bulk_redirect" {
  domain     = peakhour_domain.main.name
  phase      = "bulk_redirect"
  name       = "Legacy Redirects"
  filter_str = "http.request.uri.path matches \"^/old-\""
  enabled    = true

  actions_json = jsonencode({
    redirect = [{
      type      = "redirect"
      from_list = peakhour_bulk_redirect_list.legacy.name
    }]
  })

  depends_on = [
    peakhour_bulk_redirect_entry.old_home,
    peakhour_bulk_redirect_entry.old_about
  ]
}

# =============================================================================
# RULE ORDERING
# =============================================================================

resource "peakhour_rule_phase_order" "firewall" {
  domain = peakhour_domain.main.name
  phase  = "firewall"

  # Set to false to only order the specified rules, ignoring any others
  include_all_rules = false

  order = [
    peakhour_rule.block_ips.uuid,
    peakhour_rule.admin_whitelist.uuid,
    peakhour_rule.block_agents.uuid,
  ]
}

resource "peakhour_rule_phase_order" "url_config" {
  domain = peakhour_domain.main.name
  phase  = "url_config"

  # Set to false to only order the specified rules, ignoring any others
  include_all_rules = false

  order = [
    peakhour_rule.nocache_api.uuid,
    peakhour_rule.cache_static.uuid,
  ]
}

# =============================================================================
# LUA (Advanced)
# =============================================================================

resource "peakhour_rp_lua_options" "main" {
  domain      = peakhour_domain.main.name
  lua_enabled = false

  depends_on = [peakhour_reverse_proxy_service.main]
}

# =============================================================================
# OUTPUTS
# =============================================================================

output "domain_id" {
  value       = peakhour_domain.main.id
  description = "Domain identifier"
}

output "domain_name" {
  value       = peakhour_domain.main.name
  description = "Domain name"
}

output "acme_certificate_state" {
  value       = peakhour_acme_certificate.main.state
  description = "ACME certificate state"
}

output "primary_origin_pool" {
  value       = peakhour_origin_pool.primary.tag
  description = "Primary origin pool tag"
}

output "api_rate_limit_zone" {
  value       = peakhour_rate_limit_zone.api.name
  description = "API rate limit zone name"
}

output "waf_custom_rules" {
  value = [
    peakhour_rp_waf_custom_rule.sql_injection.uuid,
    peakhour_rp_waf_custom_rule.protect_login.uuid,
  ]
  description = "WAF custom rule UUIDs"
}

# =============================================================================
# IMPORTS (for existing resources)
# =============================================================================

import {
  to = peakhour_domain.main
  id = "myapp.example.com"
}

import {
  to = peakhour_domain_plan.main
  id = "myapp.example.com"
}

import {
  to = peakhour_reverse_proxy_service.main
  id = "myapp.example.com"
}
