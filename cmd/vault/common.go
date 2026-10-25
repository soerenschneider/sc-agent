package vault

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sync"
	"time"

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
			cidrs, err := staticCidrs(conf.ApproleCidrLoginResolver.Args)
			if err != nil {
				return nil, fmt.Errorf("invalid args for static cidr_login_resolver: %w", err)
			}
			opts = append(opts, vault_common.WithStaticCidrResolver(cidrs))
		}
		if conf.ApproleCidrLoginResolver.Type == "dynamic" {
			opts = append(opts, vault_common.WithDynamicCidrResolver(conf.Address))
		}
	}

	if conf.ApproleCidrTokenResolver != nil {
		if conf.ApproleCidrTokenResolver.Type == "static" {
			cidrs, err := staticCidrs(conf.ApproleCidrTokenResolver.Args)
			if err != nil {
				return nil, fmt.Errorf("invalid args for static cidr_token_resolver: %w", err)
			}
			opts = append(opts, vault_common.WithStaticCidrTokenResolver(cidrs))
		}
		if conf.ApproleCidrTokenResolver.Type == "dynamic" {
			opts = append(opts, vault_common.WithDynamicCidrTokenResolver(conf.Address))
		}
	}

	return opts, nil
}

// staticCidrs returns the cidrs configured for a static cidr resolver. YAML lists are decoded as []any.
func staticCidrs(args any) ([]string, error) {
	switch v := args.(type) {
	case []string:
		return v, nil
	case []any:
		cidrs := make([]string, 0, len(v))
		for _, item := range v {
			cidr, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("expected cidr to be a string, got %T", item)
			}
			cidrs = append(cidrs, cidr)
		}
		return cidrs, nil
	default:
		return nil, fmt.Errorf("expected a list of cidrs, got %T", args)
	}
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

	if conf.RotationOnly {
		client, err := buildRotationOnlyClient(clientId, conf, vaultClient, auth)
		if err != nil {
			return err
		}
		clients[clientId] = client
		return nil
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

func buildRotationOnlyClient(clientId string, conf vault_config.Vault, vaultClient *vault.Client, auth vault.AuthMethod) (*vault_common.VaultCommon, error) {
	opts, err := getOpts(conf)
	if err != nil {
		return nil, err
	}
	approleClient, err := vault_common.NewClient(vaultClient.Logical(), conf.MountApprole, opts...)
	if err != nil {
		return nil, err
	}
	secretIdRotator, err := vault_common.NewApproleUpdater(approleClient, clientId, &conf, vault_common.WithOnDemandLogin(vaultClient, auth))
	if err != nil {
		return nil, err
	}
	return vault_common.NewRotationOnlyVaultClient(auth, vaultClient, secretIdRotator)
}

// StartTokenRenewal starts logging in and renewing tokens for all Vault clients. After a client has successfully
// logged in, its approle secret_id rotation is started. Rotation-only clients don't keep a token, their rotation is
// started immediately.
func StartTokenRenewal(ctx context.Context) {
	mutex.Lock()
	defer mutex.Unlock()

	for _, client := range clients {
		if client.RotationOnly() {
			go client.StartApproleSecretIdRotation(ctx)
			continue
		}

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
	if client.RotationOnly() {
		return fmt.Errorf("vault client %q is rotation-only and can not be used by other components", clientId)
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
		if conf.SecretIdFileUser != "" {
			loginOpts = append(loginOpts, auth.WithSecretIDFileOwner(conf.SecretIdFileUser))
		}

		isWrappedToken, err := pkg_vault.ContainsFileWrappedToken(conf.SecretIdFile)
		// a missing secret_id file is fetched from approle-courier on login
		bootstrapMissingFile := conf.Bootstrap != nil && errors.Is(err, fs.ErrNotExist)
		if err != nil && !bootstrapMissingFile {
			return nil, err
		}
		if isWrappedToken {
			log.Info().Msg("Trying to authenticate using wrapped secret_id token")
			loginOpts = append(loginOpts, auth.WithWrappingToken())
		}

		if conf.Bootstrap != nil {
			bootstrapper, err := buildSecretIdBootstrapper(conf)
			if err != nil {
				return nil, fmt.Errorf("could not build secret_id bootstrapper: %w", err)
			}
			loginOpts = append(loginOpts, auth.WithBootstrapper(bootstrapper))
		}

		return auth.NewAppRoleAuth(conf.RoleId, secretId, loginOpts...)
	default:
		return nil, errors.New("unknown auth module requested")
	}
}

func buildSecretIdBootstrapper(conf vault_config.Vault) (*auth.SecretIdBootstrapper, error) {
	machineId := conf.Bootstrap.MachineId
	if machineId == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return nil, fmt.Errorf("could not determine machine_id from hostname: %w", err)
		}
		machineId = hostname
	}

	roleName := conf.Bootstrap.RoleName
	if roleName == "" {
		roleName = conf.ApproleRoleName()
	}

	opts := []auth.BootstrapOption{auth.WithBootstrapMetricsLabel(conf.SecretIdFile)}
	if conf.Bootstrap.WrapTtl != "" {
		wrapTtl, err := time.ParseDuration(conf.Bootstrap.WrapTtl)
		if err != nil {
			return nil, fmt.Errorf("invalid wrap_ttl: %w", err)
		}
		opts = append(opts, auth.WithBootstrapWrapTtl(wrapTtl))
	}

	return auth.NewSecretIdBootstrapper(conf.Bootstrap.Address, machineId, conf.MountApprole, roleName, conf.Bootstrap.AgeIdentityFile, opts...)
}
