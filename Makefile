.PHONY: build install test clean fmt vet

build:
	go build -o terraform-provider-peakhour

install: build
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/peakhour-io/peakhour/0.1.0/linux_amd64
	cp terraform-provider-peakhour ~/.terraform.d/plugins/registry.terraform.io/peakhour-io/peakhour/0.1.0/linux_amd64/

test:
	go test -v ./...

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
