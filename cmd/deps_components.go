package main

import (
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/rs/zerolog/log"
	deps "github.com/soerenschneider/sc-agent/cmd/conditional_reboot"
	"github.com/soerenschneider/sc-agent/cmd/vault"
	"github.com/soerenschneider/sc-agent/internal"
	"github.com/soerenschneider/sc-agent/internal/config"
	ports2 "github.com/soerenschneider/sc-agent/internal/core/ports"
	"github.com/soerenschneider/sc-agent/internal/services/components/conditional-reboot/app"
	"github.com/soerenschneider/sc-agent/internal/services/components/conditional-reboot/group"
	"github.com/soerenschneider/sc-agent/internal/services/components/libvirt"
	"github.com/soerenschneider/sc-agent/internal/services/components/packages"
	"github.com/soerenschneider/sc-agent/internal/services/components/release_watcher"
	"github.com/soerenschneider/sc-agent/internal/services/components/system"
	"github.com/soerenschneider/sc-agent/internal/services/components/systemd"
	"github.com/soerenschneider/sc-agent/internal/services/components/wol"
	"github.com/soerenschneider/sc-agent/pkg/reboot"
	"go.uber.org/multierr"
)

func BuildDeps(conf config.Config) (*ports2.Services, error) {
	ret := &ports2.Services{}
	var err, errs error

	ret.Packages, err = buildDnf(conf)
	if err != nil {
		errs = multierr.Append(errs, err)
	}

	ret.Machine, err = buildMachine(conf)
	if err != nil {
		errs = multierr.Append(errs, err)
	}

	ret.Libvirt, err = buildLibvirt(conf)
	if err != nil {
		errs = multierr.Append(errs, err)
	}

	ret.Systemd, err = buildSystemd(conf)
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

	if err := vault.BuildVaultClients(conf); err != nil {
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

	if conf.SecretsReplication != nil && conf.SecretsReplication.Enabled {
		ret.SecretSyncer, err = vault.BuildSecretReplication(conf.SecretsReplication)
		if err != nil {
			errs = multierr.Append(errs, err)
		}
	}

	if conf.SshPki != nil && conf.SshPki.Enabled {
		ret.SshSigner, err = vault.BuildSshService(*conf.SshPki)
		if err != nil {
			errs = multierr.Append(errs, err)
		}
	}

	if conf.X509Pki != nil && conf.X509Pki.Enabled {
		ret.Pki, err = vault.BuildX509Service(*conf.X509Pki)
		if err != nil {
			errs = multierr.Append(errs, err)
		}
	}

	return ret, errs
}

func buildConditionalReboot(config config.Config) (ports2.ConditionalReboot, error) {
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

func buildDnf(conf config.Config) (ports2.SystemPackages, error) {
	if conf.Packages == nil || !conf.Packages.Enabled {
		return nil, nil
	}
	return packages.NewDnf()
}

func buildMachine(conf config.Config) (ports2.SystemPowerStatus, error) {
	if conf.PowerStatus == nil || !conf.PowerStatus.Enabled {
		return nil, nil
	}

	return system.New(*conf.PowerStatus)
}

func buildSystemd(conf config.Config) (ports2.Systemd, error) {
	if conf.Services == nil || !conf.Services.Enabled {
		return nil, nil
	}

	return systemd.New(*conf.Services)
}

func buildLibvirt(conf config.Config) (ports2.Libvirt, error) {
	if conf.Libvirt == nil || !conf.Libvirt.Enabled {
		return nil, nil
	}

	return libvirt.New(*conf.Libvirt)
}

func buildWol(conf config.Config) (ports2.WakeOnLan, error) {
	if conf.Wol == nil || !conf.Wol.Enabled {
		return nil, nil
	}

	return wol.New(*conf.Wol)
}

func buildReleaseWatcher(conf config.Config) (*release_watcher.ReleaseWatcher, error) {
	client := retryablehttp.NewClient().HTTPClient
	return release_watcher.New(client, internal.BuildVersion)
}
