package vault_common

import (
	"errors"
	"reflect"
	"testing"

	"github.com/hashicorp/vault/api"
)

type fakeVaultClient struct {
	listResp   map[string]*api.Secret
	listPaths  []string
	writePaths []string
}

func (f *fakeVaultClient) List(path string) (*api.Secret, error) {
	f.listPaths = append(f.listPaths, path)
	return f.listResp[path], nil
}

func (f *fakeVaultClient) Read(path string) (*api.Secret, error) {
	return nil, nil
}

func (f *fakeVaultClient) Write(path string, data map[string]any) (*api.Secret, error) {
	f.writePaths = append(f.writePaths, path)
	return &api.Secret{Data: map[string]any{"secret_id": "new-secret-id"}}, nil
}

type fakeCidrResolver struct {
	cidrs []string
	err   error
}

func (f *fakeCidrResolver) GetCidr() ([]string, error) {
	return f.cidrs, f.err
}

func TestApproleSecretIdRotatorClient_GenerateSecretIdFailsClosed(t *testing.T) {
	failing := &fakeCidrResolver{err: errors.New("local address 127.0.0.1 is a loopback address")}
	working := &fakeCidrResolver{cidrs: []string{"192.168.1.10/32"}}

	tests := []struct {
		name          string
		cidrList      CidrResolver
		tokenCidrList CidrResolver
		wantErr       bool
	}{
		{name: "no resolvers"},
		{name: "working resolvers", cidrList: working, tokenCidrList: working},
		{name: "cidr list resolver fails", cidrList: failing, tokenCidrList: working, wantErr: true},
		{name: "token cidr list resolver fails", cidrList: working, tokenCidrList: failing, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vaultClient := &fakeVaultClient{}
			client, err := NewClient(vaultClient, "approle")
			if err != nil {
				t.Fatal(err)
			}
			client.cidrListResolver = tt.cidrList
			client.tokenCidrResolver = tt.tokenCidrList

			secretId, err := client.GenerateSecretId("restic")
			if (err != nil) != tt.wantErr {
				t.Fatalf("GenerateSecretId() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if len(vaultClient.writePaths) != 0 {
					t.Fatalf("expected no secret_id to be created, got writes to %v", vaultClient.writePaths)
				}
				return
			}
			if secretId != "new-secret-id" {
				t.Fatalf("unexpected secret_id %q", secretId)
			}
		})
	}
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
