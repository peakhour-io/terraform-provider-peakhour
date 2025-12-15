terraform {
  required_providers {
    peakhour = {
      source = "peakhour-io/peakhour"
    }
  }
}

provider "peakhour" {
  # API key can be set via PEAKHOUR_API_KEY environment variable
}

resource "peakhour_domain" "example" {
  name = "example.com"
}

resource "peakhour_reverse_proxy_service" "example" {
  domain = peakhour_domain.example.name
}

# These resources are "actions" (purge requests), not stable desired-state.
# To re-run a purge, change run_id (e.g. bump a number or use a release identifier).

resource "peakhour_rp_cdn_purge_resources" "static_assets" {
  domain = peakhour_domain.example.name
  run_id = "release-2025-12-13"

  paths = ["/index.html", "/assets/app.js"]
  soft  = true

  depends_on = [peakhour_reverse_proxy_service.example]
}

resource "peakhour_rp_cdn_purge_wildcard" "images" {
  domain = peakhour_domain.example.name
  run_id = "release-2025-12-13-images"

  paths = ["/images/*"]

  depends_on = [peakhour_reverse_proxy_service.example]
}

resource "peakhour_rp_cdn_purge_tags" "by_tag" {
  domain = peakhour_domain.example.name
  run_id = "release-2025-12-13-tags"

  tags = ["marketing"]

  depends_on = [peakhour_reverse_proxy_service.example]
}

