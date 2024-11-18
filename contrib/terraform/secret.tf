resource "vault_kv_secret_v2" "example" {
  mount                      = "/secret"
  name                       = "test"
  cas                        = 1
  delete_all_versions        = true
  data_json                  = jsonencode(
    {
      zip       = "zap",
      foo       = "bar"
    }
  )
  custom_metadata {
    max_versions = 5
    data = {
      foo = "vault@example.com",
      bar = "12345"
    }
  }
}


