package vault_pki_ssh

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/domain"
	"github.com/soerenschneider/vault-ssh-cli/pkg/ssh"
	"github.com/spf13/afero"
	"go.uber.org/multierr"
)

const (
	sshSignerComponent  = "ssh-signer"
	percentageThreshold = 50
)

type SshClient interface {
	SignSshPublicKey(ctx context.Context, publicKeyData []byte, req domain.SshSignatureRequest) (string, error)
}

type SshService struct {
	once   sync.Once
	fsImpl afero.Fs
	client SshClient

	users map[string]domain.SshSignatureRequest
	hosts map[string]domain.SshSignatureRequest
}

func NewSshService(client SshClient, users, hosts map[string]domain.SshSignatureRequest) (*SshService, error) {
	return &SshService{
		fsImpl: afero.NewOsFs(),
		client: client,
		users:  users,
		hosts:  hosts,
	}, nil
}

func (v *SshService) SignAndUpdateCert(ctx context.Context, req domain.SshSignatureRequest, forceNewCert bool) (*domain.SshSignatureResult, error) {
	ret := &domain.SshSignatureResult{
		Action:          domain.ActionNewCertificate,
		PublicKeyFile:   req.PublicKeyFile,
		CertificateFile: req.GetCertificateFile(),
	}

	certData, err := parseCertificateData(v.fsImpl, req.GetCertificateFile())
	if err != nil && !errors.Is(err, domain.ErrPkiSshCertificateNotFound) {
		log.Error().Str("component", sshSignerComponent).Str("public_key", req.PublicKeyFile).Err(err).Msg("could not parse existing certificate, requesting new signature")
	} else {
		ret.CertData = certData
		percentage := certData.GetPercentage()
		if percentage > percentageThreshold && !forceNewCert {
			log.Info().Str("component", sshSignerComponent).Str("public_key", req.PublicKeyFile).Float32("percentage", percentage).Msg("not requesting new cert")
			ret.Action = domain.ActionNotUpdate
			return ret, nil
		}
	}

	publicKeyData, err := afero.ReadFile(v.fsImpl, req.PublicKeyFile)
	if err != nil {
		return nil, domain.ErrPkiSshPubkeyNotFound
	}
	signedCertData, err := v.client.SignSshPublicKey(ctx, publicKeyData, req)
	if err != nil {
		return nil, err
	}

	certInfo, err := ssh.ParseCertData([]byte(signedCertData))
	if err == nil {
		ret.CertData = &certInfo
	}

	return ret, afero.WriteFile(v.fsImpl, req.GetCertificateFile(), []byte(signedCertData), 0644)
}

func (v *SshService) WatchCertificates(ctx context.Context) {
	v.once.Do(func() {
		if len(v.hosts)+len(v.users) == 0 {
			log.Info().Str("component", sshSignerComponent).Msg("No certificates defined, not scheduling auto-renewals")
			return
		}

		ticker := time.NewTicker(10 * time.Minute)
		v.autoRenew(ctx)

		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				v.autoRenew(ctx)
			}
		}
	})
}

func (v *SshService) GetHostRequest(id string) (domain.SshSignatureRequest, error) {
	req, found := v.hosts[id]
	if !found {
		return domain.SshSignatureRequest{}, domain.ErrPkiSshCertificateNotFound
	}
	return req, nil
}

func (v *SshService) GetUserRequest(id string) (domain.SshSignatureRequest, error) {
	req, found := v.users[id]
	if !found {
		return domain.SshSignatureRequest{}, domain.ErrPkiSshCertificateNotFound
	}
	return req, nil
}

func (v *SshService) autoRenew(ctx context.Context) {
	log.Info().Str("component", sshSignerComponent).Msg("Auto-renewing ssh certificates")
	seen := 0
	var errs error
	for _, req := range v.users {
		seen++
		select {
		case <-ctx.Done():
			return
		default:
			_, err := v.SignAndUpdateCert(ctx, req, false)
			if err != nil {
				errs = multierr.Append(errs, err)
			}
		}
	}

	for _, req := range v.hosts {
		seen++
		select {
		case <-ctx.Done():
			return
		default:
			_, err := v.SignAndUpdateCert(ctx, req, false)
			if err != nil {
				errs = multierr.Append(errs, err)
			}
		}
	}

	log.Info().Str("component", sshSignerComponent).Msgf("Finished auto-renwing %d ssh certificates", seen)
	if errs != nil {
		log.Error().Str("component", sshSignerComponent).Err(errs).Msg("encountered error(s) while automatically renewing certificates")
	}
}
