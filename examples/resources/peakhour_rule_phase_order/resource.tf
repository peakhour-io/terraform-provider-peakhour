resource "peakhour_rule_phase_order" "example" {
  domain            = "example.com"
  phase             = "firewall"
  include_all_rules = false
  order             = ["11111111-1111-1111-1111-111111111111"]
}
