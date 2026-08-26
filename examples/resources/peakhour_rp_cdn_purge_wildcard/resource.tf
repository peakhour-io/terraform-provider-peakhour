resource "peakhour_rp_cdn_purge_wildcard" "example" {
  domain = "example.com"
  run_id = "deploy-2026-08-26"
  paths  = ["/images/*"]
  soft   = true
}
