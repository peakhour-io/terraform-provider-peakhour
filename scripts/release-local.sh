#!/usr/bin/env bash

set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 vMAJOR.MINOR.PATCH" >&2
  exit 2
fi

release_tag="$1"
prerelease_identifier='(0|[1-9][0-9]*|[0-9A-Za-z-]*[A-Za-z-][0-9A-Za-z-]*)'
semver_pattern="^v(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)(-(${prerelease_identifier})(\\.${prerelease_identifier})*)?(\\+[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?$"
if [[ ! "$release_tag" =~ $semver_pattern ]]; then
  echo "release tag must be a v-prefixed semantic version: $release_tag" >&2
  exit 2
fi

github_token="${GITHUB_TOKEN:-}"
if [[ -z "$github_token" ]]; then
  echo "set GITHUB_TOKEN to a fine-grained token with Contents write access to the public repository" >&2
  exit 1
fi
unset GITHUB_TOKEN GH_TOKEN

for command_name in git go terraform goreleaser gpg gh; do
  if ! command -v "$command_name" >/dev/null 2>&1; then
    echo "required command is unavailable: $command_name" >&2
    exit 1
  fi
done
: "${GPG_FINGERPRINT:?set GPG_FINGERPRINT to the Registry signing key fingerprint}"

if [[ -n "$(git status --porcelain)" ]]; then
  echo "the worktree must be clean before a release" >&2
  exit 1
fi

github_remote=""
while IFS= read -r candidate_remote; do
  candidate_url="$(git remote get-url "$candidate_remote")"
  case "$candidate_url" in
    git@github.com:peakhour-io/terraform-provider-peakhour|\
    git@github.com:peakhour-io/terraform-provider-peakhour.git|\
    ssh://git@github.com/peakhour-io/terraform-provider-peakhour|\
    ssh://git@github.com/peakhour-io/terraform-provider-peakhour.git|\
    https://github.com/peakhour-io/terraform-provider-peakhour|\
    https://github.com/peakhour-io/terraform-provider-peakhour.git)
      github_remote="$candidate_remote"
      break
      ;;
  esac
done < <(git remote)
if [[ -z "$github_remote" ]]; then
  echo "no remote points to the public GitHub repository peakhour-io/terraform-provider-peakhour" >&2
  exit 1
fi

if git show-ref --verify --quiet "refs/heads/$release_tag" ||
  [[ -n "$(git ls-remote "$github_remote" "refs/heads/$release_tag")" ]]; then
  echo "a branch conflicts with release tag $release_tag" >&2
  exit 1
fi

if ! git show-ref --verify --quiet "refs/tags/$release_tag"; then
  echo "create and push release tag $release_tag before running this script" >&2
  exit 1
fi

head_commit="$(git rev-parse HEAD)"
tag_commit="$(git rev-parse "refs/tags/$release_tag^{commit}")"
if [[ "$tag_commit" != "$head_commit" ]]; then
  echo "release tag $release_tag does not point to HEAD" >&2
  exit 1
fi

remote_tag_commit="$(git ls-remote "$github_remote" "refs/tags/$release_tag^{}" | awk 'NR == 1 {print $1}')"
if [[ -z "$remote_tag_commit" ]]; then
  remote_tag_commit="$(git ls-remote "$github_remote" "refs/tags/$release_tag" | awk 'NR == 1 {print $1}')"
fi
if [[ "$remote_tag_commit" != "$head_commit" ]]; then
  echo "release tag $release_tag is not present on the GitHub remote at HEAD" >&2
  exit 1
fi

secret_key_info="$(gpg --with-colons --list-secret-keys "$GPG_FINGERPRINT" 2>/dev/null)"
if [[ -z "$secret_key_info" ]]; then
  echo "the GPG secret key is unavailable: $GPG_FINGERPRINT" >&2
  exit 1
fi
primary_key_algorithm="$(awk -F: '$1 == "sec" { print $4; exit }' <<<"$secret_key_info")"
case "$primary_key_algorithm" in
  1|2|3|17) ;;
  *)
    echo "the Registry signing key must be RSA or DSA, not algorithm $primary_key_algorithm" >&2
    exit 1
    ;;
esac

signing_probe_dir="$(mktemp -d)"
cleanup_signing_probe() {
  find "$signing_probe_dir" -type f -delete
  rmdir "$signing_probe_dir"
}
trap cleanup_signing_probe EXIT
printf 'peakhour provider release signing probe\n' >"$signing_probe_dir/payload"
gpg --batch --local-user "$GPG_FINGERPRINT" \
  --output "$signing_probe_dir/payload.sig" --detach-sign "$signing_probe_dir/payload"
gpg --batch --verify "$signing_probe_dir/payload.sig" "$signing_probe_dir/payload"

if ! goreleaser --version | grep -Eq '^GitVersion:[[:space:]]+2\.18\.0$'; then
  echo "GoReleaser v2.18.0 is required" >&2
  exit 1
fi

go mod download
go test -count=1 ./...
go vet ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...
go run github.com/zricethezav/gitleaks/v8@v8.30.1 git . --redact --no-banner
make generate
git diff --exit-code
scripts/validate-registry-examples.sh
goreleaser check
goreleaser release --snapshot --clean --skip=sign
scripts/verify-release-assets.sh

GITHUB_TOKEN="$github_token" goreleaser release --clean
REQUIRE_SIGNATURE=1 scripts/verify-release-assets.sh
GH_TOKEN="$github_token" gh release edit "$release_tag" \
  --repo peakhour-io/terraform-provider-peakhour --draft=false
github_token=""
