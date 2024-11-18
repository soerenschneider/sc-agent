resource "vault_auth_backend" "approle" {
  type = "approle"
}

locals {
  secret_token_file_1 = pathexpand("~/secret_id_1.txt")
}

resource "local_file" "secret_id_token_1" {
  content  = vault_approle_auth_backend_role_secret_id.role_1.wrapping_token
  filename = local.secret_token_file_1
}

resource "vault_approle_auth_backend_role" "role_1" {
  backend   = vault_auth_backend.approle.path
  role_name = "test-role"
  role_id   = "test-role"

  token_policies         = ["default", vault_policy.approle_1.name, "prod", "ssh_client", "pki"]
  token_max_ttl          = 300
  token_explicit_max_ttl = 300
  token_ttl              = 300
  secret_id_ttl          = 1800
}

resource "vault_approle_auth_backend_role_secret_id" "role_1" {
  backend               = vault_auth_backend.approle.path
  role_name             = vault_approle_auth_backend_role.role_1.role_name
  with_wrapped_accessor = true
  wrapping_ttl          = "50m"

  metadata = jsonencode(
    {
      "hello"     = "world"
      "role_name" = "test-role"
    }
  )
}

resource "vault_policy" "approle_1" {
  name = "approle_1"

  policy = <<EOT
path "secret/data/test" {
  capabilities = ["read"]
}

path "auth/approle/role/test-role/secret-id/destroy" {
  capabilities = ["update"]
}

path "auth/approle/role/test-role/secret-id/lookup" {
  capabilities = ["update"]
}

path "auth/approle/role/test-role/secret-id" {
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
