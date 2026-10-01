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

const (
	vaultTokenRenewerComponent = "token-renewer"
	loginRetryInterval         = 15 * time.Second
)

type TokenRenewer struct {
	client     *vault.Client
	clientName string
	auth       vault.AuthMethod
	once       sync.Once
	loggedIn   chan struct{}
}

func NewTokenRenewer(client *vault.Client, auth vault.AuthMethod, clientName string) (*TokenRenewer, error) {
	if client == nil {
		return nil, errors.New("empty client passed")
	}

	if auth == nil {
		return nil, errors.New("empty authmethod passed")
	}

	return &TokenRenewer{
		client:     client,
		auth:       auth,
		clientName: clientName,
		loggedIn:   make(chan struct{}),
	}, nil
}

// LoggedIn returns a channel that is closed after the first successful login.
func (t *TokenRenewer) LoggedIn() <-chan struct{} {
	return t.loggedIn
}

// StartTokenRenewal logs in to Vault and keeps the token renewed. Failing logins are not fatal, they are retried
// indefinitely while the client is flagged as degraded.
func (t *TokenRenewer) StartTokenRenewal(ctx context.Context) {
	t.once.Do(func() {
		successfulLogin := false
		// the client is not operational until it has logged in successfully
		component := DegradedComponentName(t.clientName)
		metrics.SetComponentDegraded(component)

		log.Info().Str("component", vaultTokenRenewerComponent).Str("client", t.clientName).Msg("Logging in to Vault")
		for {
			vaultLoginResp, err := t.client.Auth().Login(ctx, t.auth)
			if err != nil {
				if ctx.Err() != nil {
					return
				}

				logEvent := log.Error().Str("component", vaultTokenRenewerComponent).Str("client", t.clientName).Err(err)
				var respErr *vault.ResponseError
				if errors.As(err, &respErr) {
					logEvent = logEvent.Int("status_code", respErr.StatusCode)
				}
				logEvent.Msg("unable to authenticate to Vault, retrying")
				metrics.VaultLoginErrors.WithLabelValues(t.clientName).Inc()
				metrics.SetComponentDegraded(component)

				select {
				case <-ctx.Done():
					return
				case <-time.After(loginRetryInterval):
				}
				continue
			}

			metrics.SetComponentHealthy(component)
			if !successfulLogin {
				close(t.loggedIn)
				successfulLogin = true
			}
			metrics.VaultLogins.WithLabelValues(t.clientName).Inc()

			tokenErr := manageTokenLifecycle(ctx, t.client, vaultLoginResp, t.clientName)
			if tokenErr != nil {
				metrics.VaultTokenRenewErrors.WithLabelValues(t.clientName).Inc()
				log.Error().Str(logComponent, "vault").Str(logSubComponent, vaultTokenRenewerComponent).Err(tokenErr).Msgf("unable to start managing token lifecycle")
			} else {
				metrics.VaultTokenRenewals.WithLabelValues(t.clientName).Inc()
			}

			if ctx.Err() != nil {
				return
			}
		}
	})
}

// DegradedComponentName returns the name a Vault client is tracked with in the degraded metrics.
func DegradedComponentName(clientName string) string {
	return "vault/" + clientName
}

// Starts token lifecycle management. Returns only fatal errors as errors,
// otherwise returns nil so we can attempt login again.
func manageTokenLifecycle(ctx context.Context, client *vault.Client, token *vault.Secret, clientName string) error {
	renew := token.Auth.Renewable // You may notice a different top-level field called Renewable. That one is used for dynamic secrets renewal, not token renewal.
	if !renew {
		return waitForTokenExpiry(ctx, token.Auth.LeaseDuration)
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
				log.Error().Str(logComponent, "vault").Str(logSubComponent, vaultTokenRenewerComponent).Err(err).Msg("Failed to renew token, re-attempting login.")
				return nil
			}
			// This occurs once the token has reached max TTL.
			log.Warn().Str(logComponent, "vault").Str(logSubComponent, vaultTokenRenewerComponent).Msg("Token can no longer be renewed. Re-attempting login.")
			return nil

		// Successfully completed renewal
		case renewal := <-watcher.RenewCh():
			metrics.TokenTtl.WithLabelValues(clientName).Set(float64(renewal.Secret.Auth.LeaseDuration))
			log.Info().Str(logComponent, "vault").Str(logSubComponent, vaultTokenRenewerComponent).Int("token_ttl", renewal.Secret.Auth.LeaseDuration).Msgf("Successfully renewed token")
		}
	}
}

// waitForTokenExpiry blocks until a non-renewable token has expired, so a new login is not attempted immediately.
// Tokens without a TTL never expire, there is nothing left to do for them.
func waitForTokenExpiry(ctx context.Context, leaseDurationSeconds int) error {
	if leaseDurationSeconds <= 0 {
		log.Info().Str(logComponent, "vault").Str(logSubComponent, vaultTokenRenewerComponent).Msg("Token is not renewable and has no TTL, no token lifecycle management needed")
		<-ctx.Done()
		return nil
	}

	log.Warn().Str(logComponent, "vault").Str(logSubComponent, vaultTokenRenewerComponent).Int("token_ttl", leaseDurationSeconds).Msg("Token is not configured to be renewable, re-attempting login after it expired")
	select {
	case <-ctx.Done():
	case <-time.After(time.Duration(leaseDurationSeconds) * time.Second):
	}
	return nil
}
