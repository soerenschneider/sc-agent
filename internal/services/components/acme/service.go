package acme

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/config/vault"
	domain "github.com/soerenschneider/sc-agent/internal/domain/x509"
	"github.com/soerenschneider/sc-agent/internal/metrics"
	pki2 "github.com/soerenschneider/sc-agent/internal/services/components/pki"
	x509_repo "github.com/soerenschneider/sc-agent/internal/services/components/pki/x509_repo"
	"github.com/soerenschneider/sc-agent/internal/storage"
	"github.com/soerenschneider/sc-agent/pkg"
	"github.com/soerenschneider/sc-agent/pkg/pki"
	"go.uber.org/multierr"
)

const (
	acmeServiceComponent       = "acme-service"
	logCommonName              = "common_name"
	logComponent               = "component"
	logExpiration              = "expiration"
	logAction                  = "action"
	defaultPercentageThreshold = 50
	defaultCheckInterval       = 10 * time.Minute
)

var ErrCertConfigNotFound = errors.New("certificate configuration not found")

type AcmeClient interface {
	ReadAcme(ctx context.Context, commonName string) (*pki.CertData, error)
}

type Service struct {
	client                 AcmeClient
	minPercentageThreshold float64
	managedCerts           map[string]domain.ManagedCertificateConfig
	once                   sync.Once
	certStorage            map[string]pki2.X509CertStore
	cached                 map[string]string
	interval               time.Duration
}

func (s *Service) GetManagedCertificateConfig(id string) (domain.ManagedCertificateConfig, error) {
	cert, found := s.managedCerts[id]
	if !found {
		return domain.ManagedCertificateConfig{}, ErrCertConfigNotFound
	}

	return cert, nil
}

func (s *Service) GetManagedCertificateConfigs() ([]domain.ManagedCertificateConfig, error) {
	ret := make([]domain.ManagedCertificateConfig, len(s.managedCerts))

	idx := 0
	for key := range s.managedCerts {
		ret[idx], _ = s.GetManagedCertificateConfig(key)
		idx++
	}

	return ret, nil
}

func NewService(client AcmeClient, conf vault.Acme) (*Service, error) {
	certStorage := map[string]pki2.X509CertStore{}
	var errs error
	for _, cert := range conf.ManagedCerts {
		storage, err := buildCertStorage(cert.Storage)
		if err != nil {
			errs = multierr.Append(errs, err)
			continue
		}
		certStorage[cert.CommonName] = storage
	}

	managedCerts := map[string]domain.ManagedCertificateConfig{}
	for _, cert := range conf.ManagedCerts {
		managedCerts[cert.CommonName] = cert.ToDomainModel()
	}

	return &Service{
		client:                 client,
		minPercentageThreshold: defaultPercentageThreshold,
		managedCerts:           managedCerts,
		certStorage:            certStorage,
		interval:               defaultCheckInterval,
		cached:                 map[string]string{},
	}, errs
}

