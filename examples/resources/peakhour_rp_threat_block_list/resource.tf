resource "peakhour_rp_threat_block_list" "example" {
  domain     = "example.com"
  blocklists = ["tor"]
}
