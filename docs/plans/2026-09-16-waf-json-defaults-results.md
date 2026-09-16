# WAF JSON defaults: implementation and validation

Date: 2026-09-16

## Result and scope

Implemented schema-driven JSON default comparison for
`peakhour_rp_waf_custom_rule.rules_json`, `action_json`, and `logging_json`.
An imported API-expanded rule now matches compact configuration without a PATCH
when only declared defaults differ. Non-default remote values remain visible.

This report records pre-release validation. At validation time the change had not
been committed or published. It was not installed for the reported customer run
or verified against that customer's live API.
Subsequent [staging integration validation](2026-09-16-waf-json-defaults-staging-results.md)
passed for unnamed rules; staging did not round-trip a configured name. All disposable
test rules were removed and the existing rule was unchanged.

## Implementation

| Part | Change |
| --- | --- |
| `cmd/generate-json-defaults` | Deterministically copies the WAF schema reference closure from the checked-in OpenAPI spec and records its SHA-256. |
| `internal/jsondefaults` | Embeds the generated bundle and compares JSON after filling only missing declared defaults. Explicit nulls, unknown keys, and array order remain significant. |
| `internal/provider/json_schema_type.go` | Adds the opt-in custom string type, semantic equality, and plan modifier. Equivalent plans retain exact prior state. |
| `internal/provider/rp_waf_custom_rule_resource.go` | Binds the three JSON attributes to their schemas and uses the new values in read/write lifecycle handling. Restores `rule_id` only for a complete no-op. |
| `scripts/verify-stable-json-contract.py --defaults` | Compares all three models' field/default declarations and full JSON persistence calls with a selected stable API revision. |
| `Makefile` and `CONTRIBUTING.md` | Add schema regeneration to `make generate` and document the separate WAF source-contract gate. |
| Documentation | Regenerated resource descriptions and added an upgrade/maintenance guide. |

The engine supports local references, objects, arrays, and unions distinguishable
by JSON type. Unsupported constructs retain the diff. It does not infer defaults
from live responses, fetch schemas at runtime, rewrite HCL, or replace API input
validation. Existing shared comparators were not changed globally.

## TDD evidence

Before integration, real Terraform CLI tests reproduced both defects:

- Import followed by compact configuration: `expected empty plan ... planned action(s): [update]`.
- API negation changed to true while omitted in configuration: `expected Update, got action(s): [no-op]`.

The comparator tests initially failed to compile because its implementation did
not exist. After implementation they passed, including symmetrical equivalence,
explicit-null distinctions, unknown keys, array order/length, exact large-number
comparison, invalid JSON, nested defaults, and schema/reference failure cases.

Review added two failing cases for required fields with defaults and semantic
siblings beside `anyOf`. Both now conservatively retain the diff.

Review also rejected blanket `UseStateForUnknown` on `rule_id`. The ordering API
can renumber rules. A regression initially failed with `Expected unknown value at
"rule_id", but found known value: "9"`. The final implementation preserves the
ID only for no-op plans, and accepts a renumbering observed after a real update.

## Terraform-level coverage

Six new tests execute 43 explicit test steps through Terraform 1.14.2 against an
isolated HTTP server. They run automatically under ordinary `go test`, with no
live credentials. The test server uses independently specified API defaults;
it does not call the production normalizer.

| Test | Evidence |
| --- | --- |
| `DefaultsImport` | CLI import persists expanded state; compact configuration and repeated refresh produce empty plans and zero writes; metadata update sends exactly one PATCH. |
| `DefaultsDrift` | Negation, transforms, action arguments, explicit action null, logging arrays, unknown properties, and expression reordering each cause an update and converge. |
| `DefaultsConfigurationChanges` | Expanded/compact configurations compare equally in both directions; removing non-default configuration resets it; null/empty action distinctions persist; destroy works. |
| `DefaultsUnknownConfiguration` | JSON derived from `terraform_data` remains unknown during both create and update planning, then converges after apply. |
| `DefaultsConcurrentReorder` | A real metadata update leaves the numeric ID unknown and accepts a simulated concurrent renumbering from 9 to 7. |
| `DefaultsImportBlock` | Declarative import with compact HCL makes no create/update API calls and leaves an empty subsequent plan. |

Existing metadata lifecycle, null/missing-property, and enabled-toggle tests also
pass. One older mock expression used `variable_part: null`; it was corrected to
the API's actual string default, `""`.

## Validation results

- `go test -json -count=1 ./...`: 92 top-level tests passed; 22 opt-in tests skipped.
  All six new Terraform tests ran; skips include the credential-dependent live
  suite and an older opt-in mocked import test.
- `go vet ./...`: passed.
- `go test -race -count=1 ./internal/jsondefaults ./internal/provider`: passed.
- `make generate`: passed; generated WAF documentation and schema bundle updated.
- `go run ./cmd/generate-json-defaults -check`: passed; stale/manual bundle changes
  are also checked by the Go suite. Generator tests verify transitive references
  and detection of source-default changes.
- `scripts/validate-registry-examples.sh`: all 39 examples passed against a freshly
  built provider.
- Contract verifier self-tests: passed, including changed/missing defaults, added
  fields, and boolean-versus-number mutations.
- WAF defaults/persistence source check: passed at stable API revision
  `f2ba41d0480bc0b209a0b53446a4502778d967dd`.
- `git diff --check`: passed.

The source check is separate from the older broader pinned null-contract gate;
this work does not update that older gate's revision or certify other API surfaces.
The sibling Python environment is unavailable, so the backing API's own integration
suite was not run. Source verification and mocked HTTP tests alone do not prove a
particular deployment, database behavior, generated edge configuration, or live
request enforcement. The subsequent staging run adds real API round-trip evidence;
it does not establish edge deployment or request enforcement.

## Impact and rollout review

1. Pure default-expansion noise disappears, including immediately after import.
   JSON state remains a string and may retain expanded defaults. No state schema
   version change, migration, resource replacement, or reimport is required.
2. This is stricter ownership than the old subset comparator. Existing omitted
   fields with non-default remote values may now plan resets. Review the plan and
   put intended non-default values into HCL before applying.
3. Genuine updates still send full rule JSON to the existing PATCH endpoint and
   trigger API configuration regeneration. Resetting negation or transforms can
   change WAF behavior; such differences are not cosmetic.
4. Explicit null is not generically equivalent to empty strings or arrays.
   Unknown properties are retained for comparison. API/schema version skew may
   therefore expose persistent differences that require a schema/contract review.
5. OWASP settings, image transforms, ordinary rule actions, WAF options, and the
   ordering resource have no comparator changes. The numeric ID stays computed;
   real updates can still show `known after apply`.
6. The six Terraform tests verify existing expanded state and the unchanged string
   wire representation; they are not a deployment test of every historical
   provider binary.

Release the provider through the normal review process, upgrade a representative
configuration, and inspect its plan before applying. Staging JSON-default validation
is complete with the documented name-support limitation. Verify deployed metadata
support before testing named rules; do not claim this local result has fixed the
customer's installed provider.
