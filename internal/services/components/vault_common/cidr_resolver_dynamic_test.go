package vault_common

import (
	"reflect"
	"testing"
)

func TestNewDynamicCidrResolver(t *testing.T) {
	type args struct {
		vaultAddr string
	}
	tests := []struct {
		name    string
		args    args
		want    *DynamicCidrResolver
		wantErr bool
	}{
		{
			name:    "https without port",
			args:    args{vaultAddr: "https://my-vault-instance"},
			want:    &DynamicCidrResolver{vaultAddress: "my-vault-instance:443"},
			wantErr: false,
		},
		{
			name:    "https with port",
			args:    args{vaultAddr: "https://my-vault-instance:8200"},
			want:    &DynamicCidrResolver{vaultAddress: "my-vault-instance:8200"},
			wantErr: false,
		},
		{
			name:    "http without port",
			args:    args{vaultAddr: "http://my-vault-instance"},
			want:    &DynamicCidrResolver{vaultAddress: "my-vault-instance:80"},
			wantErr: false,
		},
		{
			name:    "http with port",
			args:    args{vaultAddr: "http://my-vault-instance:8200"},
			want:    &DynamicCidrResolver{vaultAddress: "my-vault-instance:8200"},
			wantErr: false,
		},
		{
			name:    "invalid input",
			args:    args{vaultAddr: "my-vault-instance:8200"},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewDynamicCidrResolver(tt.args.vaultAddr)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDynamicCidrResolver() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewDynamicCidrResolver() got = %v, want %v", got, tt.want)
			}
		})
	}
}
