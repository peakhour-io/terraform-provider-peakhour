#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
terraform_bin="${TERRAFORM:-terraform}"
go_bin="${GO:-go}"
validation_dir="$(mktemp -d)"

cleanup() {
  find "$validation_dir" -depth -delete
}
trap cleanup EXIT

"$go_bin" build -o "$validation_dir/terraform-provider-peakhour" "$repo_root"

{
  printf 'provider_installation {\n'
  printf '  dev_overrides {\n'
  printf '    "peakhour-io/peakhour" = "%s"\n' "$validation_dir"
  printf '    "hashicorp/peakhour" = "%s"\n' "$validation_dir"
  printf '  }\n'
  printf '  direct {}\n'
  printf '}\n'
} >"$validation_dir/terraform.tfrc"

example_dirs=(
  "$repo_root/examples/provider"
  "$repo_root"/examples/resources/peakhour_*
  "$repo_root"/examples/data-sources/peakhour_*
)

for example_dir in "${example_dirs[@]}"; do
  TF_CLI_CONFIG_FILE="$validation_dir/terraform.tfrc" \
    TF_DATA_DIR="$validation_dir/terraform-data" \
    "$terraform_bin" -chdir="$example_dir" validate -no-color >/dev/null
done

echo "validated ${#example_dirs[@]} Registry examples against the provider schema"
