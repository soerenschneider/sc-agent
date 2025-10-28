package vault_common

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/user"
	"runtime"
	"strconv"
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
	defaultInterval      = 10 * time.Minute
	defaultMinPercentage = 50
	logComponent         = "component"
	logSecretIdFile      = "secret_id_file"
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
	checkInterval     time.Duration
	client            ApproleClient
	minPercentage     float64
	vaultConfig       *vault_config.Vault
	approleIdentifier string

	once   sync.Once
	fsImpl afero.Fs
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
		jitter := 5 * time.Minute
		checkInterval := a.checkInterval - (jitter / 2)
		ticker := time.NewTicker(checkInterval)
		if err := a.ConditionallyRotateSecretId(*a.vaultConfig, false); err != nil {
			log.Error().Str(logComponent, approleComponentName).Str("id", a.approleIdentifier).Err(err).Msg("Conditionally rotating secret_id failed")
		}

		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				if err := a.ConditionallyRotateSecretId(*a.vaultConfig, false); err != nil {
					log.Error().Str(logComponent, approleComponentName).Str("id", a.approleIdentifier).Err(err).Msg("Conditionally rotating secret_id failed")
				}
			}
		}
	})
}

func (a *ApproleSecretIdRotatorService) ConditionallyRotateSecretId(cnf vault_config.Vault, isAccessor bool) error {
	secretId, err := a.readSecretId(cnf.SecretIdFile)
	if err != nil {
		log.Error().Str(logComponent, approleComponentName).Str(logSecretIdFile, cnf.SecretIdFile).Str("id", a.approleIdentifier).Err(err).Msg("could not read secret_id")
		metrics.SecretIdRotationErrors.WithLabelValues(cnf.SecretIdFile, "read_secret_id").Inc()
		return err
	}

	isWrapped := vault.IsWrappedToken(secretId)
	if isWrapped {
		log.Warn().Str(logComponent, approleComponentName).Str(logSecretIdFile, cnf.SecretIdFile).Str("id", a.approleIdentifier).Msg("Detected a wrapped secret_id, trying to rotate immediately")
	} else {
		secretIdInfo, err := a.client.Lookup(cnf.RoleId, secretId, isAccessor)
		if err != nil && !errors.Is(err, errAlreadyExpired) {
			log.Error().Str(logComponent, approleComponentName).Str("id", a.approleIdentifier).Str(logSecretIdFile, cnf.SecretIdFile).Err(err).Msg("could not lookup secret_id")
			return err
		}

		if secretIdInfo == nil {
			log.Warn().Str(logComponent, approleComponentName).Str("id", a.approleIdentifier).Msg("empty response for looking up secret_id, this indicates an expired secret_id")
		} else {
			secretIdPercentage := secretIdInfo.GetPercentage()
			metrics.SecretIdPercentage.WithLabelValues(cnf.SecretIdFile).Set(secretIdPercentage)
			metrics.SecretIdTtl.WithLabelValues(cnf.SecretIdFile).Set(float64(secretIdInfo.Ttl))
			if secretIdPercentage >= a.minPercentage {
				log.Debug().Str(logComponent, approleComponentName).Str("id", a.approleIdentifier).Str(logSecretIdFile, cnf.SecretIdFile).Str("expiration", secretIdInfo.Expiration.String()).Float64("lifetime", secretIdPercentage).Msg("not renewing secret_id")
				return nil
			}
			log.Info().Str(logComponent, approleComponentName).Str("id", a.approleIdentifier).Str(logSecretIdFile, cnf.SecretIdFile).Str("expiration", secretIdInfo.Expiration.String()).Float64("lifetime", secretIdPercentage).Msg("Trying to renew secret_id")
		}
	}

	log.Info().Str(logComponent, approleComponentName).Str("id", a.approleIdentifier).Str(logSecretIdFile, cnf.SecretIdFile).Msg("generating new secret_id")
	newSecretId, err := a.client.GenerateSecretId(cnf.RoleId)
	if err != nil {
		log.Error().Str(logComponent, approleComponentName).Str(logSecretIdFile, cnf.SecretIdFile).Err(err).Msg("could not generate new secret_id")
		metrics.SecretIdRotationErrors.WithLabelValues(cnf.SecretIdFile, "generate_secret_id").Inc()
		return err
	}

	if err := a.writeSecretIdFile(newSecretId, cnf); err != nil {
		log.Error().Str(logComponent, approleComponentName).Str(logSecretIdFile, cnf.SecretIdFile).Err(err).Msg("could not write new secret_id to file - not going to destroy old secret_id")
		metrics.SecretIdRotationErrors.WithLabelValues(cnf.SecretIdFile, "write_file").Inc()
		return err
	}

	err = a.client.DestroySecretId(cnf.RoleId, secretId, isAccessor)
	if err != nil {
		log.Error().Str(logComponent, approleComponentName).Str("id", a.approleIdentifier).Str(logSecretIdFile, cnf.SecretIdFile).Err(err).Msg("could not destroy secret_id")
		metrics.SecretIdRotationErrors.WithLabelValues(cnf.SecretIdFile, "destroy_secret_id").Inc()
	}
	log.Info().Str(logComponent, approleComponentName).Str(logSecretIdFile, cnf.SecretIdFile).Msg("successfully rotated secret_id")
	return nil
}

var ErrFileOwnership = errors.New("could not update file ownership")

func (a *ApproleSecretIdRotatorService) writeSecretIdFile(newSecretId string, cnf vault_config.Vault) error {
	if err := afero.WriteFile(a.fsImpl, cnf.SecretIdFile, []byte(newSecretId), 0600); err != nil {
		return err
	}

	if cnf.SecretIdFileUser != "" && runtime.GOOS != "windows" {
		usr, err := user.Lookup(cnf.SecretIdFileUser)
		if err != nil {
			return fmt.Errorf("%w: could not lookup user %s", ErrFileOwnership, cnf.SecretIdFileUser)
		}

		uid, _ := strconv.Atoi(usr.Uid)
		if err := os.Chown(cnf.SecretIdFile, uid, 0); err != nil {
			return fmt.Errorf("%w: %w", ErrFileOwnership, err)
		}
	}

	return nil
}

func (a *ApproleSecretIdRotatorService) readSecretId(secretIdFile string) (string, error) {
	secretIdBytes, err := afero.ReadFile(a.fsImpl, secretIdFile)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(secretIdBytes)), nil
}
