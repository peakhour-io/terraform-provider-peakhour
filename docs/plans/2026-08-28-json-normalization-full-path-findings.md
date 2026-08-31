# JSON normalization full-path findings

Date: 2026-08-28

## Purpose

This record defines where Terraform JSON equivalence is safe after tracing the
affected values through the complete product path. It is a decision record and
follow-up plan, not evidence that the sibling fixes have been deployed.

This public record was written against provider base commit
`1ec4e783cb7511ada58fef8d31e343609520d62d` plus the uncommitted JSON semantic
type change set reviewed on 2026-08-28. It is not immutable release evidence;
replace that description with the resulting provider commit and checked-in API
contract fixture revision before release. The complete server and consumer
revision matrix is retained in the internal compatibility review.

## Decision summary

Null equivalence is not a repository-wide JSON rule. It is valid only where
the owning API and final consumer give null and omission the same meaning.

| Provider surface | Current decision | Reason |
| --- | --- | --- |
| `peakhour_rule.actions_json` | Use exact JSON equality after removing null-valued object keys | The stable API expands optional action fields with null; a non-null remote value remains drift. |
| WAF custom `rules_json` and `logging_json` | Use configured-subset equality with null-valued object keys omitted | Stable expands optional schema fields; configured non-null values must still match. |
| WAF custom `action_json` | Retain the current configured-subset comparator, with configured primitive values compared exactly | Do not add null/empty equivalence until a compatible canonical server contract is deployed. |
| OWASP `settings_json` | Keep current comparator; do not claim safe partial updates | The server PATCH implementation can reset omitted siblings and does not implement a consistent null contract. |
| Image transform `config_json` | Keep out of blanket null equivalence; leave subset-versus-full ownership open pending a server contract decision | PUT is whole-document replacement, but the canonical treatment of unknown and null-valued options is not yet a verified public contract. |
| Computed WAF exclusion JSON | Keep exact normalized computed values | These are API observations, not Terraform-owned PATCH documents. |

## Current and desired behavior matrix

| Surface/input | Current storage or GET | Final consumer effect | Current provider comparison | Desired behavior | Prerequisite and proof |
| --- | --- | --- | --- | --- | --- |
| Rule action object key omitted vs key set to null | Server may expand an omitted optional key to null | Reviewed action writer gives both the same result | Exact JSON after recursive null-object-key removal in this change set | Equivalent; a remote non-null value remains drift | Pinned API fixture and provider lifecycle test; live endpoint proof still required |
| WAF custom rule/logging optional key omitted vs null | Server expands optional members | Reviewed writer omits or defaults the same optional value | Configured subset after recursive null-object-key removal in this change set | Equivalent only for verified optional fields | Pinned source vector and exact writer test; live apply-refresh-plan required |
| WAF custom action sub-variable omitted/null/empty | Different request and stored representations are possible | Source-level writer argument collapses null/empty; exact rendered rule awaits integration proof | Configured subset with configured primitive values exact | Product/API must choose one canonical empty representation; non-empty remains significant | Server canonicalization deployed before any provider comparator relaxation |
| OWASP field omitted/null/value | Current PATCH loses presence in some cases, injects defaults, and shallow-replaces sections | Non-null flattened values become WAF tx vars; cleared/default behavior needs runtime proof | Configured subset semantics | Omission is no-op; null is a deliberate clear/reset distinct from omission; storage and GET representation remain a product decision | Corrected API behavioral matrix, deployment, VConf test, and live provider lifecycle test |
| Image option omitted/null/unknown non-null | Whole-document PUT replaces stored config; end-to-end canonical representation is not yet verified | Event-based optimiser path, not `rp_api` | Configured subset | Product must decide null and unknown-field policy, then align ownership and comparison | Server/consumer decision, cross-repository vector, deployment, and provider lifecycle test |
| Computed WAF exclusions | API returns relationship and catalog arrays | Source inspection maps membership to removed IDs and active files | Exact normalized computed JSON | Keep exact pending a deterministic API ordering contract | Deterministic response/config tests and runtime reverse-proxy verification |

