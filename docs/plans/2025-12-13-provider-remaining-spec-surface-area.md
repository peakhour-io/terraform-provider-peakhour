# Provider Remaining Spec Surface Area (SSL/ACME/WAF/etc.) — Gap Review + Plan

**Source of truth:** `docs/spec/peakhour-api-v1.json`

## What the Terraform provider covers today

The provider currently models these API areas:
- Domains: `peakhour_domain`
- RP service enablement: `peakhour_reverse_proxy_service`
- RP config (incl “vhost aliases”): `peakhour_reverse_proxy_config` (maps `ReverseProxyConfig.aliases`)
- Origin pools: `peakhour_origin_pool` (`/api/v1/domains/{domain}/origins`)
- RP transforms: `peakhour_transform_settings` (`/services/rp/transforms`)
- Rules + lists + bulk redirects: `peakhour_rule`, `peakhour_rule_list`, `peakhour_bulk_redirect_list`, `peakhour_bulk_redirect_entry`
- Rate limiting (partial): `peakhour_rate_limit_global`, `peakhour_rate_limit_zone`
- Image transforms (presets): `peakhour_image_transform`

## “Vhost settings” in the spec

There is **no standalone “vhost settings” endpoint** in the OpenAPI paths.

What looks “vhost-ish” is split across:
- Reverse proxy aliases: `ReverseProxyConfig.aliases` (already supported by `peakhour_reverse_proxy_config`).
- Rate limiting modes: `RateLimitSettings.mode` includes `vhost` and `vhost-busy` (currently **not** modeled by Terraform).

## Big spec areas not yet modeled by Terraform (high value)

### TLS / SSL and ACME (missing)
- SSL config: `GET/PUT /api/v1/domains/{domain}/services/rp/ssl` (`SSLConfig.ciphers`)
- SSL certificate: `GET/PUT /api/v1/domains/{domain}/services/rp/ssl/certificate`
- ACME settings: `GET/PUT /api/v1/domains/{domain}/services/acme/settings` (`AcmeSettings.domain_names`)
- ACME certificate status/contents + issuance trigger: `GET/POST /api/v1/domains/{domain}/services/acme/certificate`

**Terraform design note:** `SSLCertificateAdd.private_key` is write-only in practice (API does not return it). If modeled as an attribute, it will end up in Terraform state (even if marked sensitive). Consider prioritizing ACME resources first if you want to avoid private key material in state.

### Reverse proxy service settings (missing)
- `GET/PATCH /api/v1/domains/{domain}/services/rp/settings` (`notification_emails`, `quickstart`, plus computed `ipv4_address/ipv6_address/cname`)

### Origin behavior (distinct from origin pools) (missing)
- `GET/POST /api/v1/domains/{domain}/services/rp/origin` (`OriginConfig.ssl_mode`, origin error/downtime thresholds, `OriginRequestHeaders` toggles)

### CDN caching config + flush actions (missing)
- Cache config: `GET/PATCH /api/v1/domains/{domain}/services/rp/cdn` (`CacheConfig`)
- Flush/purge endpoints exist (resources/tag/wildcard). These are “actions”, not stable desired-state; if we model them in Terraform, they should be explicit “run once” style resources or handled outside Terraform.

### Bots config (missing)
- `GET/PATCH /api/v1/domains/{domain}/services/rp/bots` (`BotConfig`)

### Firewall config (missing)
- `GET/POST /api/v1/domains/{domain}/services/rp/firewall` (`FirewallSettings`)
- `GET/POST /api/v1/domains/{domain}/services/rp/firewall/error_page`

### Lua hooks (missing)
- `GET/PUT /api/v1/domains/{domain}/services/rp/lua` (`LuaOptions`)

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

## Proposed next implementation order (small → large)

1. **Rate limiting modes resource** (unblocks `vhost` mode in spec)
   - New: `peakhour_rate_limit_settings` backed by `POST /services/rp/rate_limit` (`RateLimitSettings.mode`)
2. **RP settings resource**
   - New: `peakhour_rp_settings` backed by `GET/PATCH /services/rp/settings`
3. **TLS/ACME resources**
   - New: `peakhour_rp_ssl_config` (`/services/rp/ssl`)
   - New: `peakhour_acme_settings` (`/services/acme/settings`)
   - Optional: certificate upload resource (with the state/private-key caveat)
4. **Origin config resource**
   - New: `peakhour_rp_origin_config` (`/services/rp/origin`)
5. **Cache/Bots/Firewall/Lua** (each is a singleton config resource)
6. **Threats lists** (list-of-rules style resources)
7. **WAF suite** (multi-resource + reorder operations; biggest effort)

For each new area:
- Add `internal/client` methods + DTOs as needed.
- Add Terraform resource(s) with consistent import IDs and “reset-on-delete” semantics where appropriate.
- Add lightweight `internal/spec/contract_test.go` checks for key schema/paths to keep spec drift visible in unit tests.
- Add `TF_ACC` acceptance tests (Jenkins gated) for at least one happy-path per new resource type.
- Update the onboarding inventory (`internal/onboard/inventory.go`) so importer+`-generate-config-out` supports these resources too.

