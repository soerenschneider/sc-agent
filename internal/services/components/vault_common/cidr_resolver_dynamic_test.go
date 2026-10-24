package vault_common

import (
	"net"
	"reflect"
	"strings"
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

func TestHostCidr(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		want    string
		wantErr bool
	}{
		{name: "ipv4", ip: "192.168.1.10", want: "192.168.1.10/32"},
		{name: "ipv4-mapped ipv6", ip: "::ffff:192.168.1.10", want: "192.168.1.10/32"},
		{name: "ipv6", ip: "2001:db8::1", want: "2001:db8::1/128"},
		{name: "ipv4 loopback", ip: "127.0.0.1", wantErr: true},
		{name: "debian hostname loopback", ip: "127.0.1.1", wantErr: true},
		{name: "ipv6 loopback", ip: "::1", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := hostCidr(net.ParseIP(tt.ip))
			if (err != nil) != tt.wantErr {
				t.Fatalf("hostCidr() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("hostCidr() got = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDynamicCidrResolver_GetCidrRejectsLoopback(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	resolver, err := NewDynamicCidrResolver("http://" + listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	_, err = resolver.GetCidr()
	if err == nil || !strings.Contains(err.Error(), "loopback") {
		t.Fatalf("expected loopback address to be rejected, got %v", err)
	}
}
