resource "peakhour_rp_waf_rule_group" "example" {
  domain    = "example.com"
  ruleset   = "owaspv33"
  file_name = "REQUEST-920-PROTOCOL-ENFORCEMENT.conf"
  enabled   = true
}
