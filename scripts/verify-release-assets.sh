#!/usr/bin/env bash
set -euo pipefail

dist_dir="${1:-dist}"
project="terraform-provider-peakhour"
manifest_source="terraform-registry-manifest.json"
temporary_files=()

cleanup() {
  if [[ ${#temporary_files[@]} -gt 0 ]]; then
    find "${temporary_files[@]}" -type f -delete
  fi
}
trap cleanup EXIT

checksum_files=("$dist_dir"/"$project"_*_SHA256SUMS)
if [[ ${#checksum_files[@]} -ne 1 || ! -f "${checksum_files[0]}" ]]; then
  echo "expected exactly one ${project}_*_SHA256SUMS file in $dist_dir" >&2
  exit 1
fi

checksum_file="${checksum_files[0]}"
release_name="$(basename "$checksum_file" _SHA256SUMS)"
version="${release_name#${project}_}"

platforms=(
  darwin_amd64
  darwin_arm64
  linux_amd64
  linux_arm64
  linux_arm
  windows_amd64
  windows_arm64
)

for platform in "${platforms[@]}"; do
  archive_name="${project}_${version}_${platform}.zip"
  archive_path="$dist_dir/$archive_name"
  if [[ ! -f "$archive_path" ]]; then
    echo "missing release archive $archive_path" >&2
    exit 1
  fi

  expected_binary="${project}_v${version}"
  if [[ "$platform" == windows_* ]]; then
    expected_binary+=".exe"
  fi
  binary_count="$(zipinfo -1 "$archive_path" | awk -v expected="$expected_binary" '$0 == expected { count++ } END { print count + 0 }')"
  if [[ "$binary_count" != 1 ]]; then
    echo "$archive_name must contain exactly one $expected_binary" >&2
    exit 1
  fi

  checksum_line="$(awk -v file="$archive_name" '$2 == file { print; found=1 } END { if (!found) exit 1 }' "$checksum_file")"
  (cd "$dist_dir" && printf '%s\n' "$checksum_line" | sha256sum --check --status)

  if [[ "$platform" == "linux_amd64" || "$platform" == "linux_arm" ]]; then
    binary_file="$(mktemp)"
    temporary_files+=("$binary_file")
    unzip -p "$archive_path" "$expected_binary" >"$binary_file"
    build_metadata="$("${GO:-go}" version -m "$binary_file")"
    grep -q 'CGO_ENABLED=0' <<<"$build_metadata"
    if [[ "$platform" == "linux_amd64" ]]; then
      grep -q 'GOARCH=amd64' <<<"$build_metadata"
    else
      grep -q 'GOARCH=arm' <<<"$build_metadata"
      grep -q 'GOARM=6' <<<"$build_metadata"
    fi
    grep -aFq "$version" "$binary_file"
  fi
done

manifest_name="${release_name}_manifest.json"
manifest_checksum="$(awk -v file="$manifest_name" '$2 == file { print $1; found=1 } END { if (!found) exit 1 }' "$checksum_file")"
actual_manifest_checksum="$(sha256sum "$manifest_source" | awk '{ print $1 }')"
if [[ "$manifest_checksum" != "$actual_manifest_checksum" ]]; then
  echo "checksum for $manifest_name does not match $manifest_source" >&2
  exit 1
fi

if [[ "${REQUIRE_SIGNATURE:-0}" == 1 ]]; then
  signature_file="${checksum_file}.sig"
  if [[ ! -f "$signature_file" ]]; then
    echo "missing detached checksum signature $signature_file" >&2
    exit 1
  fi
  gpg --batch --verify "$signature_file" "$checksum_file"
fi

echo "verified Registry release assets for $version"
