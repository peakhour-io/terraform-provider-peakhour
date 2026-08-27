package spec

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/peakhour-io/terraform-provider-peakhour/internal/client"
	"github.com/peakhour-io/terraform-provider-peakhour/internal/provider"
)

const providerProjectName = "terraform-provider-peakhour"

func repositoryRoot(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
}

func readRepositoryFile(t *testing.T, name string) string {
	t.Helper()

	b, err := os.ReadFile(filepath.Join(repositoryRoot(t), name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(b)
}

func requireContains(t *testing.T, value string, fragments ...string) {
	t.Helper()

	for _, fragment := range fragments {
		if !strings.Contains(value, fragment) {
			t.Errorf("missing required publication configuration %q", fragment)
		}
	}
}

func TestRegistryManifestDeclaresProtocolVersionSix(t *testing.T) {
	var manifest struct {
		Version  int `json:"version"`
		Metadata struct {
			ProtocolVersions []string `json:"protocol_versions"`
		} `json:"metadata"`
	}

	if err := json.Unmarshal([]byte(readRepositoryFile(t, "terraform-registry-manifest.json")), &manifest); err != nil {
		t.Fatalf("parse terraform-registry-manifest.json: %v", err)
	}
	if manifest.Version != 1 {
		t.Errorf("manifest version = %d, want 1", manifest.Version)
	}
	if len(manifest.Metadata.ProtocolVersions) != 1 || manifest.Metadata.ProtocolVersions[0] != "6.0" {
		t.Errorf("protocol_versions = %v, want [6.0]", manifest.Metadata.ProtocolVersions)
	}
}

func TestGoReleaserProducesRegistryArtifacts(t *testing.T) {
	config := readRepositoryFile(t, ".goreleaser.yml")
	requireContains(t, config,
		"CGO_ENABLED=0",
		"-X main.version={{.Version}}",
		"binary: '{{ .ProjectName }}_v{{ .Version }}'",
		"formats:",
		"- zip",
		"{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}",
		"{{ .ProjectName }}_{{ .Version }}_manifest.json",
		"{{ .ProjectName }}_{{ .Version }}_SHA256SUMS",
		"artifacts: checksum",
		"--detach-sign",
		"draft: true",
		"owner: peakhour-io",
		"name: terraform-provider-peakhour",
		"- '6'",
	)

	for _, target := range []string{"darwin", "linux", "windows", "amd64", "arm64"} {
		if !strings.Contains(config, "- "+target) {
			t.Errorf("release configuration does not include %s", target)
		}
	}
}

func TestLocalReleaseUsesSignedTagRelease(t *testing.T) {
	releaseScript := readRepositoryFile(t, "scripts/release-local.sh")
	requireContains(t, releaseScript,
		"prerelease_identifier=",
		"semver_pattern=",
		"GITHUB_TOKEN",
		"GH_TOKEN",
		"GPG_FINGERPRINT",
		"git status --porcelain",
		"refs/heads/$release_tag",
		"refs/tags/$release_tag",
		`git ls-remote "$github_remote" "refs/heads/$release_tag"`,
		"go test -count=1 ./...",
		"go vet ./...",
		"golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...",
		"zricethezav/gitleaks/v8@v8.30.1 git",
		"make generate",
		"git diff --exit-code",
		"goreleaser check",
		"goreleaser release --clean",
		"REQUIRE_SIGNATURE=1 scripts/verify-release-assets.sh",
		`gh release edit "$release_tag"`,
		"--draft=false",
		"scripts/validate-registry-examples.sh",
	)

	if _, err := os.Stat(filepath.Join(repositoryRoot(t), ".github/workflows/release.yml")); !os.IsNotExist(err) {
		t.Errorf(".github/workflows/release.yml must be absent for local-only releases; stat error = %v", err)
	}
}

func TestContinuousIntegrationChecksTestsGenerationAndReleaseConfig(t *testing.T) {
	workflow := readRepositoryFile(t, ".github/workflows/test.yml")
	requireContains(t, workflow,
		"go test -count=1 ./...",
		"go vet ./...",
		"govulncheck ./...",
		"zricethezav/gitleaks/v8@v8.30.1 git",
		"actionlint/cmd/actionlint@v1.7.12",
		"make generate",
		"git diff --exit-code",
		"args: check",
		"args: release --snapshot --clean --skip=sign",
		"scripts/verify-release-assets.sh",
		"scripts/validate-registry-examples.sh",
	)
}

func TestRegistryDocumentationCoversProviderSurface(t *testing.T) {
	root := repositoryRoot(t)
	for _, name := range []string{"docs/index.md", "examples/provider/provider.tf"} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Errorf("required provider documentation %s: %v", name, err)
		}
	}

	p := provider.New("test")()
	for _, factory := range p.Resources(context.Background()) {
		res := factory()
		var response resource.MetadataResponse
		res.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "peakhour"}, &response)
		_, supportsImport := res.(resource.ResourceWithImportState)
		assertRegistryDoc(t, root, "resources", response.TypeName, supportsImport)
	}
	for _, factory := range p.DataSources(context.Background()) {
		dataSource := factory()
		var response datasource.MetadataResponse
		dataSource.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "peakhour"}, &response)
		assertRegistryDoc(t, root, "data-sources", response.TypeName, false)
	}
}

func assertRegistryDoc(t *testing.T, root, category, typeName string, supportsImport bool) {
	t.Helper()

	docName := strings.TrimPrefix(typeName, "peakhour_") + ".md"
	path := filepath.Join(root, "docs", category, docName)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("%s documentation: %v", typeName, err)
		return
	}
	if !strings.Contains(string(b), "generated by tfplugindocs") {
		t.Errorf("%s is not generated by tfplugindocs", path)
	}
	if !strings.Contains(string(b), "## Example Usage") {
		t.Errorf("%s has no example usage", path)
	}
	if supportsImport && !strings.Contains(string(b), "## Import") {
		t.Errorf("%s has no import example", path)
	}
}

