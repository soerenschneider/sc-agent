package ports

import (
	"context"
	"reflect"
	"sync"
	"time"

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
	ConditionalReboot  ConditionalReboot
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

func (s *Components) StartServices(ctx context.Context, conf config.Config, scAgentFatalErrors chan error) {
	if s.HttpReplication != nil {
		go s.HttpReplication.StartReplication(ctx)
	}

	if s.ConditionalReboot != nil {
		go func() {
			_ = s.ConditionalReboot.Start()
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

	vaultAuthReady := &sync.WaitGroup{}
	vaultLogins := len(conf.Vault)
	vaultAuthReady.Add(vaultLogins)
	vault.StartTokenRenewal(ctx, vaultAuthReady, scAgentFatalErrors)

	// wait on all members of the waitgroup but end forcefully after the timeout has passed
	vaultLoginWait := make(chan struct{})

	go func() {
		log.Info().Str(logComponent, mainComponentName).Msgf("Waiting for %d Vault logins to succeed...", vaultLogins)
		vaultAuthReady.Wait()
		close(vaultLoginWait)
	}()

	select {
	case <-vaultLoginWait:
		log.Info().Str(logComponent, mainComponentName).Msg("Vault login successful")
	case <-time.After(60 * time.Second):
		log.Error().Str(logComponent, mainComponentName).Msg("Vault login exceeded timeout")
	}

	go func() {
		vault.StartApproleSecretIdRotation(ctx)
		if s.SecretsReplication != nil {
			log.Info().Str(logComponent, mainComponentName).Msg("starting continuous secret syncer process")
			go s.SecretsReplication.StartContinuousReplication(ctx)
		}
		if s.SshCertificates != nil {
			log.Info().Str(logComponent, mainComponentName).Msg("starting management of ssh certificates")
			go s.SshCertificates.WatchCertificates(ctx)
		}
		if s.Pki != nil {
			log.Info().Str(logComponent, mainComponentName).Msg("starting management of x509 certificates")
			go s.Pki.WatchCertificates(ctx)
		}
		if s.Acme != nil {
			log.Info().Str(logComponent, mainComponentName).Msg("starting management of acme certificates")
			go s.Acme.WatchCertificates(ctx)
		}
	}()
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
