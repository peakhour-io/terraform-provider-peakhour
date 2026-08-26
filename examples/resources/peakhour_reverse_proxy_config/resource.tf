resource "peakhour_reverse_proxy_config" "example" {
  domain = "example.com"
  gzip   = true
  brotli = true
}
