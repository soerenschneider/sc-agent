package pki

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/config/vault"
	domain "github.com/soerenschneider/sc-agent/internal/domain/x509"
	"github.com/soerenschneider/sc-agent/internal/metrics"
	stores2 "github.com/soerenschneider/sc-agent/internal/services/components/pki/x509_repo"
	"github.com/soerenschneider/sc-agent/internal/storage"
	"github.com/soerenschneider/sc-agent/pkg"
	"github.com/soerenschneider/sc-agent/pkg/pki"
	"go.uber.org/multierr"
)

const (
	pkiServiceComponent        = "pki-service"
	logCommonName              = "common_name"
	logComponent               = "component"
	logExpiration              = "expiration"
	logAction                  = "action"
	defaultPercentageThreshold = 50
	defaultCheckInterval       = 10 * time.Minute
)

var ErrCertConfigNotFound = errors.New("certificate configuration not found")

type X509CertStore interface {
	WriteCert(cert *pki.CertData) error
	ReadCert() (*x509.Certificate, error)
}

type X509Client interface {
	Issue(ctx context.Context, req domain.CertificateConfig) (*pki.CertData, error)
	ReadCa(ctx context.Context, binary bool) ([]byte, error)
	ReadCaChain(ctx context.Context) ([]byte, error)
	ReadCrl(ctx context.Context, binary bool) ([]byte, error)
}

type Service struct {
	client                 X509Client
	minPercentageThreshold float32
	managedCerts           map[string]domain.ManagedCertificateConfig
	certStorage            map[string]X509CertStore
	checkInterval          time.Duration
	once                   sync.Once
}

func (s *Service) GetManagedCertificateConfig(id string) (domain.ManagedCertificateConfig, error) {
	cert, found := s.managedCerts[id]
	if !found {
		return domain.ManagedCertificateConfig{}, ErrCertConfigNotFound
	}

	certStorage, found := s.certStorage[id]
	if !found {
		return domain.ManagedCertificateConfig{}, errors.New("storage not found")
	}

	certificate, err := certStorage.ReadCert()
	if err != nil {
		return domain.ManagedCertificateConfig{}, err
	}

	parsed := domain.ParseX509Certificate(*certificate)
	cert.Certificate = &parsed

	return cert, nil
}

func NewService(client X509Client, conf vault.X509Pki) (*Service, error) {
	certStorage := map[string]X509CertStore{}
	var errs error
	for _, cert := range conf.ManagedCerts {
		storage, err := buildCertStorage(cert.Storage)
		if err != nil {
			errs = multierr.Append(errs, err)
			continue
		}
		certStorage[cert.Id] = storage
	}

	managedCerts := map[string]domain.ManagedCertificateConfig{}
	for _, cert := range conf.ManagedCerts {
		managedCerts[cert.Id] = cert.ToDomainModel()
	}

	return &Service{
		client:                 client,
		minPercentageThreshold: defaultPercentageThreshold,
		managedCerts:           managedCerts,
		certStorage:            certStorage,
		checkInterval:          defaultCheckInterval,
	}, errs
}

