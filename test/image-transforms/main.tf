terraform {
  required_providers {
    peakhour = {
      source = "peakhour-io/peakhour"
    }
  }
}

provider "peakhour" {
  api_key = "your-api-key"
}

resource "peakhour_domain" "images" {
  name = "images.example.com"
}

resource "peakhour_reverse_proxy_service" "images" {
  domain = peakhour_domain.images.name
}

resource "peakhour_image_transform" "thumbnail" {
  domain = peakhour_domain.images.name
  name   = "thumbnail"
  config_json = jsonencode({
    width   = 200
    height  = 200
    fit     = "cover"
    quality = 80
  })

  depends_on = [peakhour_reverse_proxy_service.images]
}

resource "peakhour_image_transform" "hero" {
  domain = peakhour_domain.images.name
  name   = "hero"
  config_json = jsonencode({
    width   = 1200
    format  = "auto"
    quality = 90
  })

  depends_on = [peakhour_reverse_proxy_service.images]
}
