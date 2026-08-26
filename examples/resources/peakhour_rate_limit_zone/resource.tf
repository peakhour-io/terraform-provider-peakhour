resource "peakhour_rate_limit_zone" "example" {
  domain                = "example.com"
  name                  = "api"
  requests_max          = 100
  requests_interval_sec = 60
  block_duration_sec    = 300
}
