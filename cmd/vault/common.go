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

	tokenRenewer, err := vault_common.NewTokenRenewer(vaultClient, auth)
	if err != nil {
		return err
	}

	var secretIdRotator *vault_common.ApproleSecretIdRotatorService
	if conf.AuthMethod == "approle" {
		approleClient, err := vault_common.NewClient(vaultClient.Logical(), conf.MountApprole)
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

func StartTokenRenewal(ctx context.Context, vaultLoginSuccess chan bool, vaultFatalError chan error) {
	for _, client := range clients {
		go client.StartTokenRenewer(ctx, vaultLoginSuccess, vaultFatalError)
	}
}

func StartApproleSecretIdRotation(ctx context.Context) {
	for _, client := range clients {
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
