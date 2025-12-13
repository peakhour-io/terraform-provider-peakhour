# Quick Start Guide

## Prerequisites

1. Go 1.22+ installed
2. Terraform 1.0+ installed
3. Peakhour API key

## Setup

### 1. Build the Provider

```bash
cd /home/accassar/src/peakhour/peakhour-terraform
make build
```

### 2. Install Locally for Testing

```bash
make install VERSION=0.1.0
```

Or manually:

```bash
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/peakhour-io/peakhour/0.1.0/linux_amd64
cp terraform-provider-peakhour ~/.terraform.d/plugins/registry.terraform.io/peakhour-io/peakhour/0.1.0/linux_amd64/terraform-provider-peakhour_v0.1.0
```

### 3. Configure Local Development

Create `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "peakhour-io/peakhour" = "/home/accassar/src/peakhour/peakhour-terraform"
  }
  direct {}
}
```

## Test with Example

```bash
cd examples/basic
export PEAKHOUR_API_KEY="your-api-key-here"
terraform validate
terraform plan
terraform apply
```

## Example Usage

```hcl
terraform {
  required_providers {
    peakhour = {
      source = "peakhour-io/peakhour"
    }
  }
}

provider "peakhour" {
  # API key from PEAKHOUR_API_KEY env var
}

# 1. Create domain
resource "peakhour_domain" "mysite" {
  name = "mysite.com"
}

# 2. Enable reverse proxy service
resource "peakhour_reverse_proxy_service" "mysite" {
  domain = peakhour_domain.mysite.name
}

# 3. Configure reverse proxy
resource "peakhour_reverse_proxy_config" "mysite" {
  domain    = peakhour_domain.mysite.name
  gzip      = true
  brotli    = true
  websocket = true

  depends_on = [peakhour_reverse_proxy_service.mysite]
}

# 4. Add origin pool
resource "peakhour_origin_pool" "backend" {
  domain = peakhour_domain.mysite.name
  tag    = "production"

  address = [
    {
      address = "backend.internal:8080"
      weight  = 100
    },
  ]

  shield_name         = "sydney"
  load_balancing_mode = "round_robin"

  depends_on = [peakhour_reverse_proxy_service.mysite]
}

# 5. Configure transforms
resource "peakhour_transform_settings" "mysite" {
  domain                   = peakhour_domain.mysite.name
  transform_html           = true
  transform_image_api      = true
  transform_image_quality  = 85

  depends_on = [peakhour_reverse_proxy_service.mysite]
}

# 6. Add rate limit zone
resource "peakhour_rate_limit_zone" "api_limit" {
  domain                = peakhour_domain.mysite.name
  name                  = "api_limit"
  requests_max          = 100
  requests_interval_sec = 60
  block_duration_sec    = 300

  depends_on = [peakhour_reverse_proxy_service.mysite]
}

# 7. Add IP blocklist
resource "peakhour_rule_list" "blocked_ips" {
  domain = peakhour_domain.mysite.name
  name   = "blocked_ips"
  type   = "ip"

  ips = ["192.0.2.0/24", "198.51.100.0/24"]

  depends_on = [peakhour_reverse_proxy_service.mysite]
}

# 8. Add firewall rule using the blocklist
resource "peakhour_rule" "block_ips" {
  domain     = peakhour_domain.mysite.name
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
    peakhour_reverse_proxy_service.mysite,
    peakhour_rule_list.blocked_ips
  ]
}

# 9. Add rate limiting rule
resource "peakhour_rule" "api_ratelimit" {
  domain     = peakhour_domain.mysite.name
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
    peakhour_reverse_proxy_service.mysite,
    peakhour_rate_limit_zone.api_limit
  ]
}
```

## API Endpoints Used

The provider calls these Peakhour API endpoints:

- `POST /api/v1/domains` - Create domain
- `GET /api/v1/domains/{domain}` - Get domain
- `DELETE /api/v1/domains/{domain}` - Delete domain
- `POST /api/v1/domains/{domain}/services` - Enable service
- `DELETE /api/v1/domains/{domain}/services/rp` - Disable service
- `PATCH /api/v1/domains/{domain}/services/rp` - Update RP config
- `GET /api/v1/domains/{domain}/services/rp` - Get RP config
- `POST /api/v1/domains/{domain}/origins` - Create origin pool
- `PUT /api/v1/domains/{domain}/origins` - Update origin pool
- `DELETE /api/v1/domains/{domain}/origins` - Delete origin pool
- `GET /api/v1/domains/{domain}/origins` - List origin pools
- `PATCH /api/v1/domains/{domain}/services/rp/transforms` - Update transforms
- `GET /api/v1/domains/{domain}/services/rp/transforms` - Get transforms
- `POST /api/v1/domains/{domain}/services/rp/rules/phases/{phase}` - Create rule
- `GET /api/v1/domains/{domain}/services/rp/rules/{uuid}` - Get rule
- `PATCH /api/v1/domains/{domain}/services/rp/rules/{uuid}` - Update rule
- `DELETE /api/v1/domains/{domain}/services/rp/rules/{uuid}` - Delete rule
- `POST /api/v1/domains/{domain}/services/rp/rate-limit-zones` - Create rate limit zone
- `GET /api/v1/domains/{domain}/services/rp/rate-limit-zones/{name}` - Get rate limit zone
- `PATCH /api/v1/domains/{domain}/services/rp/rate-limit-zones/{name}` - Update rate limit zone
- `DELETE /api/v1/domains/{domain}/services/rp/rate-limit-zones/{name}` - Delete rate limit zone
- `POST /api/v1/domains/{domain}/services/rp/lists` - Create rule list
- `GET /api/v1/domains/{domain}/services/rp/lists/{uuid}` - Get rule list
- `PATCH /api/v1/domains/{domain}/services/rp/lists/{uuid}` - Update rule list
- `DELETE /api/v1/domains/{domain}/services/rp/lists/{uuid}` - Delete rule list

## Troubleshooting

### Provider not found

Make sure you've either:
1. Installed it: `make install`
2. Or configured dev overrides in `~/.terraformrc`

## Private distribution (clients)

For shipping a private binary bundle to clients (filesystem mirror), see `docs/provider-distribution.md`.

### API Authentication errors

Ensure `PEAKHOUR_API_KEY` is set:
```bash
export PEAKHOUR_API_KEY="your-key"
```

### Resource already exists

Import existing resources:
```bash
terraform import peakhour_domain.example example.com
terraform import peakhour_reverse_proxy_service.example example.com
terraform import peakhour_origin_pool.backend example.com/production
terraform import peakhour_rule.my_rule example.com/rule-uuid-here
terraform import peakhour_rate_limit_zone.api_limit example.com/api_limit
terraform import peakhour_rule_list.blocked_ips example.com/list-uuid-here
```

## More Examples

See the `examples/` directory for comprehensive examples:
- `examples/basic/` - Simple domain setup
- `examples/full-setup/` - Complete configuration
- `examples/rules/` - 8 rule examples covering all phases
- `examples/rate-limiting/` - Rate limiting zones and usage
- `examples/rule-lists/` - IP/string/integer lists with firewall rules

## Next Steps

1. Review the [RULES_GUIDE.md](RULES_GUIDE.md) for advanced rule configuration
2. Publish to Terraform Registry
3. Set up CI/CD for automated testing and releases
