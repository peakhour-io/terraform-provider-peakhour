resource "peakhour_rp_firewall_settings" "example" {
  domain               = "example.com"
  challenge_cookie_key = ["fingerprint_tls", "ip"]
}
