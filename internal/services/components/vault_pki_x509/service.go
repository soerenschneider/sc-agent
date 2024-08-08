package vault_x509

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/config/vault"
	"github.com/soerenschneider/sc-agent/internal/domain"
	"github.com/soerenschneider/sc-agent/pkg/pki"
	stores "github.com/soerenschneider/sc-agent/pkg/pki/x509_repo"
	"github.com/soerenschneider/sc-agent/pkg/pki/x509_repo/storage_backend"
	"github.com/soerenschneider/vault-pki-cli/pkg"
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

type X509Client interface {
	Issue(ctx context.Context, req domain.CertConfig) (*pki.CertData, error)
	ReadCa(ctx context.Context, binary bool) ([]byte, error)
	ReadCaChain(ctx context.Context) ([]byte, error)
	ReadCrl(ctx context.Context, binary bool) ([]byte, error)
}

type X509Service struct {
	client                 X509Client
	minPercentageThreshold float64
	managedCerts           map[string]domain.CertConfig
	certStorage            map[string]X509CertStore
	checkInterval          time.Duration
	once                   sync.Once
}

func NewService(client X509Client, conf vault.X509Pki) (*X509Service, error) {
	certStorage := map[string]X509CertStore{}
	var errs error
	for key, cert := range conf.ManagedCerts {
		storage, err := buildCertStorage(cert.Storage)
		if err != nil {
			errs = multierr.Append(errs, err)
			continue
		}
		certStorage[key] = storage
	}

	return &X509Service{
		client:                 client,
		minPercentageThreshold: defaultPercentageThreshold,
		managedCerts:           conf.GetManagedCerts(),
		certStorage:            certStorage,
		checkInterval:          defaultCheckInterval,
	}, errs
}

func (s *X509Service) GetManagedCert(id string) (*domain.X509CertInfo, error) {
	storage, ok := s.certStorage[id]
	if !ok {
		return nil, ErrCertConfigNotFound
	}

	cert, err := storage.ReadCert()
	if err != nil {
		return nil, err
	}

	return &domain.X509CertInfo{
		Issuer: domain.Issuer{
			CommonName:   cert.Issuer.CommonName,
			SerialNumber: cert.Issuer.SerialNumber,
		},
		Subject:        cert.Subject.CommonName,
		Serial:         pkg.FormatSerial(cert.SerialNumber),
		NotBefore:      cert.NotBefore,
		NotAfter:       cert.NotAfter,
		EmailAddresses: cert.EmailAddresses,
	}, nil
}

func (v *X509Service) GetManagedCertConfig(id string) (domain.CertConfig, error) {
	conf, found := v.managedCerts[id]
	if !found {
		return domain.CertConfig{}, ErrCertConfigNotFound
	}

	return conf, nil
}

func (v *X509Service) Start(ctx context.Context) {
	v.once.Do(func() {
		if len(v.managedCerts) == 0 {
			log.Info().Str("component", pkiServiceComponent).Msg("No certificates defined, not scheduling auto-renewals")
			return
		}

		jitter := 2 * time.Minute
		checkInterval := v.checkInterval - (jitter / 2)
		ticker := time.NewTicker(checkInterval)
		v.autoRenew(ctx)

		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				time.Sleep(rand.N(jitter))
				v.autoRenew(ctx)
			}
		}
	})
}

