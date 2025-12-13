# Peakhour Provider Spec Alignment (v1) — Review & Continuation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use `superpowers:executing-plans` to implement/verify this plan task-by-task.

**Goal:** Bring the Terraform provider into alignment with `docs/spec/peakhour-api-v1.json` (nullable semantics, phase-aware rule endpoints, reverse proxy config correctness, and image transform workflow support), and leave a clear audit trail for review.

**Architecture:**
- `internal/client` mirrors the OpenAPI schemas and endpoints. Nullable schema fields are represented as pointers (e.g. `*bool`, `*int`, `*string`).
- `internal/provider` resources map API `nil` → Terraform `Null` and provide “reset on delete” semantics for config-like resources.

**Tech Stack:** Go 1.22+, Terraform Plugin Framework `v1.12.0`, Go unit tests.

---

## Current State (What’s Done)

### Commits (chronological)
- `e7f89d2` — `chore: align toolchain and add Peakhour API spec reference`
- `d557ae7` — `feat: update client models and endpoints to match v1 spec`
- `5e28679` — `feat: reverse proxy config nils and reset-on-delete`
- `8d88051` — `feat(client): align DTOs with spec and consolidate transform logic`
- `6d2e755` — `feat: phase-aware rule crud and json normalization`
- `bfc90d4` — `fix(provider): implement full reset logic and fix pointer usage`
- `3b6cadd` — `fix: strict import validation and consistent drift handling`
- `47473c4` — `chore: fmt/vet/test after spec alignment`

### Local-only artifacts (NOT committed)
- Untracked: `.toolchains/` (a locally downloaded Go toolchain used for QA)
- Untracked: `foo_test` (empty file; safe to delete)

---

## Verification (known-good commands)

These were run successfully against `HEAD`:

```bash
# If your system Go is already 1.22+, you can use `go` directly.
# In this environment, Go 1.22 was run from .toolchains/go1.22.10/bin.

go vet ./...
GOFLAGS=-count=1 go test ./... -short
```

**Note:** `terraform` was not installed in the environment where this work was executed, so `examples/*` validation was not run.

---

## Task 0: Reproduce environment and confirm baseline

**Files:**
- Verify: `go.mod`

**Step 1: Confirm Go version**
Run: `go version`
Expected: `go version go1.22.x ...`

**Step 2: Confirm dependency versions**
Run: `sed -n '1,40p' go.mod`
Expected:
- `go 1.22.0` (or `go 1.22`)
- `github.com/hashicorp/terraform-plugin-framework v1.12.0`

**Step 3: Run baseline QA**
Run:
```bash
go vet ./...
GOFLAGS=-count=1 go test ./... -short
```
Expected: PASS

---

## Task 1: Spec reference is present

**Files:**
- Verify: `docs/spec/peakhour-api-v1.json`

**Step 1: Confirm spec file exists**
Run: `ls -l docs/spec/peakhour-api-v1.json`
Expected: file exists

---

## Task 2: Client DTOs and phase-aware rule endpoints are correct

**Files:**
- Verify: `internal/client/types.go`
- Verify: `internal/client/rules.go`
- Verify: `internal/client/transform.go`
- Verify tests: `internal/client/client_test.go`

**Step 1: Spot-check nullable DTO fields are pointers**
Examples to check:
- Reverse proxy config fields use pointers: `*bool`, `*int`, `*string`.

**Step 2: Spot-check rule paths include phase**
Expected path shape:
- `/api/v1/domains/{domain}/services/rp/rules/phases/{phase}/rule/{uuid}`

**Step 3: Run focused client tests**
Run: `go test ./internal/client -v`
Expected: PASS

---

## Task 3: Reverse proxy config resource matches nullable semantics and resets on Delete

**Files:**
- Verify: `internal/provider/reverse_proxy_config_resource.go`
- Verify tests: `internal/provider/reverse_proxy_config_resource_test.go`

**Step 1: Verify Read maps API nil → Terraform Null**
Look for patterns:
- `if config.Websocket != nil { ... } else { state.Websocket = types.BoolNull() }`

**Step 2: Verify Delete does “reset” via Update/PATCH**
Look for a method like `deleteConfig` that sends an “empty”/zeroed config.

**Step 3: Run provider tests**
Run: `go test ./internal/provider -v`
Expected: PASS

---

## Task 4: Transform settings and image transform resources exist

**Files:**
- Verify: `internal/provider/transform_settings_resource.go`
- Verify: `internal/provider/image_transform_resource.go`
- Verify tests: `internal/provider/transform_settings_resource_test.go`

**Step 1: Verify TransformSettings Delete resets**
Expected: Delete calls an API update with `client.TransformSettings{}` (empty struct).

**Step 2: Verify new image transform resource exists**
Expected resource file exists and implements CRUD.

