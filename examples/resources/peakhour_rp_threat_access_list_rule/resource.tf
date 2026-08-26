resource "peakhour_rp_threat_access_list_rule" "example" {
  domain      = "example.com"
  rule_type   = "whitelist"
  content     = "203.0.113.10"
  description = "Allow the office IP"
}
