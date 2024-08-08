package vault_secrets_replication

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	vault "github.com/hashicorp/vault/api"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/domain"
	"github.com/soerenschneider/sc-agent/internal/metrics"
	"github.com/spf13/afero"
	"go.uber.org/multierr"
	"golang.org/x/exp/maps"
)

const (
	vaultSecretSyncer     = "secrets-replication"
	defaultTickerInterval = 5 * time.Minute
)

type ReplicationClient interface {
	ReadSecret(ctx context.Context, path string) (*vault.KVSecret, error)
}

type SecretsReplicationOpts func(syncer *SecretReplicationService) error

type SecretReplicationService struct {
	client ReplicationClient

	replicationItems    map[string]domain.SecretReplicationItem
	fsImpl              afero.Fs
	once                sync.Once
	mutex               sync.Mutex
	replicationInterval time.Duration

	hashes map[string]string
}

func NewService(client ReplicationClient, syncItems []domain.SecretReplicationItem, opts ...SecretsReplicationOpts) (*SecretReplicationService, error) {
	if client == nil {
		return nil, errors.New("empty kv2client passed")
	}

	if syncItems == nil {
		return nil, errors.New("no syncitems passed")
	}

	fs := afero.NewOsFs()

	// convert to map
	syncItemsMap := map[string]domain.SecretReplicationItem{}
	for _, req := range syncItems {
		syncItemsMap[req.SecretPath] = req
	}

	ret := &SecretReplicationService{
		client:              client,
		replicationItems:    syncItemsMap,
		hashes:              map[string]string{},
		fsImpl:              fs,
		replicationInterval: defaultTickerInterval,
	}

	var errs error
	for _, opt := range opts {
		if err := opt(ret); err != nil {
			errs = multierr.Append(errs, err)
		}
	}

	return ret, errs
}

func (s *SecretReplicationService) StartContinuousReplication(ctx context.Context) {
	s.once.Do(func() {
		ticker := time.NewTicker(s.replicationInterval)
		s.syncAllSecrets(ctx)

		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				s.syncAllSecrets(ctx)
			}
		}
	})
}

func (s *SecretReplicationService) syncAllSecrets(ctx context.Context) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var errs error
	for _, req := range s.replicationItems {
		select {
		case <-ctx.Done():
			return
		default:
			_, err := s.Replicate(ctx, req)
			if err != nil {
				errs = multierr.Append(errs, err)
			}
		}
	}
	if errs != nil {
		log.Error().Err(errs).Msg("encountered problems while syncing secrets")
	}
}

func (s *SecretReplicationService) GetReplicationItem(secretPath string) (domain.SecretReplicationItem, error) {
	request, found := s.replicationItems[secretPath]
	if !found {
		return domain.SecretReplicationItem{}, domain.ErrSecretReplicationItemNotFound
	}

	return request, nil
}

func (s *SecretReplicationService) GetReplicationItems() []domain.SecretReplicationItem {
	return maps.Values(s.replicationItems)
}

func (s *SecretReplicationService) Replicate(ctx context.Context, item domain.SecretReplicationItem) (bool, error) {
	read, err := s.client.ReadSecret(ctx, item.SecretPath)
	if err != nil {
		errorLabel := "vault_unknown"
		var respErr *vault.ResponseError
		if errors.As(err, &respErr) {
			errorLabel = fmt.Sprintf("vault_%d", respErr.StatusCode)
		}
		metrics.SecretReplicationErrors.WithLabelValues(item.SecretPath, errorLabel).Inc()
		return false, err
	}

	formatted, err := item.Formatter.Format(read.Data)
	if err != nil {
		metrics.SecretReplicationErrors.WithLabelValues(item.SecretPath, "formatter").Inc()
		return false, err
	}

	newHash := hash(formatted)
	oldHash, found := s.hashes[item.SecretPath]
	if found {
		if newHash == oldHash {
			metrics.SecretsCacheHit.WithLabelValues(item.SecretPath).Inc()
			log.Debug().Str("component", vaultSecretSyncer).Msgf("secret %s has not been updated since last time we read it", item.SecretPath)
			return false, nil
		}
		log.Info().Str("component", vaultSecretSyncer).Str("secret_path", item.SecretPath).Str("dest", item.DestUri).Msg("detected update in secret")
	}
	s.hashes[item.SecretPath] = newHash

	if err := s.saveFormattedSecret(formatted, item.DestUri); err != nil {
		metrics.SecretReplicationErrors.WithLabelValues(item.SecretPath, "save").Inc()
		return false, err
	}

	metrics.SecretsRead.WithLabelValues(item.SecretPath).Inc()
	log.Info().Str("component", vaultSecretSyncer).Str("secret_path", item.SecretPath).Str("dest", item.DestUri).Msg("successfully synced secret")
	return true, nil
}

func (s *SecretReplicationService) saveFormattedSecret(data []byte, uri string) error {
	return afero.WriteFile(s.fsImpl, uri, data, 0600)
}

func hash(data []byte) string {
	hasher := md5.New()
	hasher.Write(data)
	hashBytes := hasher.Sum(nil)
	hashString := hex.EncodeToString(hashBytes)
	return hashString
}