## Concrete risks

### OWASP partial updates can be destructive

If the remote `protocol` section contains a custom request-body limit and
restricted HTTP versions, applying only `protocol.max_num_args` can replace the
section after Pydantic has injected defaults. The unrelated body limit and HTTP
version policy can therefore reset. Explicit null also does not consistently
mean reset: top-level and nested nulls currently take different paths.

Provider consequence:

- Do not describe omitted fields as reliably unchanged or explicit null as a
  reliable clear operation until the website/API fix is deployed.
- Do not attempt to repair this solely with a Terraform semantic comparator;
  the risk occurs during PATCH, before refresh comparison.
- Require API/UI/DB/VConf and live apply-refresh-plan tests before declaring
  partial OWASP management safe.

### Image transform ownership is inconsistent

The image API uses full replacement. A remote preset such as
`{"w":300,"q":80,"future_option":true}` may compare equal to Terraform
`{"w":300}` under configured-subset comparison. A later width update sends a
full PUT and deletes `q` and `future_option`. The UI has the same unknown-field
loss. Image events are also published immediately; the image commit resource
does not currently gate optimiser delivery.

Provider consequence:

- Decide and document whether null keys are removed and whether unknown
  non-null keys are preserved. If the product chooses whole-document ownership,
  compare the complete API-canonical document so remote extra fields are not
  silently hidden.
- Until deployment behavior is resolved, do not document image commit as the
  boundary that makes changes live.

### WAF custom actions require action-aware semantics

For an optional target sub-variable, missing, null, and empty string all become
the same empty ModSecurity argument before `rp_api`. A non-empty value such as
`password` changes the target from all `ARGS` to `ARGS:password` and is
security-significant. The API also accepts `setvar`, but the configuration
writer currently emits no action for it.

Provider consequence:

- Retain its current configured-subset comparison, with configured primitive
  values exact, until the API canonicalizes action arguments.
- After that deployment, use an action-specific comparator; never use blanket
  null/empty/omitted equality.
- A successful API write is not sufficient proof. Tests must assert the exact
  generated ModSecurity text and reverse-proxy load behavior.

### Computed WAF exclusions expose response defects

The Python API returns boolean `false`, while generated TypeScript currently
models the field as string literal `"False"`. This breaks UI display and local
toggle state. Source inspection shows the Python DB-to-`rp_api` path turning
excluded rules into removed rule IDs and excluded groups into omission from
active rule files; runtime reverse-proxy verification remains required.

Provider consequence:

- Leave both JSON attributes computed and exact.
- Retain exact computed response values pending a documented deterministic
  ordering contract. Add deterministic server ordering so set-like membership
  does not cause noisy state refreshes.

## Required cross-repository gates

1. Website/API fixes and tests must define canonical request, stored, and GET
   representations for each affected document.
2. Generated JavaScript types and React Hook Form payload tests must match the
   Python contract.
3. Persistence tests must cover JSONB or event state, including empty objects
   and unknown non-null image options.
4. Writer tests must assert exact `rp_api.Vconf` values or, for image transforms,
   exact RabbitMQ/optimiser state.
5. Reverse-proxy tests must load the resulting WAF custom rule and OWASP tx vars.
6. Provider lifecycle tests must perform apply, authoritative refresh, and a
   second plan, while also proving meaningful non-null drift is not hidden.

## Repository ownership

- `peakhour-terraform`: scoped semantic comparators, safety documentation,
  lifecycle tests, and deployment prerequisites.
- `peakhour-website-stable`: React forms, generated JS contract, Python PATCH
  semantics, JSONB/event persistence, WAF writers, and deterministic responses.
- `peakhour-optimiser`: image event validation, the decided policy for unknown
  options, empty configuration handling, and SQLite state.
- `rp-api` and `reverse-proxy`: no production change identified by this review;
  retain integration tests proving the final serialized configuration.
