package vault_common

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	vault_config "github.com/soerenschneider/sc-agent/internal/config/vault"
	"github.com/soerenschneider/sc-agent/internal/metrics"
	"github.com/soerenschneider/sc-agent/pkg/vault"
	"github.com/spf13/afero"
	"go.uber.org/multierr"
)

const (
	defaultMinPercentage = 50
	approleComponentName = "secret-id-rotator"
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
	client        ApproleClient
	minPercentage float64
	vaultConfig   *vault_config.Vault

	once   sync.Once
	fsImpl afero.Fs
}

type ApproleSecretIdRotationOption func(a *ApproleSecretIdRotatorService) error

func NewApproleUpdater(client ApproleClient, config *vault_config.Vault, opts ...ApproleSecretIdRotationOption) (*ApproleSecretIdRotatorService, error) {
	ret := &ApproleSecretIdRotatorService{
		client:        client,
		vaultConfig:   config,
		minPercentage: defaultMinPercentage,
		fsImpl:        afero.NewOsFs(),
	}

	var errs error
	for _, opt := range opts {
		if err := opt(ret); err != nil {
			errs = multierr.Append(errs, err)
		}
	}

	return ret, errs
}

func (a *ApproleSecretIdRotatorService) StartSecretIdRotation(ctx context.Context) {
	a.once.Do(func() {
		ticker := time.NewTicker(30 * time.Second)
		if err := a.ConditionallyRotateSecretId(a.vaultConfig.RoleId, a.vaultConfig.SecretIdFile, false); err != nil {
			log.Error().Err(err).Msg("could not watch")
		}

		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				if err := a.ConditionallyRotateSecretId(a.vaultConfig.RoleId, a.vaultConfig.SecretIdFile, false); err != nil {
					log.Error().Err(err).Msg("could not watch")
				}
			}
		}
	})
}

func (a *ApproleSecretIdRotatorService) ConditionallyRotateSecretId(roleName, secretIdFile string, isAccessor bool) error {
	secretId, err := a.readSecretId(secretIdFile)
	if err != nil {
		log.Error().Str("component", approleComponentName).Str("secret_id_file", secretIdFile).Err(err).Msg("could not read secret_id")
		metrics.SecretIdRotationErrors.WithLabelValues(secretIdFile, "read_secret_id").Inc()
		return err
	}

	if vault.IsWrappedToken(secretIdFile) {
		log.Warn().Str("component", approleComponentName).Str("secret_id_file", secretIdFile).Msg("Detected a wrapped secret_id, trying to rotate immediately")
	} else {
		secretIdInfo, err := a.client.Lookup(roleName, secretId, isAccessor)
		if err != nil && !errors.Is(err, errAlreadyExpired) {
			log.Error().Str("component", approleComponentName).Str("secret_id_file", secretIdFile).Err(err).Msg("could not lookup secret_id")
			return err
		}

		if secretIdInfo == nil {
			log.Warn().Str("component", approleComponentName).Msg("empty response for looking up secret_id, this indicates an expired secret_id")
		} else {
			secretIdPercentage := secretIdInfo.GetPercentage()
			metrics.SecretIdPercentage.WithLabelValues(secretIdFile).Set(secretIdPercentage)
			metrics.SecretIdTtl.WithLabelValues(secretIdFile).Set(float64(secretIdInfo.Ttl))
			if secretIdPercentage >= a.minPercentage {
				log.Debug().Str("component", approleComponentName).Str("secret_id_file", secretIdFile).Float64("lifetime", secretIdPercentage).Msg("not renewing secret_id")
				return nil
			}
			log.Info().Str("component", approleComponentName).Str("secret_id_file", secretIdFile).Float64("lifetime", secretIdPercentage).Msg("Trying to renew secret_id")
		}
	}

	log.Info().Str("secret_id_file", secretIdFile).Msg("generating new secret_id")
	newSecretId, err := a.client.GenerateSecretId(roleName)
	if err != nil {
		log.Error().Str("component", approleComponentName).Str("secret_id_file", secretIdFile).Err(err).Msg("could not generate new secret_id")
		metrics.SecretIdRotationErrors.WithLabelValues(secretIdFile, "generate_secret_id").Inc()
		return err
	}

	if err := a.writeSecretIdFile(newSecretId, secretIdFile); err != nil {
		log.Error().Str("component", approleComponentName).Str("secret_id_file", secretIdFile).Err(err).Msg("could not write new secret_id to file - not going to destroy old secret_id")
		metrics.SecretIdRotationErrors.WithLabelValues(secretIdFile, "write_file").Inc()
		return err
	}

	err = a.client.DestroySecretId(roleName, secretId, isAccessor)
	if err != nil {
		log.Error().Str("component", approleComponentName).Str("secret_id_file", secretIdFile).Err(err).Msg("could not destroy secret_id")
		metrics.SecretIdRotationErrors.WithLabelValues(secretIdFile, "destroy_secret_id").Inc()
	}
	log.Info().Str("component", approleComponentName).Str("secret_id_file", secretIdFile).Msg("successfully rotated secret_id")
	return nil
}

func (a *ApproleSecretIdRotatorService) writeSecretIdFile(newSecretId string, dest string) error {
	return afero.WriteFile(a.fsImpl, dest, []byte(newSecretId), 0600)
}

func (a *ApproleSecretIdRotatorService) readSecretId(secretIdFile string) (string, error) {
	secretIdBytes, err := afero.ReadFile(a.fsImpl, secretIdFile)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(secretIdBytes)), nil
}
