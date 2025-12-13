# Acceptance Tests: Origin Pools + Rule Phases — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use `superpowers:executing-plans` to implement this plan task-by-task.

**Goal:** Expand Terraform acceptance (E2E) coverage to include `peakhour_origin_pool` and `peakhour_rule` across all phases defined in `docs/spec/peakhour-api-v1.json`.

**Architecture:** Use `terraform-plugin-testing` (TF_ACC) with an existing test domain (`PEAKHOUR_TEST_DOMAIN`) and create/destroy resources inside that domain. Keep tests deterministic by generating unique names (acctest random suffixes) and by isolating per-phase rules into separate resources.

**Tech Stack:** Go 1.22+, Terraform CLI, `github.com/hashicorp/terraform-plugin-testing`.

---

## Intent & Learnings (Session 2025-12-13)

**Intent:**
- Make provider behavior and examples **spec-aligned** (using `docs/spec/peakhour-api-v1.json` as source of truth) and prevent regressions with automated checks.
- Expand “real API” confidence by adding **Terraform acceptance (E2E) tests** for origin pools and rules, covering every supported rules phase.

**Learnings / decisions:**
- **Spec ↔ code verification:** We added lightweight “contract tests” for the provider-supported subset (paths/schemas/critical shape checks) instead of full OpenAPI codegen. These run in `go test` and are suitable for Jenkins unit stage.
- **Acceptance harness namespace:** `terraform-plugin-testing` defaults to the `hashicorp/*` namespace; this provider uses `peakhour-io/*`, so acceptance tests must set `TF_ACC_PROVIDER_NAMESPACE=peakhour-io` (handled in `internal/provider/acceptance_test.go`).
- **Rate limiting (spec-aligned):**
  - `concurrent_connections` belongs to **global** rate limiting (`RateLimitGlobal`), not zones (`RateLimitZone`).
  - Zones are referenced by rules; global settings apply separately. This is why this plan includes rule tests that reference zones, and why examples/docs should use `peakhour_rate_limit_global` for global limits.
- **Raw config endpoint shape:** `/api/v1/domains/{domain}/services/rp/config` returns `RawConfig` (wrapper object with `config`), not `ReverseProxyConfig`. Keep client models aligned even if unused by resources.
- **Rule phases to cover:** `PhaseName` enum includes `request_rewrite`, `url_config`, `firewall`, `rate_limit_request`, `rate_limit_request_late`, `rate_limit_response`, `request_headers`, `response_headers`, `load_balance`, `bulk_redirect`.
- **Action payloads:** Spec uses a discriminated union keyed by `type`; some actions require additional required fields (e.g. `redirect` requires `from_list`, `firewall` requires `action`). Acceptance tests should use minimal spec-valid action objects.
- **Bulk redirects required new resources:** the `bulk_redirect` phase relies on bulk redirect list/entry endpoints, so the provider needs list/entry resources to test the phase end-to-end.

## Current Status (as of 2025-12-13)

- Acceptance tests added for `peakhour_origin_pool`, `peakhour_rule` (all phases), and bulk redirect list/entry resources.
- Docs/examples updated to cover missing phases and bulk redirects.
- Next step is to run `make testacc` in Jenkins with real credentials (`RUN_ACCEPTANCE=true`) and iterate on any API-environment specific failures.

## Prereqs (one-time)

- Ensure Terraform CLI is available in `PATH`.
- Ensure Go 1.22+ is used for `go test`.
- Required env vars for acceptance tests:
  - `TF_ACC=1`
  - `PEAKHOUR_API_KEY=...`
  - `PEAKHOUR_TEST_DOMAIN=...` (an existing domain in the account)
  - Optional: `PEAKHOUR_BASE_URL=...` (non-prod environments)

Smoke run:
```bash
TF_ACC=1 PEAKHOUR_API_KEY="..." go test ./internal/provider -run '^TestAccProvider_Smoke$' -v
```

---

## Task 1: `peakhour_origin_pool` acceptance test

**Files:**
- Create: `internal/provider/origin_pool_acceptance_test.go`

**Step 1: Write the failing test**

Add:
```go
package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccOriginPool_basic(t *testing.T) {
	testAccPreCheck(t)

	domain := testAccEnv(t, "PEAKHOUR_TEST_DOMAIN")
	tag := fmt.Sprintf("tfacc-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOriginPoolConfig(domain, tag, "192.0.2.10:8080"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("peakhour_origin_pool.test", "domain", domain),
					resource.TestCheckResourceAttr("peakhour_origin_pool.test", "tag", tag),
					resource.TestCheckResourceAttr("peakhour_origin_pool.test", "address.#", "1"),
				),
			},
			{
				ResourceName:      "peakhour_origin_pool.test",
				ImportState:       true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) { return fmt.Sprintf("%s/origins/%s", domain, tag), nil },
				ImportStateVerify: true,
			},
		},
	})
}

func testAccOriginPoolConfig(domain, tag, addr string) string {
	return fmt.Sprintf(`
