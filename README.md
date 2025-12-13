# Terraform Provider for Peakhour

Terraform provider for managing [Peakhour](https://www.peakhour.io) CDN and edge security platform resources.

## Features

- **Domain Management**: Create and manage domains
- **Reverse Proxy Service**: Enable CDN/proxy service for domains
- **Configuration**: Configure compression, WebSocket, redirects, and aliases
- **Origin Pools**: Manage backend servers with load balancing
- **Transform Settings**: Configure image optimization and HTML processing
- **Rules**: Create conditional rules in different phases (firewall, rate limiting, headers, caching, etc.)
- **Rate Limit Zones**: Define rate limiting policies referenced by rules
- **Rule Lists**: Manage IP, string, and integer lists for use in rules

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.22 (for development)

## Installation

### Using the Provider

Add this to your Terraform configuration:

```hcl
terraform {
  required_providers {
    peakhour = {
      source = "peakhour-io/peakhour"
    }
  }
}

provider "peakhour" {
  api_key = var.peakhour_api_key
  # Optional: override API endpoint (defaults to https://console.peakhour.io)
  # base_url = "https://console.staging.peakhour.io"
}
```

### Authentication

The provider requires a Peakhour API key. Set it via:

1. **Provider configuration**:
   ```hcl
   provider "peakhour" {
     api_key = "your-api-key"
   }
   ```

2. **Environment variable** (recommended):
   ```bash
   export PEAKHOUR_API_KEY="your-api-key"
   ```

Optional: override API endpoint via environment variable:
```bash
export PEAKHOUR_BASE_URL="https://console.staging.peakhour.io"
```

## API Errors

When the Peakhour API returns an error (validation, permission issues, etc.), Terraform will fail the operation and surface the API status code plus the error message.

- If the API returns `{"error":"...", "validation_errors":[...]}`, the provider includes the `validation_errors` details in the Terraform error.
- If the API returns `{"error":"..."}`, the provider uses the `error` field.
- Otherwise, it falls back to the raw response body.

## Usage

### Basic Example

```hcl
# Create a domain
resource "peakhour_domain" "example" {
  name = "example.com"
}

# Enable reverse proxy service
resource "peakhour_reverse_proxy_service" "example" {
  domain = peakhour_domain.example.name
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
  ]

  depends_on = [peakhour_reverse_proxy_service.example]
}
```

### Full Configuration Example

See [examples/full-setup/main.tf](examples/full-setup/main.tf) for a complete example including:
- Domain creation
- Service enablement
- Reverse proxy configuration (compression, WebSocket, aliases)
- TLS/ACME certificate configuration
- Origin pools with load balancing
- Transform settings (image optimization, HTML processing)

See [examples/tls-acme/main.tf](examples/tls-acme/main.tf) for a focused TLS/ACME example.

## Onboarding Existing Config (Many Domains)

If you already have domains/rules configured in Peakhour, see `docs/onboarding-existing-config.md` for an import workflow using `cmd/peakhour-tf-onboard` and Terraform config generation (`terraform plan -generate-config-out=...`, Terraform >= 1.5).

## Resources

### `peakhour_domain`

Manages a Peakhour domain.

```hcl
resource "peakhour_domain" "example" {
  name = "example.com"
}
```

**Arguments:**
- `name` (Required, String) - Domain name

**Attributes:**
- `id` (String) - Domain identifier

---

### `peakhour_reverse_proxy_service`

Enables the Reverse Proxy (CDN) service for a domain.

```hcl
resource "peakhour_reverse_proxy_service" "example" {
  domain = "example.com"
}
```

**Arguments:**
- `domain` (Required, String) - Domain name

**Attributes:**
- `id` (String) - Service identifier

---

### `peakhour_reverse_proxy_config`

Manages Reverse Proxy configuration.

**Note on Partial Updates:** This resource supports partial updates. Fields that are not specified (or set to null) in the configuration are NOT reset to defaults; they retain their existing values on the server. To reset a field to its default value, it must be explicitly defined.

```hcl
resource "peakhour_reverse_proxy_config" "example" {
  domain    = "example.com"
  gzip      = true
  brotli    = true
  websocket = true
  aliases   = ["www.example.com"]
}
```

**Arguments:**
- `domain` (Required, String) - Domain name
- `websocket` (Optional, Bool) - Enable WebSocket support (default: false)
- `gzip` (Optional, Bool) - Enable gzip compression (default: true)
- `brotli` (Optional, Bool) - Enable Brotli compression (default: false)
- `aliases` (Optional, List of String) - Domain aliases
- `track_sessions` (Optional, Bool) - Enable session tracking (default: false)
- `debug` (Optional, Bool) - Enable debug mode (default: false)
- `segment` (Optional, Bool) - Enable segment analytics (default: false)
- `redirect_mode` (Optional, String) - Redirect mode (e.g., "all", "http")
- `redirect_location` (Optional, String) - Redirect target URL
- `redirect_status_code` (Optional, Number) - HTTP redirect status code (default: 301)

---

### `peakhour_rp_settings`

Manages Reverse Proxy service settings (notifications, quickstart, computed addresses).

```hcl
resource "peakhour_rp_settings" "example" {
  domain = "example.com"

  notification_emails = ["ops@example.com"]
  quickstart          = true
}
```

See [examples/tls-acme/main.tf](examples/tls-acme/main.tf).

---

### `peakhour_rp_ssl_config`

Manages the SSL/TLS cipher profile.

```hcl
resource "peakhour_rp_ssl_config" "example" {
  domain  = "example.com"
  ciphers = "intermediate"
}
```

See [examples/full-setup/main.tf](examples/full-setup/main.tf).

---

### `peakhour_rp_ssl_certificate`

Uploads a custom SSL certificate and private key.

**Important:** The API does not return the uploaded private key (or certificate PEM), so Terraform cannot automatically verify drift of `certificate_pem`/`private_key_pem`. The private key is stored in Terraform state (marked sensitive). Prefer ACME when possible.

```hcl
resource "peakhour_rp_ssl_certificate" "example" {
  domain = "example.com"

  certificate_pem = file("cert.pem")
  private_key_pem = file("key.pem")
}
```

See [examples/tls-acme/main.tf](examples/tls-acme/main.tf).

---

### `peakhour_acme_settings`

Manages ACME settings (hostnames/SANs) for a domain.

```hcl
resource "peakhour_acme_settings" "example" {
  domain = "example.com"

  domain_names = [
    "example.com",
    "www.example.com",
  ]
}
```

See [examples/tls-acme/main.tf](examples/tls-acme/main.tf).

---

### `peakhour_acme_certificate`

Reads ACME certificate status and can optionally trigger issuance (async).

```hcl
resource "peakhour_acme_certificate" "example" {
  domain = "example.com"

  # Optional: toggle to trigger issuance
  # issue = true
}
```

See [examples/tls-acme/main.tf](examples/tls-acme/main.tf).

---

### `peakhour_rp_origin_config`

Manages RP origin behavior settings (distinct from origin pools).

```hcl
resource "peakhour_rp_origin_config" "example" {
  domain   = "example.com"
  ssl_mode = "https"

  origin_request_headers = {
    geoip = true
  }
}
```

See [examples/full-setup/main.tf](examples/full-setup/main.tf).

---

### `peakhour_rp_cdn_cache`

Manages CDN caching settings.

```hcl
resource "peakhour_rp_cdn_cache" "example" {
  domain        = "example.com"
  cache_enabled = true
  cdn_query_mode = "full"
}
```

See [examples/full-setup/main.tf](examples/full-setup/main.tf).

---

### `peakhour_rp_cdn_purge_resources`

Triggers a CDN purge for specific resources (action).

**Note:** This is not stable desired-state. The API call is performed when the resource is created. To run another purge, change `run_id` (or taint the resource).

```hcl
resource "peakhour_rp_cdn_purge_resources" "example" {
  domain = "example.com"
  run_id = "release-2025-12-13"

  paths = ["/index.html", "/assets/app.js"]
  soft  = true
}
```

See [examples/cdn-purge/main.tf](examples/cdn-purge/main.tf).

---

### `peakhour_rp_cdn_purge_wildcard`

Triggers a CDN purge for wildcard paths (action).

```hcl
resource "peakhour_rp_cdn_purge_wildcard" "example" {
  domain = "example.com"
  run_id = "release-2025-12-13-images"

  paths = ["/images/*"]
}
```

See [examples/cdn-purge/main.tf](examples/cdn-purge/main.tf).

---

### `peakhour_rp_cdn_purge_tags`

Triggers a CDN purge by cache tag (action).

```hcl
resource "peakhour_rp_cdn_purge_tags" "example" {
  domain = "example.com"
  run_id = "release-2025-12-13-tags"

  tags = ["marketing"]
}
```

See [examples/cdn-purge/main.tf](examples/cdn-purge/main.tf).

---

### `peakhour_rp_bots`

Manages bots verification settings.

```hcl
resource "peakhour_rp_bots" "example" {
  domain = "example.com"

  bots_verify_list = ["google", "bing"]
  bots_verify_rdns = true
}
```

See [examples/full-setup/main.tf](examples/full-setup/main.tf).

---

### `peakhour_rp_threat_access_list_rule`

Manages a reverse proxy threat access list rule for a domain.

```hcl
resource "peakhour_rp_threat_access_list_rule" "example" {
  domain      = "example.com"
  rule_type   = "whitelist"
  content     = "203.0.113.10"
  description = "Allow office IP"
}
```

See [examples/threats/main.tf](examples/threats/main.tf).

---

### `peakhour_rp_threat_block_list`

Manages the set of enabled reverse proxy threat block lists for a domain.

```hcl
resource "peakhour_rp_threat_block_list" "example" {
  domain     = "example.com"
  blocklists = ["tor"]
}
```

See [examples/threats/main.tf](examples/threats/main.tf).

---

### `peakhour_rp_waf_options`

Manages WAF options for a domain.

```hcl
resource "peakhour_rp_waf_options" "example" {
  domain = "example.com"

  waf_mode                        = "enabled"
  waf_ruleset                     = "owaspv33"
  waf_set_exposed_password_header = true
}
```

See [examples/waf/main.tf](examples/waf/main.tf).

---

### `peakhour_rp_waf_owasp_settings`

Manages WAF OWASP settings for a domain (as JSON).

```hcl
resource "peakhour_rp_waf_owasp_settings" "example" {
  domain = "example.com"

  settings_json = jsonencode({
    methods = {
      allowed_methods = ["GET", "HEAD", "POST", "OPTIONS"]
    }
    protocol = {
      max_num_args = 100
    }
  })
}
```

See [examples/waf/main.tf](examples/waf/main.tf).

---

### `peakhour_rp_waf_custom_rule`

Manages a WAF custom rule for a domain.

```hcl
resource "peakhour_rp_waf_custom_rule" "example" {
  domain = "example.com"

  name        = "Block curl user-agent"
  description = "Example custom rule (pass action)"
  enabled     = true

  rules_json = jsonencode([
    {
      variable      = "REQUEST_HEADERS"
      variable_part = "user-agent"
      operator      = "@contains"
      operator_arg  = "curl"
    }
  ])

  action_json = jsonencode({
    action_name = "pass"
  })

  logging_json = jsonencode({
    message  = "matched tf custom rule"
    severity = "INFO"
    tags     = ["terraform"]
  })
}
```

See [examples/waf/main.tf](examples/waf/main.tf).

---

### `peakhour_rp_waf_rule_group`

Manages the enabled state of a WAF rule group (file) within a ruleset.

```hcl
resource "peakhour_rp_waf_rule_group" "example" {
  domain = "example.com"

  ruleset   = "owaspv33"
  file_name = "REQUEST-913-SCANNER-DETECTION.conf"
  enabled   = false
}
```

See [examples/waf/main.tf](examples/waf/main.tf).

---

### `peakhour_rp_firewall_settings`

Manages firewall settings such as challenge cookie key configuration.

```hcl
resource "peakhour_rp_firewall_settings" "example" {
  domain = "example.com"

  challenge_cookie_key = ["fingerprint_tls", "ip"]
}
```

See [examples/full-setup/main.tf](examples/full-setup/main.tf).

---

### `peakhour_rp_firewall_error_page`

Uploads a custom firewall error page. Note: the API does not return the configured content (only whether a page exists), so Terraform cannot detect drift of the content.

```hcl
resource "peakhour_rp_firewall_error_page" "example" {
  domain = "example.com"

  content = "<html><body><h1>Access denied</h1></body></html>"
}
```

See [examples/full-setup/main.tf](examples/full-setup/main.tf).

---

### `peakhour_rp_lua_options`

Manages Lua settings.

```hcl
resource "peakhour_rp_lua_options" "example" {
  domain      = "example.com"
  lua_enabled = false
}
```

See [examples/full-setup/main.tf](examples/full-setup/main.tf).

---

### `peakhour_origin_pool`

Manages an origin pool (backend servers).

```hcl
resource "peakhour_origin_pool" "backend" {
  domain = "example.com"
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
}
```

**Arguments:**
- `domain` (Required, String) - Domain name
- `tag` (Required, String) - Origin pool tag/name
- `address` (Required, List of Object) - Backend server addresses
  - `address` (Required, String) - Backend address (IP:port, domain:port, or URL)
  - `weight` (Optional, Number) - Load balancing weight
- `shield_name` (Optional, String) - Shield location name
- `load_balancing_mode` (Optional, String) - Load balancing mode (e.g., "round_robin", "least_conn")
- `load_balancing_key` (Optional, String) - Load balancing key for consistent hashing
- `load_balancing_overload_percent` (Optional, Number) - Overload percentage

---

### `peakhour_transform_settings`

Manages transform settings (HTML/image processing).

---

### `peakhour_rule`

Manages a rule in a specific phase. Rules allow conditional logic based on request properties.

```hcl
resource "peakhour_rule" "block_admin" {
  domain     = "example.com"
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
}
```

**Arguments:**
- `domain` (Required, String) - Domain name
- `phase` (Required, String) - Phase name (e.g., "firewall", "request_headers", "url_config", "rate_limit_request", "request_rewrite", "load_balance")
- `name` (Required, String) - Rule name
- `filter_str` (Required, String) - Filter expression using Wirefilter syntax
- `enabled` (Optional, Bool) - Whether the rule is enabled (default: true)
- `actions_json` (Required, String) - Actions as JSON string

**Attributes:**
- `uuid` (String) - Rule UUID (computed)

**Available Phases:**
- `request_rewrite` - Modify request URL before processing
- `url_config` - Override configuration based on URL (vconf actions)
- `firewall` - Block, allow, or challenge requests
- `rate_limit_request` - Rate limiting (early phase)
- `rate_limit_request_late` - Rate limiting (late phase)
- `rate_limit_response` - Rate limiting on responses
- `request_headers` - Modify request headers
- `response_headers` - Modify response headers
- `load_balance` - Origin selection
- `bulk_redirect` - Bulk redirects

**Common Action Types:**
- `firewall` - Block/allow/challenge requests
- `header` - Add/remove/modify headers
- `vconf` - Override configuration (caching, WAF, transforms, etc.)
- `rate_limit_request` - Rate limit configuration
- `cache` - Add cache tags
- `request_rewrite` - Rewrite request URI
- `origin_selection` - Select origin pool
- `redirect` - Redirect using bulk redirect list

See [examples/rules/main.tf](examples/rules/main.tf) and [RULES_GUIDE.md](RULES_GUIDE.md) for detailed examples.

---

### `peakhour_rule_phase_order`

Manages the order of rules within a phase.

```hcl
resource "peakhour_rule_phase_order" "firewall" {
  domain = "example.com"
  phase  = "firewall"

  # Full order for the phase (default: include_all_rules=true)
  order = [
    peakhour_rule.block_admin.uuid,
    peakhour_rule.challenge_bots.uuid,
  ]
}
```

See [examples/rules/main.tf](examples/rules/main.tf).

---

### `peakhour_bulk_redirect_list`

Manages a bulk redirect list. Lists can be referenced from rules in the `bulk_redirect` phase using the `redirect` action.

```hcl
resource "peakhour_bulk_redirect_list" "legacy" {
  domain = "example.com"
  name   = "legacy_redirects"
}
```

**Arguments:**
- `domain` (Required, String) - Domain name
- `name` (Required, String) - List name (referenced by rules via `from_list`)
- `description` (Optional, String) - List description

**Attributes:**
- `uuid` (String) - List UUID (computed)
- `entries_count` (Number) - Number of entries in the list

---

### `peakhour_bulk_redirect_entry`

Manages an entry in a bulk redirect list.

```hcl
resource "peakhour_bulk_redirect_entry" "home" {
  domain             = "example.com"
  bulk_redirect_uuid = peakhour_bulk_redirect_list.legacy.uuid

  source_path = "/old-home"
  target_url  = "https://example.com/new-home"
  status_code = 301
}
```

**Arguments:**
- `domain` (Required, String) - Domain name
- `bulk_redirect_uuid` (Required, String) - Bulk redirect list UUID
- `source_path` (Required, String) - Path to match (e.g. `/old-home`)
- `target_url` (Required, String) - Full URL to redirect to
- `status_code` (Optional, Number) - HTTP status code (301, 302, 307, 308)
- `enabled` (Optional, Bool) - Enable/disable entry (default: true)
- `preserve_query_string` (Optional, Bool) - Preserve query string (default: true)
- `source_domain` (Optional, String) - Optional domain to match
- `source_scheme` (Optional, String) - Optional scheme to match

**Attributes:**
- `entry_id` (String) - Entry identifier (computed)

---

### `peakhour_rate_limit_settings`

Manages rate limit mode settings for a domain (enables/disables global/vhost/zone rate limiting).

```hcl
resource "peakhour_rate_limit_settings" "example" {
  domain = "example.com"
  mode   = ["zone", "global"]
}
```

See [examples/rate-limiting/main.tf](examples/rate-limiting/main.tf).

---

### `peakhour_rate_limit_zone`

Manages a rate limit zone. Zones define rate limiting rules that can be referenced in rules.

```hcl
resource "peakhour_rate_limit_zone" "api_limit" {
  domain = "example.com"
  name   = "api_limit"

  # Allow 100 requests per 60 seconds
  requests_max         = 100
  requests_interval_sec = 60

  # Block for 300 seconds when limit exceeded
  block_duration_sec = 300
}

# Use the zone in a rule
resource "peakhour_rule" "api_ratelimit" {
  domain     = "example.com"
  phase      = "rate_limit_request"
  name       = "API Rate Limit"
  filter_str = "http.request.uri.path matches \"^/api/\""
  enabled    = true

  actions_json = jsonencode({
    rate_limit_request = [{
      type                            = "rate_limit_request"
      check_zone                      = "api_limit"
      check_zone_action               = "block"
      check_zone_action_status_code   = 429
      zone_key                        = ["ip"]
    }]
  })

  depends_on = [peakhour_rate_limit_zone.api_limit]
}
```

**Arguments:**
- `domain` (Required, String) - Domain name
- `name` (Required, String) - Zone name (used to reference in rules)
- `block_duration_sec` (Optional, Number) - How long to block when limit exceeded (seconds)
- `connections_max` (Optional, Number) - Maximum connections in interval
- `connections_interval_sec` (Optional, Number) - Time window for connection limit (seconds)
- `requests_max` (Optional, Number) - Maximum requests in interval
- `requests_interval_sec` (Optional, Number) - Time window for request limit (seconds)
- `response_errors_max` (Optional, Number) - Maximum response errors in interval
- `response_errors_interval_sec` (Optional, Number) - Time window for response error limit (seconds)

**Attributes:**
- `id` (String) - Zone identifier (domain/name)

See [examples/rate-limiting/main.tf](examples/rate-limiting/main.tf) for detailed examples.

---

### `peakhour_rate_limit_global`

Manages global rate limiting settings for a domain. Global settings are not tied to rate limit zones.

**Arguments:**
- `domain` (Required, String) - Domain name
- `block_duration_sec` (Optional, Number) - How long to block when limit exceeded (seconds)
- `concurrent_connections` (Optional, Number) - Maximum concurrent connections
- `connections_max` (Optional, Number) - Maximum connections in interval
- `connections_interval_sec` (Optional, Number) - Time window for connection limit (seconds)
- `requests_max` (Optional, Number) - Maximum requests in interval
- `requests_interval_sec` (Optional, Number) - Time window for request limit (seconds)
- `response_errors_max` (Optional, Number) - Maximum response errors in interval
- `response_errors_interval_sec` (Optional, Number) - Time window for response error limit (seconds)

---

### `peakhour_rule_list`

Manages a rule list. Lists store collections of IPs, strings, or integers that can be referenced in rules using the `$list_name` syntax.

```hcl
# IP blocklist
resource "peakhour_rule_list" "ip_blocklist" {
  domain = "example.com"
  name   = "blocked_ips"
  type   = "ip"

  ips = [
    "192.0.2.0/24",
    "198.51.100.0/24",
    "203.0.113.45",
  ]
}

# Use the list in a firewall rule
resource "peakhour_rule" "block_listed_ips" {
  domain     = "example.com"
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

  depends_on = [peakhour_rule_list.ip_blocklist]
}

# String list (e.g., user agents)
resource "peakhour_rule_list" "bad_user_agents" {
  domain = "example.com"
  name   = "blocked_user_agents"
  type   = "string"

  strs = [
    "BadBot",
    "EvilScraper",
    "MaliciousCrawler",
  ]
}

# Integer list (e.g., HTTP status codes)
resource "peakhour_rule_list" "error_codes" {
  domain = "example.com"
  name   = "monitored_error_codes"
  type   = "integer"

  ints = [500, 502, 503, 504]
}
```

**Arguments:**
- `domain` (Required, String) - Domain name
- `name` (Required, String) - List name (used to reference in rules with `$name`)
- `type` (Required, String) - List type: `"ip"`, `"string"`, or `"integer"`
- `ips` (Optional, List of String) - IP addresses/networks (for type="ip")
- `strs` (Optional, List of String) - String values (for type="string")
- `ints` (Optional, List of Number) - Integer values (for type="integer")

**Attributes:**
- `uuid` (String) - List UUID (computed)
- `id` (String) - List identifier (domain/uuid)

**Usage in Rules:**
Reference lists in filter expressions using `$list_name`:
- `ip.src in $blocked_ips` - Check if source IP is in list
- `http.user_agent in $blocked_user_agents` - Check if user agent is in list
- `http.request.country in $allowed_country_codes` - Check if country is in list
- `not ip.src in $allowed_ips` - Check if IP is NOT in list

See [examples/rule-lists/main.tf](examples/rule-lists/main.tf) for detailed examples.

---

### `peakhour_transform_settings`

Manages transform settings (HTML/image processing).

```hcl
resource "peakhour_transform_settings" "example" {
  domain = "example.com"

  transform_html          = true
  transform_lazy_sizes    = true
  transform_image_api     = true
  transform_image_quality = 85
}
```

**Arguments:**
- `domain` (Required, String) - Domain name
- `transform_html` (Optional, Bool) - Enable HTML transformation (default: false)
- `transform_beacon` (Optional, Bool) - Enable beacon injection (default: false)
- `transform_lazy_sizes` (Optional, Bool) - Enable lazy loading for images (default: false)
- `transform_mixed_content` (Optional, Bool) - Fix mixed content (default: false)
- `transform_img_dims_to_query_args` (Optional, Bool) - Convert image dimensions to query args (default: false)
- `transform_image_quality` (Optional, Number) - Image quality 1-100 (default: 85)
- `transform_image_format` (Optional, Bool) - Enable automatic format conversion (default: false)
- `transform_image_optimise` (Optional, Bool) - Enable image optimization (default: false)
- `transform_image_api` (Optional, Bool) - Enable image transformation API (default: false)
- `transform_http_header_value` (Optional, String) - HTTP header value for transforms
- `transform_esi` (Optional, Bool) - Enable Edge Side Includes (default: false)
- `transform_rewrite_domains` (Optional, List of String) - Domains to rewrite in content

---

### `peakhour_image_transform`

Manages a named image transformation that can be referenced by name in image URLs.

```hcl
resource "peakhour_image_transform" "thumbnail" {
  domain = "example.com"
  name   = "thumbnail"
  config_json = jsonencode({
    width   = 200
    height  = 200
    fit     = "cover"
    quality = 80
  })
}
```

**Arguments:**
- `domain` (Required, String) - Domain name
- `name` (Required, String) - Transform name (used in URLs e.g. `example.com/img.jpg?peak_transform=thumbnail`)
- `config_json` (Required, String) - Transformation configuration JSON

## Data Sources

### `peakhour_domain`

Fetches information about an existing domain.

```hcl
data "peakhour_domain" "example" {
  name = "example.com"
}
```

**Arguments:**
- `name` (Required, String) - Domain name to look up

**Attributes:**
- `id` (String) - Domain identifier

## API Specification

For a comprehensive list of all available fields and their types, please refer to the [API Specification](docs/spec/peakhour-api-v1.json). This specification serves as the source of truth for the API.

## Development

### Building the Provider

```bash
cd peakhour-terraform
go mod download
go build -o terraform-provider-peakhour
```

### Testing Locally

1. Build and install the provider locally:
   ```bash
   # If you previously ran Terraform in examples/ and changed the provider binary,
   # clear old lock/state files to avoid checksum mismatch errors:
   make clean
   make build
   make install
   ```

2. Run Terraform:
   ```bash
   cd examples/basic
   export PEAKHOUR_API_KEY="your-api-key"
   terraform init -backend=false
   terraform validate
   terraform plan
   terraform apply
   ```

### Running Tests

```bash
go test ./...
```

### Acceptance Tests (E2E)

Acceptance tests run Terraform against a real Peakhour account.

**Prerequisites:**
- Terraform CLI available in `PATH`
- `PEAKHOUR_API_KEY` set
- `PEAKHOUR_TEST_DOMAIN` set to an existing domain in your Peakhour account (ideally a dedicated test domain)
- Optional: `PEAKHOUR_BASE_URL` for non-production environments

Run:

```bash
export PEAKHOUR_API_KEY="your-api-key"
export PEAKHOUR_TEST_DOMAIN="test.example.com"
make testacc
```

### Jenkins

This repo includes a `Jenkinsfile` that runs unit tests by default and can run acceptance tests when `RUN_ACCEPTANCE=true`.

- Configure a Jenkins string credential named `peakhour-api-key` (or update `Jenkinsfile` to match your credential ID).
- Provide `PEAKHOUR_TEST_DOMAIN` as a Jenkins parameter (and optionally `PEAKHOUR_BASE_URL`).
- Optional: override tool versions via `GO_VERSION` and `TERRAFORM_VERSION` parameters.

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## License

Mozilla Public License 2.0

## Support

For issues and questions:
- GitHub Issues: https://github.com/peakhour-io/terraform-provider-peakhour/issues
- Peakhour Support: https://www.peakhour.io/support
