SHELL := /usr/bin/env bash

.PHONY: build install test testacc clean fmt vet lint generate onboard dist dist-clean dist-mirror \
        build-linux build-linux-arm64 build-darwin build-darwin-arm64 build-windows build-all \
        beta-bundle

VERSION ?= 0.1.2
DIST_DIR ?= dist
RELEASE_PLATFORMS ?= linux_amd64 linux_arm64 darwin_amd64 darwin_arm64 windows_amd64
BINARY_NAME = terraform-provider-peakhour

build:
	go build -o $(BINARY_NAME)

# Build flags for smaller binaries (-s strips symbol table, -w strips DWARF)
LDFLAGS = -s -w -X main.version=$(VERSION)
GOBUILD = GOTOOLCHAIN=local CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)"

# Individual platform builds (stripped)
build-linux:
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME)_linux_amd64

build-linux-arm64:
	GOOS=linux GOARCH=arm64 $(GOBUILD) -o $(BINARY_NAME)_linux_arm64

build-darwin:
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME)_darwin_amd64

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -o $(BINARY_NAME)_darwin_arm64

build-windows:
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME)_windows_amd64.exe

build-all: build-linux build-linux-arm64 build-darwin build-darwin-arm64 build-windows
	@echo "Built binaries for all platforms:"
	@ls -lh $(BINARY_NAME)_*

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
	go test -count=1 -v ./...

testacc:
	TF_ACC=1 go test ./... -run '^TestAcc' -v -timeout 60m

fmt:
	go fmt ./...

vet:
	go vet ./...

clean:
	rm -f terraform-provider-peakhour terraform-provider-peakhour_* peakhour-tf-onboard_*
	rm -rf examples/*/.terraform
	rm -rf examples/*/.terraform.lock.hcl
	rm -rf examples/*/terraform.tfstate*

lint: fmt vet
	go mod tidy

generate:
	terraform fmt -recursive examples/
	cd tools && go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-dir .. --provider-name peakhour

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
	  CGO_ENABLED=0 GOOS="$$OS" GOARCH="$$ARCH" \
	    go build -o "$$OUTDIR/peakhour-tf-onboard_v$$V$${EXT}" ./cmd/peakhour-tf-onboard; \
	done; \
	cd "$$D"; \
	if command -v sha256sum >/dev/null 2>&1; then \
	  sha256sum $$(find "$$HOST" -type f) > SHA256SUMS; \
	else \
	  shasum -a 256 $$(find "$$HOST" -type f) > SHA256SUMS; \
	fi; \
	tar -cJf "peakhour-provider_$$V.tar.xz" "$$HOST" SHA256SUMS; \
	echo "Wrote $$D/peakhour-provider_$$V.tar.xz"

# Beta bundle for distribution to testers
beta-bundle: build-all
	@set -euo pipefail; \
	V="$(VERSION)"; \
	rm -rf peakhour-terraform-beta-$$V-*; \
	for PLATFORM in $(RELEASE_PLATFORMS); do \
	  OS="$${PLATFORM%_*}"; \
	  ARCH="$${PLATFORM#*_}"; \
	  EXT=""; \
	  if [[ "$$OS" == "windows" ]]; then EXT=".exe"; fi; \
	  BUNDLE_DIR="peakhour-terraform-beta-$$V-$${OS}_$${ARCH}"; \
	  rm -rf "$$BUNDLE_DIR" "$$BUNDLE_DIR.tar.xz" "$$BUNDLE_DIR.zip"; \
	  mkdir -p "$$BUNDLE_DIR"; \
	  cp "$(BINARY_NAME)_$${OS}_$${ARCH}$${EXT}" "$$BUNDLE_DIR/"; \
	  CGO_ENABLED=0 GOOS="$$OS" GOARCH="$$ARCH" \
	    go build -o "peakhour-tf-onboard_$${OS}_$${ARCH}$${EXT}" ./cmd/peakhour-tf-onboard; \
	  cp "peakhour-tf-onboard_$${OS}_$${ARCH}$${EXT}" "$$BUNDLE_DIR/"; \
	  sed "s/^\*\*Provider Version:\*\*.*/**Provider Version:** $$V (Beta)/" \
	    BETA_TESTER_GUIDE.md > "$$BUNDLE_DIR/README.md"; \
	  rsync -a --exclude='.terraform' --exclude='.terraform.lock.hcl' \
	    --exclude='terraform.tfstate*' --exclude='*.tfrc' \
	    examples/ "$$BUNDLE_DIR/examples/"; \
	  cd "$$BUNDLE_DIR" && \
	  if command -v sha256sum >/dev/null 2>&1; then \
	    sha256sum $(BINARY_NAME)_* peakhour-tf-onboard_* > SHA256SUMS; \
	  else \
	    shasum -a 256 $(BINARY_NAME)_* peakhour-tf-onboard_* > SHA256SUMS; \
	  fi; \
	  cd ..; \
	  tar -cJf "$$BUNDLE_DIR.tar.xz" "$$BUNDLE_DIR"; \
	  if command -v zip >/dev/null 2>&1; then \
	    zip -9 -r "$$BUNDLE_DIR.zip" "$$BUNDLE_DIR"; \
	  else \
	    python3 -m zipfile -c "$$BUNDLE_DIR.zip" "$$BUNDLE_DIR"; \
	  fi; \
	  rm -rf "$$BUNDLE_DIR"; \
	done; \
	ls -lh peakhour-terraform-beta-$$V-*.tar.xz peakhour-terraform-beta-$$V-*.zip
