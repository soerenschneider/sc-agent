resource "vault_mount" "pki" {
  path                      = "pki"
  type                      = "pki"
  default_lease_ttl_seconds = 3600
  max_lease_ttl_seconds     = 864000
}

resource "vault_pki_secret_backend_root_cert" "test" {
  depends_on           = [vault_mount.pki]
  backend              = vault_mount.pki.path
  type                 = "internal"
  common_name          = "Root CA"
  ttl                  = "3153600000"
  format               = "pem"
  private_key_format   = "der"
  key_type             = "rsa"
  key_bits             = 2048
  exclude_cn_from_sans = true
  ou                   = "My OU"
  organization         = "My organization"
}

resource "vault_pki_secret_backend_role" "role" {
  backend          = vault_mount.pki.path
  name             = "my_role"
  ttl              = 3600
  allow_ip_sans    = true
  key_type         = "rsa"
  key_bits         = 2048
  allowed_domains  = ["example.com", "my.domain"]
  allow_subdomains = true
}

resource "vault_policy" "pki_issue_policy" {
  name = "pki"

  policy = <<EOT
path "pki/*" {
  capabilities = ["create", "update", "read", "list", "delete"]
}

path "pki/issue/*" {
  capabilities = ["create", "update"]
}

path "pki/sign/*" {
  capabilities = ["create", "update"]
}

path "pki/roles/*" {
  capabilities = ["create", "update", "read", "delete", "list"]
}
EOT
}