func TestRegistryDocumentationIsNotIgnored(t *testing.T) {
	for _, line := range strings.Split(readRepositoryFile(t, ".gitignore"), "\n") {
		if strings.TrimSpace(line) == "docs/" || strings.TrimSpace(line) == "/docs/" {
			t.Fatal(".gitignore must not ignore Registry documentation")
		}
	}
}

func TestProviderAddressMatchesPublicModule(t *testing.T) {
	main := readRepositoryFile(t, "main.go")
	module := readRepositoryFile(t, "go.mod")
	requireContains(t, main, `Address: "registry.terraform.io/peakhour-io/peakhour"`)
	requireContains(t, module, "module github.com/peakhour-io/"+providerProjectName)
}

func TestDocumentedDefaultAPIEndpointMatchesClient(t *testing.T) {
	const want = "https://www.peakhour.io"
	const legacy = "console." + "peakhour.io"

	if got := client.NewClient("test", "").BaseURL; got != want {
		t.Fatalf("default client API URL = %q, want %q", got, want)
	}

	p := provider.New("test")()
	var response frameworkprovider.SchemaResponse
	p.Schema(context.Background(), frameworkprovider.SchemaRequest{}, &response)
	description := response.Schema.Attributes["base_url"].GetDescription()
	if !strings.Contains(description, want) {
		t.Errorf("provider base_url description %q does not document %s", description, want)
	}

	root := repositoryRoot(t)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && path != root && strings.HasPrefix(entry.Name(), ".") {
			return filepath.SkipDir
		}
		if entry.IsDir() && (entry.Name() == "dist" || entry.Name() == "vendor") {
			return filepath.SkipDir
		}
		if entry.IsDir() {
			return nil
		}
		extension := filepath.Ext(path)
		if extension != ".go" && extension != ".md" && extension != ".tf" && entry.Name() != "Jenkinsfile" {
			return nil
		}
		contents, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(string(contents), legacy) {
			t.Errorf("%s documents the non-existent %s endpoint", path, legacy)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan documented API endpoints: %v", err)
	}
}

func TestLicenseContainsCompleteMPLTwoText(t *testing.T) {
	license := readRepositoryFile(t, "LICENSE")
	requireContains(t, license,
		"Mozilla Public License Version 2.0",
		`10.2. Effect of New Versions`,
		`Exhibit A - Source Code Form License Notice`,
	)
	if strings.Contains(license, "abbreviated") {
		t.Fatal("LICENSE contains abbreviated placeholder text")
	}
}

func TestPublicRepositoryGovernanceFiles(t *testing.T) {
	for _, name := range []string{
		"SECURITY.md",
		"CONTRIBUTING.md",
		".github/ISSUE_TEMPLATE/bug_report.yml",
		".github/pull_request_template.md",
		".github/dependabot.yml",
		".gitleaks.toml",
	} {
		contents := readRepositoryFile(t, name)
		if strings.TrimSpace(contents) == "" {
			t.Errorf("%s must not be empty", name)
		}
	}
}

func TestReleaseAssetVerifierCoversRequiredArtifacts(t *testing.T) {
	verifier := readRepositoryFile(t, "scripts/verify-release-assets.sh")
	requireContains(t, verifier,
		"darwin_amd64",
		"darwin_arm64",
		"linux_amd64",
		"linux_arm64",
		"linux_arm",
		"windows_amd64",
		"SHA256SUMS",
		"manifest.json",
		"sha256sum",
		"zipinfo",
		"CGO_ENABLED=0",
		"GOARM=6",
	)
}

func TestRegistryExampleValidatorUsesProviderSchema(t *testing.T) {
	validator := readRepositoryFile(t, "scripts/validate-registry-examples.sh")
	requireContains(t, validator,
		"build -o",
		"dev_overrides",
		"TF_DATA_DIR",
		"terraform-provider-peakhour",
		"validate -no-color",
		"examples/resources/peakhour_*",
		"examples/data-sources/peakhour_*",
	)
}

func TestPublicGuidesMatchReleaseToolingAndProviderSurface(t *testing.T) {
	readme := readRepositoryFile(t, "README.md")
	quickstart := readRepositoryFile(t, "QUICKSTART.md")
	distribution := readRepositoryFile(t, "docs/provider-distribution.md")
	betaGuide := readRepositoryFile(t, "BETA_TESTER_GUIDE.md")
	jenkins := readRepositoryFile(t, "Jenkinsfile")
	projectSummary := readRepositoryFile(t, "PROJECT_SUMMARY.md")

	requireContains(t, readme, "Go](https://go.dev/doc/install) >= 1.26.7")
	requireContains(t, quickstart, "Go 1.26.7+")
	requireContains(t, distribution, ".tar.xz", ".goreleaser.yml", "publication-checklist.md")
	requireContains(t, jenkins, "defaultValue: '1.26.7'")
	requireContains(t, projectSummary, "terraform-plugin-framework` v1.19.0", "37 Resources", "33 stateful resources", "Go 1.26.7+")
	for _, nonexistent := range []string{"peakhour_tls_acme", "peakhour_bulk_redirect`", "peakhour_cdn_purge`"} {
		if strings.Contains(betaGuide, nonexistent) {
			t.Errorf("BETA_TESTER_GUIDE.md advertises nonexistent resource %s", nonexistent)
		}
	}
}
