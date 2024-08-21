package main

import (
	"context"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/rs/zerolog/log"
	deps "github.com/soerenschneider/sc-agent/cmd/conditional_reboot"
	"github.com/soerenschneider/sc-agent/cmd/vault"
	"github.com/soerenschneider/sc-agent/internal"
	"github.com/soerenschneider/sc-agent/internal/config"
	"github.com/soerenschneider/sc-agent/internal/core/ports"
	"github.com/soerenschneider/sc-agent/internal/domain"
	"github.com/soerenschneider/sc-agent/internal/domain/http_replication"
	"github.com/soerenschneider/sc-agent/internal/services/components/conditional_reboot/app"
	"github.com/soerenschneider/sc-agent/internal/services/components/conditional_reboot/group"
	http_replication2 "github.com/soerenschneider/sc-agent/internal/services/components/http_replication"
	"github.com/soerenschneider/sc-agent/internal/services/components/libvirt"
	"github.com/soerenschneider/sc-agent/internal/services/components/packages"
	"github.com/soerenschneider/sc-agent/internal/services/components/release_watcher"
	"github.com/soerenschneider/sc-agent/internal/services/components/system"
	"github.com/soerenschneider/sc-agent/internal/services/components/systemd"
	"github.com/soerenschneider/sc-agent/internal/services/components/wol"
	"github.com/soerenschneider/sc-agent/internal/storage"
	"github.com/soerenschneider/sc-agent/pkg/reboot"
	"go.uber.org/multierr"
)

var httpClient = retryablehttp.NewClient().HTTPClient

//nolint:cyclop
func BuildDeps(conf config.Config) (*ports.Components, error) {
	ret := &ports.Components{}
	var err, errs error

	ret.Packages, err = buildPackages(conf)
	if err != nil {
		errs = multierr.Append(errs, err)
	}

	ret.PowerStatus, err = buildPowerstatus(conf)
	if err != nil {
		errs = multierr.Append(errs, err)
	}

	ret.Libvirt, err = buildLibvirt(conf)
	if err != nil {
		errs = multierr.Append(errs, err)
	}

	ret.Services, err = buildServices(conf)
	if err != nil {
		errs = multierr.Append(errs, err)
	}

	ret.ConditionalReboot, err = buildConditionalReboot(conf)
	if err != nil {
		errs = multierr.Append(errs, err)
	}

	ret.Wol, err = buildWol(conf)
	if err != nil {
		errs = multierr.Append(errs, err)
	}

	if strings.HasPrefix(internal.BuildVersion, "v") {
		ret.ReleaseWatcher, err = buildReleaseWatcher(conf)
		if err != nil {
			errs = multierr.Append(errs, err)
		}
	} else {
		log.Warn().Str("build_version", internal.BuildVersion).Msg("not building release watcher, no valid BuildVersion")
	}

	if err := vault.BuildVaultClients(conf); err != nil {
		errs = multierr.Append(errs, err)
	}

	if conf.SecretsReplication != nil && conf.SecretsReplication.Enabled {
		ret.SecretsReplication, err = vault.BuildSecretReplication(conf.SecretsReplication)
		if err != nil {
			errs = multierr.Append(errs, err)
		}
	}

	if conf.SshSigner != nil && conf.SshSigner.Enabled {
		ret.SshCertificates, err = vault.BuildSshService(*conf.SshSigner)
		if err != nil {
			errs = multierr.Append(errs, err)
		}
	}

	if conf.X509Pki != nil && conf.X509Pki.Enabled {
		ret.Pki, err = vault.BuildPkiService(*conf.X509Pki)
		if err != nil {
			errs = multierr.Append(errs, err)
		}
	}

	if conf.Acme != nil && conf.Acme.Enabled {
		ret.Acme, err = vault.BuildAcmeService(*conf.Acme)
		if err != nil {
			errs = multierr.Append(errs, err)
		}
	}

	if conf.HttpReplication != nil && conf.HttpReplication.Enabled {
		ret.HttpReplication, err = buildHttpReplication(*conf.HttpReplication)
		if err != nil {
			errs = multierr.Append(errs, err)
		} else {
			go ret.HttpReplication.StartReplication(context.Background())
		}
	}

	return ret, errs
}

func buildHttpReplication(conf config.HttpReplication) (*http_replication2.Service, error) {
	items := []http_replication.ReplicationItem{}
	for key, val := range conf.ReplicationItems {
		storage, err := buildCertStorage(val.Storage)
		if err != nil {
			return nil, err
		}
		postHooks := []domain.PostHook{}
		for key, hook := range val.PostHooks {
			postHooks = append(postHooks, domain.PostHook{
				Name: key,
				Cmd:  hook,
			})
		}
		items = append(items, http_replication.ReplicationItem{
			PostHooks: postHooks,
			ReplicationConf: http_replication.ReplicationConf{
				Id:          key,
				Source:      val.Source,
				Destination: val.Storage,
				Sha256Sum:   val.Sha256Sum,
			},
			Destination: storage,
		})
	}

	return http_replication2.New(httpClient, items)
}

func buildConditionalReboot(config config.Config) (ports.ConditionalReboot, error) {
	if config.ConditionalReboot == nil || !config.ConditionalReboot.Enabled {
		return nil, nil
	}

	groupUpdates := make(chan *group.Group, 1)

	groups, err := deps.BuildGroups(groupUpdates, config.ConditionalReboot)
	if err != nil {
		log.Fatal().Err(err).Msg("could not build groups")
	}

	rebootImpl := &reboot.DefaultRebootImpl{}

	var opts []app.ConditionalRebootOpts
	app, err := app.NewConditionalReboot(groups, rebootImpl, groupUpdates, opts...)
	if err != nil {
		return nil, err
	}

	go func() {
		if err := app.Start(); err != nil {
			log.Fatal().Err(err).Msgf("could not start conditional-reboot")
		}
	}()

	return app, nil
}

func buildPackages(conf config.Config) (ports.SystemPackages, error) {
	if conf.Packages == nil || !conf.Packages.Enabled {
		return nil, nil
	}
	return packages.NewDnf()
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

func buildCertStorage(storageConf string) (*storage.FilesystemStorage, error) {
	return storage.NewFilesystemStorageFromUri(storageConf)
}
