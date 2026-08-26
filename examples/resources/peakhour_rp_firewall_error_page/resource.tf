resource "peakhour_rp_firewall_error_page" "example" {
  domain  = "example.com"
  content = "<!doctype html><title>Request blocked</title>"
}
