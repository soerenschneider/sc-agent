resource "vault_auth_backend" "approle" {
  type = "approle"
}

locals {
  filename = pathexpand("~/secretId.txt")
}

resource "local_file" "private_key" {
  content  = vault_approle_auth_backend_role_secret_id.id.wrapping_token
  filename = local.filename
}

resource "vault_approle_auth_backend_role" "example" {
  backend   = vault_auth_backend.approle.path
  role_name = "test-role"
  role_id   = "test-role"

  token_policies         = ["default", vault_policy.example.name, "prod", "ssh_client", "pki"]
  token_max_ttl          = 300
  token_explicit_max_ttl = 300
  token_ttl              = 300
  secret_id_ttl          = 1800
}

resource "vault_approle_auth_backend_role_secret_id" "id" {
  backend               = vault_auth_backend.approle.path
  role_name             = vault_approle_auth_backend_role.example.role_name
  with_wrapped_accessor = true
  wrapping_ttl          = "5m"

  metadata = jsonencode(
    {
      "hello"     = "world"
      "role_name" = "test-role"
    }
  )
}

resource "vault_policy" "example" {
  name = "dev"

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