func (v *X509Service) autoRenew(ctx context.Context) {
	log.Info().Str("component", pkiServiceComponent).Msg("Auto-renewing pki certificates")
	seen := 0
	var errs error
	for _, req := range v.managedCerts {
		seen++
		select {
		case <-ctx.Done():
			return
		default:
			err := v.Issue(ctx, req)
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

func (s *X509Service) ReadCa(ctx context.Context) ([]byte, error) {
	return s.client.ReadCa(ctx, false)
}

func (s *X509Service) Issue(ctx context.Context, certConf domain.CertConfig) error {
	storage, ok := s.certStorage[certConf.Id]
	if !ok {
		return ErrCertConfigNotFound
	}

	issueNewCertificate, err := s.shouldIssueNewCertificate(ctx, storage)
	if err == nil && !issueNewCertificate {
		log.Info().Str(logComponent, pkiServiceComponent).Str(logCommonName, certConf.CommonName).Str(logAction, "nop").Msg("Cert exists and does not need a renewal")
		return nil
	} else {
		log.Info().Str(logComponent, pkiServiceComponent).Str(logCommonName, certConf.CommonName).Str(logAction, "issuing").Err(err).Msg("Going to issue certificate")
	}

	req := domain.CertConfig{
		Role:       certConf.Role,
		CommonName: certConf.CommonName,
		Ttl:        certConf.Ttl,
		AltNames:   certConf.AltNames,
		IpSans:     certConf.IpSans,
	}

	cert, err := s.client.Issue(ctx, req)
	if err != nil {
		return err
	}
	x509Cert, err := pkg.ParseCertPem(cert.Certificate)
	if err != nil {
		log.Error().Str("component", pkiServiceComponent).Str(logCommonName, certConf.CommonName).Msgf("Could not parse certificate data: %v", err)
	} else {
		log.Info().Str(logComponent, pkiServiceComponent).Str(logCommonName, certConf.CommonName).Str(logAction, "issued").Int64(logExpiration, x509Cert.NotAfter.Unix()).Msgf("New certificate valid until %v (%s)", x509Cert.NotAfter, time.Until(x509Cert.NotAfter).Round(time.Second))
	}

	return storage.WriteCert(cert)
}

func (p *X509Service) shouldIssueNewCertificate(ctx context.Context, sink X509CertStore) (bool, error) {
	cert, err := sink.ReadCert()
	if err != nil || cert == nil {
		if errors.Is(err, storage_backend.ErrNoCertFound) {
			log.Info().Str("component", pkiServiceComponent).Msg("No existing certificate found")
			return true, nil
		} else {
			log.Warn().Str("component", pkiServiceComponent).Msgf("Could not read certificate: %v", err)
			return true, err
		}
	}

	if !pkg.IsCertExpired(*cert) {
		if err := p.Verify(ctx, cert); err != nil {
			return true, fmt.Errorf("cert exists but can not be verified against ca: %w", err)
		}
	}

	return p.isLifetimeExceeded(cert)
}

func (p *X509Service) Verify(ctx context.Context, cert *x509.Certificate) error {
	var caData []byte
	op := func() error {
		var err error
		caData, err = p.client.ReadCaChain(ctx)
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

	log.Info().Str(logComponent, pkiServiceComponent).Str("serial", pkg.FormatSerial(ca.SerialNumber)).Msg("Received CA")
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

func (p *X509Service) isLifetimeExceeded(cert *x509.Certificate) (bool, error) {
	if cert == nil {
		return true, errors.New("empty certificate provided")
	}

	from := cert.NotBefore
	expiry := cert.NotAfter

	secondsTotal := expiry.Sub(from).Seconds()
	durationUntilExpiration := time.Until(expiry)

	percentage := math.Max(0, durationUntilExpiration.Seconds()*100./secondsTotal)
	log.Info().Str(logComponent, pkiServiceComponent).Str(logCommonName, cert.DNSNames[0]).Int64(logExpiration, expiry.Unix()).Msgf("Lifetime at %.2f%%, %s left (valid from '%v', until '%v')", percentage, durationUntilExpiration.Round(time.Second), from, expiry)

	return percentage <= 50, nil
}

func buildCertStorage(storage []vault.CertStorage) (*stores.MultiKeyPairSink, error) {
	var storageDing []*stores.KeyPairSink
	for _, storage := range storage {
		var ca, crt, key stores.StorageImplementation
		var err error
		if len(storage.CaFile) > 0 {
			ca, err = storage_backend.NewFilesystemStorageFromUri(storage.CaFile)
			if err != nil {
				return nil, err
			}
		}

		if len(storage.CertFile) > 0 {
			crt, err = storage_backend.NewFilesystemStorageFromUri(storage.CertFile)
			if err != nil {
				return nil, err
			}
		}

		if len(storage.KeyFile) > 0 {
			key, err = storage_backend.NewFilesystemStorageFromUri(storage.KeyFile)
			if err != nil {
				return nil, err
			}
		}

		sink, err := stores.NewKeyPairSink(crt, key, ca)
		if err != nil {
			return nil, err
		}

		storageDing = append(storageDing, sink)
	}

	return stores.NewMultiKeyPairSink(storageDing...)
}
