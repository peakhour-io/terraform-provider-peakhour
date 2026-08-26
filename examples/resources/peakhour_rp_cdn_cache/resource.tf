resource "peakhour_rp_cdn_cache" "example" {
  domain        = "example.com"
  cache_enabled = true
  edge_ttl_sec  = 3600
}
