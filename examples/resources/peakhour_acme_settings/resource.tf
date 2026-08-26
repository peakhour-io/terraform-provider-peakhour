resource "peakhour_acme_settings" "example" {
  domain       = "example.com"
  domain_names = ["example.com", "www.example.com"]
}
