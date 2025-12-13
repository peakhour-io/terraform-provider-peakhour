# Multi-Domain Terraform Onboarding (Importer + Drift Checks) — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use `superpowers:executing-plans` to implement this plan task-by-task.

**Goal:** Adopt Terraform for an existing Peakhour account with many domains/rules by generating deterministic `import {}` blocks from the Peakhour API, using Terraform’s `-generate-config-out` to scaffold HCL, and adding Jenkins drift detection.

**Architecture:**
- **One root module per domain** (recommended for scale): each domain lives in its own directory with its own state, enabling fast plans and low blast radius.
- A small Go CLI (`cmd/peakhour-tf-onboard`) uses `internal/client` to inventory existing resources for a domain and emits:
  - `provider.tf` (minimal provider wiring)
  - `imports.tf` (Terraform `import {}` blocks for all discovered resources)
- Terraform is then run per-domain to materialize state and optionally generate initial HCL via `terraform plan -generate-config-out=generated.tf`.

**Tech Stack:** Go 1.22, Terraform CLI >= 1.5, Jenkins, existing `internal/client` API client.

---

## Intent & Learnings (Session 2025-12-13)

**Intent:**
- Make Terraform adoption practical for an account with *many* domains and large rulesets by automating imports and making drift visible.

**Learnings / decisions:**
- Terraform can scaffold HCL from imported state via `terraform plan -generate-config-out=...` (Terraform >= 1.5).
- While this provider isn’t published to the public registry yet, `terraform init` works if the provider binary is installed locally (e.g. via `make build && make install`) and the configuration has a lockfile.
- The key operational risk is “out of band” changes: adopting Terraform needs a drift job (nightly `plan -refresh-only -detailed-exitcode`) and a team policy for handling UI/API edits.

## Decisions (make once, write down)

1. **State layout**
   - Recommended: `infra/domains/<domain>/` (one state per domain)
   - Alternative: group domains by environment/account (slower plans, higher blast radius)

2. **Terraform backend**
   - Recommended for teams: remote backend (S3/GCS/TFC/etc.)
   - Onboarding can start with local state, but migrate before “enforcing Terraform”

3. **Adoption policy**
   - “Terraform is source-of-truth”: UI/API changes must be reverted or backfilled immediately.
   - Add a scheduled drift job in Jenkins (`plan -refresh-only -detailed-exitcode`).

---

## Scope (what we can import/export)

This plan covers resources implemented by this provider:

- `peakhour_domain` (import id: `domain`)
- `peakhour_reverse_proxy_service` (import id: `domain`)
- `peakhour_reverse_proxy_config` (import id: `domain`)
- `peakhour_transform_settings` (import id: `domain`)
- `peakhour_rate_limit_global` (import id: `domain`)
- `peakhour_rate_limit_zone` (import id: `domain/name`)
- `peakhour_origin_pool` (import id: `domain/origins/tag`)
- `peakhour_rule_list` (import id: `domain/uuid`)
- `peakhour_rule` (import id: `domain/phase/uuid`)
- `peakhour_image_transform` (import id: `domain/uuid`)
- `peakhour_bulk_redirect_list` (import id: `domain/bulk_redirects/uuid`)
- `peakhour_bulk_redirect_entry` (import id: `domain/bulk_redirects/uuid/entries/entry_id`)

Anything not modeled by provider resources is out-of-scope for automated Terraform generation and will remain “unmanaged” unless new resources are added.

---

## Task 1: Fix provider import/read consistency (required for `-generate-config-out`)

**Why:** Terraform config generation and reliable import need resources to (a) parse import IDs into required attributes, and (b) set computed IDs in `Read`.

**Files:**
- Modify: `internal/provider/rate_limit_zone_resource.go`
- Modify: `internal/provider/reverse_proxy_service_resource.go`
- Modify: `internal/provider/reverse_proxy_config_resource.go`
- Modify: `internal/provider/transform_settings_resource.go`
- Modify: `internal/provider/origin_pool_resource.go`
- Modify: `internal/provider/rule_list_resource.go`
- Modify: `internal/provider/rule_resource.go`
- Modify: `internal/provider/image_transform_resource.go`
- (Optional) Add tests: `internal/provider/import_state_test.go`

**Step 1: Write failing tests for ImportState parsing**

Add a minimal unit test that calls `ImportState` and asserts required attrs are set (at least for `rate_limit_zone`).

