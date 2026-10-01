package vault

import (
	"context"
	"errors"
	"fmt"
	"sync"

	vault "github.com/hashicorp/vault/api"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/config"
	vault_config "github.com/soerenschneider/sc-agent/internal/config/vault"
	"github.com/soerenschneider/sc-agent/internal/metrics"
	"github.com/soerenschneider/sc-agent/internal/services/components/vault_common"
	"github.com/soerenschneider/sc-agent/internal/services/components/vault_common/auth"
	pkg_vault "github.com/soerenschneider/sc-agent/pkg/vault"
)

var (
	clients = map[string]*vault_common.VaultCommon{}
	mutex   sync.Mutex
)

func getVaultClient(key string) *vault_common.VaultCommon {
	mutex.Lock()
	defer mutex.Unlock()
	return clients[key]
}

// BuildVaultClients builds all configured Vault clients. Clients that can not be built are skipped and flagged as
// degraded, components depending on them will subsequently fail to build.
func BuildVaultClients(conf config.Config) {
	for clientId, vaultConf := range conf.Vault {
		if err := buildVaultClient(clientId, vaultConf); err != nil {
			log.Error().Str("component", "vault").Str("client", clientId).Err(err).Msg("could not build vault client, running in degraded mode")
			metrics.SetComponentDegraded(vault_common.DegradedComponentName(clientId))
		}
	}
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

func buildVaultClient(clientId string, conf vault_config.Vault) error {
	mutex.Lock()
	defer mutex.Unlock()

	_, found := clients[clientId]
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

	tokenRenewer, err := vault_common.NewTokenRenewer(vaultClient, auth, clientId)
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
		secretIdRotator, err = vault_common.NewApproleUpdater(approleClient, clientId, &conf)
		if err != nil {
			return err
		}
	}

	client, err := vault_common.NewVaultClient(auth, vaultClient, tokenRenewer, secretIdRotator)
	if err != nil {
		return err
	}

	clients[clientId] = client
	return nil
}

// StartTokenRenewal starts logging in and renewing tokens for all Vault clients. After a client has successfully
// logged in, its approle secret_id rotation is started.
func StartTokenRenewal(ctx context.Context) {
	mutex.Lock()
	defer mutex.Unlock()

	for _, client := range clients {
		go client.StartTokenRenewer(ctx)
		go func() {
			select {
			case <-ctx.Done():
			case <-client.LoggedIn():
				client.StartApproleSecretIdRotation(ctx)
			}
		}()
	}
}

// WaitForLogin blocks until the Vault client with the given id has successfully logged in. It returns an error if
// the client is not available or the context is canceled before.
func WaitForLogin(ctx context.Context, clientId string) error {
	client := getVaultClient(clientId)
	if client == nil {
		return fmt.Errorf("vault client %q not found", clientId)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-client.LoggedIn():
		return nil
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

		isWrappedToken, err := pkg_vault.ContainsFileWrappedToken(conf.SecretIdFile)
		if err != nil {
			return nil, err
		}
		if isWrappedToken {
			log.Info().Msg("Trying to authenticate using wrapped secret_id token")
			loginOpts = append(loginOpts, auth.WithWrappingToken())
		}

		return auth.NewAppRoleAuth(conf.RoleId, secretId, loginOpts...)
	default:
		return nil, errors.New("unknown auth module requested")
	}
}
