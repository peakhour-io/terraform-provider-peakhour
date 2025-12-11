# Peakhour Rules Guide

Rules allow you to create conditional logic that executes at different phases of request processing. Each rule has a filter expression and one or more actions.

## Rule Structure

```hcl
resource "peakhour_rule" "example" {
  domain     = "example.com"
  phase      = "firewall"
  name       = "Rule Name"
  filter_str = "http.request.uri.path matches \"^/api/\""
  enabled    = true

  actions_json = jsonencode({
    action_type = [{
      type = "action_type"
      # ... action-specific fields
    }]
  })
}
```

## Phases

Rules execute in specific phases during request processing:

| Phase | When It Runs | Common Uses |
|-------|-------------|-------------|
| `request_rewrite` | Before any processing | URL rewriting, path normalization |
| `url_config` | After rewrite, before cache lookup | Per-URL configuration overrides (caching, WAF, etc.) |
| `firewall` | Early in request | Block/allow/challenge based on conditions |
| `rate_limit_request` | Before origin request | Rate limiting (early phase) |
| `request_headers` | Before sending to origin | Modify headers sent to backend |
| `load_balance` | Origin selection | Choose which backend pool to use |
| `rate_limit_request_late` | After origin response | Rate limiting (late phase) |
| `rate_limit_response` | Response processing | Rate limit based on response |
| `response_headers` | Before sending to client | Modify headers sent to client |
| `bulk_redirect` | Redirect processing | Bulk URL redirects |

## Filter Syntax (Wirefilter)

Filters use Wirefilter syntax to match requests:

### Basic Operators
- `==` - Equals
- `!=` - Not equals
- `contains` - String contains
- `matches` - Regex match
- `in` - In list
- `>`, `<`, `>=`, `<=` - Comparisons

### Logical Operators
- `and` - Both conditions
- `or` - Either condition
- `not` - Negate condition

### Common Fields
- `http.request.uri.path` - Request path
- `http.request.uri.query` - Query string
- `http.host` - Host header
- `http.user_agent` - User agent
- `http.request.method` - HTTP method (GET, POST, etc.)
- `ip.src` - Client IP address
- `http.referer` - Referer header
- `http.cookie` - Cookie header
- `http.request.headers["X-Custom"]` - Custom header

### Examples
```
# Match specific path
http.request.uri.path == "/api/users"

# Regex match
http.request.uri.path matches "^/api/.*"

# Multiple conditions
http.request.uri.path matches "^/admin/" and ip.src != 192.0.2.1

# Check user agent
http.user_agent contains "bot" or http.user_agent contains "crawler"

# Query parameter check
http.request.uri.query contains "debug=true"

# Header check
http.request.headers["X-API-Key"] == "secret123"
```

## Action Types

### Firewall Actions

Block, allow, or challenge requests.

```hcl
actions_json = jsonencode({
  firewall = [{
    type   = "firewall"
    action = "deny"      # or "allow", "challenge", "log"
    reason = "Blocked by rule"
  }]
})
```

### Header Actions

Modify request or response headers.

```hcl
actions_json = jsonencode({
  header = [{
    type = "header"
    set_headers = {
      "X-Custom-Header" = "value"
      "X-API-Version"   = "v2"
    }
    remove_headers = ["X-Powered-By", "Server"]
  }]
})
```

### VConf Actions (Configuration Override)

Override configuration for matching requests.

```hcl
actions_json = jsonencode({
  vconf = [{
    type              = "vconf"
    continue_on_match = false

    # Caching
    cache_enabled   = true
    force_cache     = true
    edge_ttl_sec    = 86400
    browser_ttl_sec = 3600

    # Compression
    gzip   = true
    brotli = true

    # WAF
    waf_mode = "on"  # or "off", "detect"

    # Other settings
    websocket = true
  }]
})
```

### Rate Limit Actions

Rate limiting based on various keys.

```hcl
actions_json = jsonencode({
  rate_limit_request = [{
    type                            = "rate_limit_request"
    check_zone                      = "api_limit"
    check_zone_action               = "block"  # or "challenge", "log"
    check_zone_action_status_code   = 429
    zone_key                        = ["ip"]  # or ["ip", "header"], ["cookie"], etc.
    zone_key_headers                = ["X-API-Key"]
    zone_key_cookies                = ["session_id"]
  }]
})
```

### Cache Actions

Add cache tags for selective purging.

```hcl
actions_json = jsonencode({
  cache = [{
    type     = "cache"
    add_tags = ["products", "catalog", "v2"]
  }]
})
```

### Request Rewrite Actions

Modify the request URI before sending to origin.

```hcl
actions_json = jsonencode({
  request_rewrite = [{
    type    = "request_rewrite"
    set_uri = "/new/path"
    # or
    # set_uri_dyn = "concat(\"/v2\", http.request.uri.path)"
  }]
})
```

### Origin Selection Actions

Choose which backend pool to use.

```hcl
actions_json = jsonencode({
  origin_selection = [{
    type     = "origin_selection"
    set_pool = "api-backend"
  }]
})
```

## Complete Examples

### 1. Block Admin Access from Internet

