package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/soerenschneider/sc-agent/internal/config/vault"
)

func TestValidate_Vault(t *testing.T) {
	dir := t.TempDir()
	secretIdFile := filepath.Join(dir, "secret-id")
	if err := os.WriteFile(secretIdFile, []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	ageIdentityFile := filepath.Join(dir, "age.key")
	if err := os.WriteFile(ageIdentityFile, []byte("AGE-SECRET-KEY-1"), 0600); err != nil {
		t.Fatal(err)
	}
	missingFile := filepath.Join(dir, "missing")

	approle := func(mod func(v *vault.Vault)) vault.Vault {
		v := vault.Vault{
			Address:      "https://vault.example.com",
			AuthMethod:   "approle",
			RoleId:       "role",
			SecretIdFile: secretIdFile,
			MountApprole: "approle",
		}
		mod(&v)
		return v
	}

	tests := []struct {
		name    string
		vault   vault.Vault
		wantErr bool
	}{
		{
			name:  "approle",
			vault: approle(func(v *vault.Vault) {}),
		},
		{
			name:  "token",
			vault: vault.Vault{AuthMethod: "token", Token: "token"},
		},
		{
			name:    "token without token",
			vault:   vault.Vault{AuthMethod: "token"},
			wantErr: true,
		},
		{
			name:    "unknown auth method",
			vault:   approle(func(v *vault.Vault) { v.AuthMethod = "kubernetes" }),
			wantErr: true,
		},
		{
			name:    "invalid address",
			vault:   approle(func(v *vault.Vault) { v.Address = "vault.example.com" }),
			wantErr: true,
		},
		{
			name:    "approle without role_id",
			vault:   approle(func(v *vault.Vault) { v.RoleId = "" }),
			wantErr: true,
		},
		{
			name:    "missing secret_id file",
			vault:   approle(func(v *vault.Vault) { v.SecretIdFile = missingFile }),
			wantErr: true,
		},
		{
			name: "missing secret_id file with bootstrap",
			vault: approle(func(v *vault.Vault) {
				v.SecretIdFile = missingFile
				v.Bootstrap = &vault.VaultBootstrap{Address: "https://approle-courier.example.com", AgeIdentityFile: ageIdentityFile}
			}),
		},
		{
			name: "bootstrap with missing age identity",
			vault: approle(func(v *vault.Vault) {
				v.Bootstrap = &vault.VaultBootstrap{Address: "https://approle-courier.example.com", AgeIdentityFile: missingFile}
			}),
			wantErr: true,
		},
		{
			name: "bootstrap with invalid wrap_ttl",
			vault: approle(func(v *vault.Vault) {
				v.Bootstrap = &vault.VaultBootstrap{Address: "https://approle-courier.example.com", AgeIdentityFile: ageIdentityFile, WrapTtl: "2 minutes"}
			}),
			wantErr: true,
		},
		{
			name: "invalid cidr resolver",
			vault: approle(func(v *vault.Vault) {
				v.ApproleCidrLoginResolver = &vault.VaultApproleCidrResolver{Type: "magic"}
			}),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := Config{Vault: map[string]vault.Vault{"default": tt.vault}}
			err := Validate(conf)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