func (s *Service) WatchCertificates(ctx context.Context) {
	s.once.Do(func() {
		if len(s.managedCerts) == 0 {
			log.Warn().Str("component", pkiServiceComponent).Msg("no certificates defined, not scheduling auto-renewals")
			return
		}

		log.Info().Str(logComponent, pkiServiceComponent).Msgf("start replication of %d certs", len(s.managedCerts))
		jitter := 5 * time.Minute
		checkInterval := s.checkInterval - (jitter / 2)
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
	seen := 0
	var errs error
	for _, req := range s.managedCerts {
		seen++
		select {
		case <-ctx.Done():
			return
		default:
			err := s.Issue(ctx, req)
			if err != nil {
				errs = multierr.Append(errs, err)
			}
		}
	}

	log.Info().Str("component", pkiServiceComponent).Msgf("Finished auto-renewing %d pki certificates", seen)
	if errs != nil {
		log.Error().Str("component", pkiServiceComponent).Err(errs).Msg("encountered error(s) while automatically renewing certificates")
	}
}

func (s *Service) ReadCa(ctx context.Context) ([]byte, error) {
	return s.client.ReadCa(ctx, false)
}

func (s *Service) Issue(ctx context.Context, conf domain.ManagedCertificateConfig) error {
	if conf.StorageConfig == nil || conf.CertificateConfig == nil {
		metrics.PkiErrors.WithLabelValues("unknown", "invalid_request").Inc()
		return errors.New("invalid request")
	}

	metrics.PkiReadRequests.WithLabelValues(conf.CertificateConfig.Id).Inc()
	metrics.PkiRequestTimestamp.WithLabelValues(conf.CertificateConfig.Id).SetToCurrentTime()

	storage, ok := s.certStorage[conf.CertificateConfig.Id]
	if !ok {
		metrics.PkiErrors.WithLabelValues(conf.CertificateConfig.Id, "no_storage").Inc()
		return ErrCertConfigNotFound
	}

	issueNewCertificate, err := s.shouldIssueNewCertificate(ctx, storage)
	if err == nil && !issueNewCertificate {
		log.Info().Str(logComponent, pkiServiceComponent).Str(logCommonName, conf.CertificateConfig.CommonName).Str(logAction, "nop").Msg("cert exists and does not need a renewal")
		return nil
	} else {
		log.Info().Str(logComponent, pkiServiceComponent).Str(logCommonName, conf.CertificateConfig.CommonName).Str(logAction, "issuing").Err(err).Msg("issuing new certificate")
	}

	cert, err := s.client.Issue(ctx, getRequest(conf))
	if err != nil {
		metrics.PkiErrors.WithLabelValues(conf.CertificateConfig.Id, "issue").Inc()
		return err
	}

	x509Cert, err := pki.ParseCertPem(cert.Certificate)
	if err != nil {
		metrics.PkiErrors.WithLabelValues(conf.CertificateConfig.Id, "parse_cert").Inc()
		log.Error().Str("component", pkiServiceComponent).Str(logCommonName, conf.CertificateConfig.CommonName).Msgf("could not parse certificate data: %v", err)
	} else {
		metrics.PkiCertPercent.WithLabelValues(conf.CertificateConfig.Id).Set(float64(pkg.GetPercentage(x509Cert.NotBefore, x509Cert.NotAfter)))
		metrics.PkiExpirationDate.WithLabelValues(conf.CertificateConfig.Id).Set(float64(x509Cert.NotAfter.Unix()))
		log.Info().Str(logComponent, pkiServiceComponent).Str(logCommonName, conf.CertificateConfig.CommonName).Str(logAction, "issued").Int64(logExpiration, x509Cert.NotAfter.Unix()).Msgf("issued certificate valid until %v (%s)", x509Cert.NotAfter, time.Until(x509Cert.NotAfter).Round(time.Second))
	}

	if err := storage.WriteCert(cert); err != nil {
		metrics.PkiErrors.WithLabelValues(conf.CertificateConfig.Id, "write_cert").Inc()
		return err
	}

	if len(conf.PostHooks) > 0 {
		metrics.PkiErrors.WithLabelValues(conf.CertificateConfig.Id, "run_hooks").Inc()
		return pkg.RunPostIssueHooks(conf.PostHooks)
	}

	return nil
}

func getRequest(conf domain.ManagedCertificateConfig) domain.CertificateConfig {
	return domain.CertificateConfig{
		Role:       conf.CertificateConfig.Role,
		CommonName: conf.CertificateConfig.CommonName,
		Ttl:        conf.CertificateConfig.Ttl,
		AltNames:   conf.CertificateConfig.AltNames,
		IpSans:     conf.CertificateConfig.IpSans,
	}
}

func (s *Service) shouldIssueNewCertificate(ctx context.Context, sink X509CertStore) (bool, error) {
	cert, err := sink.ReadCert()
	if err != nil || cert == nil {
		if errors.Is(err, storage.ErrNoCertFound) {
			log.Info().Str("component", pkiServiceComponent).Msg("No existing certificate found")
			return true, nil
		} else {
			log.Warn().Str("component", pkiServiceComponent).Msgf("Could not read certificate: %v", err)
			return true, err
		}
	}

	metrics.PkiCertPercent.WithLabelValues(cert.Subject.CommonName).Set(float64(pkg.GetPercentage(cert.NotBefore, cert.NotAfter)))
	metrics.PkiExpirationDate.WithLabelValues(cert.Subject.CommonName).Set(float64(cert.NotAfter.Unix()))

	if !pki.IsCertExpired(*cert) {
		if err := s.Verify(ctx, cert); err != nil {
			return true, fmt.Errorf("cert exists but can not be verified against ca: %w", err)
		}
	}

	return s.isLifetimeExceeded(cert)
}

func (s *Service) GetManagedCertificatesConfigs() []domain.ManagedCertificateConfig {
	ret := make([]domain.ManagedCertificateConfig, len(s.managedCerts))
	idx := 0
	for key := range s.managedCerts {
		ret[idx], _ = s.GetManagedCertificateConfig(key)
		idx++
	}

	return ret
}

func (s *Service) Verify(ctx context.Context, cert *x509.Certificate) error {
	var caData []byte
	op := func() error {
		var err error
		caData, err = s.client.ReadCaChain(ctx)
		return err
	}

	backoffImpl := backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 3)
	if err := backoff.Retry(op, backoffImpl); err != nil {
		return err
	}

	caBlock, _ := pem.Decode(caData)
	ca, err := x509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		return err
	}

	log.Info().Str(logComponent, pkiServiceComponent).Str("serial", pki.FormatSerial(ca.SerialNumber)).Msg("Received CA")
	return verifyCertAgainstCa(cert, ca)
}

