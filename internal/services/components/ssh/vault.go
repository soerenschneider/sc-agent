package ssh

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	vault "github.com/hashicorp/vault/api"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/domain"
	"github.com/soerenschneider/sc-agent/internal/domain/ssh"
	"github.com/spf13/afero"
)

const (
	defaultTtl = "4h"
)

type VaultClient interface {
	SignKeyWithContext(ctx context.Context, role string, reqData map[string]any) (*vault.Secret, error)
	ReadRawWithContext(ctx context.Context, path string) (*vault.Response, error)
}

type VaultSshClient struct {
	client    VaultClient
	mountPath string
}

type VaultSshSignerOpts func(v *VaultSshClient) error

func NewVaultClient(client VaultClient, mountPath string) (*VaultSshClient, error) {
	if client == nil {
		return nil, errors.New("emtpy client passed")
	}

	if len(mountPath) == 0 {
		return nil, errors.New("empty mount path")
	}

	ret := &VaultSshClient{
		client:    client,
		mountPath: mountPath,
	}

	return ret, nil
}

func parseCertificateData(fsImpl afero.Fs, certificateFile string) (*ssh.Certificate, error) {
	exists, err := afero.Exists(fsImpl, certificateFile)
	if !exists || err != nil {
		return nil, ssh.ErrPkiSshCertificateNotFound
	}

	certData, err := afero.ReadFile(fsImpl, certificateFile)
	if err != nil {
		return nil, ssh.ErrPkiSshCertificateNotFound
	}

	cert, err := ssh.ParseCertData(certData)
	if err != nil {
		return nil, ssh.ErrPkiSshBadCertificate
	}

	return &cert, nil
}

func (v *VaultSshClient) ReadCaData(ctx context.Context) (string, error) {
	path := fmt.Sprintf("%s/public_key", v.mountPath)
	resp, err := v.client.ReadRawWithContext(ctx, path)
	if err != nil {
		return "", fmt.Errorf("reading cert failed: %v", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("could not read body from response: %v", err)
	}

	return string(data), nil
}

func getDefaultTtl(req ssh.CertificateConfig) string {
	if req.CertType == "user" {
		return cmp.Or(req.Ttl, "16h")
	}
	if req.CertType == "host" {
		return cmp.Or(req.Ttl, "30d")
	}

	log.Warn().Str(logComponent, sshSignerComponent).Msgf("invalid certtype specified: %q", req.CertType)
	return defaultTtl
}

func (v *VaultSshClient) SignSshPublicKey(ctx context.Context, publicKeyData []byte, req ssh.CertificateConfig) (string, error) {
	reqData := map[string]interface{}{
		"public_key":       string(publicKeyData),
		"cert_type":        req.CertType,
		"ttl":              getDefaultTtl(req),
		"critical_options": req.CriticalOptions,
		"extensions":       req.Extensions,
	}

	if len(req.Principals) > 0 {
		reqData["valid_principals"] = strings.Join(req.Principals, ",")
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
