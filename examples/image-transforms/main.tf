terraform {
  required_providers {
    peakhour = {
      source = "peakhour.io/peakhour/peakhour"
    }
  }
}

provider "peakhour" {
  api_key = "your-api-key"
}

resource "peakhour_image_transform" "thumbnail" {
  domain = "example.com"
  name   = "thumbnail"
  config_json = jsonencode({
    width   = 200
    height  = 200
    fit     = "cover"
    quality = 80
  })
}

resource "peakhour_image_transform" "hero" {
  domain = "example.com"
  name   = "hero"
  config_json = jsonencode({
    width   = 1200
    format  = "auto"
    quality = 90
  })
}
