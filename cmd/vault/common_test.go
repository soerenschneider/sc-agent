package vault

import (
	"reflect"
	"testing"

	vault_config "github.com/soerenschneider/sc-agent/internal/config/vault"
	"gopkg.in/yaml.v3"
)

func TestStaticCidrs(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		want    []string
		wantErr bool
	}{
		{
			name: "list of cidrs",
			yaml: "type: static\nargs:\n  - 192.168.0.0/24\n  - 10.0.0.1/32\n",
			want: []string{"192.168.0.0/24", "10.0.0.1/32"},
		},
		{
			name:    "no list",
			yaml:    "type: static\nargs: 192.168.0.0/24\n",
			wantErr: true,
		},
		{
			name:    "no strings",
			yaml:    "type: static\nargs:\n  - 42\n",
			wantErr: true,
		},
		{
			name:    "missing args",
			yaml:    "type: static\n",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resolver vault_config.VaultApproleCidrResolver
			if err := yaml.Unmarshal([]byte(tt.yaml), &resolver); err != nil {
				t.Fatal(err)
			}
			got, err := staticCidrs(resolver.Args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("staticCidrs() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("staticCidrs() got = %v, want %v", got, tt.want)
			}
		})
	}
}
