resource "peakhour_rp_waf_options" "example" {
  domain      = "example.com"
  waf_mode    = "enabled"
  waf_ruleset = "owaspv33"
}
