terraform {
  required_providers {
    peakhour = {
      source = "peakhour-io/peakhour"
    }
  }
}

provider "peakhour" {
  # The API key can also be set with the PEAKHOUR_API_KEY environment variable.
  api_key = "example-api-key"
}
