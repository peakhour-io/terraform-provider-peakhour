package spec

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const releaseTestCommit = "0123456789abcdef0123456789abcdef01234567"

func writeExecutable(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
		t.Fatalf("write executable %s: %v", path, err)
	}
}

func releaseScriptFixture(t *testing.T, overrides ...string) (string, string, []string) {
	t.Helper()

	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(filepath.Join(root, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}

	releaseScript := readRepositoryFile(t, "scripts/release-local.sh")
	writeExecutable(t, filepath.Join(root, "scripts", "release-local.sh"), releaseScript)
	logPath := filepath.Join(root, "commands.log")

	shim := `#!/usr/bin/env bash
set -euo pipefail
name="$(basename "$0")"
printf '%s %s|github_token=%s|gh_token=%s\n' "$name" "$*" "${GITHUB_TOKEN:+set}" "${GH_TOKEN:+set}" >>"$RELEASE_TEST_LOG"
if [[ "$name" == git ]]; then
  case "$*" in
    "status --porcelain") [[ "${MOCK_DIRTY:-0}" == 1 ]] && echo dirty ;;
    "remote") echo origin ;;
    "remote get-url origin")
      if [[ "${MOCK_WRONG_REMOTE:-0}" == 1 ]]; then
        echo git@github.com:somewhere/else.git
      else
        echo git@github.com:peakhour-io/terraform-provider-peakhour.git
      fi ;;
    "show-ref --verify --quiet refs/heads/v1.2.3") [[ "${MOCK_LOCAL_BRANCH:-0}" == 1 ]] ;;
    "show-ref --verify --quiet refs/tags/v1.2.3") [[ "${MOCK_MISSING_LOCAL_TAG:-0}" != 1 ]] ;;
    "rev-parse HEAD") echo "` + releaseTestCommit + `" ;;
    "rev-parse refs/tags/v1.2.3^{commit}")
      if [[ "${MOCK_LOCAL_TAG_MISMATCH:-0}" == 1 ]]; then echo bad; else echo "` + releaseTestCommit + `"; fi ;;
    "ls-remote origin refs/heads/v1.2.3")
      [[ "${MOCK_REMOTE_BRANCH:-0}" == 1 ]] && echo "` + releaseTestCommit + ` refs/heads/v1.2.3" ;;
    "ls-remote origin refs/tags/v1.2.3^{}")
      if [[ "${MOCK_MISSING_REMOTE_TAG:-0}" != 1 ]]; then
        if [[ "${MOCK_REMOTE_TAG_MISMATCH:-0}" == 1 ]]; then echo "bad refs/tags/v1.2.3^{}"; else echo "` + releaseTestCommit + ` refs/tags/v1.2.3^{}"; fi
      fi ;;
    "ls-remote origin refs/tags/v1.2.3") ;;
  esac
elif [[ "$name" == gpg ]]; then
  if [[ "$*" == "--with-colons --list-secret-keys "* ]]; then
    if [[ "${MOCK_MISSING_KEY:-0}" != 1 ]]; then
      if [[ "${MOCK_ECC_KEY:-0}" == 1 ]]; then echo 'sec:u:255:22:ABCDEF:0:0:::::::'; else echo 'sec:u:3072:1:ABCDEF:0:0:::::::'; fi
    fi
  elif [[ "$*" == *"--detach-sign"* ]]; then
    [[ "${MOCK_SIGN_FAILURE:-0}" != 1 ]] || exit 1
    while [[ $# -gt 0 ]]; do
      if [[ "$1" == --output ]]; then shift; : >"$1"; break; fi
      shift
    done
  fi
elif [[ "$name" == goreleaser && "$*" == --version ]]; then
  if [[ "${MOCK_OLD_GORELEASER:-0}" == 1 ]]; then echo 'GitVersion: 2.17.0'; else echo 'GitVersion: 2.18.0'; fi
elif [[ "$name" == go && "$*" == "test -count=1 ./..." && "${MOCK_TEST_FAILURE:-0}" == 1 ]]; then
  exit 1
fi
`
	shimPath := filepath.Join(binDir, "shim")
	writeExecutable(t, shimPath, shim)
	for _, name := range []string{"git", "go", "terraform", "goreleaser", "gpg", "make", "gh"} {
		if err := os.Symlink("shim", filepath.Join(binDir, name)); err != nil {
			t.Fatalf("symlink %s: %v", name, err)
		}
	}

	writeExecutable(t, filepath.Join(root, "scripts", "validate-registry-examples.sh"), `#!/usr/bin/env bash
printf 'validate|github_token=%s\n' "${GITHUB_TOKEN:+set}" >>"$RELEASE_TEST_LOG"
`)
	writeExecutable(t, filepath.Join(root, "scripts", "verify-release-assets.sh"), `#!/usr/bin/env bash
printf 'verify|required=%s|github_token=%s\n' "${REQUIRE_SIGNATURE:-0}" "${GITHUB_TOKEN:+set}" >>"$RELEASE_TEST_LOG"
`)
	writeExecutable(t, filepath.Join(root, "scripts", "verify-stable-json-contract.py"), `#!/usr/bin/env bash
printf 'stable-contract %s|github_token=%s\n' "$*" "${GITHUB_TOKEN:+set}" >>"$RELEASE_TEST_LOG"
[[ "${MOCK_CONTRACT_FAILURE:-0}" != 1 ]]
`)

	environment := append(os.Environ(),
		"PATH="+binDir+":/usr/bin:/bin",
		"RELEASE_TEST_LOG="+logPath,
		"GITHUB_TOKEN=test-token",
		"GPG_FINGERPRINT=ABCDEF",
	)
	environment = append(environment, overrides...)
	return root, logPath, environment
}

func runReleaseFixture(t *testing.T, overrides ...string) (string, string, error) {
	t.Helper()
	root, logPath, environment := releaseScriptFixture(t, overrides...)
	command := exec.Command("bash", "scripts/release-local.sh", "v1.2.3")
	command.Dir = root
	command.Env = environment
	output, err := command.CombinedOutput()
	log, readErr := os.ReadFile(logPath)
	if readErr != nil && !os.IsNotExist(readErr) {
		t.Fatalf("read command log: %v", readErr)
	}
	return string(output), string(log), err
}

func TestLocalReleaseRejectsInvalidSemver(t *testing.T) {
	root, _, environment := releaseScriptFixture(t)
	command := exec.Command("bash", "scripts/release-local.sh", "v1.2.3-01")
	command.Dir = root
	command.Env = environment
	output, err := command.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "v-prefixed semantic version") {
		t.Fatalf("invalid SemVer result: err=%v output=%s", err, output)
	}
}

