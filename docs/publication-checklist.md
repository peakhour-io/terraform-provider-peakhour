# Public provider publication checklist

The repository-local preparation is automated and tested. The actions in this
document intentionally require a maintainer because they create public state,
grant credentials, or publish a release.

## Public repository controls

- Approve the files and history intended for public disclosure. In particular,
  review `docs/spec/peakhour-api-v1.json`, historical planning documents, and
  the beta tester guide for proprietary or customer information.
- Run a history-aware secret scan over every ref that will be pushed. Do not
  publish a ref until every finding is reviewed.
- Confirm the lowercase public GitHub repository remains
  `peakhour-io/terraform-provider-peakhour` and configure branch protection.
- Push only the approved refs. Do not create a branch with the same name as a
  release tag.

## Signing and automation bootstrap

- Create a dedicated RSA or DSA OpenPGP release key. The public Terraform
  Registry does not accept ECC keys. Protect and back up the private key.
- Register the public key with the Terraform Registry publisher account.
- Keep the private key on the maintainer release machine. Before releasing, set
  `GPG_FINGERPRINT` to its fingerprint and cache its passphrase with GPG.
- Create a fine-grained GitHub token limited to this repository with Contents
  read and write access, or a classic token with the `public_repo` scope, and
  expose it to the release process as `GITHUB_TOKEN`. Do not store either secret
  in the repository or shell history.
- Review the pinned CI actions and enable Dependabot and required status checks.

## First release and Registry enrollment

- Confirm the intended version is a `v`-prefixed semantic version and that no
  branch has that name.
- Confirm the public `Tests` workflow is green at the release commit.
- Git tag pushes do not publish this provider. Releases are created only by the
  local release script; CI remains validation-only.
- Create the approved tag at that commit and push it to the `github` remote.
- From a clean checkout of the tag, install GoReleaser v2.18.0, export
  `GITHUB_TOKEN` and `GPG_FINGERPRINT`, and run
  `scripts/release-local.sh <tag>`. The script reruns the quality gates and must
  produce a draft GitHub Release with platform ZIP archives, the renamed
  protocol manifest, `SHA256SUMS`, and a detached binary GPG signature. The
  script verifies those local production assets before finalizing the draft.
- Verify the release checksums and signature independently before enrollment.
- Sign in to the public Terraform Registry with the owning GitHub account and
  publish `peakhour-io/peakhour`. The Registry-created GitHub webhook will ingest
  later finalized releases automatically.

## Release verification

For every release, verify that CI passed, generated docs are current, all ZIPs
contain the correctly named versioned provider binary, Linux AMD64 is built with CGO
disabled, the manifest declares protocol `6.0`, and the checksum signature
validates against the Registry-registered public key.
