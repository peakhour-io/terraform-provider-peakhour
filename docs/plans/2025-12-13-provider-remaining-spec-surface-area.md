# Provider Remaining Spec Surface Area (SSL/ACME/WAF/etc.) — Gap Review + Plan

**Source of truth:** `docs/spec/peakhour-api-v1.json`

## Status (as of 2025-12-13)

Completed from this plan (merged):
- Added resources:
  - `peakhour_rate_limit_settings`
  - `peakhour_rp_settings`
  - `peakhour_rp_ssl_config`
  - `peakhour_rp_ssl_certificate`
  - `peakhour_acme_settings`
  - `peakhour_acme_certificate`
  - `peakhour_rp_origin_config`
  - `peakhour_rp_cdn_cache`
  - `peakhour_rp_bots`
  - `peakhour_rp_firewall_settings`
  - `peakhour_rp_firewall_error_page`
  - `peakhour_rp_lua_options`
- Updated onboarding inventory to include these resources (`internal/onboard/inventory.go`).
- Extended unit/contract checks to cover spec paths and provider registration (`internal/spec/contract_test.go`).
- Updated docs/examples for the new resources (`README.md`, `examples/full-setup/main.tf`, `examples/rate-limiting/main.tf`).

## What the Terraform provider covers today

The provider currently models these API areas:
- Domains: `peakhour_domain`
- RP service enablement: `peakhour_reverse_proxy_service`
- RP config (incl “vhost aliases”): `peakhour_reverse_proxy_config` (maps `ReverseProxyConfig.aliases`)
- RP settings: `peakhour_rp_settings` (`/services/rp/settings`)
- RP SSL/TLS cipher profile: `peakhour_rp_ssl_config` (`/services/rp/ssl`)
- RP SSL/TLS certificate upload + info: `peakhour_rp_ssl_certificate` (`/services/rp/ssl/certificate`)
- Origin pools: `peakhour_origin_pool` (`/api/v1/domains/{domain}/origins`)
- Origin behavior: `peakhour_rp_origin_config` (`/services/rp/origin`)
- CDN cache config: `peakhour_rp_cdn_cache` (`/services/rp/cdn`)
- Bots config: `peakhour_rp_bots` (`/services/rp/bots`)
- Firewall settings: `peakhour_rp_firewall_settings` (`/services/rp/firewall`)
- Firewall error page: `peakhour_rp_firewall_error_page` (`/services/rp/firewall/error_page`)
- Lua options: `peakhour_rp_lua_options` (`/services/rp/lua`)
- RP transforms: `peakhour_transform_settings` (`/services/rp/transforms`)
- Rules + lists + bulk redirects: `peakhour_rule`, `peakhour_rule_list`, `peakhour_bulk_redirect_list`, `peakhour_bulk_redirect_entry`
- Rate limiting: `peakhour_rate_limit_settings`, `peakhour_rate_limit_global`, `peakhour_rate_limit_zone`
- ACME settings: `peakhour_acme_settings` (`/services/acme/settings`)
- ACME certificate status/issue trigger: `peakhour_acme_certificate` (`/services/acme/certificate`)
- Image transforms (presets): `peakhour_image_transform`

## “Vhost settings” in the spec

There is **no standalone “vhost settings” endpoint** in the OpenAPI paths.

What looks “vhost-ish” is split across:
- Reverse proxy aliases: `ReverseProxyConfig.aliases` (already supported by `peakhour_reverse_proxy_config`).
- Rate limiting modes: `RateLimitSettings.mode` includes `vhost` and `vhost-busy` (now supported by `peakhour_rate_limit_settings`).

## Big spec areas not yet modeled by Terraform (high value)

### TLS / SSL and ACME (partial)
- ✅ SSL config: `GET/PUT /api/v1/domains/{domain}/services/rp/ssl` (`peakhour_rp_ssl_config`)
- ✅ SSL certificate: `GET/PUT /api/v1/domains/{domain}/services/rp/ssl/certificate` (`peakhour_rp_ssl_certificate`)
- ✅ ACME settings: `GET/PUT /api/v1/domains/{domain}/services/acme/settings` (`peakhour_acme_settings`)
- ✅ ACME certificate status/contents + issuance trigger: `GET/POST /api/v1/domains/{domain}/services/acme/certificate` (`peakhour_acme_certificate`)

