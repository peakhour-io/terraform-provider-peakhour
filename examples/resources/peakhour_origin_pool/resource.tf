resource "peakhour_origin_pool" "example" {
  domain = "example.com"
  tag    = "primary"

  address = [{
    address = "origin.example.com:443"
    weight  = 100
  }]

  load_balancing_mode = "round_robin"
}
