resource "peakhour_rp_waf_custom_rule_order" "example" {
  domain            = "example.com"
  include_all_rules = false
  order             = ["11111111-1111-1111-1111-111111111111"]
}
