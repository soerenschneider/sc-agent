package vault

import (
	"context"
	"errors"
	"sync"

	vault "github.com/hashicorp/vault/api"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/config"
	vault_config "github.com/soerenschneider/sc-agent/internal/config/vault"
	"github.com/soerenschneider/sc-agent/internal/services/components/vault_common"
	"github.com/soerenschneider/sc-agent/internal/services/components/vault_common/auth"
	pkg_vault "github.com/soerenschneider/sc-agent/pkg/vault"
	"go.uber.org/multierr"
)

var (
	clients = map[string]*vault_common.VaultCommon{}
	mutex   sync.Mutex
)

func getVaultClient(key string) *vault_common.VaultCommon {
	return clients[key]
}

func BuildVaultClients(conf config.Config) error {
	var errs error
	for clientId, vaultConf := range conf.Vault {
		if err := buildVaultClient(clientId, vaultConf); err != nil {
			errs = multierr.Append(errs, err)
		}
	}

	return errs
}

func getOpts(conf vault_config.Vault) ([]vault_common.ApproleSecretIdRotationOpts, error) {
	var opts []vault_common.ApproleSecretIdRotationOpts
	if conf.ApproleCidrLoginResolver != nil {
		if conf.ApproleCidrLoginResolver.Type == "static" {
			cidrs, ok := conf.ApproleCidrLoginResolver.Args.([]string)
			if !ok {
				return nil, errors.New("can not cast ApproleCidrLoginResolver.Args to []string")
			}
			opts = append(opts, vault_common.WithStaticCidrResolver(cidrs))
		}
		if conf.ApproleCidrLoginResolver.Type == "dynamic" {
			opts = append(opts, vault_common.WithDynamicCidrResolver(conf.Address))
		}
	}

	if conf.ApproleCidrTokenResolver != nil {
		if conf.ApproleCidrTokenResolver.Type == "static" {
			cidrs, ok := conf.ApproleCidrTokenResolver.Args.([]string)
			if !ok {
				return nil, errors.New("can not cast ApproleCidrTokenResolver.Args to []string")
			}
			opts = append(opts, vault_common.WithStaticCidrTokenResolver(cidrs))
		}
		if conf.ApproleCidrTokenResolver.Type == "dynamic" {
			opts = append(opts, vault_common.WithDynamicCidrTokenResolver(conf.Address))
		}
	}

	return opts, nil
}

func buildVaultClient(key string, conf vault_config.Vault) error {
	mutex.Lock()
	defer mutex.Unlock()

	_, found := clients[key]
	if found {
		return nil
	}

	vaultConf := vault.DefaultConfig()
	vaultConf.Address = conf.Address
	vaultConf.MaxRetries = 5

	auth, err := buildVaultAuth(conf)
	if err != nil {
		return err
	}

	vaultClient, err := vault.NewClient(vaultConf)
	if err != nil {
		return err
	}

	tokenRenewer, err := vault_common.NewTokenRenewer(vaultClient, auth, key)
	if err != nil {
		return err
	}

	var secretIdRotator *vault_common.ApproleSecretIdRotatorService
	if conf.AuthMethod == "approle" {
		opts, err := getOpts(conf)
		if err != nil {
			return err
		}
		approleClient, err := vault_common.NewClient(vaultClient.Logical(), conf.MountApprole, opts...)
		if err != nil {
			return err
		}
		secretIdRotator, err = vault_common.NewApproleUpdater(approleClient, &conf)
		if err != nil {
			return err
		}
	}

	client, err := vault_common.NewVaultClient(auth, vaultClient, tokenRenewer, secretIdRotator)
	if err != nil {
		return err
	}

	clients[key] = client
	return nil
}

func StartTokenRenewal(ctx context.Context, wg *sync.WaitGroup, vaultFatalError chan error) {
	for key := range clients {
		client := clients[key]
		go client.StartTokenRenewer(ctx, wg, vaultFatalError)
	}
}

func StartApproleSecretIdRotation(ctx context.Context) {
	for key := range clients {
		client := clients[key]
		go client.StartApproleSecretIdRotation(ctx)
	}
}

func buildVaultAuth(conf vault_config.Vault) (vault.AuthMethod, error) {
	switch conf.AuthMethod {
	case "token":
		return auth.NewTokenAuth(conf.Token)

	case "approle":
		secretId := &auth.SecretID{
			FromFile: conf.SecretIdFile,
		}

		var loginOpts []auth.LoginOption
		if conf.MountApprole != "" {
			loginOpts = append(loginOpts, auth.WithMountPath(conf.MountApprole))
		}

		wrappedToken := pkg_vault.IsWrappedToken(conf.SecretIdFile)
		if wrappedToken {
			log.Info().Msg("Trying to authenticate using wrapped secret_id token")
			loginOpts = append(loginOpts, auth.WithWrappingToken())
		}

		return auth.NewAppRoleAuth(conf.RoleId, secretId, loginOpts...)
	default:
		return nil, errors.New("unknown auth module requested")
	}
}
