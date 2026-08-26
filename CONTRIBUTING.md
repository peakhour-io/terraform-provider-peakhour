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

`make generate` formats the Terraform examples and regenerates the public
Registry documentation. Commit every generated documentation change with the
schema change that caused it.

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
