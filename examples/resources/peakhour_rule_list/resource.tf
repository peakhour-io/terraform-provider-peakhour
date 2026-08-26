resource "peakhour_rule_list" "example" {
  domain = "example.com"
  name   = "trusted-networks"
  type   = "ip"
  ips    = ["203.0.113.0/24"]
}
