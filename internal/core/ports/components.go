package ports

import (
	"context"
	"reflect"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/cmd/vault"
	"github.com/soerenschneider/sc-agent/internal/config"
	"github.com/soerenschneider/sc-agent/internal/services/components/packages"
)

const (
	logComponent      = "component"
	mainComponentName = "main"
)

var (
	once              sync.Once
	enabledComponents []string
)

type Components struct {
	Acme               Acme
	RebootManager      RebootManager
	HttpReplication    HttpReplication
	K0s                K0s
	Libvirt            Libvirt
	Packages           SystemPackages
	Pki                X509Pki
	PowerStatus        SystemPowerStatus
	ReleaseWatcher     ReleaseWatcher
	SecretsReplication SecretsReplication
	Services           Systemd
	SshCertificates    SshPki
	Wol                WakeOnLan
}

func (s *Components) UsesVault() bool {
	return s.SshCertificates != nil || s.Pki != nil || s.SecretsReplication != nil || s.Acme != nil
}

func (s *Components) StartServices(ctx context.Context, conf config.Config) {
	if s.HttpReplication != nil {
		go s.HttpReplication.StartReplication(ctx)
	}

	if s.RebootManager != nil {
		go func() {
			_ = s.RebootManager.Start(ctx)
		}()
	}

	if s.ReleaseWatcher != nil {
		go func() {
			s.ReleaseWatcher.WatchReleases(ctx)
		}()
	}

	if s.Packages != nil {
		go func() {
			updatesAvailableChecker, _ := packages.NewUpdatesAvailableChecker(s.Packages)
			updatesAvailableChecker.Start(ctx)
		}()
	}

	if !s.UsesVault() {
		return
	}

	// Vault logins are not awaited here, a Vault instance that is not reachable must not keep the other components
	// from working. Each Vault based component is started as soon as its Vault client has logged in.
	vault.StartTokenRenewal(ctx)

	if s.SecretsReplication != nil {
		startAfterVaultLogin(ctx, conf.SecretsReplication.VaultId, "continuous secret syncer process", s.SecretsReplication.StartContinuousReplication)
	}
	if s.SshCertificates != nil {
		startAfterVaultLogin(ctx, conf.SshSigner.VaultId, "management of ssh certificates", s.SshCertificates.WatchCertificates)
	}
	if s.Pki != nil {
		startAfterVaultLogin(ctx, conf.X509Pki.VaultId, "management of x509 certificates", s.Pki.WatchCertificates)
	}
	if s.Acme != nil {
		startAfterVaultLogin(ctx, conf.Acme.VaultId, "management of acme certificates", s.Acme.WatchCertificates)
	}
}

func startAfterVaultLogin(ctx context.Context, vaultId string, name string, start func(ctx context.Context)) {
	go func() {
		log.Info().Str(logComponent, mainComponentName).Str("vault_client", vaultId).Msgf("waiting for vault login before starting %s", name)
		if err := vault.WaitForLogin(ctx, vaultId); err != nil {
			if ctx.Err() == nil {
				log.Error().Str(logComponent, mainComponentName).Str("vault_client", vaultId).Err(err).Msgf("not starting %s", name)
			}
			return
		}
		log.Info().Str(logComponent, mainComponentName).Str("vault_client", vaultId).Msgf("starting %s", name)
		start(ctx)
	}()
}

// HasOperationalComponents returns whether at least one component that does actual work has been built. The release
// watcher only reports on sc-agent itself and is therefore not taken into account.
func (s *Components) HasOperationalComponents() bool {
	for _, component := range s.EnabledComponents() {
		if component != "ReleaseWatcher" {
			return true
		}
	}
	return false
}

func (s *Components) EnabledComponents() []string {
	once.Do(func() {
		v := reflect.ValueOf(s).Elem() // Get the value of the pointer to the struct
		t := v.Type()

		for i := 0; i < v.NumField(); i++ {
			fieldValue := v.Field(i)
			if !fieldValue.IsNil() { // Check if the member is not nil
				enabledComponents = append(enabledComponents, t.Field(i).Name)
			}
		}
	})

	return enabledComponents
}