func (s *Service) ReadAcme(ctx context.Context, managedCertConfig domain.ManagedCertificateConfig) error {
	if managedCertConfig.CertificateConfig == nil {
		metrics.AcmeErrors.WithLabelValues("unknown", "invalid_req").Inc()
		return errors.New("invalid request")
	}

	commonName := managedCertConfig.CertificateConfig.CommonName
	metrics.AcmeReadRequests.WithLabelValues(commonName).Inc()
	metrics.AcmeRequestsTimestamp.WithLabelValues(commonName).SetToCurrentTime()

	storage, ok := s.certStorage[commonName]
	if !ok {
		metrics.AcmeErrors.WithLabelValues(commonName, "no_storage").Inc()
		return ErrCertConfigNotFound
	}

	cert, err := s.client.ReadAcme(ctx, commonName)
	if err != nil {
		metrics.AcmeErrors.WithLabelValues(commonName, "read_cert_vault").Inc()
		return err
	}

	x509Cert, err := pki.ParseCertPem(cert.Certificate)
	if err != nil {
		metrics.AcmeErrors.WithLabelValues(commonName, "parse_cert").Inc()
		log.Error().Err(err).Str(logComponent, acmeServiceComponent).Str(logCommonName, commonName).Msg("could not parse cert data read from Vault")
		return err
	}

	metrics.AcmeExpirationDate.WithLabelValues(commonName).Set(float64(x509Cert.NotAfter.Unix()))
	metrics.AcmeCertPercent.WithLabelValues(commonName).Set(float64(pkg.GetPercentage(x509Cert.NotBefore, x509Cert.NotAfter)))
	log.Info().Str(logComponent, acmeServiceComponent).Str(logCommonName, commonName).Str(logAction, "read").Int64(logExpiration, x509Cert.NotAfter.Unix()).Msgf("certificate valid until %v (%s)", x509Cert.NotAfter, time.Until(x509Cert.NotAfter).Round(time.Second))

	_, found := s.cached[commonName]
	if !found {
		existingCert, err := storage.ReadCert()
		if err == nil {
			s.cached[commonName] = hash(existingCert.Raw)
		}
	}

	certHash := hash(x509Cert.Raw)
	oldHash, found := s.cached[commonName]
	if found && oldHash == certHash {
		return nil
	}

	s.cached[commonName] = certHash

	log.Info().Str(logComponent, acmeServiceComponent).Str(logCommonName, commonName).Msg("writing cert data")
	if err := storage.WriteCert(cert); err != nil {
		metrics.AcmeErrors.WithLabelValues(commonName, "write_cert").Inc()
		return fmt.Errorf("could not write acme cert to disk: %w", err)
	}

	if len(managedCertConfig.PostHooks) > 0 {
		err := pkg.RunPostIssueHooks(managedCertConfig.PostHooks)
		if err != nil {
			metrics.AcmeErrors.WithLabelValues(commonName, "run_hooks").Inc()
		}
		return err
	}

	return nil
}

func (s *Service) WatchCertificates(ctx context.Context) {
	s.once.Do(func() {
		if len(s.managedCerts) == 0 {
			log.Warn().Str(logComponent, acmeServiceComponent).Msg("no certificates defined, not scheduling auto-renewals")
			return
		}

		log.Info().Str(logComponent, acmeServiceComponent).Msgf("start replication of %d items", len(s.managedCerts))
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
	for _, req := range s.managedCerts {
		select {
		case <-ctx.Done():
			return
		default:
			if err := s.ReadAcme(ctx, req); err != nil {
				log.Error().Err(err).Str(logComponent, acmeServiceComponent).Str(logCommonName, req.CertificateConfig.CommonName).Msg("error while handling acme certificate")
			}
		}
	}
}

func buildCertStorage(storageConf []vault.CertStorage) (*x509_repo.MultiKeyPairSink, error) {
	var storageDing []*x509_repo.KeyPairSink
	for _, conf := range storageConf {
		var ca, crt, key x509_repo.StorageImplementation
		var err error
		if len(conf.CaFile) > 0 {
			ca, err = storage.NewFilesystemStorageFromUri(conf.CaFile)
			if err != nil {
				return nil, err
			}
		}

		if len(conf.CertFile) > 0 {
			crt, err = storage.NewFilesystemStorageFromUri(conf.CertFile)
			if err != nil {
				return nil, err
			}
		}

		if len(conf.KeyFile) > 0 {
			key, err = storage.NewFilesystemStorageFromUri(conf.KeyFile)
			if err != nil {
				return nil, err
			}
		}

		sink, err := x509_repo.NewKeyPairSink(crt, key, ca)
		if err != nil {
			return nil, err
		}

		storageDing = append(storageDing, sink)
	}

	return x509_repo.NewMultiKeyPairSink(storageDing...)
}

func hash(data []byte) string {
	hasher := sha256.New()
	hasher.Write(data)
	hashBytes := hasher.Sum(nil)
	hashString := hex.EncodeToString(hashBytes)
	return hashString
}
