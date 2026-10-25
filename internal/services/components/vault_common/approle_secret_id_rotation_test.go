package vault_common

import (
	"testing"

	vault_config "github.com/soerenschneider/sc-agent/internal/config/vault"
	"github.com/soerenschneider/sc-agent/pkg/vault"
	"github.com/spf13/afero"
)

// fakeApproleClient treats every secret_id as expired and records the role names it was called with.
type fakeApproleClient struct {
	roleNames []string
}

func (f *fakeApproleClient) DestroySecretId(roleName, secretId string, isAccessor bool) error {
	f.roleNames = append(f.roleNames, roleName)
	return nil
}

func (f *fakeApproleClient) Lookup(roleName, secretId string, isAccessor bool) (*vault.SecretIdInfo, error) {
	f.roleNames = append(f.roleNames, roleName)
	return nil, errAlreadyExpired
}

func (f *fakeApproleClient) GetSecretIdAccessors(roleName string) ([]string, error) {
	f.roleNames = append(f.roleNames, roleName)
	return nil, nil
}

func (f *fakeApproleClient) GenerateSecretId(roleName string) (string, error) {
	f.roleNames = append(f.roleNames, roleName)
	return "new-secret-id", nil
}

func (f *fakeApproleClient) ReadRoleId(roleName string) (string, error) {
	f.roleNames = append(f.roleNames, roleName)
	return "", nil
}

func TestConditionallyRotateSecretId_RoleName(t *testing.T) {
	tests := []struct {
		name         string
		conf         vault_config.Vault
		wantRoleName string
	}{
		{
			name:         "role_name configured",
			conf:         vault_config.Vault{RoleId: "3c5e8a0b-1f2d-4e6a-9b7c-8d0e1f2a3b4c", RoleName: "demo", SecretIdFile: "/secret-id"},
			wantRoleName: "demo",
		},
		{
			name:         "defaults to role_id",
			conf:         vault_config.Vault{RoleId: "demo", SecretIdFile: "/secret-id"},
			wantRoleName: "demo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeApproleClient{}
			rotator, err := NewApproleUpdater(client, "default", &tt.conf)
			if err != nil {
				t.Fatal(err)
			}
			rotator.fsImpl = afero.NewMemMapFs()
			if err := afero.WriteFile(rotator.fsImpl, tt.conf.SecretIdFile, []byte("old-secret-id"), 0600); err != nil {
				t.Fatal(err)
			}

			if err := rotator.ConditionallyRotateSecretId(tt.conf, false); err != nil {
				t.Fatalf("expected rotation to succeed, got %v", err)
			}

			if len(client.roleNames) != 3 {
				t.Fatalf("expected lookup, generate and destroy calls, got %v", client.roleNames)
			}
			for _, roleName := range client.roleNames {
				if roleName != tt.wantRoleName {
					t.Fatalf("expected role name %q, got %v", tt.wantRoleName, client.roleNames)
				}
			}
			data, err := afero.ReadFile(rotator.fsImpl, tt.conf.SecretIdFile)
			if err != nil || string(data) != "new-secret-id" {
				t.Fatalf("expected new secret_id to be written, got %q (%v)", data, err)
			}
		})
	}
}
