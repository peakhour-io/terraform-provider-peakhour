# Rules Support Update

## What Was Added

### New Resource: `peakhour_rule`

Added support for Peakhour's powerful rules engine, allowing conditional logic at different phases of request processing.

**Features:**
- ✅ Create rules in any phase (firewall, headers, caching, rate limiting, etc.)
- ✅ Flexible filter expressions using Wirefilter syntax
- ✅ Support for all action types via JSON
- ✅ Enable/disable rules
- ✅ Full CRUD operations
- ✅ Import existing rules

### Phases Supported

- `request_rewrite` - URL rewriting before processing
- `url_config` - Per-URL configuration overrides (vconf)
- `firewall` - Block/allow/challenge requests
- `rate_limit_request` - Rate limiting (early)
- `rate_limit_request_late` - Rate limiting (late)
- `rate_limit_response` - Response rate limiting
- `request_headers` - Modify request headers
- `response_headers` - Modify response headers
- `load_balance` - Origin pool selection
- `bulk_redirect` - Bulk redirects

### Action Types Supported

- `firewall` - Block, allow, challenge, log
- `header` - Set/remove request/response headers
- `vconf` - Override configuration (caching, WAF, compression, etc.)
- `rate_limit_request` - Rate limiting with zones and keys
- `rate_limit_response` - Response rate limiting
- `cache` - Add cache tags
- `request_rewrite` - Rewrite request URI
- `origin_selection` - Select origin pool
- `redirect` - Bulk redirect lists

## Files Added/Modified

### New Files
- `internal/client/rules.go` - Rules API client methods
- `internal/provider/rule_resource.go` - Rule resource implementation
- `examples/rules/main.tf` - 8 comprehensive examples
- `RULES_GUIDE.md` - Complete rules documentation

### Modified Files
- `internal/client/types.go` - Added rule types
- `internal/provider/provider.go` - Registered rule resource
- `README.md` - Added rules documentation
- `PROJECT_SUMMARY.md` - Updated with rules info
- `QUICKSTART.md` - Added rules example

## Usage Examples

### Basic Firewall Rule
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
    }]
  })
}
```

### Rate Limiting
```hcl
resource "peakhour_rule" "api_limit" {
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
}
```

### Cache Configuration
```hcl
resource "peakhour_rule" "cache_images" {
  domain     = "example.com"
  phase      = "url_config"
  name       = "Cache Images"
  filter_str = "http.request.uri.path matches \"\\.(jpg|png|gif)$\""
  enabled    = true

  actions_json = jsonencode({
    vconf = [{
      type              = "vconf"
      force_cache       = true
      edge_ttl_sec      = 86400
      continue_on_match = false
    }]
  })
}
```

See `examples/rules/main.tf` for 8 complete examples!

## API Endpoints Used

- `GET /api/v1/domains/{domain}/services/rp/rules` - List all rules
- `GET /api/v1/domains/{domain}/services/rp/rules/phases/{phase}` - List rules in phase
- `GET /api/v1/domains/{domain}/services/rp/rules/{uuid}` - Get rule details
- `POST /api/v1/domains/{domain}/services/rp/rules/phases/{phase}` - Create rule
- `PATCH /api/v1/domains/{domain}/services/rp/rules/{uuid}` - Update rule
- `DELETE /api/v1/domains/{domain}/services/rp/rules/{uuid}` - Delete rule

## Filter Syntax (Wirefilter)

Rules use Wirefilter expressions to match requests:

```
# Path matching
http.request.uri.path == "/api/users"
http.request.uri.path matches "^/api/.*"

# User agent
http.user_agent contains "bot"

# IP address
ip.src != 192.0.2.1

# Multiple conditions
http.request.uri.path matches "^/admin/" and ip.src != 192.0.2.1

# Headers
http.request.headers["X-API-Key"] == "secret123"
```

## Why Rules Are Powerful

1. **Flexible**: Conditional logic based on any request property
2. **Fast**: Execute at edge, no origin impact
3. **Safe**: Test with `enabled = false` first
4. **Ordered**: Rules process in sequence, first match wins (with continue_on_match)
5. **Comprehensive**: Cover firewall, caching, headers, rate limiting, routing

## Common Use Cases

### Security
- Block malicious IPs/user agents
- Challenge suspicious bots
- Rate limit API endpoints
- Add security headers

### Performance
- Force caching for static assets
- Add cache tags for selective purging
- Enable compression for specific paths

### Routing
- Route API traffic to different backends
- Rewrite legacy URLs
- Select origin pools based on path

### Compliance
- Add CORS headers
- Remove sensitive headers
- Enforce HTTPS

## Integration with Existing Resources

Rules work alongside other resources:

```hcl
# 1. Create domain
resource "peakhour_domain" "app" {
  name = "myapp.com"
}

# 2. Enable service
resource "peakhour_reverse_proxy_service" "app" {
  domain = peakhour_domain.app.name
}

# 3. Add origins
resource "peakhour_origin_pool" "backend" {
  domain = peakhour_domain.app.name
  tag    = "main"
  # ...
}

# 4. Add rules
resource "peakhour_rule" "security" {
  domain     = peakhour_domain.app.name
  phase      = "firewall"
  name       = "Security Rules"
  filter_str = "..."
  actions_json = jsonencode({ ... })
  depends_on = [peakhour_reverse_proxy_service.app]
}
```

## Next Steps

1. **Try It**: Check out `examples/rules/main.tf`
2. **Read Guide**: See `RULES_GUIDE.md` for comprehensive documentation
3. **Test Safely**: Start with `enabled = false` and firewall action "log"
4. **Import Existing**: Import your existing rules from the UI
5. **Build Complex Logic**: Combine multiple rules in different phases

## Build & Test

```bash
# Rebuild provider
make build

# Test with examples
cd examples/rules
export PEAKHOUR_API_KEY="your-key"
terraform init
terraform plan
terraform apply
```

## Summary

- **New Resource**: `peakhour_rule`
- **10 Phases Supported**: From request rewrite to response headers
- **8+ Action Types**: Firewall, headers, caching, rate limiting, routing
- **Comprehensive Examples**: 8 real-world examples in `examples/rules/`
- **Full Documentation**: `RULES_GUIDE.md` with syntax and best practices
- **Production Ready**: Full CRUD, import support, proper state management

Rules give your clients the full power of Peakhour's edge logic in Terraform! 🚀
