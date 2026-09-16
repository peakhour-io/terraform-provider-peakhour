# WAF custom-rule JSON defaults

The provider compares `peakhour_rp_waf_custom_rule.rules_json`, `action_json`,
and `logging_json` using the defaults declared in its bundled API schema.
For example, `{"action_name":"deny"}` is equivalent to the expanded API object
with `action_arg_val: null`, `action_arg_val_param: null`, and
`action_arg_val_param_val: ""`.

Compact configuration can therefore match an imported rule without issuing a
PATCH solely to change its JSON representation. The provider retains an
equivalent prior-state string rather than rewriting HCL or storing an arbitrary
normalized representation. State JSON can still contain the expanded defaults.

## Upgrade effects

The old subset comparison could hide extra API fields even when their values
were not defaults. The new comparison owns the complete configured JSON document:

| Configuration | Remote value | Result |
| --- | --- | --- |
| `operator_negated` omitted | `false` | Equivalent |
| `operator_negated` omitted | `true` | Update to restore the default |
| `transforms` omitted | `[]` | Equivalent |
| `transforms` omitted | `["lowercase"]` | Update to remove the transform |
| `action_arg_val_param_val` omitted | `""` | Equivalent |
| `action_arg_val_param_val` omitted | `null` | Different; explicit null is preserved |
| `logging_json = "{}"` | `{"tags":null}` | Equivalent |
| `logging_json = "{}"` | `{"tags":[]}` | Different; empty arrays are not null |
| Unknown property omitted | Unknown property present, including null | Different |

Before applying after an upgrade, inspect the plan. Copy any intended non-default
remote values into HCL. Applying a reset sends the existing full custom-rule PATCH
and causes the API to regenerate configuration; this can change WAF behavior.
Do not treat all newly visible differences as cosmetic.

Resource addresses, UUID/import IDs, JSON string attribute types, and the state
schema version are unchanged. Reimport and state migration are not required.
Names, descriptions, enabled state, and rule ordering retain their existing
management semantics. `rule_id` is preserved for a complete no-op, but remains
unknown for a real update because the separate ordering API can renumber rules.

OWASP settings, image transforms, ordinary rule actions, and computed WAF exclusion
JSON keep their existing comparison behavior. This change does not make partial
OWASP updates safe or change the API's treatment of unknown fields.

## Schema maintenance and limits

Defaults are generated from `docs/spec/peakhour-api-v1.json`, embedded in the
provider, and identified by the source SHA-256 in
`internal/jsondefaults/schemas.json`. There is no runtime schema download.
Maintainers regenerate with `go generate ./internal/jsondefaults` or `make generate`.
Tests reject a stale or manually changed generated bundle.

Comparison fills only missing properties with an explicitly declared default.
Explicit nulls, unknown properties, array lengths, and array order remain
significant. It does not create absent parent objects unless the parent itself has
a default. Required fields are not supplied from defaults.

The normalizer supports objects, arrays, local component references, and unions
whose branches are distinguishable by JSON type. Conditional schemas, ambiguous
unions, unsupported schema keywords, and reference/recursion failures retain the
diff rather than guessing equivalence. It is not a replacement for API validation.

API/schema version skew may expose additional differences or repeated plans. Verify
the deployed API defaults and update the bundled schema before relaxing comparison.
Use `make test-waf-defaults-contract` to check the sibling API source; that check
does not prove deployment or live endpoint behavior.
