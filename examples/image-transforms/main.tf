terraform {
  required_providers {
    peakhour = {
      source = "peakhour-io/peakhour"
    }
  }
}

provider "peakhour" {
  # API key from PEAKHOUR_API_KEY environment variable
}

resource "peakhour_domain" "images" {
  name = "images.example.com"
}

resource "peakhour_domain_plan" "images" {
  domain = peakhour_domain.images.name
  code   = "business2"
}

resource "peakhour_reverse_proxy_service" "images" {
  domain     = peakhour_domain.images.name
  depends_on = [peakhour_domain_plan.images]
}

resource "peakhour_image_transform" "thumbnail" {
  domain = peakhour_domain.images.name
  name   = "thumbnail"
  config_json = jsonencode({
    w   = 200
    h   = 200
    fit = "crop" # Valid: clip, crop, fill, scale, facearea
    q   = 80     # Quality 0-100 or "auto", "auto:high", "auto:med", "auto:low"
  })

  depends_on = [peakhour_reverse_proxy_service.images]
}

resource "peakhour_image_transform" "hero" {
  domain = peakhour_domain.images.name
  name   = "hero"
  config_json = jsonencode({
    w  = 1200
    fm = "WEBP" # Valid: GIF, JPEG, PNG, WEBP, SVG, AVIF, JXL
    q  = "auto"
  })

  depends_on = [peakhour_reverse_proxy_service.images]
}

# Commit all image transform changes at once
resource "peakhour_image_transform_commit" "images" {
  domain = peakhour_domain.images.name

  # Triggers commit when any transform changes
  triggers = {
    thumbnail = peakhour_image_transform.thumbnail.id
    hero      = peakhour_image_transform.hero.id
  }
}
