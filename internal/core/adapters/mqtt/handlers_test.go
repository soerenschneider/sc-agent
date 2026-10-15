package mqtt

import (
	"context"
	"errors"
	"testing"

	"github.com/soerenschneider/sc-agent/internal/core/ports"
	"github.com/soerenschneider/sc-agent/internal/domain"
	"github.com/soerenschneider/sc-agent/internal/domain/system"
)

type fakePower struct {
	rebooted, shutdown bool
}

func (f *fakePower) SetCpuGovernor(_ system.Governor) error { return nil }
func (f *fakePower) GetCurrentCpuGovernor() (string, error) { return "", nil }
func (f *fakePower) Shutdown() error                        { f.shutdown = true; return nil }
func (f *fakePower) Reboot() error                          { f.rebooted = true; return nil }

type fakeNftables struct {
	req ports.NftablesAddSetElementRequest
	err error
}

func (f *fakeNftables) AddSetElement(req ports.NftablesAddSetElementRequest) error {
	f.req = req
	return f.err
}

func TestPowerStatusHandler(t *testing.T) {
	power := &fakePower{}
	h := &PowerStatusHandler{services: &ports.Components{PowerStatus: power}}

	if _, err := h.Handle(context.Background(), "sc-agent/host/power/reboot", nil); err != nil {
		t.Fatalf("reboot: unexpected error %v", err)
	}
	if _, err := h.Handle(context.Background(), "sc-agent/host/power/shutdown", nil); err != nil {
		t.Fatalf("shutdown: unexpected error %v", err)
	}
	if !power.rebooted || !power.shutdown {
		t.Errorf("expected reboot and shutdown to be called, got %+v", power)
	}

	if _, err := h.Handle(context.Background(), "sc-agent/host/power/unknown", nil); !errors.Is(err, domain.ErrNotImplemented) {
		t.Errorf("unknown verb: got %v, want ErrNotImplemented", err)
	}

	disabled := &PowerStatusHandler{services: &ports.Components{}}
	if _, err := disabled.Handle(context.Background(), "sc-agent/host/power/reboot", nil); !errors.Is(err, domain.ErrComponentDisabled) {
		t.Errorf("disabled: got %v, want ErrComponentDisabled", err)
	}
}

func TestNftablesHandler(t *testing.T) {
	nft := &fakeNftables{}
	h := &NftablesHandler{services: &ports.Components{Nftables: nft}}

	payload := []byte(`{"set": "ssh-allowlist", "element": "192.0.2.10", "timeout": "1h"}`)
	if _, err := h.Handle(context.Background(), "sc-agent/host/nftables/add", payload); err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	want := ports.NftablesAddSetElementRequest{Set: "ssh-allowlist", Element: "192.0.2.10", Timeout: "1h"}
	if nft.req != want {
		t.Errorf("got request %+v, want %+v", nft.req, want)
	}

	if _, err := h.Handle(context.Background(), "sc-agent/host/nftables/add", []byte("not json")); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("invalid payload: got %v, want ErrInvalidInput", err)
	}

	nft.err = domain.ErrNftablesNoSuchSet
	if _, err := h.Handle(context.Background(), "sc-agent/host/nftables/add", payload); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("unknown set: got %v, want ErrNotFound", err)
	}

	disabled := &NftablesHandler{services: &ports.Components{}}
	if _, err := disabled.Handle(context.Background(), "sc-agent/host/nftables/add", payload); !errors.Is(err, domain.ErrComponentDisabled) {
		t.Errorf("disabled: got %v, want ErrComponentDisabled", err)
	}
}