**Step 3: Run provider tests**
Run: `go test ./internal/provider -v`
Expected: PASS

---

## Task 5: Rule resource is phase-aware, import format is domain/phase/uuid, JSON canonicalization avoids drift

**Files:**
- Verify: `internal/provider/rule_resource.go`
- Verify helpers: `internal/provider/utils.go`
- Verify tests: `internal/provider/rule_resource_test.go`

**Step 1: Confirm import parsing**
Expected:
- Import ID format: `domain/phase/uuid`
- Uses `parseCompositeID(req.ID, 3)`

**Step 2: Confirm actions JSON normalization**
Expected:
- Helper `normalizeJSON` exists.
- Create/Update store canonical JSON string.

---

## Task 6: Import validation and drift handling are consistent across resources

**Files:**
- Verify: `internal/provider/import_validation_test.go`
- Verify drift handling updates:
  - `internal/provider/domain_resource.go`
  - `internal/provider/reverse_proxy_service_resource.go`
  - `internal/provider/rule_list_resource.go`
  - `internal/provider/rate_limit_zone_resource.go`
  - `internal/provider/origin_pool_resource.go`

**Step 1: Confirm drift handling uses typed 404**
Expected pattern:
```go
if client.IsNotFoundError(err) {
  resp.State.RemoveResource(ctx)
  return
}
```

**Step 2: Run import validation tests**
Run: `go test ./internal/provider -run ImportID -v`
Expected: PASS

---

## Task 7: Go toolchain decision (Go 1.22)

**Files:**
- Review: `go.mod`

**Decision:** Use Go 1.22+ in CI/dev environments (no `toolchain` pin in `go.mod`).

**Step 1: Verify CI/dev Go 1.22**
Run:
- `go version` (expect `go1.22.x`)
- `go mod tidy`
- `go test ./... -short`

---

## Task 8: Examples validation (requires terraform)

**Files:**
- Verify: `examples/*/main.tf`

**Prereq:** install Terraform.

**Note:** The provider is not published on the public Terraform Registry. For local validation, use a provider development override (see `README.md` “Testing Locally”) and skip `terraform init` (it will attempt to query the registry and fail).

**Step 1: Validate each example**
Run:
```bash
# Build the local provider binary (repo root)
go build -o terraform-provider-peakhour

# Point Terraform at the local provider directory
cat > /tmp/terraformrc <<EOF
provider_installation {
  dev_overrides {
    "peakhour-io/peakhour" = "$(pwd)"
  }
  direct {}
}
EOF
export TF_CLI_CONFIG_FILE=/tmp/terraformrc

# Validate examples (no init needed with dev overrides)
for dir in examples/*; do
  [ -d "$dir" ] || continue
  echo "Validating $dir..."
  terraform -chdir="$dir" validate
done
```
Expected: PASS

---

## Task 9: Workspace hygiene (local)

**Step 1: Delete local-only artifacts**
- `.toolchains/` (only if not needed)
- `foo_test`

Run:
```bash
rm -rf .toolchains foo_test
```
Expected: `git status --porcelain` is clean

---

## Task 10: Acceptance Tests (E2E) + Jenkins wiring

**Files:**
- Verify tests:
  - `internal/provider/provider_acceptance_test.go`
  - `internal/provider/rule_list_acceptance_test.go`
  - `internal/provider/image_transform_acceptance_test.go`
- Verify helper: `internal/provider/acceptance_test.go`
- Verify make target: `Makefile` (`testacc`)
- Verify CI: `Jenkinsfile`
- Verify docs: `README.md`

**Prereqs:**
- Terraform CLI available in `PATH`
- `TF_ACC=1`
- `PEAKHOUR_API_KEY` set
- `PEAKHOUR_TEST_DOMAIN` set (for resource tests)

**Step 1: Run smoke acceptance test**
Run:
```bash
TF_ACC=1 PEAKHOUR_API_KEY="..." go test ./internal/provider -run '^TestAccProvider_Smoke$' -v
```
Expected: PASS

**Step 2: Run full acceptance suite**
Run:
```bash
TF_ACC=1 PEAKHOUR_API_KEY="..." PEAKHOUR_TEST_DOMAIN="test.example.com" make testacc
```
Expected: PASS

**Step 3: Jenkins**
Acceptance tests run when `RUN_ACCEPTANCE=true`, using the `peakhour-api-key` credential and the `PEAKHOUR_TEST_DOMAIN` parameter.

---

## Handoff Notes / Learnings

- The original plan file was lost from the repo; this doc is the reconstructed source-of-truth.
- Drift handling should use `client.IsNotFoundError(err)` (typed 404 via `internal/client/APIError`) rather than string matching.
- Terraform example validation can be run locally via `terraform validate` + provider dev overrides (see Task 8).
- Do not commit `.toolchains/` — it’s a local convenience only.
