package main

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/rs/zerolog/log"
	deps "github.com/soerenschneider/sc-agent/cmd/reboot_manager"
	"github.com/soerenschneider/sc-agent/cmd/vault"
	"github.com/soerenschneider/sc-agent/internal"
	"github.com/soerenschneider/sc-agent/internal/config"
	"github.com/soerenschneider/sc-agent/internal/core/ports"
	"github.com/soerenschneider/sc-agent/internal/domain"
	"github.com/soerenschneider/sc-agent/internal/domain/http_replication"
	"github.com/soerenschneider/sc-agent/internal/events"
	"github.com/soerenschneider/sc-agent/internal/metrics"
	http_replication_svc "github.com/soerenschneider/sc-agent/internal/services/components/http_replication"
	"github.com/soerenschneider/sc-agent/internal/services/components/libvirt"
	"github.com/soerenschneider/sc-agent/internal/services/components/packages"
	"github.com/soerenschneider/sc-agent/internal/services/components/reboot_manager/app"
	"github.com/soerenschneider/sc-agent/internal/services/components/reboot_manager/group"
	"github.com/soerenschneider/sc-agent/internal/services/components/release_watcher"
	"github.com/soerenschneider/sc-agent/internal/services/components/system"
	"github.com/soerenschneider/sc-agent/internal/services/components/systemd"
	"github.com/soerenschneider/sc-agent/internal/services/components/wol"
	"github.com/soerenschneider/sc-agent/internal/storage"
	"github.com/soerenschneider/sc-agent/internal/sysinfo"
	"github.com/soerenschneider/sc-agent/pkg/reboot"
)

var httpClient = retryablehttp.NewClient().HTTPClient

// BuildDeps builds all configured components. Components that can not be built are not fatal: they are left
// disabled, flagged as degraded via metrics and the remaining components keep working.
//
//nolint:cyclop
func BuildDeps(conf config.Config) *ports.Components {
	ret := &ports.Components{}

	if sink, err := buildEventSink(conf, ret); err != nil {
		degrade("event_sink", err)
	} else {
		events.ConfiguredEventSink = sink
	}

	if pkgs, err := buildPackages(conf); err != nil {
		degrade("packages", err)
	} else {
		ret.Packages = pkgs
	}

	if powerStatus, err := buildPowerstatus(conf); err != nil {
		degrade("power_status", err)
	} else {
		ret.PowerStatus = powerStatus
	}

	if libvirtSvc, err := buildLibvirt(conf); err != nil {
		degrade("libvirt", err)
	} else {
		ret.Libvirt = libvirtSvc
	}

	if services, err := buildServices(conf); err != nil {
		degrade("services", err)
	} else {
		ret.Services = services
	}

	if rebootManager, err := buildRebootManager(conf); err != nil {
		degrade("reboot_manager", err)
	} else {
		ret.RebootManager = rebootManager
	}

	if wolSvc, err := buildWol(conf); err != nil {
		degrade("wol", err)
	} else {
		ret.Wol = wolSvc
	}

	if strings.HasPrefix(internal.BuildVersion, "v") {
		if releaseWatcher, err := buildReleaseWatcher(conf); err != nil {
			degrade("release_watcher", err)
		} else {
			ret.ReleaseWatcher = releaseWatcher
		}
	} else {
		log.Warn().Str("build_version", internal.BuildVersion).Msg("not building release watcher, no valid BuildVersion")
	}

	// failing vault clients are flagged as degraded by the vault package itself
	vault.BuildVaultClients(conf)

	if conf.SecretsReplication != nil && conf.SecretsReplication.Enabled {
		if svc, err := vault.BuildSecretReplication(conf.SecretsReplication); err != nil {
			degrade("secrets_replication", err)
		} else {
			ret.SecretsReplication = svc
		}
	}

	if conf.SshSigner != nil && conf.SshSigner.Enabled {
		if svc, err := vault.BuildSshService(*conf.SshSigner); err != nil {
			degrade("ssh_certificates", err)
		} else {
			ret.SshCertificates = svc
		}
	}

	if conf.X509Pki != nil && conf.X509Pki.Enabled {
		if svc, err := vault.BuildPkiService(*conf.X509Pki); err != nil {
			degrade("pki", err)
		} else {
			ret.Pki = svc
		}
	}

	if conf.Acme != nil && conf.Acme.Enabled {
		if svc, err := vault.BuildAcmeService(*conf.Acme); err != nil {
			degrade("acme", err)
		} else {
			ret.Acme = svc
		}
	}

	if conf.HttpReplication != nil && conf.HttpReplication.Enabled {
		if svc, err := buildHttpReplication(*conf.HttpReplication); err != nil {
			degrade("http_replication", err)
		} else {
			ret.HttpReplication = svc
		}
	}

	return ret
}

