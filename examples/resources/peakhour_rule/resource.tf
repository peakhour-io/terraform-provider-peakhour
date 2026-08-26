resource "peakhour_rule" "example" {
  domain     = "example.com"
  phase      = "firewall"
  name       = "Block private paths"
  filter_str = "http.request.uri.path matches \"^/private/\""
  enabled    = true

  actions_json = jsonencode({
    firewall = [{
      type   = "firewall"
      action = "deny"
    }]
  })
}
