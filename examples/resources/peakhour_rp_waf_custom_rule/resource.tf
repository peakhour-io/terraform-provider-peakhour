resource "peakhour_rp_waf_custom_rule" "example" {
  domain      = "example.com"
  description = "Log curl user agents"
  enabled     = true

  rules_json = jsonencode([{
    variable      = "REQUEST_HEADERS"
    variable_part = "user-agent"
    operator      = "@contains"
    operator_arg  = "curl"
  }])

  action_json = jsonencode({
    action_name = "pass"
  })

  logging_json = jsonencode({
    message  = "curl user agent"
    severity = "INFO"
    tags     = ["terraform"]
  })
}
