package vault_common

import (
	"reflect"
	"testing"

	"github.com/hashicorp/vault/api"
)

type fakeVaultClient struct {
	listResp  map[string]*api.Secret
	listPaths []string
}

func (f *fakeVaultClient) List(path string) (*api.Secret, error) {
	f.listPaths = append(f.listPaths, path)
	return f.listResp[path], nil
}

func (f *fakeVaultClient) Read(path string) (*api.Secret, error) {
	return nil, nil
}

func (f *fakeVaultClient) Write(path string, data map[string]any) (*api.Secret, error) {
	return nil, nil
}

func TestApproleSecretIdRotatorClient_GetSecretIdAccessors(t *testing.T) {
	tests := []struct {
		name     string
		mount    string
		listResp map[string]*api.Secret
		want     []string
	}{
		{
			name:  "custom mount",
			mount: "approle-apps",
			listResp: map[string]*api.Secret{
				"auth/approle-apps/role/restic/secret-id": {Data: map[string]any{"keys": []any{"accessor-1", "accessor-2"}}},
			},
			want: []string{"accessor-1", "accessor-2"},
		},
		{
			name:  "no secret_ids",
			mount: "approle",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vaultClient := &fakeVaultClient{listResp: tt.listResp}
			client, err := NewClient(vaultClient, tt.mount)
			if err != nil {
				t.Fatal(err)
			}

			got, err := client.GetSecretIdAccessors("restic")
			if err != nil {
				t.Fatalf("GetSecretIdAccessors() unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("GetSecretIdAccessors() got = %v, want %v", got, tt.want)
			}
			wantPath := "auth/" + tt.mount + "/role/restic/secret-id"
			if len(vaultClient.listPaths) != 1 || vaultClient.listPaths[0] != wantPath {
				t.Fatalf("expected list on %q, got %v", wantPath, vaultClient.listPaths)
			}
		})
	}
}
