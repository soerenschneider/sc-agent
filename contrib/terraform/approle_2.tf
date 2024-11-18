locals {
  secret_token_file_2 = pathexpand("~/secret_id_2.txt")
}

resource "local_file" "secret_id_token_2" {
  content  = vault_approle_auth_backend_role_secret_id.role2.wrapping_token
  filename = local.secret_token_file_2
}

resource "vault_approle_auth_backend_role" "role2" {
  backend   = vault_auth_backend.approle.path
  role_name = "test-role-2"
  role_id   = "test-role-2"

  token_policies         = ["default", vault_policy.approle_2.name, "prod", "ssh_client", "pki"]
  token_max_ttl          = 300
  token_explicit_max_ttl = 300
  token_ttl              = 300
  secret_id_ttl          = 1800
}

resource "vault_approle_auth_backend_role_secret_id" "role2" {
  backend               = vault_auth_backend.approle.path
  role_name             = vault_approle_auth_backend_role.role2.role_name
  with_wrapped_accessor = true
  wrapping_ttl          = "5m"

  metadata = jsonencode(
    {
      "hello"     = "world"
      "role_name" = "test-role"
    }
  )
}

resource "vault_policy" "approle_2" {
  name = "approle_2"

  policy = <<EOT
path "secret/data/test" {
  capabilities = ["read"]
}

path "auth/approle/role/test-role-2/secret-id/destroy" {
  capabilities = ["update"]
}

path "auth/approle/role/test-role-2/secret-id/lookup" {
  capabilities = ["update"]
}

path "auth/approle/role/test-role-2/secret-id" {
  capabilities = ["update", "list"]
}

path "auth/token/revoke-self" {
  capabilities = ["update"]
}

path "auth/token/lookup-self" {
  capabilities = ["update"]
}
EOT
}
