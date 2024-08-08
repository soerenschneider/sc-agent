package vault_common

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	vault "github.com/hashicorp/vault/api"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/metrics"
)

const vaultTokenRenewer = "token-renewer"

type TokenRenewer struct {
	client *vault.Client
	auth   vault.AuthMethod
	once   sync.Once
}

func NewTokenRenewer(client *vault.Client, auth vault.AuthMethod) (*TokenRenewer, error) {
	return &TokenRenewer{
		client: client,
		auth:   auth,
	}, nil
}

func (t *TokenRenewer) StartTokenRenewal(ctx context.Context, vaultAuthReady chan bool, vaultAuthError chan error) {
	t.once.Do(func() {
		successfulLogin := false

		for {
			log.Info().Str("component", vaultTokenRenewer).Msg("Logging in to VaultId")
			vaultLoginResp, err := t.client.Auth().Login(ctx, t.auth)

			if err != nil {
				var respErr *vault.ResponseError
				if errors.As(err, &respErr) {
					log.Error().Str("component", vaultTokenRenewer).Err(err).Int("status_code", respErr.StatusCode).Msgf("unable to authenticate to VaultId")
					if respErr.StatusCode >= 400 && respErr.StatusCode <= 500 && !successfulLogin {
						vaultAuthError <- respErr
						close(vaultAuthReady)
						successfulLogin = true
					}
				}
				metrics.VaultLoginErrors.Inc()
				time.Sleep(15 * time.Second)
				continue
			}
			if !successfulLogin {
				// Only write to the channel once and close it afterwards
				vaultAuthReady <- true
				close(vaultAuthReady)
				successfulLogin = true
			}
			metrics.VaultLogins.Inc()

			tokenErr := manageTokenLifecycle(ctx, t.client, vaultLoginResp)
			if tokenErr != nil {
				metrics.VaultTokenRenewErrors.Inc()
				log.Error().Str("component", vaultTokenRenewer).Err(err).Msgf("unable to start managing token lifecycle")
			} else {
				metrics.VaultTokenRenewals.Inc()
			}
		}
	})
}

// Starts token lifecycle management. Returns only fatal errors as errors,
// otherwise returns nil so we can attempt login again.
func manageTokenLifecycle(ctx context.Context, client *vault.Client, token *vault.Secret) error {
	renew := token.Auth.Renewable // You may notice a different top-level field called Renewable. That one is used for dynamic secrets renewal, not token renewal.
	if !renew {
		log.Warn().Msg("Token is not configured to be renewable. Re-attempting login.")
		return nil
	}

	watcher, err := client.NewLifetimeWatcher(&vault.LifetimeWatcherInput{
		Secret:    token,
		Increment: 3600, // Learn more about this optional value in https://www.vaultproject.io/docs/concepts/lease#lease-durations-and-renewal
	})
	if err != nil {
		return fmt.Errorf("unable to initialize new lifetime watcher for renewing auth token: %w", err)
	}

	go watcher.Start()
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		// `DoneCh` will return if renewal fails, or if the remaining lease
		// duration is under a built-in threshold and either renewing is not
		// extending it or renewing is disabled. In any case, the caller
		// needs to attempt to log in again.
		case err := <-watcher.DoneCh():
			if err != nil {
				log.Error().Str("component", vaultTokenRenewer).Err(err).Msg("Failed to renew token, re-attempting login.")
				return nil
			}
			// This occurs once the token has reached max TTL.
			log.Warn().Str("component", vaultTokenRenewer).Msg("Token can no longer be renewed. Re-attempting login.")
			return nil

		// Successfully completed renewal
		case renewal := <-watcher.RenewCh():
			metrics.TokenTtl.WithLabelValues("hm").Set(float64(renewal.Secret.Auth.LeaseDuration))
			log.Info().Str("component", vaultTokenRenewer).Int("token_ttl", renewal.Secret.Auth.LeaseDuration).Msgf("Successfully renewed token")
		}
	}
}
