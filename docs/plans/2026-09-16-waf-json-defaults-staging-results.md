# WAF JSON defaults: staging integration results

Date: 2026-09-16

## Outcome

The schema-default comparison passed against the deployed staging API at
`https://www-staging.peakhour.xyz`, using the existing `example.com` test domain.
The completed run passed 110 assertions from 02:13:38 to 02:14:11 UTC.

There is a separate deployment compatibility finding: staging did not return a
configured custom-rule `name`. The initial named create failed Terraform's
consistency check because the planned name became null. After removing that
disposable rule, defaults validation used `name` omitted and a unique description.
This result validates unnamed rules and description updates; it does not certify
named rules against this staging deployment.

The provider was built from the current local working tree, not a published
release. Binary SHA-256:
`6f108c52d8a0c5ec9fad86c90b369fb7b2bbaa84ed1c25b6b545c22ec710e578`.
Terraform version: 1.14.2. Embedded schema source SHA-256:
`a551e755233ef0e84360340b9237547423e725c2184128429f6cec17c56e4be4`.

## Isolation and evidence

- Built the provider into a private temporary directory and selected it with a
  Terraform development override. No provider installation used by other work
  was replaced.
- Used new Terraform configurations and local state files. Existing
  `examples/waf` state and configuration were not modified or applied.
- Snapshotted the existing custom-rule collection, which contained one rule.
- Created a disposable rule with a unique test-header/path condition, `pass`
  action, and `enabled = false`. The provider used the enable-toggle endpoint
  during create; every subsequent rule assertion confirmed it was disabled.
- A loopback HTTP recorder forwarded requests to the real staging API over TLS.
  It did not generate mock responses. It restricted requests to the custom-rule
  collection and the newly created UUID, and counted reads and writes separately.
  Credentials were kept out of Terraform configuration and recorded output.
- Transferred management from the create state to a fresh import state, removing
  the original local state reference after successful import.

## Verified behavior

| Scenario | Result |
| --- | --- |
| Create from compact JSON | Succeeded; direct API reads confirmed expanded action, expression, selector, and logging defaults. |
| First normal plan after create | Empty, exit code 0, no API writes. |
| CLI import into fresh state with compact HCL | Succeeded without writes; the next normal plan was empty with exit code 0. |
| Refresh-only plan/apply, then repeated normal plan | No resource drift or remote writes; subsequent normal plan was empty. |
| Configure `operator_negated = true` | Update planned; exactly one PATCH persisted the value; subsequent plan empty. |
| Remove that non-default property from HCL | Update planned; API value reset to false; subsequent plan empty. |
| Change description | Metadata persisted; JSON defaults remained stable; subsequent plan empty. |
| Independently PATCH negation to true | Terraform detected the remote drift, reconciled it with one PATCH, and returned to an empty plan. |
| Explicit `action_arg_val_param_val = null` | API preserved null and the subsequent plan was empty. |
| Remove that explicit null | Update restored the documented empty-string default; subsequent plan empty. |
| Destroy | Only the disposable UUID was deleted; final managed state contained zero resources. |

Every plan was checked for its expected resource action and restricted to the
disposable resource address. Every plan and the refresh-only apply issued zero
mutating API calls.

Completed-run request accounting:

| Caller | Requests |
| --- | --- |
| Terraform | 32 GET, 1 PUT, 1 enable-toggle PATCH, 6 rule PATCH, 1 DELETE |
| Independent assertions/drift injection | 13 GET, 1 rule PATCH |

All completed-run HTTP responses were successful: 46 responses with status 200
and 9 with status 204.

## Findings and cleanup

1. **Deployed metadata contract differs from local source.** The configured name
   became null on the first create. The UUID was captured, the rule was confirmed
   disabled, and it was deleted explicitly. Do not treat the locally verified
   metadata source contract as deployed. Deploy/verify that support before
   accepting named-rule behavior on staging.
2. **Refresh-only plan handling required a harness correction.** Terraform's
   JSON omitted `resource_changes` for a no-change refresh-only plan and reported
   no resource drift; the CLI printed “No changes” but returned detailed exit
   code 2. The harness initially expected a normal-plan no-op entry and exit 0.
   It now checks the JSON actions/drift and absence of API writes for refresh-only
   operations. Normal empty plans still require exit 0. No provider code change
   was needed for this.
3. **All temporary rules were removed.** Four attempts created temporary rules:
   one exposed the metadata mismatch, two stopped while correcting the harness,
   and the fourth completed the full test. Cleanup was verified for every UUID.
   The final collection exactly matched each attempt's baseline: the original
   rule was unchanged. No domain, service, WAF-option, OWASP, or ordering endpoint
   was mutated.

Private run artifacts (plans, isolated state, API snapshots, request audit, and
assertion results) were retained under `/tmp/peakhour-waf-staging-y38xsbh2` for the
successful run. Earlier diagnostic artifacts are in
`/tmp/peakhour-waf-staging-w50qo23c`, `/tmp/peakhour-waf-staging-c2w1ot_k`, and
`/tmp/peakhour-waf-staging-8pvkbb47`. These paths are local, temporary evidence,
not checked-in fixtures or reusable managed Terraform environments.

## Limits and rollout implications

The JSON-default feature now has live staging control-plane evidence in addition
to unit, mocked Terraform lifecycle, and source-contract coverage. This does not
prove edge deployment or request enforcement; no traffic-enforcement test was
performed. Concurrent renumbering remains covered by the local Terraform test;
the staging ordering endpoint was intentionally not changed.

The stricter comparison can reveal previously hidden non-default values and plan
resets. Review upgrade plans and retain intended non-default values in HCL.
The staging run preceded publication and did not upgrade or apply the customer
configuration.
