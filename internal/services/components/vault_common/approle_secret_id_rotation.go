package vault_common

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/rs/zerolog/log"
	vault_config "github.com/soerenschneider/sc-agent/internal/config/vault"
	"github.com/soerenschneider/sc-agent/internal/fsutil"
	"github.com/soerenschneider/sc-agent/internal/metrics"
	"github.com/soerenschneider/sc-agent/pkg/vault"
	"github.com/spf13/afero"
	"go.uber.org/multierr"
)

const (
	defaultInterval      = 10 * time.Minute
	defaultMinPercentage = 50
	logComponent         = "component"
	logSubComponent      = "subcomponent"
	logSecretIdFile      = "secret_id_file"
	approleComponentName = "secret-id-rotator"

	// minOnDemandInterval and maxOnDemandInterval bound the delay between checks of clients that log in on demand
	minOnDemandInterval = 10 * time.Minute
	maxOnDemandInterval = 24 * time.Hour
)

var errAlreadyExpired = errors.New("secret_id already expired")

type ApproleClient interface {
	DestroySecretId(roleName, secretId string, isAccessor bool) error
	Lookup(roleName, secretId string, isAccessor bool) (*vault.SecretIdInfo, error)
	GetSecretIdAccessors(roleName string) ([]string, error)
	GenerateSecretId(roleName string) (string, error)
	ReadRoleId(roleName string) (string, error)
}

type ApproleSecretIdRotatorService struct {
	checkInterval     time.Duration
	client            ApproleClient
	minPercentage     float64
	vaultConfig       *vault_config.Vault
	approleIdentifier string

	// onDemand is set for rotation-only clients that log in for each check instead of keeping a token
	onDemand *onDemandLogin

	once   sync.Once
	fsImpl afero.Fs
	now    func() time.Time
}

type onDemandLogin struct {
	client *api.Client
	auth   api.AuthMethod
}

type ApproleSecretIdRotationOption func(a *ApproleSecretIdRotatorService) error

func NewApproleUpdater(client ApproleClient, approleIdentifier string, config *vault_config.Vault, opts ...ApproleSecretIdRotationOption) (*ApproleSecretIdRotatorService, error) {
	ret := &ApproleSecretIdRotatorService{
		client:            client,
		vaultConfig:       config,
		minPercentage:     defaultMinPercentage,
		fsImpl:            afero.NewOsFs(),
		checkInterval:     defaultInterval,
		approleIdentifier: approleIdentifier,
		now:               time.Now,
	}

	var errs error
	for _, opt := range opts {
		if err := opt(ret); err != nil {
			errs = multierr.Append(errs, err)
		}
	}

	return ret, errs
}

// WithOnDemandLogin makes the rotator log in before each check and revoke the token afterwards, so no token is kept
// between checks. Checks are scheduled based on the secret_id's remaining lifetime to keep the number of logins low.
func WithOnDemandLogin(client *api.Client, auth api.AuthMethod) ApproleSecretIdRotationOption {
	return func(a *ApproleSecretIdRotatorService) error {
		if client == nil || auth == nil {
			return errors.New("empty client or authmethod passed")
		}
		a.onDemand = &onDemandLogin{client: client, auth: auth}
		return nil
	}
}

func (a *ApproleSecretIdRotatorService) StartSecretIdRotation(ctx context.Context) {
	a.once.Do(func() {
		for {
			delay := a.check(ctx)
			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
			}
		}
	})
}

// check conditionally rotates the secret_id and returns the delay until the next check.
func (a *ApproleSecretIdRotatorService) check(ctx context.Context) time.Duration {
	if a.onDemand == nil {
		if err := a.ConditionallyRotateSecretId(*a.vaultConfig, false); err != nil {
			log.Error().Str(logComponent, "vault").Str(logSubComponent, approleComponentName).Str("id", a.approleIdentifier).Err(err).Msg("Conditionally rotating secret_id failed")
		}
		jitter := 5 * time.Minute
		return a.checkInterval - (jitter / 2)
	}

	// rotation-only clients have no token renewer, they are degraded while checks fail
	component := DegradedComponentName(a.approleIdentifier)
	logout, err := a.login(ctx)
	if err != nil {
		log.Error().Str(logComponent, "vault").Str(logSubComponent, approleComponentName).Str("id", a.approleIdentifier).Err(err).Msg("could not log in to check secret_id")
		metrics.VaultLoginErrors.WithLabelValues(a.approleIdentifier).Inc()
		metrics.SecretIdRotationErrors.WithLabelValues(a.vaultConfig.SecretIdFile, "login").Inc()
		metrics.SetComponentDegraded(component)
		return minOnDemandInterval
	}
	metrics.VaultLogins.WithLabelValues(a.approleIdentifier).Inc()
	defer logout()

	nextCheck, err := a.conditionallyRotateSecretId(*a.vaultConfig, false)
	if err != nil {
		log.Error().Str(logComponent, "vault").Str(logSubComponent, approleComponentName).Str("id", a.approleIdentifier).Err(err).Msg("Conditionally rotating secret_id failed")
		metrics.SetComponentDegraded(component)
		return minOnDemandInterval
	}
	metrics.SetComponentHealthy(component)
	delay := min(max(nextCheck, minOnDemandInterval), maxOnDemandInterval)
	log.Info().Str(logComponent, "vault").Str(logSubComponent, approleComponentName).Str("id", a.approleIdentifier).Str("next_check", a.now().Add(delay).Format(time.RFC3339)).Msg("scheduled next secret_id check")
	return delay
}

