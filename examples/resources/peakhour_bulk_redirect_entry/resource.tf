resource "peakhour_bulk_redirect_entry" "example" {
  domain             = "example.com"
  bulk_redirect_uuid = "11111111-1111-1111-1111-111111111111"
  source_path        = "/old-page"
  target_url         = "https://example.com/new-page"
  status_code        = 301
}
