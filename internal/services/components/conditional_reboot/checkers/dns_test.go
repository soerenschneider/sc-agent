package checkers

import (
	"context"
	"testing"
)

func TestDnsChecker_IsHealthy(t *testing.T) {

	tests := []struct {
		name    string
		host    string
		want    bool
		wantErr bool
	}{
		{
			host:    "google.com",
			want:    true,
			wantErr: false,
		},
		{
			host:    "google.commmm",
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &DnsChecker{
				host: tt.host,
			}
			got, err := c.IsHealthy(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("IsHealthy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IsHealthy() got = %v, want %v", got, tt.want)
			}
		})
	}
}