Example skeleton (plan-level; adjust to actual framework helpers you choose):
```go
func TestRateLimitZone_ImportState_SetsDomainAndName(t *testing.T) {
  res := provider.NewRateLimitZoneResource()
  // Call res.ImportState(...) and assert state has domain/name populated for ID "example.com/my-zone".
}
```

Run: `go test ./internal/provider -run ImportState -v`
Expected: FAIL until implementation is fixed.

**Step 2: Fix `peakhour_rate_limit_zone` ImportState**

Change `ImportState` to parse `domain/name` and set:
- `domain`
- `name`
- `id` (to full import ID)

Run: `go test ./internal/provider -run ImportState -v`
Expected: PASS.

**Step 3: Ensure singleton-style resources set `id` in Read**

Update these Read methods to always set `state.ID` deterministically:
- `peakhour_reverse_proxy_service`: `domain + "/rp"`
- `peakhour_reverse_proxy_config`: `domain + "/config"`
- `peakhour_transform_settings`: `domain + "/transforms"`
- `peakhour_rate_limit_zone`: `domain + "/" + name"`
- `peakhour_origin_pool`: `domain + "/origins/" + tag"`
- `peakhour_rule_list`: `domain + "/" + uuid"`
- `peakhour_rule`: `domain + "/" + phase + "/" + uuid"`
- `peakhour_image_transform`: `domain + "/" + uuid"`

Run: `go test ./... -short`
Expected: PASS.

**Step 4: Commit**
```bash
git add internal/provider/*.go internal/provider/*_test.go
git commit -m "fix(provider): make imports/read IDs consistent"
```

---

## Task 2: Add “list domains” to the API client (for many-domain automation)

**Files:**
- Modify: `internal/client/domain.go`
- Test: `internal/client/client_test.go`

**Step 1: Write failing unit test**

Add a test that stands up an `httptest` server and verifies `GET /api/v1/domains` is used and decoded.

Run: `go test ./internal/client -run ListDomains -v`
Expected: FAIL (method missing).

**Step 2: Implement client method**

Add:
- `func (c *Client) ListDomains() (*Domains, error)` or `([]string, error)`
- Include both owned `Domains.Domains[].Name` and `Domains.GrantedDomains[].Name` (dedupe + sort).

Run: `go test ./internal/client -run ListDomains -v`
Expected: PASS.

**Step 3: Commit**
```bash
git add internal/client/domain.go internal/client/client_test.go
git commit -m "feat(client): add domain listing for onboarding"
```

---

## Task 3: Implement inventory collection for a domain

**Files:**
- Create: `internal/onboard/inventory.go`
- Create: `internal/onboard/inventory_test.go`

**Step 1: Define import target model**

Create:
```go
type ImportTarget struct {
  TypeName   string // e.g. "peakhour_rule"
  Name       string // terraform local name (sanitized)
  ImportID   string // provider import ID
  DependsOn  []string // optional terraform addresses for ordering/readability
}
```

**Step 2: Write unit tests with stubbed inventory**

Use a fake client interface (define a small interface in `internal/onboard` that `*client.Client` satisfies) so you can unit test without real API calls.

Run: `go test ./internal/onboard -v`
Expected: FAIL until implemented.

**Step 3: Implement `CollectDomainInventory`**

Inventory calls (best-effort; skip on 404):
- Domain always: `peakhour_domain` (`id=domain`)
- Reverse proxy service: `GetDomainService(domain,"rp")`
- Reverse proxy config: `GetReverseProxyConfig(domain)`
- Transform settings: `GetTransformSettings(domain)`
- Rate limit global: `GetRateLimit(domain)`
- Rate limit zones: `ListRateLimitZones(domain)`
- Origin pools: `GetOriginPools(domain)`
- Rule lists: `ListRuleLists(domain)`
- Rules per phase (from spec enum):
  - `request_rewrite`, `url_config`, `firewall`, `rate_limit_request`, `rate_limit_request_late`, `rate_limit_response`, `request_headers`, `response_headers`, `load_balance`, `bulk_redirect`
  - call `ListRulesInPhase(domain, phase)`
- Image transforms: `ListImageTransformPresets(domain)`
- Bulk redirect lists: `ListBulkRedirectLists(domain)`
  - for each list UUID: `ListBulkRedirectEntries(domain, uuid)`

Ensure deterministic ordering:
- sort domains, then stable sort targets by `TypeName`, then `Name`.

Run: `go test ./internal/onboard -v`
Expected: PASS.

**Step 4: Commit**
```bash
git add internal/onboard
git commit -m "feat(onboard): collect per-domain import inventory"
```