terraform {
  required_providers {
    peakhour = {
      source = "peakhour-io/peakhour"
    }
  }
}

provider "peakhour" {}

resource "peakhour_origin_pool" "test" {
  domain = %q
  tag    = %q

  address = [{
    address = %q
    weight  = 100
  }]
}
`, domain, tag, addr)
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
TF_ACC=1 PEAKHOUR_API_KEY="..." PEAKHOUR_TEST_DOMAIN="..." go test ./internal/provider -run '^TestAccOriginPool_basic$' -v
```
Expected: FAIL (test not found / compile error until file is added).

**Step 3: Make it pass**

- Add missing imports (e.g. `github.com/hashicorp/terraform-plugin-testing/terraform`) and fix compile errors.

**Step 4: Run test to verify it passes**

Run same command; expected: PASS.

**Step 5: Commit**
```bash
git add internal/provider/origin_pool_acceptance_test.go
git commit -m "test(acc): add origin pool acceptance test"
```

---

## Task 2: `peakhour_rule` acceptance tests for all phases (except `bulk_redirect`)

**Files:**
- Create: `internal/provider/rule_acceptance_test.go`

**Step 1: Write failing tests**

Create one test per phase:
- `firewall` (action: `firewall`)
- `request_headers` (action: `header`)
- `response_headers` (action: `header`)
- `url_config` (action: `vconf`)
- `request_rewrite` (action: `request_rewrite`)
- `load_balance` (action: `origin_selection`) — requires an origin pool created in the same config
- `rate_limit_request` (action: `rate_limit_request`) — requires a zone created in the same config
- `rate_limit_request_late` (action: `rate_limit_request_late`) — requires a zone
- `rate_limit_response` (action: `rate_limit_response`) — requires a zone

**Step 2: Run a single phase test to verify it fails**

Example:
```bash
TF_ACC=1 PEAKHOUR_API_KEY="..." PEAKHOUR_TEST_DOMAIN="..." go test ./internal/provider -run '^TestAccRule_firewall$' -v
```

**Step 3: Make it pass**

- Ensure each rule uses a minimal spec-valid action payload (required `type` and required fields).
- Prefer `jsonencode(...)` in HCL to avoid JSON formatting issues.

**Step 4: Run the whole rule acceptance suite**
```bash
TF_ACC=1 PEAKHOUR_API_KEY="..." PEAKHOUR_TEST_DOMAIN="..." go test ./internal/provider -run '^TestAccRule_' -v -timeout 60m
```

**Step 5: Commit**
```bash
git add internal/provider/rule_acceptance_test.go
git commit -m "test(acc): cover rule phases with acceptance tests"
```

---

## Task 3: Cover the `bulk_redirect` phase (needs new resources)

The spec phase enum (`PhaseName`) includes `bulk_redirect`, but the API uses bulk redirect list endpoints:
- `/api/v1/domains/{domain}/services/rp/rules/bulk_redirects`
- `/api/v1/domains/{domain}/services/rp/rules/bulk_redirects/{bulk_redirect}/entries`

**Files:**
- Create: `internal/client/bulk_redirects.go`
- Create: `internal/provider/bulk_redirect_list_resource.go`
- Create: `internal/provider/bulk_redirect_entry_resource.go` (or model entries inline)
- Create tests:
  - `internal/client/bulk_redirects_test.go` (or extend `internal/client/client_test.go`)
  - `internal/provider/bulk_redirect_acceptance_test.go`

**Approach:**
1. Implement resources to create a bulk redirect list and at least one entry.
2. Add a `peakhour_rule` in phase `bulk_redirect` with:
   ```hcl
   actions_json = jsonencode({
     redirect = [{
       type      = "redirect"
       from_list = peakhour_bulk_redirect_list.test.name
     }]
   })
   ```
3. Add acceptance tests that create list + entry + rule, and verify import works.

**Commit split (recommended):**
- `feat: add bulk redirect list resources`
- `test(acc): add bulk redirect phase coverage`

---

## Task 4: Examples/docs coverage for missing phases

**Files:**
- Update: `examples/rules/main.tf`
- Update: `README.md`

Add examples for:
- `response_headers`
- `rate_limit_request_late`
- `rate_limit_response`
- `bulk_redirect` (once Task 3 lands)

---

## Jenkins note

No Jenkins changes required: `Jenkinsfile` already runs `make testacc` when `RUN_ACCEPTANCE=true`.
