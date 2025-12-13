SHELL := /usr/bin/env bash

.PHONY: build install test testacc clean fmt vet lint onboard dist dist-clean dist-mirror

VERSION ?= 0.1.0
DIST_DIR ?= dist
RELEASE_PLATFORMS ?= linux_amd64 linux_arm64 darwin_amd64 darwin_arm64 windows_amd64

build:
	go build -o terraform-provider-peakhour

install: build
	@set -euo pipefail; \
	OS="$$(go env GOOS)"; \
	ARCH="$$(go env GOARCH)"; \
	EXT=""; \
	if [[ "$$OS" == "windows" ]]; then EXT=".exe"; fi; \
	SRC="terraform-provider-peakhour$${EXT}"; \
	DEST="$$HOME/.terraform.d/plugins/registry.terraform.io/peakhour-io/peakhour/$(VERSION)/$${OS}_$${ARCH}"; \
	mkdir -p "$$DEST"; \
	cp "$$SRC" "$$DEST/terraform-provider-peakhour_v$(VERSION)$${EXT}"

test:
	go test -v ./...

testacc:
	TF_ACC=1 go test ./... -run '^TestAcc' -v -timeout 60m

fmt:
	go fmt ./...

vet:
	go vet ./...

clean:
	rm -f terraform-provider-peakhour
	rm -rf examples/*/.terraform
	rm -rf examples/*/.terraform.lock.hcl
	rm -rf examples/*/terraform.tfstate*

lint: fmt vet
	go mod tidy

onboard:
	go build -o peakhour-tf-onboard ./cmd/peakhour-tf-onboard

dist: dist-mirror

dist-clean:
	rm -rf $(DIST_DIR)

dist-mirror:
	@set -euo pipefail; \
	V="$(VERSION)"; \
	D="$(DIST_DIR)"; \
	HOST="registry.terraform.io"; \
	NAMESPACE="peakhour-io"; \
	NAME="peakhour"; \
	rm -rf "$$D/$$HOST/$$NAMESPACE/$$NAME/$$V"; \
	for PLATFORM in $(RELEASE_PLATFORMS); do \
	  OS="$${PLATFORM%_*}"; \
	  ARCH="$${PLATFORM#*_}"; \
	  EXT=""; \
	  if [[ "$$OS" == "windows" ]]; then EXT=".exe"; fi; \
	  OUTDIR="$$D/$$HOST/$$NAMESPACE/$$NAME/$$V/$${OS}_$${ARCH}"; \
	  mkdir -p "$$OUTDIR"; \
	  CGO_ENABLED=0 GOOS="$$OS" GOARCH="$$ARCH" \
	    go build -ldflags "-X main.version=$$V" \
	    -o "$$OUTDIR/terraform-provider-$${NAME}_v$$V$${EXT}" .; \
	done; \
	cd "$$D"; \
	if command -v sha256sum >/dev/null 2>&1; then \
	  sha256sum $$(find "$$HOST" -type f) > SHA256SUMS; \
	else \
	  shasum -a 256 $$(find "$$HOST" -type f) > SHA256SUMS; \
	fi; \
	tar -czf "peakhour-provider_$$V.tar.gz" "$$HOST" SHA256SUMS; \
	echo "Wrote $$D/peakhour-provider_$$V.tar.gz"
