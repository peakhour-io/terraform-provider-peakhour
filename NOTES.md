# Terraform Provider Development Notes

## Current Issues

### 1. WAF Rule Group (`peakhour_rp_waf_rule_group`)

**Status:** Not working - needs backend investigation

**Error:** API returns 500 error due to `sqlalchemy.exc.NoResultFound`

**Root Cause:** The backend lookup at `peakhour/reverseproxy/waf/api/resources.py:77-79` queries by `file_name` only:
```python
result = WafRuleGroup.query.filter(
    WafRuleGroup.file_name == slug,
).one()
```

The `file_name` must match an existing row in the `waf_file` database table. The staging database may not have the same WAF rule groups as production.

**Next Steps:**
- Query staging database to find available `file_name` values for the `peakhour` ruleset
- Or expose an API endpoint to list available rule groups
- Consider returning 404 instead of 500 when rule group not found

### 2. Transform Settings (`peakhour_transform_settings`)

**Status:** Not working - backend 500 error

**Root Cause:** The `set_transform_options` helper at `peakhour/reverseproxy/helpers.py:250` calls:
```python
canned.images(self.model)
```

This function in `peakhour/reverseproxy/rules/canned.py` may be failing due to missing staging environment configuration.

**Next Steps:**
- Check server error logs for detailed stack trace
- Investigate what `canned.images()` requires to succeed

## Design Patterns Used

### JSON Fields with Server-Side Defaults

Resources with JSON fields (e.g., `rules_json`, `action_json`, `settings_json`) use a custom `JSONNormalizedType` with **subset equality semantics**:

- User config is a **subset** of API response (API adds server defaults)
- Drift detection only compares keys the user explicitly set
- This prevents constant "changes detected" due to server-added defaults

**Pattern for Create/Update:**
```go
// Don't read from API after Create/Update - trust the plan values
// Semantic equality in Read() handles drift detection
resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
```

**Resources using this pattern:**
- `rp_waf_custom_rule` - rules_json, action_json, logging_json
- `rp_waf_owasp_settings` - settings_json
- `rp_threat_block_list` - blocklists
- `rule` - actions_json

### Resources That Read After Create/Update

Some resources need to read from API after Create/Update to populate computed fields with server-side values:

- `reverse_proxy_config` - computed boolean fields
- `rate_limit_global` - computed rate limit settings

### Preserving Computed Fields in Update

For resources where computed fields don't change during Update, preserve them from state:
```go
func (r *Resource) Update(...) {
    var plan, state Model
    req.Plan.Get(ctx, &plan)
    req.State.Get(ctx, &state)

    // ... API update ...

    // Preserve computed fields from state
    plan.RuleID = state.RuleID
    plan.Created = state.Created
}
```

## Origin Pool Address Format

Origin addresses must use full URL format with scheme:
- `http://hostname:port` (e.g., `http://origin.example.com:8080`)
- `https://hostname:port` (e.g., `https://origin.example.com:8443`)

**Not valid:** `hostname:port` without scheme

## API Quirks

### redirect_mode "none"

The API returns `redirect_mode: "none"` as the default/disabled state. The provider treats this as `null` in Terraform state to match user expectations.

### redirect_status_code Validation

The API validates `redirect_status_code` as an enum (301, 302, 307, 308). Sending `0` fails validation. On delete, omit this field entirely rather than sending zero.

### BulkRedirectEntry.id Type

API returns `id` as integer but schema expects string. Custom `StringOrInt` type handles unmarshaling both.
