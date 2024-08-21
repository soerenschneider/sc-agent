package secret_replication

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	vault "github.com/hashicorp/vault/api"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/domain"
	"github.com/soerenschneider/sc-agent/internal/metrics"
	"github.com/spf13/afero"
	"go.uber.org/multierr"
)

const (
	vaultSecretSyncer     = "secrets-replication"
	defaultTickerInterval = 5 * time.Minute
)

type ReplicationClient interface {
	ReadSecret(ctx context.Context, path string) (*vault.KVSecret, error)
}

type SecretsReplicationOpts func(syncer *Service) error

type Service struct {
	client ReplicationClient

	replicationItems    map[string]domain.SecretReplicationItem
	fsImpl              afero.Fs
	once                sync.Once
	mutex               sync.Mutex
	replicationInterval time.Duration

	hashes map[string]string
}

func NewService(client ReplicationClient, syncItems []domain.SecretReplicationItem, opts ...SecretsReplicationOpts) (*Service, error) {
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

	ret := &Service{
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

func (s *Service) StartContinuousReplication(ctx context.Context) {
	s.once.Do(func() {
		if len(s.replicationItems) == 0 {
			log.Warn().Str("component", vaultSecretSyncer).Msg("no items defined, not scheduling auto-renewals")
			return
		}

		log.Info().Str("component", vaultSecretSyncer).Msgf("start replication of %d secrets", len(s.replicationItems))
		jitter := 5 * time.Minute
		checkInterval := s.replicationInterval - (jitter / 2)
		ticker := time.NewTicker(checkInterval)
		s.syncAllSecrets(ctx)

		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				time.Sleep(rand.N(jitter)) // #nosec G404
				s.syncAllSecrets(ctx)
			}
		}
	})
}

func (s *Service) syncAllSecrets(ctx context.Context) {
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

func (s *Service) GetReplicationItem(id string) (domain.SecretReplicationItem, error) {
	item, found := s.replicationItems[id]
	if !found {
		return domain.SecretReplicationItem{}, domain.ErrSecretsReplicationItemNotFound
	}

	item.Status = domain.FailedStatus
	if found {
		item.Status = domain.SynchronizedStatus
	}

	return item, nil
}

func (s *Service) GetReplicationItems() []domain.SecretReplicationItem {
	ret := make([]domain.SecretReplicationItem, len(s.replicationItems))

	idx := 0
	for key := range s.replicationItems {
		ret[idx], _ = s.GetReplicationItem(key)
		idx++
	}

	return ret
}

func (s *Service) Replicate(ctx context.Context, item domain.SecretReplicationItem) (bool, error) {
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

func (s *Service) saveFormattedSecret(data []byte, uri string) error {
	return afero.WriteFile(s.fsImpl, uri, data, 0600)
}

func hash(data []byte) string {
	hasher := sha256.New()
	hasher.Write(data)
	hashBytes := hasher.Sum(nil)
	hashString := hex.EncodeToString(hashBytes)
	return hashString
}
