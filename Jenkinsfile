pipeline {
  agent any

  options {
    timestamps()
  }

  parameters {
    string(name: 'GO_VERSION', defaultValue: '1.22.10', description: 'Go toolchain version for builds/tests.')
    booleanParam(name: 'RUN_ACCEPTANCE', defaultValue: false, description: 'Run Terraform acceptance tests (requires Peakhour credentials).')
    string(name: 'PEAKHOUR_TEST_DOMAIN', defaultValue: '', description: 'Existing Peakhour domain to use for acceptance tests.')
    string(name: 'PEAKHOUR_BASE_URL', defaultValue: '', description: 'Optional Peakhour API base URL (defaults to console.peakhour.io).')
    string(name: 'TERRAFORM_VERSION', defaultValue: '1.14.2', description: 'Terraform CLI version for acceptance tests.')
  }

  environment {
    GOFLAGS = '-count=1'
  }

  stages {
    stage('Unit') {
      steps {
        sh '''#!/usr/bin/env bash
set -euo pipefail

GO_VERSION="${GO_VERSION}"
OS="linux"
ARCH="amd64"
TAR="go${GO_VERSION}.${OS}-${ARCH}.tar.gz"

mkdir -p .toolchains/tmp
if [[ -x ".toolchains/go/bin/go" ]] && [[ "$(.toolchains/go/bin/go env GOVERSION)" == "go${GO_VERSION}" ]]; then
  echo "Using cached Go $(.toolchains/go/bin/go env GOVERSION)"
else
  rm -rf .toolchains/go
  pushd .toolchains/tmp >/dev/null
    rm -f "${TAR}" "${TAR}.sha256"
    curl -fsSLo "${TAR}" "https://go.dev/dl/${TAR}"
    curl -fsSLo "${TAR}.sha256" "https://go.dev/dl/${TAR}.sha256"
    sha256sum -c "${TAR}.sha256"
    tar -C .. -xzf "${TAR}"
  popd >/dev/null
fi

export PATH="${PWD}/.toolchains/go/bin:${PATH}"
go version
make vet
make test
'''
      }
    }

    stage('Acceptance') {
      when {
        expression { return params.RUN_ACCEPTANCE }
      }

      options {
        timeout(time: 60, unit: 'MINUTES')
      }

      steps {
        withCredentials([string(credentialsId: 'peakhour-api-key', variable: 'PEAKHOUR_API_KEY')]) {
          sh '''#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${PEAKHOUR_TEST_DOMAIN}" ]]; then
  echo "PEAKHOUR_TEST_DOMAIN Jenkins parameter is required for acceptance tests" >&2
  exit 1
fi

TF_VERSION="${TERRAFORM_VERSION}"
OS="linux"
ARCH="amd64"
ZIP="terraform_${TF_VERSION}_${OS}_${ARCH}.zip"

mkdir -p .toolchains/bin .toolchains/tmp
pushd .toolchains/tmp >/dev/null
  rm -f "${ZIP}" SHA256SUMS
  curl -fsSLo "${ZIP}" "https://releases.hashicorp.com/terraform/${TF_VERSION}/${ZIP}"
  curl -fsSLo SHA256SUMS "https://releases.hashicorp.com/terraform/${TF_VERSION}/terraform_${TF_VERSION}_SHA256SUMS"
  grep "${ZIP}$" SHA256SUMS | sha256sum -c -
  unzip -o "${ZIP}" -d ../bin >/dev/null
popd >/dev/null

export PATH="${PWD}/.toolchains/go/bin:${PWD}/.toolchains/bin:${PATH}"
go version
terraform version | head -n 2

export TF_ACC=1
export PEAKHOUR_TEST_DOMAIN="${PEAKHOUR_TEST_DOMAIN}"

if [[ -n "${PEAKHOUR_BASE_URL}" ]]; then
  export PEAKHOUR_BASE_URL="${PEAKHOUR_BASE_URL}"
fi

make testacc
'''
        }
      }
    }
  }
}
