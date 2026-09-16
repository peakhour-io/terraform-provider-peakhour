# Contributing

Thank you for contributing to the Peakhour Terraform provider.

## Development setup

Install the Go and Terraform versions declared by `go.mod` and the examples,
then download dependencies:

```shell
go mod download
```

Run the local quality gates before opening a pull request:

```shell
go test -count=1 ./...
go vet ./...
make generate
scripts/validate-registry-examples.sh
git diff --exit-code
```

When changing JSON normalization or preparing a local release, verify the
provider's checked-in contract vectors against the reviewed stable API source:

```shell
make test-stable-contract
```

The target uses `../peakhour-website-stable` by default. Set
`PEAKHOUR_WEBSITE_STABLE_PATH` when the stable API checkout is elsewhere. This
check intentionally runs during the maintainer's local release workflow rather
than public CI, which does not have access to the sibling repository. It checks
the pinned Python source contract and provider lifecycle fixtures; it does not
replace a live stable-API endpoint test.

`make generate` regenerates the embedded WAF JSON schema bundle, formats the
Terraform examples, and regenerates the public Registry documentation. Commit
generated schema and documentation changes with the source change that caused them.

For WAF custom-rule default comparison, run:

```shell
go generate ./internal/jsondefaults
go test -count=1 ./internal/jsondefaults ./internal/provider -run 'TestEqual|TestSchema|TestGenerated|TestJSONSchema|TestAccRPWAFCustomRule_'
make test-waf-defaults-contract
```

The generator reads `docs/spec/peakhour-api-v1.json`, copies the relevant schema
reference closure, and records the source SHA-256. Normal builds and Terraform
operations use the embedded bundle; they do not fetch a schema. The Go test suite
checks that regeneration produces exactly the committed bundle.

`test-waf-defaults-contract` compares every declared field/default and WAF JSON
persistence call against the sibling API's HEAD source. To verify a particular
deployment revision instead, run
`python3 scripts/verify-stable-json-contract.py --defaults /path/to/stable REVISION`.
This is a source contract check, not a live API test. It is separate from the
older, broader pinned null-contract check above. Defaults factories, conditional
defaults, or new schema constructs require review before extending comparison.

The schema comparator is opt-in per attribute. Its current scope and upgrade
effects are documented in [WAF JSON defaults](docs/guides/waf-json-defaults.md).

CI also runs `govulncheck` and scans Git history with Gitleaks. Before
committing, run `gitleaks dir . --redact` with Gitleaks 8.30.1 or later so
uncommitted files are included in the scan.

## Acceptance tests

Tests that use mock HTTP servers run as part of `go test ./...`. Live acceptance
tests can change real Peakhour configuration and require credentials; maintainers
run those deliberately in an isolated account. Never add credentials, Terraform
state, `.tfvars` files, or customer data to the repository.

## Pull requests

Keep changes focused, add a regression test before fixing a defect, and describe
the user-visible behavior. Confirm whether a schema change is backward
compatible and update examples and generated docs when applicable.

Only maintainers create version tags and signed releases. See
[`docs/publication-checklist.md`](docs/publication-checklist.md) for the release
and public Registry process.
