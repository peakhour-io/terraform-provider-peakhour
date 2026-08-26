resource "peakhour_image_transform" "example" {
  domain = "example.com"
  name   = "thumbnail"

  config_json = jsonencode({
    w   = 200
    h   = 200
    fit = "crop"
  })
}
