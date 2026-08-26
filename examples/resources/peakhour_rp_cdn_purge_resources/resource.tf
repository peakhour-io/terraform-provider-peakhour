resource "peakhour_rp_cdn_purge_resources" "example" {
  domain = "example.com"
  run_id = "deploy-2026-08-26"
  paths  = ["/assets/app.css", "/assets/app.js"]
  soft   = true
}
