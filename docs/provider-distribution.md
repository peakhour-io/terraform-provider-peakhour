# Distributing the Peakhour Terraform Provider (Private Binary → Public Registry)

This repo’s provider address is:

- `peakhour-io/peakhour` (Terraform `source`)

## Phase 1: Private binary distribution (recommended first)

### Build a versioned provider bundle

From this repo:

```bash
make dist-mirror VERSION=0.1.0
```

Outputs:

- `dist/peakhour-provider_0.1.0.tar.gz` (filesystem mirror + checksums)
- `dist/SHA256SUMS`

### Client install

1. Extract the tarball on the client host:

```bash
sudo mkdir -p /opt/peakhour/terraform-providers
sudo tar -xzf peakhour-provider_0.1.0.tar.gz -C /opt/peakhour/terraform-providers
```

2. Create `~/.terraformrc` (or `/etc/terraformrc`) pointing Terraform at the mirror:

```hcl
provider_installation {
  filesystem_mirror {
    path    = "/opt/peakhour/terraform-providers"
    include = ["peakhour-io/peakhour"]
  }
  direct {}
}
```

3. In the client Terraform config, pin the provider version:

```hcl
terraform {
  required_providers {
    peakhour = {
      source  = "peakhour-io/peakhour"
      version = "0.1.0"
    }
  }
}

provider "peakhour" {}
```

4. Set credentials/environment:

```bash
export PEAKHOUR_API_KEY="..."
# Optional (staging, etc.)
export PEAKHOUR_BASE_URL="https://console.staging.peakhour.io"
```

### Identifying Terraform traffic

Every API request includes:

- `User-Agent: terraform-provider-peakhour/<version>`
- `X-Peakhour-Client: terraform-provider-peakhour/<version>`

## Phase 2: Publish to the Terraform Registry

When you’re ready:

1. Choose a semver tag (`vX.Y.Z`) and ensure builds embed that version (this repo uses `-ldflags "-X main.version=..."`).
2. Produce release artifacts for each platform (the same set as `make dist-mirror`).
3. Publish under `peakhour-io/peakhour`.
4. Update client configs to remove the mirror override and let `terraform init` install from the registry.

