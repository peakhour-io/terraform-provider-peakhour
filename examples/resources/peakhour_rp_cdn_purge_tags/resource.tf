resource "peakhour_rp_cdn_purge_tags" "example" {
  domain = "example.com"
  run_id = "deploy-2026-08-26"
  tags   = ["product-123", "catalog"]
  soft   = true
}
