# WAF Custom Rule Contract Implementation Plan

> **Compatibility update:** The provider no longer requires this stable REST
> extension. Custom-rule names are not exposed by the Terraform resource,
> enabled state is reconciled through the existing toggle endpoint, and valid
> complete-order requests work with the pre-existing stable reorder endpoint.
> The stable changes below remain optional API and UI enhancements.

> **For agentic workers:** REQUIRED: Use `subagent-driven-development` (if subagents available) or `executing-plans` to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make stable REST, its WAF UI, and Terraform converge on named custom rules, explicit enabled state, deterministic ordering, and legacy rules whose name is absent.

**Acceptance Criteria:**
- Stable REST create/list/update persists `name`, defaults omitted `enabled` to true, and explicitly applies true or false.
- Existing rows with `name = NULL` remain valid and the UI renders a useful fallback label.
- Stable custom-rule reorder rejects missing, extra, or duplicate UUIDs with HTTP 400 and logs one config update on success.
- Terraform rereads authoritative custom-rule state after create/update and can manage complete custom-rule order with drift detection.
- Existing UUIDs, custom-rule import IDs, stable config logging, and asynchronous deployment behavior remain unchanged.

**Non-Goals:**
- Replace stable's config-log/listener deployment pipeline.
- Remove the legacy toggle endpoint.
- Refactor unrelated WAF or provider resources.
- Add a live cross-repository dependency to tests or runtime code.

**Architecture:** `targeted-refactor`; plan against current structure. Stable continues to own persistence, request validation, WAF mutation, and config logging. Terraform continues to own declarative reconciliation, with ordering isolated in a resource patterned after `RulePhaseOrderResource`; the UI presents the API field and provides a deterministic fallback for legacy NULL values.

**Tech Stack:** Python, SQLAlchemy, Alembic, Pyramid/Pydantic, React/TypeScript, Go, Terraform Plugin Framework.

---

### Task 1: Stable REST contract and persistence

**Task Goal:** Persist WAF rule names and apply enabled state idempotently while preserving one config-log entry per mutation.

**Acceptance Criteria:**
- Create and update round-trip nullable names.
- Omitted enabled defaults to true on create and preserves current state on update.
- Explicit false and true are persisted.

**Definition of Done:** Focused helper/API tests pass and an Alembic migration adds the nullable column.

**Out of Scope:** Changing the toggle endpoint or deployment pipeline.

**Review Gates:** spec=exempt, quality=required

**Files:**
- Modify: `../peakhour-website-stable/peakhour/reverseproxy/models.py`
- Modify: `../peakhour-website-stable/peakhour/reverseproxy/helpers.py`
- Modify: `../peakhour-website-stable/peakhour/reverseproxy/waf/api/views.py`
- Create: `../peakhour-website-stable/alembic/versions/*_add_waf_custom_rule_name.py`
- Test: `../peakhour-website-stable/peakhour/reverseproxy/waf/test/test_helper.py`
- Test: `../peakhour-website-stable/peakhour/reverseproxy/waf/api/test_functional.py`

- [x] Write failing round-trip and explicit-enabled tests.
- [x] Run them and verify failures identify ignored fields.
- [x] Add the model/migration/helper changes.
- [x] Run focused tests to green.

### Task 2: Stable custom-rule order validation

**Task Goal:** Give custom-rule ordering the same complete-permutation contract as phase-rule ordering.

**Acceptance Criteria:** Missing, extra, and duplicate UUIDs return HTTP 400; a valid order updates every rule position and logs once.

**Definition of Done:** Functional tests prove invalid and valid cases.

**Out of Scope:** Changing the REST API's complete-order request contract.

**Review Gates:** spec=exempt, quality=required

**Files:**
- Modify: `../peakhour-website-stable/peakhour/reverseproxy/helpers.py`
- Modify: `../peakhour-website-stable/peakhour/reverseproxy/waf/api/views.py`
- Test: `../peakhour-website-stable/peakhour/reverseproxy/waf/api/test_functional.py`

- [x] Write failing validation tests.
- [x] Verify RED.
- [x] Reuse the existing `RuleReorderError` boundary and map failures to HTTP 400.
- [x] Verify GREEN.

### Task 3: Stable UI names and legacy fallback

**Task Goal:** Allow users to edit a rule name and render unnamed legacy rules safely.

**Acceptance Criteria:** Create/edit forms submit `name`; configured panels show `name` when present and `Custom rule <rule_id>` otherwise.

**Definition of Done:** Focused component tests pass and TypeScript builds.

**Out of Scope:** General WAF UI redesign.

**Review Gates:** spec=exempt, quality=required

**Files:**
- Modify: `../peakhour-website-stable/js/components/waf/WafCustomRule.tsx`
- Modify: `../peakhour-website-stable/js/components/waf/WafCustomRuleConfigured.tsx`
- Test: existing or new focused WAF component test.

- [x] Write failing rendering/form tests.
- [x] Verify RED.
- [x] Add the field and fallback label.
- [x] Verify GREEN and build.

### Task 4: Terraform authoritative custom-rule state

**Task Goal:** Stop storing unverified plan values after create/update.

**Acceptance Criteria:** Create/update reread by UUID and store the API's name, enabled state, normalized JSON, rule ID, and timestamp.

**Definition of Done:** Provider tests fail before and pass after the change.

**Out of Scope:** Changing custom-rule import identity.

**Review Gates:** spec=exempt, quality=required

**Files:**
- Modify: `internal/provider/rp_waf_custom_rule_resource.go`
- Test: `internal/provider/rp_waf_custom_rule_resource_test.go`

- [x] Write failing create/update reread tests.
- [x] Verify RED.
- [x] Call the existing read helper after writes.
- [x] Verify GREEN.

### Task 5: Terraform custom-rule ordering

**Task Goal:** Manage deterministic custom-rule order as a separate owner.

**Acceptance Criteria:** The resource validates a complete unique UUID set, applies order, rereads drift, imports by domain, and does not reorder on Delete.

**Definition of Done:** Unit/contract tests, docs, examples, provider registration, and onboarding inventory pass.

**Out of Scope:** Partial ordering.

**Review Gates:** spec=exempt, quality=required

**Files:**
- Create: `internal/provider/rp_waf_custom_rule_order_resource.go`
- Create: `internal/provider/rp_waf_custom_rule_order_resource_test.go`
- Modify: `internal/provider/provider.go`
- Modify: `internal/spec/contract_test.go`
- Modify: `internal/onboard/inventory.go`
- Modify: `README.md`
- Modify: `examples/waf/main.tf`

- [x] Write failing resource/registration tests.
- [x] Verify RED.
- [x] Implement by adapting `RulePhaseOrderResource` without a new abstraction.
- [x] Update docs/onboarding and verify GREEN.

### Task 6: Cross-repository verification

**Task Goal:** Prove both sides satisfy the shared contract without relying on sibling paths at runtime.

**Acceptance Criteria:** Focused tests, broader relevant suites, formatting/type checks, and clean diffs pass.

**Definition of Done:** Results and any pre-existing failures are recorded in the handoff.

**Out of Scope:** Publishing or merging without explicit instruction.

**Review Gates:** spec=exempt, quality=required

- [x] Run stable Python and JS verification.
- [x] Run Terraform Go tests, vet, formatting, and example validation where available.
- [x] Inspect both diffs and worktree status.
