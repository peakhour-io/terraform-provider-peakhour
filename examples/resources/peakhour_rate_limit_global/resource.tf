resource "peakhour_rate_limit_global" "example" {
  domain                 = "example.com"
  concurrent_connections = 1000
}