// login logs in and returns a function that revokes the token again.
func (a *ApproleSecretIdRotatorService) login(ctx context.Context) (func(), error) {
	if _, err := a.onDemand.client.Auth().Login(ctx, a.onDemand.auth); err != nil {
		return nil, err
	}

	return func() {
		// revoking is best-effort, the token expires anyway
		if err := a.onDemand.client.Auth().Token().RevokeSelfWithContext(ctx, ""); err != nil {
			log.Warn().Str(logComponent, "vault").Str(logSubComponent, approleComponentName).Str("id", a.approleIdentifier).Err(err).Msg("could not revoke token after checking secret_id")
		}
		a.onDemand.client.ClearToken()
	}, nil
}

func (a *ApproleSecretIdRotatorService) ConditionallyRotateSecretId(cnf vault_config.Vault, isAccessor bool) error {
	_, err := a.conditionallyRotateSecretId(cnf, isAccessor)
	return err
}

// conditionallyRotateSecretId rotates the secret_id if needed and returns the time until it needs to be checked
// again, which is 0 if that is unknown.
func (a *ApproleSecretIdRotatorService) conditionallyRotateSecretId(cnf vault_config.Vault, isAccessor bool) (time.Duration, error) {
	secretId, err := a.readSecretId(cnf.SecretIdFile)
	if err != nil {
		log.Error().Str(logComponent, "vault").Str(logSubComponent, approleComponentName).Str(logSecretIdFile, cnf.SecretIdFile).Str("id", a.approleIdentifier).Err(err).Msg("could not read secret_id")
		metrics.SecretIdRotationErrors.WithLabelValues(cnf.SecretIdFile, "read_secret_id").Inc()
		return 0, err
	}

	isWrapped := vault.IsWrappedToken(secretId)
	if isWrapped {
		log.Warn().Str(logComponent, "vault").Str(logSubComponent, approleComponentName).Str(logSecretIdFile, cnf.SecretIdFile).Str("id", a.approleIdentifier).Msg("Detected a wrapped secret_id, trying to rotate immediately")
	} else {
		secretIdInfo, err := a.client.Lookup(cnf.ApproleRoleName(), secretId, isAccessor)
		if err != nil && !errors.Is(err, errAlreadyExpired) {
			log.Error().Str(logComponent, "vault").Str(logSubComponent, approleComponentName).Str("id", a.approleIdentifier).Str(logSecretIdFile, cnf.SecretIdFile).Err(err).Msg("could not lookup secret_id")
			metrics.SecretIdRotationErrors.WithLabelValues(cnf.SecretIdFile, "lookup").Inc()
			return 0, err
		}

		if secretIdInfo == nil {
			log.Warn().Str(logComponent, "vault").Str(logSubComponent, approleComponentName).Str("id", a.approleIdentifier).Msg("empty response for looking up secret_id, this indicates an expired secret_id")
		} else {
			secretIdPercentage := a.recordSecretIdInfo(cnf, *secretIdInfo)
			if secretIdPercentage >= a.minPercentage {
				log.Debug().Str(logComponent, "vault").Str(logSubComponent, approleComponentName).Str("id", a.approleIdentifier).Str(logSecretIdFile, cnf.SecretIdFile).Str("expiration", secretIdInfo.Expiration.String()).Float64("lifetime", secretIdPercentage).Msg("not renewing secret_id")
				return a.untilRotationDue(*secretIdInfo), nil
			}
			log.Info().Str(logComponent, "vault").Str(logSubComponent, approleComponentName).Str("id", a.approleIdentifier).Str(logSecretIdFile, cnf.SecretIdFile).Str("expiration", secretIdInfo.Expiration.String()).Float64("lifetime", secretIdPercentage).Msg("Trying to renew secret_id")
		}
	}

	log.Info().Str(logComponent, "vault").Str(logSubComponent, approleComponentName).Str("id", a.approleIdentifier).Str(logSecretIdFile, cnf.SecretIdFile).Msg("generating new secret_id")
	newSecretId, err := a.client.GenerateSecretId(cnf.ApproleRoleName())
	if err != nil {
		log.Error().Str(logComponent, "vault").Str(logSubComponent, approleComponentName).Str(logSecretIdFile, cnf.SecretIdFile).Err(err).Msg("could not generate new secret_id")
		metrics.SecretIdRotationErrors.WithLabelValues(cnf.SecretIdFile, "generate_secret_id").Inc()
		return 0, err
	}

	if err := a.writeSecretIdFile(newSecretId, cnf); err != nil {
		log.Error().Str(logComponent, "vault").Str(logSubComponent, approleComponentName).Str(logSecretIdFile, cnf.SecretIdFile).Err(err).Msg("could not write new secret_id to file - not going to destroy old secret_id")
		metrics.SecretIdRotationErrors.WithLabelValues(cnf.SecretIdFile, "write_file").Inc()
		// nobody knows the new secret_id, don't leave it valid until it expires
		if destroyErr := a.client.DestroySecretId(cnf.ApproleRoleName(), newSecretId, false); destroyErr != nil {
			log.Error().Str(logComponent, "vault").Str(logSubComponent, approleComponentName).Str("id", a.approleIdentifier).Str(logSecretIdFile, cnf.SecretIdFile).Err(destroyErr).Msg("could not destroy unused new secret_id")
			metrics.SecretIdRotationErrors.WithLabelValues(cnf.SecretIdFile, "destroy_secret_id").Inc()
		}
		return 0, err
	}

	err = a.client.DestroySecretId(cnf.ApproleRoleName(), secretId, isAccessor)
	if err != nil {
		log.Error().Str(logComponent, "vault").Str(logSubComponent, approleComponentName).Str("id", a.approleIdentifier).Str(logSecretIdFile, cnf.SecretIdFile).Err(err).Msg("could not destroy secret_id")
		metrics.SecretIdRotationErrors.WithLabelValues(cnf.SecretIdFile, "destroy_secret_id").Inc()
	}
	log.Info().Str(logComponent, "vault").Str(logSubComponent, approleComponentName).Str(logSecretIdFile, cnf.SecretIdFile).Msg("successfully rotated secret_id")

	// look up the new secret_id, so the metrics don't show the expiration of the old one
	newSecretIdInfo, err := a.client.Lookup(cnf.ApproleRoleName(), newSecretId, false)
	if err != nil || newSecretIdInfo == nil {
		log.Warn().Str(logComponent, "vault").Str(logSubComponent, approleComponentName).Str("id", a.approleIdentifier).Str(logSecretIdFile, cnf.SecretIdFile).Err(err).Msg("could not lookup new secret_id")
		metrics.SecretIdExpiration.DeleteLabelValues(cnf.SecretIdFile)
		return 0, nil
	}
	a.recordSecretIdInfo(cnf, *newSecretIdInfo)
	return a.untilRotationDue(*newSecretIdInfo), nil
}