---

## Task 4: Emit deterministic `imports.tf` + `provider.tf`

**Files:**
- Create: `internal/onboard/render.go`
- Create: `internal/onboard/render_test.go`
- Create: `internal/onboard/testdata/*` (goldens)

**Step 1: Name sanitization helper**

Implement `sanitizeTFName(string) string`:
- lowercase
- replace invalid chars with `_`
- trim leading digits/underscores
- ensure non-empty (fallback: `x`)

Add collision handling:
- maintain `map[string]int` and append `_<n>` deterministically on repeats.

**Step 2: Render output**

Render `provider.tf`:
```hcl
terraform {
  required_providers {
    peakhour = {
      source = "peakhour-io/peakhour"
    }
  }
}

provider "peakhour" {}
```

Render `imports.tf`:
```hcl
import {
  to = peakhour_rule.rule_firewall_10_ab12cd34
  id = "example.com/firewall/ab12cd34-..."
}
```

**Step 3: Golden tests**

Create a small static inventory (no API) and assert the rendered HCL matches `testdata/imports.tf.golden`.

Run: `go test ./internal/onboard -run Render -v`
Expected: PASS.

**Step 4: Commit**
```bash
git add internal/onboard
git commit -m "feat(onboard): render provider and import blocks"
```

---

## Task 5: Build the CLI `peakhour-tf-onboard`

**Files:**
- Create: `cmd/peakhour-tf-onboard/main.go`
- Modify: `Makefile` (optional convenience target)
- Update docs: `README.md` (optional)

**Step 1: CLI flags**

Implement flags:
- `--out` (default `./out`)
- `--domain` (repeatable; if empty + `--all-domains`, enumerate)
- `--all-domains`
- `--concurrency` (default `4`)

Auth/base URL:
- Use `PEAKHOUR_API_KEY` and optional `PEAKHOUR_BASE_URL`.

**Step 2: Write directories**

For each domain, create:
- `<out>/domains/<domain>/provider.tf`
- `<out>/domains/<domain>/imports.tf`

**Step 3: Verification**

Run:
```bash
go build ./cmd/peakhour-tf-onboard
PEAKHOUR_API_KEY="..." ./peakhour-tf-onboard --all-domains --out /tmp/peakhour-tf
find /tmp/peakhour-tf -maxdepth 3 -type f -name '*.tf' | head
```
Expected: per-domain folders created.

**Step 4: Commit**
```bash
git add cmd/peakhour-tf-onboard Makefile README.md
git commit -m "feat: add multi-domain Terraform onboarding CLI"
```

---

## Task 6: Document the onboarding workflow (operator runbook)

**Files:**
- Create: `docs/onboarding-existing-config.md` (or add README section)

**Runbook must include:**

1. Generate imports:
```bash
PEAKHOUR_API_KEY="..." peakhour-tf-onboard --all-domains --out infra
```

2. For each domain, generate config + state:
```bash
cd infra/domains/example.com
# If the provider isn't published yet, install it locally first:
#   (in provider repo) make build && make install
terraform init
terraform plan -generate-config-out=generated.tf
terraform apply -auto-approve
```

3. Refactor `generated.tf` into your desired module layout (optional, iterative).

4. Enable drift check (below).

**Commit**
```bash
git add docs/onboarding-existing-config.md README.md
git commit -m "docs: add onboarding runbook for existing Peakhour config"
```

---

## Task 7: Jenkins drift detection job (config repo)

**Goal:** detect changes made in Peakhour UI/API that drift from Terraform.

**Files (in the Terraform config repo, not provider repo):**
- Create: `scripts/drift_check.sh`
- Create/Modify: `Jenkinsfile`

**Step 1: Drift script**

`scripts/drift_check.sh` should:
- iterate `infra/domains/*`
- run `terraform init` (backend enabled)
- run `terraform plan -refresh-only -detailed-exitcode`
- treat exit code `2` as “drift detected” (fail/alert)

**Step 2: Jenkins schedule**

Create a nightly scheduled job that:
- injects `PEAKHOUR_API_KEY`
- runs drift check

---

## Execution Handoff

Plan complete and saved to `docs/plans/2025-12-13-multi-domain-onboarding-importer.md`.

Two execution options:
1. **Subagent-Driven (this session)** — use `superpowers:subagent-driven-development`
2. **Parallel Session (separate)** — open a new session in a worktree and use `superpowers:executing-plans`

Which approach do you want?
