# Peakhour Terraform Provider - Beta Tester Guide

This guide explains how to install and use the Peakhour Terraform Provider from a binary (not yet in the public Terraform Registry).

## Prerequisites

- Terraform v1.0+ installed
- A Peakhour account with API access
- Your Peakhour API key

## Step 1: Install the Provider Binary

Place the `terraform-provider-peakhour` binary somewhere on your system (e.g., `/opt/peakhour/`):

```bash
mkdir -p /opt/peakhour
cp terraform-provider-peakhour /opt/peakhour/
chmod +x /opt/peakhour/terraform-provider-peakhour
```

## Step 2: Configure Terraform to Use the Local Binary

Create or edit `~/.terraformrc` (Linux/macOS) or `%APPDATA%\terraform.rc` (Windows):

```hcl
provider_installation {
  dev_overrides {
    "peakhour-io/peakhour" = "/opt/peakhour"
  }
  direct {}
}
```

> **Note:** With `dev_overrides`, you skip `terraform init` for this provider. Terraform uses the binary directly.

## Step 3: Set Environment Variables

```bash
# Required - Your Peakhour API key
export PEAKHOUR_API_KEY="your-api-key-here"

# Optional - Override API endpoint (defaults to https://www.peakhour.io)
# export PEAKHOUR_BASE_URL="https://console.staging.peakhour.io"
```

**Windows (PowerShell):**
```powershell
$env:PEAKHOUR_API_KEY = "your-api-key-here"
```

## Step 4: Write Your Terraform Configuration

Create a `main.tf` file:

```hcl
terraform {
  required_providers {
    peakhour = {
      source = "peakhour-io/peakhour"
    }
  }
}

provider "peakhour" {}

# Create a domain
resource "peakhour_domain" "example" {
  name = "example.com"
}

# Assign a plan to the domain
resource "peakhour_domain_plan" "example" {
  domain = peakhour_domain.example.name
  code   = "basic"  # Your plan code: basic, business, business2, etc.
}

# Enable reverse proxy (CDN) service
resource "peakhour_reverse_proxy_service" "example" {
  domain     = peakhour_domain.example.name
  depends_on = [peakhour_domain_plan.example]
}

# Add an origin pool
resource "peakhour_origin_pool" "backend" {
  domain = peakhour_domain.example.name
  tag    = "production"

  address = [
    {
      address = "http://192.0.2.1:8080"
      weight  = 100
    }
  ]

  load_balancing_mode = "round_robin"
  depends_on          = [peakhour_reverse_proxy_service.example]
}
```

## Step 5: Run Terraform

```bash
# Preview changes (no 'terraform init' needed with dev_overrides)
terraform plan

# Apply changes
terraform apply
```

You'll see a warning about dev_overrides - this is expected and can be ignored.

## Environment Variables Reference

| Variable | Required | Description |
|----------|----------|-------------|
| `PEAKHOUR_API_KEY` | Yes | Your Peakhour API key |
| `PEAKHOUR_BASE_URL` | No | API endpoint (default: `https://www.peakhour.io`) |

## Available Resources

| Resource | Description |
|----------|-------------|
| `peakhour_domain` | Create/manage domains |
| `peakhour_domain_plan` | Assign a plan to a domain |
| `peakhour_reverse_proxy_service` | Enable CDN/reverse proxy |
| `peakhour_reverse_proxy_config` | Configure compression, WebSockets, aliases |
| `peakhour_origin_pool` | Origin servers with load balancing |
| `peakhour_acme_settings` | ACME domain-name settings |
| `peakhour_acme_certificate` | ACME certificate issuance |
| `peakhour_rule` | Firewall, caching, headers, rate limiting rules |
| `peakhour_rate_limit_zone` | Rate limiting zones |
| `peakhour_rule_list` | IP/string/integer lists for rules |
| `peakhour_bulk_redirect_list` | Redirect lists |
| `peakhour_bulk_redirect_entry` | Redirect entries |
| `peakhour_rp_waf_options` | WAF settings |
| `peakhour_rp_waf_custom_rule` | Custom WAF rules |
| `peakhour_transform_settings` | HTML/image transform settings |
| `peakhour_image_transform` | Image optimization rules |
| `peakhour_rp_cdn_purge_resources` | Purge individual cached resources |
| `peakhour_rp_cdn_purge_wildcard` | Purge cached wildcard paths |
| `peakhour_rp_cdn_purge_tags` | Purge cached tags |

## Troubleshooting

### "Could not find provider" error

Check that:
1. The binary path in `~/.terraformrc` matches where you placed the binary
2. The binary is executable (`chmod +x`)

### "401 Unauthorized" error

Verify your API key: `echo $PEAKHOUR_API_KEY`

### Provider not being used

With `dev_overrides`, you should see this warning when running `terraform plan`:
```
Warning: Provider development overrides are in effect
```
If you don't see this, check your `~/.terraformrc` syntax.

---

**Provider Version:** 0.1.2 (Beta)
