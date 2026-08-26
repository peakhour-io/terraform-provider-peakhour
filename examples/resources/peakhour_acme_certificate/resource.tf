resource "peakhour_acme_certificate" "example" {
  domain = "example.com"
  issue  = true
}
