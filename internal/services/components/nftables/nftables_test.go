package nftables

import (
	"errors"
	"testing"

	"github.com/soerenschneider/sc-agent/internal/config"
	"github.com/soerenschneider/sc-agent/internal/core/ports"
	"github.com/soerenschneider/sc-agent/internal/domain"
)

func TestParseElement(t *testing.T) {
	tests := []struct {
		element string
		want    string
		wantErr bool
	}{
		{element: "192.0.2.10", want: "192.0.2.10"},
		{element: " 192.0.2.10 ", want: "192.0.2.10"},
		{element: "2001:db8::1", want: "2001:db8::1"},
		{element: "192.0.2.10/24", want: "192.0.2.0/24"},
		{element: "2001:db8::/32", want: "2001:db8::/32"},
		{element: "fe80::1%eth0", wantErr: true},
		{element: "192.0.2.10 }; flush ruleset; add element inet filter x { 1.1.1.1", wantErr: true},
		{element: "example.com", wantErr: true},
		{element: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.element, func(t *testing.T) {
			got, err := parseElement(tt.element)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseElement() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !errors.Is(err, domain.ErrNftablesInvalidElement) {
				t.Fatalf("parseElement() error = %v, want ErrNftablesInvalidElement", err)
			}
			if got != tt.want {
				t.Errorf("parseElement() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddSetElementValidation(t *testing.T) {
	svc, err := New(config.Nftables{
		Enabled: true,
		Sets: map[string]config.NftablesSet{
			"allowlist": {Family: "inet", Table: "filter", Set: "allowed", MaxTimeout: "1h"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		req     ports.NftablesAddSetElementRequest
		wantErr error
	}{
		{
			name:    "unknown set",
			req:     ports.NftablesAddSetElementRequest{Set: "unknown", Element: "192.0.2.10", Timeout: "1m"},
			wantErr: domain.ErrNftablesNoSuchSet,
		},
		{
			name:    "invalid element",
			req:     ports.NftablesAddSetElementRequest{Set: "allowlist", Element: "invalid", Timeout: "1m"},
			wantErr: domain.ErrNftablesInvalidElement,
		},
		{
			name:    "malformed timeout",
			req:     ports.NftablesAddSetElementRequest{Set: "allowlist", Element: "192.0.2.10", Timeout: "1 hour"},
			wantErr: domain.ErrNftablesInvalidTimeout,
		},
		{
			name:    "timeout too short",
			req:     ports.NftablesAddSetElementRequest{Set: "allowlist", Element: "192.0.2.10", Timeout: "1ms"},
			wantErr: domain.ErrNftablesInvalidTimeout,
		},
		{
			name:    "timeout exceeds max",
			req:     ports.NftablesAddSetElementRequest{Set: "allowlist", Element: "192.0.2.10", Timeout: "2h"},
			wantErr: domain.ErrNftablesInvalidTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.AddSetElement(tt.req)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("AddSetElement() error = %v, want %v", err, tt.wantErr)
			}
			if !errors.Is(err, domain.ErrNotFound) && !errors.Is(err, domain.ErrInvalidInput) {
				t.Errorf("AddSetElement() error = %v, want error caused by caller", err)
			}
		})
	}
}