func degrade(component string, err error) {
	log.Error().Str(logComponent, mainComponentName).Str("degraded_component", component).Err(err).Msg("could not build component, running in degraded mode")
	metrics.SetComponentDegraded(component)
}

func buildHttpReplication(conf config.HttpReplication) (*http_replication_svc.Service, error) {
	items := make([]http_replication.ReplicationItem, 0, len(conf.ReplicationItems))

	for key, val := range conf.ReplicationItems {
		destStorage, err := buildCertStorage(val.Destinations)
		if err != nil {
			return nil, err
		}
		postHooks := make([]domain.PostHook, 0, len(val.PostHooks))
		for key, hook := range val.PostHooks {
			postHooks = append(postHooks, domain.PostHook{
				Name: key,
				Cmd:  hook,
			})
		}

		var fileValidationConf *http_replication.FileValidation
		if val.Validation != nil {
			fileValidationConf = &http_replication.FileValidation{
				Test:         val.Validation.Test,
				Arg:          val.Validation.Arg,
				InvertResult: val.Validation.InvertResult,
			}
		}

		items = append(items, http_replication.ReplicationItem{
			PostHooks: postHooks,
			ReplicationConf: http_replication.ReplicationConf{
				Id:             key,
				Source:         val.Source,
				Destinations:   val.Destinations,
				FileValidation: fileValidationConf,
			},
			Destination: destStorage,
		})
	}

	return http_replication_svc.New(httpClient, items)
}

func buildRebootManager(config config.Config) (ports.RebootManager, error) {
	if config.RebootManager == nil || !config.RebootManager.Enabled {
		return nil, nil
	}

	groupUpdates := make(chan *group.Group, 1)

	groups, err := deps.BuildGroups(groupUpdates, config.RebootManager)
	if err != nil {
		return nil, fmt.Errorf("could not build groups: %w", err)
	}

	rebootImpl := &reboot.DefaultRebootImpl{}

	var opts []app.RebootManagerOpts
	if config.RebootManager.DryRun {
		opts = append(opts, app.DryRun())
	}
	app, err := app.NewRebootManager(groups, rebootImpl, groupUpdates, opts...)
	if err != nil {
		return nil, err
	}

	return app, nil
}

func buildPackages(conf config.Config) (ports.SystemPackages, error) {
	if conf.Packages == nil || !conf.Packages.Enabled {
		return nil, nil
	}

	if sysinfo.Sysinfo.IsDebian() {
		return packages.NewAptPackageManager()
	}

	if sysinfo.Sysinfo.IsRedHat() {
		return packages.NewDnfPackageManager()
	}

	return nil, fmt.Errorf("unknown/unsupported system: %v", sysinfo.Sysinfo.OS)
}

func buildPowerstatus(conf config.Config) (ports.SystemPowerStatus, error) {
	if conf.PowerStatus == nil || !conf.PowerStatus.Enabled {
		return nil, nil
	}

	return system.New(*conf.PowerStatus)
}

func buildServices(conf config.Config) (ports.Systemd, error) {
	if conf.Services == nil || !conf.Services.Enabled {
		return nil, nil
	}

	return systemd.New(*conf.Services)
}

func buildLibvirt(conf config.Config) (ports.Libvirt, error) {
	if conf.Libvirt == nil || !conf.Libvirt.Enabled {
		return nil, nil
	}

	return libvirt.New(*conf.Libvirt)
}

func buildWol(conf config.Config) (ports.WakeOnLan, error) {
	if conf.Wol == nil || !conf.Wol.Enabled {
		return nil, nil
	}

	return wol.New(*conf.Wol)
}

func buildReleaseWatcher(conf config.Config) (*release_watcher.ReleaseWatcher, error) {
	return release_watcher.New(httpClient, internal.BuildVersion)
}

func buildCertStorage(storageConf []string) (http_replication_svc.StorageImplementation, error) {
	return storage.NewMultiFilesystemStorage(storageConf...)
}
