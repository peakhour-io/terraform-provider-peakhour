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
  - `peakhour_rp_cdn_purge_resources`
  - `peakhour_rp_cdn_purge_wildcard`
  - `peakhour_rp_cdn_purge_tags`
  - `peakhour_rp_bots`
  - `peakhour_rp_threat_access_list_rule`
  - `peakhour_rp_threat_block_list`
  - `peakhour_rp_waf_options`
  - `peakhour_rp_waf_owasp_settings`
  - `peakhour_rp_firewall_settings`
  - `peakhour_rp_firewall_error_page`
  - `peakhour_rp_lua_options`
- Updated onboarding inventory to include config resources (`internal/onboard/inventory.go`) (purge resources are actions and are not inventoried).
- Extended unit/contract checks to cover spec paths and provider registration (`internal/spec/contract_test.go`).
- Updated docs/examples for the new resources (`README.md`, `examples/full-setup/main.tf`, `examples/rate-limiting/main.tf`, `examples/cdn-purge/main.tf`, `examples/threats/main.tf`, `examples/waf/main.tf`).

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
- CDN purge actions (run-once): `peakhour_rp_cdn_purge_resources`, `peakhour_rp_cdn_purge_wildcard`, `peakhour_rp_cdn_purge_tags`
- Bots config: `peakhour_rp_bots` (`/services/rp/bots`)
- Threat access list rules: `peakhour_rp_threat_access_list_rule` (`/services/rp/threats/access_list`)
- Threat block list selection: `peakhour_rp_threat_block_list` (`/services/rp/threats/block_list`)
- WAF options: `peakhour_rp_waf_options` (`/services/rp/waf`)
- WAF OWASP settings: `peakhour_rp_waf_owasp_settings` (`/services/rp/waf/owasp`)
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
- ✅ Flush/purge endpoints: modeled as explicit “run once” Terraform resources:
  - `peakhour_rp_cdn_purge_resources` (`POST /api/v1/domains/{domain}/services/rp/cdn/resources/flush`)
  - `peakhour_rp_cdn_purge_wildcard` (`POST /api/v1/domains/{domain}/services/rp/cdn/wildcard/flush`)
  - `peakhour_rp_cdn_purge_tags` (`POST /api/v1/domains/{domain}/services/rp/cdn/tag/flush`)

**Terraform design note:** These are actions, not stable desired-state. They run on `Create` (and on changes via `RequiresReplace`). Re-run by bumping `run_id` (or tainting).

### Bots config (done)
- ✅ `GET/PATCH /api/v1/domains/{domain}/services/rp/bots` (`peakhour_rp_bots`)

### Firewall config (done)
- ✅ `GET/POST /api/v1/domains/{domain}/services/rp/firewall` (`peakhour_rp_firewall_settings`)
- ✅ `GET/PUT /api/v1/domains/{domain}/services/rp/firewall/error_page` (`peakhour_rp_firewall_error_page`)

### Lua hooks (done)
- ✅ `GET/PUT /api/v1/domains/{domain}/services/rp/lua` (`peakhour_rp_lua_options`)

### Threats lists (done)
- ✅ Access list rules: `GET/PUT /api/v1/domains/{domain}/services/rp/threats/access_list` (`peakhour_rp_threat_access_list_rule`)
- ✅ Access list rule CRUD: `GET/PATCH/DELETE /api/v1/domains/{domain}/services/rp/threats/access_list/{rule}` (`peakhour_rp_threat_access_list_rule`)
- ✅ Block lists selection: `GET/POST /api/v1/domains/{domain}/services/rp/threats/block_list` (`peakhour_rp_threat_block_list`)

### WAF (large surface area; partial)
- ✅ WAF options: `GET/PATCH /api/v1/domains/{domain}/services/rp/waf` (`peakhour_rp_waf_options`)
- ✅ OWASP settings: `GET/PATCH /api/v1/domains/{domain}/services/rp/waf/owasp` (`peakhour_rp_waf_owasp_settings`)
- ⏳ Custom rules: `.../waf/customrule*`
- ⏳ Rulesets/rulegroups/rules: `.../waf/ruleset*` and `.../waf/rule/{rule}`

## Additional spec modules not modeled (likely separate milestones)

- Domain plans/billing: `/api/v1/plans*`, `/api/v1/domains/{domain}/plan*`
- “Edge Access” product suite: `/api/v1/edge_access/*` (lists/policies/rules/secrets/tokens + config log/commit workflow)

## Learnings / notes (from implementation)

- **Omit vs clear matters**: many endpoints accept `null` to clear values; for PATCH/POST-based config we used `map[string]any` to preserve the distinction between “unset” and “explicitly null”.
- **Some config is not readable**: `/services/rp/firewall/error_page` indicates whether a page exists, but doesn’t return the content; Terraform can’t auto-verify drift of `content`.
- **Some secrets are write-only**: `/services/rp/ssl/certificate` does not return uploaded PEM/key material; Terraform can store what you upload, but cannot validate drift.
- **Spec-required-but-nullable fields**: `LuaOptions` fields are required by the schema but can be `null`; we send a full object (including `null` values) to satisfy “required” without forcing defaults.
- **Action endpoints in Terraform**: purge/flush endpoints are best modeled as “run once” resources with a `run_id` input (and `RequiresReplace`), not as normal desired-state resources.
- **Go toolchain**: the repo requires Go 1.22; `go.mod` must use `go 1.22` (not `go 1.22.0`) and local tests should run with the pinned toolchain in `.toolchains/go1.22.10/bin/go` (or any Go 1.22+).

## Proposed next implementation order (remaining)

1. **WAF suite (remaining)**
   - Custom rules CRUD + reorder.
   - Ruleset/rulegroup/rule toggles (enable/disable).

For each new area:
- Add `internal/client` methods + DTOs as needed.
- Add Terraform resource(s) with consistent import IDs and “reset-on-delete” semantics where appropriate.
- Add lightweight `internal/spec/contract_test.go` checks for key schema/paths to keep spec drift visible in unit tests.
- Add `TF_ACC` acceptance tests (Jenkins gated) for at least one happy-path per new resource type.
- Update the onboarding inventory (`internal/onboard/inventory.go`) so importer+`-generate-config-out` supports these resources too.