func verifyCertAgainstCa(cert, ca *x509.Certificate) error {
	if cert == nil || ca == nil {
		return errors.New("empty cert(s) supplied")
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(ca)

	verifyOptions := x509.VerifyOptions{
		Roots: certPool,
	}
	_, err := cert.Verify(verifyOptions)
	return err
}

func (s *Service) isLifetimeExceeded(cert *x509.Certificate) (bool, error) {
	if cert == nil {
		return true, errors.New("empty certificate provided")
	}

	percentage := pkg.GetPercentage(cert.NotBefore, cert.NotAfter)
	log.Info().Str(logComponent, pkiServiceComponent).Str(logCommonName, cert.DNSNames[0]).Int64(logExpiration, cert.NotAfter.Unix()).Msgf("Lifetime at %.2f%%, %s left (valid from '%v', until '%v')", percentage, time.Until(cert.NotAfter).Round(time.Second), cert.NotBefore, cert.NotAfter)

	return percentage <= s.minPercentageThreshold, nil
}

func buildCertStorage(storageConf []vault.CertStorage) (*stores2.MultiKeyPairSink, error) {
	var storageDing []*stores2.KeyPairSink
	for _, conf := range storageConf {
		var ca, caChain, crt, key stores2.StorageImplementation
		var err error
		if len(conf.CaFile) > 0 {
			ca, err = storage.NewFilesystemStorageFromUri(conf.CaFile)
			if err != nil {
				return nil, err
			}
		}

		if len(conf.CaChainFile) > 0 {
			caChain, err = storage.NewFilesystemStorageFromUri(conf.CaChainFile)
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

		sink, err := stores2.NewKeyPairSink(crt, key, ca, caChain)
		if err != nil {
			return nil, err
		}

		storageDing = append(storageDing, sink)
	}

	return stores2.NewMultiKeyPairSink(storageDing...)
}