**Terraform design note:** `SSLCertificateAdd.private_key` is write-only in practice (API does not return it). If you use `peakhour_rp_ssl_certificate`, the private key is stored in Terraform state (marked sensitive) and drift cannot be automatically verified.

### Reverse proxy service settings (done)
- ✅ `GET/PATCH /api/v1/domains/{domain}/services/rp/settings` (`peakhour_rp_settings`)

### Origin behavior (distinct from origin pools) (done)
- ✅ `GET/POST /api/v1/domains/{domain}/services/rp/origin` (`peakhour_rp_origin_config`)

### CDN caching config + flush actions (partial)
- ✅ Cache config: `GET/PATCH /api/v1/domains/{domain}/services/rp/cdn` (`peakhour_rp_cdn_cache`)
- Flush/purge endpoints exist (resources/tag/wildcard). These are “actions”, not stable desired-state; if we model them in Terraform, they should be explicit “run once” style resources or handled outside Terraform.

### Bots config (done)
- ✅ `GET/PATCH /api/v1/domains/{domain}/services/rp/bots` (`peakhour_rp_bots`)

### Firewall config (done)
- ✅ `GET/POST /api/v1/domains/{domain}/services/rp/firewall` (`peakhour_rp_firewall_settings`)
- ✅ `GET/PUT /api/v1/domains/{domain}/services/rp/firewall/error_page` (`peakhour_rp_firewall_error_page`)

### Lua hooks (done)
- ✅ `GET/PUT /api/v1/domains/{domain}/services/rp/lua` (`peakhour_rp_lua_options`)

### Threats lists (missing)
- Access list rules: `GET/PUT /api/v1/domains/{domain}/services/rp/threats/access_list`
- Access list rule CRUD: `GET/PATCH/DELETE /api/v1/domains/{domain}/services/rp/threats/access_list/{rule}`
- Block lists: `GET/POST /api/v1/domains/{domain}/services/rp/threats/block_list`

### WAF (large surface area; missing)
- WAF options: `GET/POST /api/v1/domains/{domain}/services/rp/waf`
- Custom rules: `.../waf/customrule*`
- OWASP toggles/settings: `.../waf/owasp`
- Rulesets/rulegroups/rules: `.../waf/ruleset*` and `.../waf/rule/{rule}`

## Additional spec modules not modeled (likely separate milestones)

- Domain plans/billing: `/api/v1/plans*`, `/api/v1/domains/{domain}/plan*`
- “Edge Access” product suite: `/api/v1/edge_access/*` (lists/policies/rules/secrets/tokens + config log/commit workflow)

## Learnings / notes (from implementation)

- **Omit vs clear matters**: many endpoints accept `null` to clear values; for PATCH/POST-based config we used `map[string]any` to preserve the distinction between “unset” and “explicitly null”.
- **Some config is not readable**: `/services/rp/firewall/error_page` indicates whether a page exists, but doesn’t return the content; Terraform can’t auto-verify drift of `content`.
- **Some secrets are write-only**: `/services/rp/ssl/certificate` does not return uploaded PEM/key material; Terraform can store what you upload, but cannot validate drift.
- **Spec-required-but-nullable fields**: `LuaOptions` fields are required by the schema but can be `null`; we send a full object (including `null` values) to satisfy “required” without forcing defaults.
- **Go toolchain**: the repo requires Go 1.22; `go.mod` must use `go 1.22` (not `go 1.22.0`) and local tests should run with the pinned toolchain in `.toolchains/go1.22.10/bin/go` (or any Go 1.22+).

## Proposed next implementation order (remaining)

1. **Cache purge/flush actions**
   - If desired, model as explicit “run once” resources (not normal config) or keep outside Terraform.
2. **Threats lists**
   - New resources for access list and block list.
3. **WAF suite**
   - Multiple resources + reorder operations; biggest effort.

For each new area:
- Add `internal/client` methods + DTOs as needed.
- Add Terraform resource(s) with consistent import IDs and “reset-on-delete” semantics where appropriate.
- Add lightweight `internal/spec/contract_test.go` checks for key schema/paths to keep spec drift visible in unit tests.
- Add `TF_ACC` acceptance tests (Jenkins gated) for at least one happy-path per new resource type.
- Update the onboarding inventory (`internal/onboard/inventory.go`) so importer+`-generate-config-out` supports these resources too.
