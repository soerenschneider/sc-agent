locals {
  ssh_role_name = "default"
}

resource "vault_mount" "ssh" {
  type = "ssh"
  path = "ssh"
}

resource "vault_ssh_secret_backend_ca" "ca" {
  backend              = vault_mount.ssh.path
  generate_signing_key = true
}

resource "vault_ssh_secret_backend_role" "roles" {
  name                    = "default"
  backend                 = vault_mount.ssh.path
  key_type                = "ca"
  allow_user_certificates = true
  allow_host_certificates = false
  ttl                     = "24h"
  max_ttl                 = "24h"
  allowed_users           = "soeren"
  default_user            = "soeren"

  allowed_user_key_config {
    type    = "rsa"
    lengths = [3072, 4096]
  }

  allowed_user_key_config {
    type    = "ed25519"
    lengths = [0]
  }
}

resource "vault_policy" "client_sign" {
  name = "ssh_client"

  policy = <<EOT
path "ssh/sign/${local.ssh_role_name}" {
  capabilities = ["update"]
}
EOT
}
