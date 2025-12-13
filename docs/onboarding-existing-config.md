# Onboarding an Existing Peakhour Account into Terraform

This provider supports importing an existing Peakhour configuration into Terraform state and scaffolding initial HCL using Terraform’s config generation (`-generate-config-out`).

## Prerequisites

- Terraform CLI **>= 1.5**
- Go **>= 1.22** (to build the helper CLI)
- A Peakhour API key in `PEAKHOUR_API_KEY`
- (Optional) non-prod API base URL in `PEAKHOUR_BASE_URL`

If the provider is not published to the Terraform Registry yet, install the provider binary locally before running `terraform init` in your config repo:

```bash
make build
make install
```

Alternatively, use a dev override (see `QUICKSTART.md`).

## Workflow (recommended per-domain state)

1. Build the onboarding CLI:

```bash
make onboard
```

2. Generate per-domain `imports.tf` files:

```bash
PEAKHOUR_API_KEY="..." ./peakhour-tf-onboard --all-domains --out infra
```

This creates:
- `infra/domains/<domain>/provider.tf`
- `infra/domains/<domain>/imports.tf`

3. For each domain, generate config and import into state:

```bash
cd infra/domains/example.com
terraform init
terraform plan -generate-config-out=generated.tf
terraform apply -auto-approve
```

4. Refactor:

- Move resources from `generated.tf` into your preferred layout (modules, files, naming).
- Run `terraform plan` until clean.

## Handling Out-of-Band Changes (Jenkins drift checks)

Once Terraform is considered the source of truth, treat UI/API edits as drift that must be reverted or captured in HCL.

Typical Jenkins drift job:
- Iterate `infra/domains/*`
- Run `terraform init` (backend enabled)
- Run `terraform plan -refresh-only -detailed-exitcode`
  - exit code `0`: no drift
  - exit code `2`: drift detected (fail / notify)

