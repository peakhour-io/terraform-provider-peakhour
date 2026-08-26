resource "peakhour_bulk_redirect_list" "example" {
  domain      = "example.com"
  name        = "legacy-pages"
  description = "Redirect retired URLs"
}