```hcl
resource "peakhour_rule" "block_admin" {
  domain     = "example.com"
  phase      = "firewall"
  name       = "Block Admin Access"
  filter_str = "http.request.uri.path matches \"^/admin/\" and ip.src != 203.0.113.0/24"
  enabled    = true

  actions_json = jsonencode({
    firewall = [{
      type   = "firewall"
      action = "deny"
      reason = "Admin access restricted"
    }]
  })
}
```

### 2. API Rate Limiting

```hcl
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
      zone_key                        = ["ip", "header"]
      zone_key_headers                = ["X-API-Key"]
    }]
  })
}
```

### 3. Cache Static Assets

```hcl
resource "peakhour_rule" "cache_static" {
  domain     = "example.com"
  phase      = "url_config"
  name       = "Cache Static Assets"
  filter_str = "http.request.uri.path matches \"\\.(jpg|png|css|js|woff2)$\""
  enabled    = true

  actions_json = jsonencode({
    vconf = [{
      type              = "vconf"
      force_cache       = true
      cache_enabled     = true
      edge_ttl_sec      = 2592000  # 30 days
      browser_ttl_sec   = 86400    # 1 day
      continue_on_match = false
    }]
  })
}
```

### 4. Route API to Different Backend

```hcl
resource "peakhour_rule" "route_api" {
  domain     = "example.com"
  phase      = "load_balance"
  name       = "Route API Traffic"
  filter_str = "http.request.uri.path matches \"^/api/\""
  enabled    = true

  actions_json = jsonencode({
    origin_selection = [{
      type     = "origin_selection"
      set_pool = "api-backend"
    }]
  })
}
```

### 5. Add Security Headers

```hcl
resource "peakhour_rule" "security_headers" {
  domain     = "example.com"
  phase      = "response_headers"
  name       = "Add Security Headers"
  filter_str = "true"  # Match all requests
  enabled    = true

  actions_json = jsonencode({
    header = [{
      type = "header"
      set_headers = {
        "X-Content-Type-Options" = "nosniff"
        "X-Frame-Options"        = "SAMEORIGIN"
        "X-XSS-Protection"       = "1; mode=block"
        "Strict-Transport-Security" = "max-age=31536000"
      }
    }]
  })
}
```

### 6. Challenge Suspicious Bots

```hcl
resource "peakhour_rule" "challenge_bots" {
  domain     = "example.com"
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
}
```

### 7. Rewrite Legacy URLs

```hcl
resource "peakhour_rule" "rewrite_legacy" {
  domain     = "example.com"
  phase      = "request_rewrite"
  name       = "Rewrite Legacy URLs"
  filter_str = "http.request.uri.path matches \"^/old-api/\""
  enabled    = true

  actions_json = jsonencode({
    request_rewrite = [{
      type    = "request_rewrite"
      set_uri = "/api/v2/"
    }]
  })
}
```

### 8. Add Cache Tags for Selective Purging

```hcl
resource "peakhour_rule" "cache_tags" {
  domain     = "example.com"
  phase      = "url_config"
  name       = "Tag Product Pages"
  filter_str = "http.request.uri.path matches \"^/products/\""
  enabled    = true

  actions_json = jsonencode({
    cache = [{
      type     = "cache"
      add_tags = ["products", "catalog"]
    }]
  })
}
```

## Best Practices

1. **Order Matters**: Rules execute in order within a phase. Use the Peakhour UI to reorder if needed.

2. **Test Filters**: Test your filter expressions carefully to avoid blocking legitimate traffic.

3. **Use Specific Filters**: More specific filters are more efficient than broad matches.

4. **Enable Gradually**: Start with `enabled = false`, test, then enable.

5. **Use continue_on_match**: In vconf actions, set `continue_on_match = false` to stop processing more rules if this one matches.

6. **Log First**: Use firewall action `log` to test before blocking with `deny`.

7. **Cache Tags**: Use descriptive cache tags to make selective purging easier.

8. **Rate Limit Keys**: Choose appropriate zone keys (ip, header, cookie) based on your API authentication.

## Importing Existing Rules

```bash
terraform import peakhour_rule.example example.com/rule-uuid-here
```

## Debugging

- View rules in Peakhour dashboard to see UUIDs
- Check rule ordering in the UI
- Use firewall action "log" to test without blocking
- Monitor logs to see which rules match

## Advanced: Multiple Actions

Some phases support multiple action types:

```hcl
actions_json = jsonencode({
  header = [{
    type = "header"
    set_headers = { "X-Custom" = "value" }
  }]
  cache = [{
    type     = "cache"
    add_tags = ["api"]
  }]
})
```

## Phase Execution Order

1. `request_rewrite` - URL normalization
2. `url_config` - Configuration overrides
3. `firewall` - Access control
4. `rate_limit_request` - Early rate limiting
5. `request_headers` - Modify request
6. `load_balance` - Origin selection
7. [Request sent to origin]
8. `rate_limit_request_late` - Late rate limiting
9. `rate_limit_response` - Response rate limiting
10. `response_headers` - Modify response
11. [Response sent to client]
