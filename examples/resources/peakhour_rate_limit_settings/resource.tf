resource "peakhour_rate_limit_settings" "example" {
  domain = "example.com"
  mode   = ["zone", "global"]
}
