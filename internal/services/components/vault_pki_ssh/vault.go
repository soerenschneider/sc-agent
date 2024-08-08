package vault_pki_ssh

import (
	"cmp"
	"context"
	"fmt"
	"strings"

	vault "github.com/hashicorp/vault/api"
	"github.com/soerenschneider/sc-agent/internal/domain"
	"github.com/soerenschneider/vault-ssh-cli/pkg/ssh"
	"github.com/spf13/afero"
	"go.uber.org/multierr"
)

const (
	defaultTtl = "4h"
)

type VaultClient interface {
	SignKeyWithContext(ctx context.Context, role string, reqData map[string]any) (*vault.Secret, error)
}

type VaultSshClient struct {
	client VaultClient
}

type VaultSshSignerOpts func(v *VaultSshClient) error

func NewVaultSshClient(client VaultClient, opts ...VaultSshSignerOpts) (*VaultSshClient, error) {
	ret := &VaultSshClient{
		client: client,
	}

	var errs error
	for _, opt := range opts {
		if err := opt(ret); err != nil {
			errs = multierr.Append(errs, err)
		}
	}

	return ret, errs
}

func parseCertificateData(fsImpl afero.Fs, certificateFile string) (*ssh.CertInfo, error) {
	exists, err := afero.Exists(fsImpl, certificateFile)
	if !exists || err != nil {
		return nil, domain.ErrPkiSshCertificateNotFound
	}

	certData, err := afero.ReadFile(fsImpl, certificateFile)
	if err != nil {
		return nil, domain.ErrPkiSshCertificateNotFound
	}

	cert, err := ssh.ParseCertData(certData)
	if err != nil {
		return nil, domain.ErrPkiSshBadCertificate
	}

	return &cert, nil
}

func (v *VaultSshClient) SignSshPublicKey(ctx context.Context, publicKeyData []byte, req domain.SshSignatureRequest) (string, error) {
	reqData := map[string]interface{}{
		"public_key":       string(publicKeyData),
		"valid_principals": strings.Join(req.Principals, ","),
		"cert_type":        req.CertType,
		"ttl":              cmp.Or(req.Ttl, defaultTtl),
		"critical_options": req.CriticalOptions,
		"extensions":       req.Extensions,
	}

	resp, err := v.client.SignKeyWithContext(ctx, req.Role, reqData)
	if err != nil {
		return "", fmt.Errorf("could not sign public key: %w", err)
	}

	if resp == nil || resp.Data == nil {
		return "", fmt.Errorf("empty response: %w", domain.ErrVaultInvalidResponse)
	}

	signedKeyData, found := resp.Data["signed_key"]
	if !found {
		return "", fmt.Errorf("response is missing signed_key data: %w", domain.ErrVaultInvalidResponse)
	}

	signedKey, conversionOk := signedKeyData.(string)
	if !conversionOk {
		return "", fmt.Errorf("could not convert signed_key data to string: %w", domain.ErrVaultInvalidResponse)
	}

	return signedKey, nil
}
