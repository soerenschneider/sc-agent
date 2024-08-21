package ssh

import (
	"context"
	"errors"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/domain/ssh"
	"github.com/soerenschneider/sc-agent/internal/metrics"
	"github.com/spf13/afero"
	"go.uber.org/multierr"
)

const (
	sshSignerComponent  = "ssh-signer"
	percentageThreshold = 50

	logComponent  = "component"
	logExpiration = "expiration"
	logId         = "id"
	logPubkey     = "pub_key"

	defaultInterval = 10 * time.Minute
)

type Client interface {
	SignSshPublicKey(ctx context.Context, publicKeyData []byte, req ssh.CertificateConfig) (string, error)
	ReadCaData(ctx context.Context) (string, error)
}

type Service struct {
	onceCertManager sync.Once
	fsImpl          afero.Fs
	client          Client
	interval        time.Duration

	managedCertificates map[string]ssh.ManagedCertificateConfig
}

func NewService(client Client, managedKeys map[string]ssh.ManagedCertificateConfig) (*Service, error) {
	return &Service{
		fsImpl:              afero.NewOsFs(),
		client:              client,
		managedCertificates: managedKeys,
		interval:            defaultInterval,
	}, nil
}

func (s *Service) SignAndUpdateCert(ctx context.Context, cert ssh.ManagedCertificateConfig, forceNewCert bool) (*ssh.RequestCertificateResult, error) {
	if cert.CertificateConfig == nil || cert.StorageConfig == nil {
		metrics.SshErrors.WithLabelValues("unknown", "invalid_req").Inc()
		return nil, errors.New("invalid request")
	}
	metrics.SshRequests.WithLabelValues(cert.StorageConfig.PublicKeyFile).Inc()
	metrics.SshRequestTimestamp.WithLabelValues(cert.StorageConfig.PublicKeyFile).SetToCurrentTime()

	ret := &ssh.RequestCertificateResult{
		Action:     ssh.ActionNewCertificate,
		CertConfig: cert.CertificateConfig,
	}

	certData, err := parseCertificateData(s.fsImpl, cert.StorageConfig.CertificateFile)
	if err == nil {
		ret.CertData = certData
		percentage := certData.GetPercentage()
		metrics.SshExpirationDate.WithLabelValues(cert.StorageConfig.PublicKeyFile).Set(float64(certData.ValidBefore.Unix()))
		metrics.SshCertPercent.WithLabelValues(cert.StorageConfig.PublicKeyFile).Set(float64(percentage))
		if percentage > percentageThreshold && !forceNewCert {
			durationUntilExpiration := time.Until(certData.ValidBefore)
			log.Info().Str(logComponent, sshSignerComponent).Str(logId, cert.CertificateConfig.Id).Str(logPubkey, cert.StorageConfig.PublicKeyFile).Int64(logExpiration, certData.ValidBefore.Unix()).Msgf("Lifetime at %.2f%%, %s left (valid from '%v', until '%v')", percentage, durationUntilExpiration.Round(time.Second), certData.ValidAfter, certData.ValidBefore)
			ret.Action = ssh.ActionNotUpdate
			return ret, nil
		}
	}

	log.Info().Str(logComponent, sshSignerComponent).Str(logId, cert.CertificateConfig.Id).Str(logPubkey, cert.StorageConfig.PublicKeyFile).Err(err).Msg("requesting new signature")
	publicKeyData, err := afero.ReadFile(s.fsImpl, cert.StorageConfig.PublicKeyFile)
	if err != nil {
		metrics.SshErrors.WithLabelValues(cert.StorageConfig.PublicKeyFile, "pubkey_not_found").Inc()
		return nil, ssh.ErrPkiSshPubkeyNotFound
	}

	signedCertData, err := s.client.SignSshPublicKey(ctx, publicKeyData, *cert.CertificateConfig)
	if err != nil {
		metrics.SshErrors.WithLabelValues(cert.StorageConfig.PublicKeyFile, "sign_cert").Inc()
		return nil, err
	}

	certInfo, err := ssh.ParseCertData([]byte(signedCertData))
	if err != nil {
		metrics.SshErrors.WithLabelValues(cert.StorageConfig.PublicKeyFile, "parse_cert").Inc()
		return nil, err
	}

	metrics.SshExpirationDate.WithLabelValues(cert.StorageConfig.PublicKeyFile).Set(float64(certInfo.ValidBefore.Unix()))
	metrics.SshCertPercent.WithLabelValues(cert.StorageConfig.PublicKeyFile).Set(float64(certInfo.Percentage))
	ret.CertData = &certInfo

	err = afero.WriteFile(s.fsImpl, cert.StorageConfig.CertificateFile, []byte(signedCertData), 0644)
	if err != nil {
		metrics.SshErrors.WithLabelValues(cert.StorageConfig.PublicKeyFile, "write_cert").Inc()
	}

	return ret, err
}

func (s *Service) ReadCaData(ctx context.Context) (string, error) {
	return s.client.ReadCaData(ctx)
}

func (s *Service) WatchCertificates(ctx context.Context) {
	s.onceCertManager.Do(func() {
		if len(s.managedCertificates) == 0 {
			log.Warn().Str(logComponent, sshSignerComponent).Msg("No certificates defined, not scheduling auto-renewals")
			return
		}

		log.Info().Str("component", sshSignerComponent).Msgf("start replication of %d certs", len(s.managedCertificates))

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

func (s *Service) GetManagedCertificatesConfigs() []ssh.ManagedCertificateConfig {
	ret := make([]ssh.ManagedCertificateConfig, len(s.managedCertificates))
	idx := 0
	for key := range s.managedCertificates {
		ret[idx], _ = s.GetManagedCertificateConfig(key)
		idx++
	}
	return ret
}

func (s *Service) GetManagedCertificateConfig(id string) (ssh.ManagedCertificateConfig, error) {
	cert, found := s.managedCertificates[id]
	if !found {
		return ssh.ManagedCertificateConfig{}, ssh.ErrPkiSshCertificateNotFound
	}

	var err error
	cert.Certificate, err = parseCertificateData(s.fsImpl, cert.StorageConfig.CertificateFile)
	if err != nil {
		log.Error().Str(logComponent, sshSignerComponent).Err(err).Msg("could not parse cert data")
	}

	return cert, nil
}

func (s *Service) autoRenew(ctx context.Context) {
	log.Info().Str(logComponent, sshSignerComponent).Msg("Auto-renewing ssh certificates")
	seen := 0
	var errs error
	for _, req := range s.managedCertificates {
		seen++
		select {
		case <-ctx.Done():
			return
		default:
			_, err := s.SignAndUpdateCert(ctx, req, false)
			if err != nil {
				errs = multierr.Append(errs, err)
			}
		}
	}

	log.Info().Str(logComponent, sshSignerComponent).Msgf("Finished auto-renewing %d ssh certificates", seen)
	if errs != nil {
		log.Error().Str(logComponent, sshSignerComponent).Err(errs).Msg("encountered error(s) while automatically renewing certificates")
	}
}
