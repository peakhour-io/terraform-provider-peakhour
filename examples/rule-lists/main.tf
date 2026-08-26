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

# Example 1: IP blocklist
resource "peakhour_rule_list" "ip_blocklist" {
  domain = peakhour_domain.example.name
  name   = "blocked_ips"
  type   = "ip"

  ips = [
    "192.0.2.0/24",
    "198.51.100.0/24",
    "203.0.113.45",
  ]

  depends_on = [peakhour_reverse_proxy_service.example]
}

# Example 2: IP allowlist for admin access
resource "peakhour_rule_list" "ip_allowlist" {
  domain = peakhour_domain.example.name
  name   = "allowed_admin_ips"
  type   = "ip"

  ips = [
    "203.0.113.0/24", # Office network
    "198.51.100.10",  # VPN endpoint
  ]

  depends_on = [peakhour_reverse_proxy_service.example]
}

# Example 3: User agent blocklist
resource "peakhour_rule_list" "bad_user_agents" {
  domain = peakhour_domain.example.name
  name   = "blocked_user_agents"
  type   = "string"

  strs = [
    "BadBot",
    "EvilScraper",
    "MaliciousCrawler",
    "SpamBot",
  ]

  depends_on = [peakhour_reverse_proxy_service.example]
}

# Example 4: Allowed countries (by code)
resource "peakhour_rule_list" "allowed_countries" {
  domain = peakhour_domain.example.name
  name   = "allowed_country_codes"
  type   = "string"

  strs = [
    "US",
    "CA",
    "GB",
    "AU",
    "NZ",
  ]

  depends_on = [peakhour_reverse_proxy_service.example]
}

# Example 5: HTTP status codes to monitor
resource "peakhour_rule_list" "error_codes" {
  domain = peakhour_domain.example.name
  name   = "monitored_error_codes"
  type   = "integer"

  ints = [
    500,
    502,
    503,
    504,
  ]

  depends_on = [peakhour_reverse_proxy_service.example]
}

# Example 6: API versions to support
resource "peakhour_rule_list" "api_versions" {
  domain = peakhour_domain.example.name
  name   = "supported_api_versions"
  type   = "string"

  strs = [
    "v1",
    "v2",
    "v3",
  ]

  depends_on = [peakhour_reverse_proxy_service.example]
}

# Use Case 1: Block IPs from blocklist
resource "peakhour_rule" "block_listed_ips" {
  domain     = peakhour_domain.example.name
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

  depends_on = [
    peakhour_reverse_proxy_service.example,
    peakhour_rule_list.ip_blocklist
  ]
}

# Use Case 2: Allow only specific IPs to admin
resource "peakhour_rule" "admin_ip_whitelist" {
  domain     = peakhour_domain.example.name
  phase      = "firewall"
  name       = "Admin IP Whitelist"
  filter_str = "http.request.uri.path matches \"^/admin/\" and not ip.src in $allowed_admin_ips"
  enabled    = true

  actions_json = jsonencode({
    firewall = [{
      type   = "firewall"
      action = "deny"
      reason = "Admin access restricted to allowed IPs"
    }]
  })

  depends_on = [
    peakhour_reverse_proxy_service.example,
    peakhour_rule_list.ip_allowlist
  ]
}

# Use Case 3: Block bad user agents
resource "peakhour_rule" "block_bad_agents" {
  domain     = peakhour_domain.example.name
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

  depends_on = [
    peakhour_reverse_proxy_service.example,
    peakhour_rule_list.bad_user_agents
  ]
}

# Use Case 4: Geographic restriction
resource "peakhour_rule" "geo_restriction" {
  domain     = peakhour_domain.example.name
  phase      = "firewall"
  name       = "Geographic Restriction"
  filter_str = "not http.request.country in $allowed_country_codes"
  enabled    = true

  actions_json = jsonencode({
    firewall = [{
      type   = "firewall"
      action = "deny"
      reason = "Access restricted to allowed countries"
    }]
  })

  depends_on = [
    peakhour_reverse_proxy_service.example,
    peakhour_rule_list.allowed_countries
  ]
}

output "ip_blocklist_uuid" {
  value       = peakhour_rule_list.ip_blocklist.uuid
  description = "IP blocklist UUID"
}

output "ip_allowlist_uuid" {
  value       = peakhour_rule_list.ip_allowlist.uuid
  description = "IP allowlist UUID"
}

output "user_agent_blocklist_uuid" {
  value       = peakhour_rule_list.bad_user_agents.uuid
  description = "User agent blocklist UUID"
}
