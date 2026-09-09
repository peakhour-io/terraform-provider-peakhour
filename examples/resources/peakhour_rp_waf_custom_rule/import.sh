# Import identity remains the domain and UUID, regardless of the rule name.
terraform import peakhour_rp_waf_custom_rule.example example.com/customrule/11111111-1111-1111-1111-111111111111

# Inspect imported metadata and copy name, description, and enabled into your
# resource configuration to preserve their existing values before applying.
terraform state show peakhour_rp_waf_custom_rule.example
