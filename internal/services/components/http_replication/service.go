package http_replication

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"math/rand/v2"
	"net/http"
	"strings"
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

func (s *Service) Replicate(ctx context.Context, conf http_replication.ReplicationItem) error {
	metrics.HttpReplicationTimestamp.WithLabelValues(conf.ReplicationConf.Id).SetToCurrentTime()
	metrics.HttpReplicationRequests.WithLabelValues(conf.ReplicationConf.Id).Inc()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, conf.ReplicationConf.Source, nil)
	if err != nil {
		return err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		metrics.HttpReplicationErrors.WithLabelValues(conf.ReplicationConf.Id, "request_errors").Inc()
		return err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode/100 != 2 {
		metrics.HttpReplicationErrors.WithLabelValues(conf.ReplicationConf.Id, "request_errors").Inc()
		return fmt.Errorf("wrong status code, expected 2xx got %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		metrics.HttpReplicationErrors.WithLabelValues(conf.ReplicationConf.Id, "data_errors").Inc()
		return err
	}

	if strings.TrimSpace(string(data)) == "" {
		metrics.HttpReplicationErrors.WithLabelValues(conf.ReplicationConf.Id, "data_errors").Inc()
		return errors.New("empty payload")
	}

	return s.updateFile(data, conf)
}

func (s *Service) updateFile(data []byte, conf http_replication.ReplicationItem) error {
	hash := hashContent(data)
	if isMismatchedChecksum(conf, hash) {
		conf.Status = http_replication.InvalidChecksum
		log.Error().Str(logComponent, httpReplicationComponent).Str("hash_expected", conf.ReplicationConf.Sha256Sum).Str("hash_actual", hash).Str("id", conf.ReplicationConf.Id).Msg("invalid hash")
		metrics.HttpReplicationErrors.WithLabelValues(conf.ReplicationConf.Id, "hash_mismatch").Inc()
		return http_replication.ErrMismatchedHash
	}

	oldHash, itemAlreadyCached := s.cache[conf.ReplicationConf.Id]
	if itemAlreadyCached && oldHash == hash {
		// item is already downloaded. let's check if the item on disk has been changed by a 3rd party since our last check.
		diskContent, err := conf.Destination.Read()
		if err == nil {
			diskHash := hashContent(diskContent)
			if diskHash == hash {
				// file exists locally and is identical to the item we downloaded, we're done
				return nil
			}
			log.Info().Str(logComponent, httpReplicationComponent).Str("id", conf.ReplicationConf.Id).Msg("noticed file has changed on disk, proceeding to overwrite")
		}
	}

	s.cache[conf.ReplicationConf.Id] = hash
	updateMetricsHash(conf.ReplicationConf.Id, data)

	if !itemAlreadyCached {
		read, err := conf.Destination.Read()
		if err == nil && hash == hashContent(read) {
			log.Debug().Str(logComponent, httpReplicationComponent).Str("id", conf.ReplicationConf.Id).Msg("file already exists locally")
			return nil
		}
	}

	log.Info().Str(logComponent, httpReplicationComponent).Str("id", conf.ReplicationConf.Id).Msg("writing item to disk")
	if err := conf.Destination.Write(data); err != nil {
		metrics.HttpReplicationErrors.WithLabelValues(conf.ReplicationConf.Id, "write_file").Inc()
		return err
	}

	if err := pkg.RunPostIssueHooks(conf.PostHooks); err != nil {
		metrics.HttpReplicationErrors.WithLabelValues(conf.ReplicationConf.Id, "post_hooks").Inc()
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
