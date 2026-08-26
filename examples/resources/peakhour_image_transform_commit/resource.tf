resource "peakhour_image_transform_commit" "example" {
  domain = "example.com"
  triggers = {
    revision = "deploy-2026-08-26"
  }
}