func TestLocalReleaseRejectsUnsafeRepositoryState(t *testing.T) {
	tests := []struct {
		name     string
		override string
		message  string
	}{
		{"dirty worktree", "MOCK_DIRTY=1", "worktree must be clean"},
		{"local branch conflict", "MOCK_LOCAL_BRANCH=1", "branch conflicts"},
		{"missing local tag", "MOCK_MISSING_LOCAL_TAG=1", "create and push release tag"},
		{"local tag mismatch", "MOCK_LOCAL_TAG_MISMATCH=1", "does not point to HEAD"},
		{"wrong remote", "MOCK_WRONG_REMOTE=1", "public GitHub repository"},
		{"remote branch conflict", "MOCK_REMOTE_BRANCH=1", "branch conflicts"},
		{"missing remote tag", "MOCK_MISSING_REMOTE_TAG=1", "not present on the GitHub remote"},
		{"remote tag mismatch", "MOCK_REMOTE_TAG_MISMATCH=1", "not present on the GitHub remote"},
		{"missing signing key", "MOCK_MISSING_KEY=1", "secret key is unavailable"},
		{"unsupported signing key", "MOCK_ECC_KEY=1", "must be RSA or DSA"},
		{"wrong GoReleaser version", "MOCK_OLD_GORELEASER=1", "GoReleaser v2.18.0 is required"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output, log, err := runReleaseFixture(t, test.override)
			if err == nil || !strings.Contains(output, test.message) {
				t.Fatalf("unsafe state result: err=%v output=%s", err, output)
			}
			if strings.Contains(log, "goreleaser release --clean") {
				t.Fatalf("unsafe state reached release command:\n%s", log)
			}
		})
	}
}

func TestLocalReleaseRequiresTokenAndFailsBeforePublishing(t *testing.T) {
	output, log, err := runReleaseFixture(t, "GITHUB_TOKEN=")
	if err == nil || !strings.Contains(output, "set GITHUB_TOKEN") {
		t.Fatalf("missing token result: err=%v output=%s", err, output)
	}
	if strings.Contains(log, "goreleaser release --clean") {
		t.Fatalf("missing token reached release command:\n%s", log)
	}

	for _, override := range []string{"MOCK_SIGN_FAILURE=1", "MOCK_TEST_FAILURE=1", "MOCK_CONTRACT_FAILURE=1"} {
		output, log, err = runReleaseFixture(t, override)
		if err == nil {
			t.Fatalf("%s did not stop release: output=%s", override, output)
		}
		if strings.Contains(log, "goreleaser release --clean") {
			t.Fatalf("%s reached release command:\n%s", override, log)
		}
	}
}

func TestLocalReleaseIsolatesTokenAndFinalizesOnlyAfterVerification(t *testing.T) {
	output, log, err := runReleaseFixture(t)
	if err != nil {
		t.Fatalf("release fixture failed: %v\n%s\n%s", err, output, log)
	}

	for _, line := range strings.Split(strings.TrimSpace(log), "\n") {
		if strings.Contains(line, "github_token=set") && !strings.HasPrefix(line, "goreleaser release --clean|") {
			t.Errorf("GitHub token leaked outside production GoReleaser: %s", line)
		}
	}

	production := strings.Index(log, "goreleaser release --clean|github_token=set")
	verification := strings.Index(log, "verify|required=1|github_token=")
	finalization := strings.Index(log, "gh release edit v1.2.3 --repo peakhour-io/terraform-provider-peakhour --draft=false")
	if production < 0 || verification < production || finalization < verification {
		t.Fatalf("release was not produced as a draft, verified, then finalized:\n%s", log)
	}
}
