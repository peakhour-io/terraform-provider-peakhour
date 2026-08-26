# Distributing the Peakhour Terraform Provider

This repo’s provider address is:

- `peakhour-io/peakhour` (Terraform `source`)

## Phase 1: Private binary distribution (recommended first)

### Build a versioned provider bundle

From this repo:

```bash
make dist-mirror VERSION=0.1.2
```

Outputs:

- `dist/peakhour-provider_0.1.2.tar.xz` (filesystem mirror + checksums)
- `dist/SHA256SUMS`

### Client install

1. Extract the tarball on the client host:

```bash
sudo mkdir -p /opt/peakhour/terraform-providers
sudo tar -xJf peakhour-provider_0.1.2.tar.xz -C /opt/peakhour/terraform-providers
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
      version = "0.1.2"
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

Public Registry releases do not use the private mirror bundle. They are built
and signed by [`.goreleaser.yml`](../.goreleaser.yml) from an approved semantic
version tag. Complete the one-time account, signing-key, public-repository, and
Registry enrollment steps in the
[publication checklist](publication-checklist.md) before creating the first tag.

After the provider is published as `peakhour-io/peakhour`, clients remove the
filesystem mirror override and run `terraform init` to install it from the
Registry.
