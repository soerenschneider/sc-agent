package checkers

import (
	"context"
	"testing"
)

func TestTcpChecker_IsHealthy(t *testing.T) {
	type fields struct {
		host string
		port string
	}

	tests := []struct {
		name    string
		fields  fields
		want    bool
		wantErr bool
	}{
		{
			name: "google dns",
			fields: fields{
				host: "8.8.8.8",
				port: "53",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "google dns",
			fields: fields{
				host: "8.8.8.8",
				port: "54",
			},
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &TcpChecker{
				host: tt.fields.host,
				port: tt.fields.port,
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
