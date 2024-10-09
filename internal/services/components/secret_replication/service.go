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
	"github.com/soerenschneider/sc-agent/internal/domain/secret_replication"
	"github.com/soerenschneider/sc-agent/internal/metrics"
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

	replicationItems    map[string]secret_replication.ReplicationItem
	once                sync.Once
	mutex               sync.Mutex
	replicationInterval time.Duration

	hashes map[string]string
}

func NewService(client ReplicationClient, syncItems []secret_replication.ReplicationItem, opts ...SecretsReplicationOpts) (*Service, error) {
	if client == nil {
		return nil, errors.New("empty kv2client passed")
	}

	if syncItems == nil {
		return nil, errors.New("no syncitems passed")
	}

	// convert to map
	syncItemsMap := map[string]secret_replication.ReplicationItem{}
	for _, req := range syncItems {
		syncItemsMap[req.ReplicationConf.SecretPath] = req
	}

	ret := &Service{
		client:              client,
		replicationItems:    syncItemsMap,
		hashes:              map[string]string{},
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

func (s *Service) GetReplicationItem(id string) (secret_replication.ReplicationItem, error) {
	item, found := s.replicationItems[id]
	if !found {
		return secret_replication.ReplicationItem{}, secret_replication.ErrSecretsReplicationItemNotFound
	}

	item.Status = secret_replication.FailedStatus
	if found {
		item.Status = secret_replication.SynchronizedStatus
	}

	return item, nil
}

func (s *Service) GetReplicationItems() []secret_replication.ReplicationItem {
	ret := make([]secret_replication.ReplicationItem, len(s.replicationItems))

	idx := 0
	for key := range s.replicationItems {
		ret[idx], _ = s.GetReplicationItem(key)
		idx++
	}

	return ret
}

func (s *Service) Replicate(ctx context.Context, item secret_replication.ReplicationItem) (bool, error) {
	read, err := s.client.ReadSecret(ctx, item.ReplicationConf.SecretPath)
	if err != nil {
		errorLabel := "vault_unknown"
		var respErr *vault.ResponseError
		if errors.As(err, &respErr) {
			errorLabel = fmt.Sprintf("vault_%d", respErr.StatusCode)
		}
		metrics.SecretReplicationErrors.WithLabelValues(item.ReplicationConf.SecretPath, errorLabel).Inc()
		return false, err
	}

	formatted, err := item.Formatter.Format(read.Data)
	if err != nil {
		metrics.SecretReplicationErrors.WithLabelValues(item.ReplicationConf.SecretPath, "formatter").Inc()
		return false, err
	}

	newHash := hash(formatted)
	oldHash, found := s.hashes[item.ReplicationConf.SecretPath]
	if found {
		if newHash == oldHash {
			metrics.SecretsCacheHit.WithLabelValues(item.ReplicationConf.SecretPath).Inc()
			log.Debug().Str("component", vaultSecretSyncer).Msgf("secret %s has not been updated since last time we read it", item.ReplicationConf.SecretPath)
			return false, nil
		}
		log.Info().Str("component", vaultSecretSyncer).Str("secret_path", item.ReplicationConf.SecretPath).Str("dest", item.ReplicationConf.DestUri).Msg("detected update in secret")
	}
	s.hashes[item.ReplicationConf.SecretPath] = newHash

	//if err := s.saveFormattedSecret(formatted, item.ReplicationConf.DestUri); err != nil {
	if err := item.Destination.Write(formatted); err != nil {
		metrics.SecretReplicationErrors.WithLabelValues(item.ReplicationConf.SecretPath, "save").Inc()
		return false, err
	}

	metrics.SecretsRead.WithLabelValues(item.ReplicationConf.SecretPath).Inc()
	log.Info().Str("component", vaultSecretSyncer).Str("secret_path", item.ReplicationConf.SecretPath).Str("dest", item.ReplicationConf.DestUri).Msg("successfully synced secret")
	return true, nil
}

func hash(data []byte) string {
	hasher := sha256.New()
	hasher.Write(data)
	hashBytes := hasher.Sum(nil)
	hashString := hex.EncodeToString(hashBytes)
	return hashString
}
