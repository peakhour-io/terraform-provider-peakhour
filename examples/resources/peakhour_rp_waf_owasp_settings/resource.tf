resource "peakhour_rp_waf_owasp_settings" "example" {
  domain = "example.com"

  settings_json = jsonencode({
    methods = {
      allowed_methods = ["GET", "HEAD", "POST", "OPTIONS"]
    }
  })
}
