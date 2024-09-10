package http_replication

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"hash/fnv"
	"io"
	"math/rand/v2"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/domain/http_replication"
	"github.com/soerenschneider/sc-agent/internal/metrics"
	"github.com/soerenschneider/sc-agent/pkg"
)

const (
	logComponent             = "component"
	httpReplicationComponent = "http-replication"
	defaultInterval          = 10 * time.Minute
)

type Client interface {
	Do(req *http.Request) (*http.Response, error)
}

type StorageImplementation interface {
	Read() ([]byte, error)
	CanRead() error
	Write([]byte) error
}

type Service struct {
	client       Client
	managedItems map[string]http_replication.ReplicationItem
	cache        map[string]string
	once         sync.Once
	interval     time.Duration
}

func New(client Client, items []http_replication.ReplicationItem) (*Service, error) {
	if client == nil {
		return nil, errors.New("empty client passed")
	}

	managedItems := map[string]http_replication.ReplicationItem{}
	for _, item := range items {
		managedItems[item.ReplicationConf.Id] = item
	}

	ret := &Service{
		client:       client,
		managedItems: managedItems,
		cache:        map[string]string{},
		interval:     defaultInterval,
	}

	return ret, nil
}

func (s *Service) StartReplication(ctx context.Context) {
	s.once.Do(func() {
		if len(s.managedItems) == 0 {
			log.Warn().Str(logComponent, httpReplicationComponent).Msg("no certificates defined, not scheduling auto-renewals")
			return
		}

		log.Info().Str(logComponent, httpReplicationComponent).Msgf("start replication of %d items", len(s.managedItems))
		jitter := 5 * time.Minute
		checkInterval := s.interval - (jitter / 2)
		ticker := time.NewTicker(checkInterval)
		s.autoRenew(ctx)

		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				time.Sleep(rand.N(jitter)) // #nosec G404
				s.autoRenew(ctx)
			}
		}
	})
}

func (s *Service) autoRenew(ctx context.Context) {
	for _, req := range s.managedItems {
		select {
		case <-ctx.Done():
			return
		default:
			if err := s.Replicate(ctx, req); err != nil {
				log.Error().Err(err).Str(logComponent, httpReplicationComponent).Str("id", req.ReplicationConf.Id).Msg("replicating item failed")
			}
		}
	}
}

func (s *Service) GetReplicationItem(id string) (http_replication.ReplicationItem, error) {
	item, found := s.managedItems[id]
	if !found {
		return http_replication.ReplicationItem{}, http_replication.ErrHttpReplicationItemNotFound
	}

	item.Status = http_replication.Unknown
	_, found = s.cache[id]
	if found {
		item.Status = http_replication.Synced
	}

	return item, nil
}

func (s *Service) GetReplicationItems() ([]http_replication.ReplicationItem, error) {
	ret := make([]http_replication.ReplicationItem, len(s.managedItems))

	idx := 0
	for key := range s.managedItems {
		ret[idx], _ = s.GetReplicationItem(key)
		idx++
	}

	return ret, nil
}

func (s *Service) Replicate(ctx context.Context, conf http_replication.ReplicationItem) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, conf.ReplicationConf.Source, nil)
	if err != nil {
		return err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	hash := hashContent(data)
	if isMismatchedChecksum(conf, hash) {
		conf.Status = http_replication.InvalidChecksum
		log.Error().Str(logComponent, httpReplicationComponent).Str("hash_expected", conf.ReplicationConf.Sha256Sum).Str("hash_actual", hash).Str("id", conf.ReplicationConf.Id).Msg("invalid hash")
		return http_replication.ErrMismatchedHash
	}

	oldHash, found := s.cache[conf.ReplicationConf.Id]
	if found && oldHash == hash {
		return nil
	}

	s.cache[conf.ReplicationConf.Id] = hash
	updateMetricsHash(conf.ReplicationConf.Id, data)

	if !found {
		read, err := conf.Destination.Read()
		if err == nil && hash == hashContent(read) {
			log.Info().Str(logComponent, httpReplicationComponent).Str("id", conf.ReplicationConf.Id).Msg("file already exists locally")
			return nil
		}
	}

	log.Info().Str(logComponent, httpReplicationComponent).Str("id", conf.ReplicationConf.Id).Msg("writing updated value to disk")
	if err := conf.Destination.Write(data); err != nil {
		return err
	}

	if err := pkg.RunPostIssueHooks(conf.PostHooks); err != nil {
		return err
	}

	return nil
}

func isMismatchedChecksum(conf http_replication.ReplicationItem, hash string) bool {
	return len(conf.ReplicationConf.Sha256Sum) > 0 && hash != conf.ReplicationConf.Sha256Sum
}

func hashContent(data []byte) string {
	hasher := sha256.New()
	hasher.Write(bytes.TrimSpace(data))
	hashBytes := hasher.Sum(nil)
	hashString := hex.EncodeToString(hashBytes)
	return hashString
}

func updateMetricsHash(id string, data []byte) {
	h := fnv.New64a()
	_, _ = h.Write(data)
	metrics.HttpReplicationFileHash.WithLabelValues(id).Set(float64(h.Sum64()))
}