// recordSecretIdInfo updates the metrics of the secret_id and returns its remaining lifetime in percent.
func (a *ApproleSecretIdRotatorService) recordSecretIdInfo(cnf vault_config.Vault, info vault.SecretIdInfo) float64 {
	percentage := info.GetPercentage()
	metrics.SecretIdPercentage.WithLabelValues(cnf.SecretIdFile).Set(percentage)
	metrics.SecretIdTtl.WithLabelValues(cnf.SecretIdFile).Set(float64(info.Ttl))
	if info.Expiration.IsZero() {
		metrics.SecretIdExpiration.DeleteLabelValues(cnf.SecretIdFile)
	} else {
		metrics.SecretIdExpiration.WithLabelValues(cnf.SecretIdFile).Set(float64(info.Expiration.Unix()))
	}
	return percentage
}

// untilRotationDue returns the time until the secret_id's remaining lifetime drops below the minimum percentage.
// secret_ids without expiration never need to be rotated.
func (a *ApproleSecretIdRotatorService) untilRotationDue(info vault.SecretIdInfo) time.Duration {
	if info.Expiration.IsZero() {
		return maxOnDemandInterval
	}
	lifetime := info.Expiration.Sub(info.CreationTime)
	rotationDue := info.Expiration.Add(-time.Duration(float64(lifetime) * a.minPercentage / 100))
	return rotationDue.Sub(a.now())
}

var ErrFileOwnership = fsutil.ErrFileOwnership

func (a *ApproleSecretIdRotatorService) writeSecretIdFile(newSecretId string, cnf vault_config.Vault) error {
	return fsutil.WriteFileAtomic(a.fsImpl, cnf.SecretIdFile, []byte(newSecretId), cnf.SecretIdFileUser)
}

func (a *ApproleSecretIdRotatorService) readSecretId(secretIdFile string) (string, error) {
	secretIdBytes, err := afero.ReadFile(a.fsImpl, secretIdFile)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(secretIdBytes)), nil
}
