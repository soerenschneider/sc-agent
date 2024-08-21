package acme

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	vault "github.com/hashicorp/vault/api"
	"github.com/soerenschneider/sc-agent/pkg/pki"
	"go.uber.org/multierr"
)

const (
	defaultTtl        = "8h"
	defaultAcmePrefix = "acmevault/prod"
	defaultMountPath  = "secret"

	// keys of the kv2 secret's map for the respective data
	acmevaultKeyPrivateKey  = "private_key"
	acmevaultKeyCertificate = "cert"
	acmevaultKeyIssuer      = "issuer"
	acmevaultVersion        = "version"

	// the secret name (without the path) of the certificate saved by acmevault
	acmevaultKv2SecretNameCertificate = "certificate"
	// the secret name (without the path) of the private key saved by acmevault
	acmevaultKv2SecretNamePrivatekey = "privatekey"
)

type VaultClient interface {
	ReadWithContext(ctx context.Context, path string) (*vault.Secret, error)
}

type VaultAcmeClient struct {
	client     VaultClient
	mountPath  string
	acmePrefix string
}

type VaultClientOpts func(v *VaultAcmeClient) error

func NewVaultClient(client VaultClient, opts ...VaultClientOpts) (*VaultAcmeClient, error) {
	if client == nil {
		return nil, errors.New("empty x509 vault client passed")
	}

	ret := &VaultAcmeClient{
		client:     client,
		acmePrefix: defaultAcmePrefix,
		mountPath:  defaultMountPath,
	}

	var errs error
	for _, opt := range opts {
		if err := opt(ret); err != nil {
			errs = multierr.Append(errs, err)
		}
	}

	return ret, errs
}

func (c *VaultAcmeClient) ReadAcme(ctx context.Context, commonName string) (*pki.CertData, error) {
	certData, err := c.readAcmeCert(ctx, commonName)
	if err != nil {
		return nil, fmt.Errorf("could not read certificate data: %w", err)
	}

	secretData, err := c.readAcmeSecret(ctx, commonName)
	if err != nil {
		return nil, fmt.Errorf("could not read secret data: %w", err)
	}

	return &pki.CertData{
		PrivateKey:  secretData.PrivateKey,
		Certificate: certData.Certificate,
		CaData:      certData.CaData,
	}, nil
}

func (c *VaultAcmeClient) getAcmevaultDataPath(domain string, leaf string) string {
	prefix := fmt.Sprintf("%s/data/%s", c.mountPath, c.acmePrefix)
	return fmt.Sprintf("%s/client/%s/%s", prefix, domain, leaf)
}

func (c *VaultAcmeClient) readKv2Secret(ctx context.Context, path string) (map[string]interface{}, error) {
	secret, err := c.client.ReadWithContext(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("could not read kv2 data '%s': %w", path, err)
	}
	if secret == nil {
		return nil, errors.New("read kv2 data is nil")
	}

	var data map[string]interface{}
	_, ok := secret.Data["data"]
	if !ok {
		return nil, errors.New("read kv2 secret contains no data")
	}
	data, ok = secret.Data["data"].(map[string]interface{})
	if !ok {
		return nil, errors.New("read kv2 data is malformed")
	}

	return data, nil
}

func (c *VaultAcmeClient) readAcmeCert(ctx context.Context, commonName string) (*pki.CertData, error) {
	path := c.getAcmevaultDataPath(commonName, acmevaultKv2SecretNameCertificate)
	data, err := c.readKv2Secret(ctx, path)
	if err != nil {
		return nil, err
	}

	rawCert, ok := data[acmevaultKeyCertificate]
	if !ok {
		return nil, errors.New("read kv2 secret does not contain certificate data")
	}
	cert, err := base64.StdEncoding.DecodeString(rawCert.(string))
	if err != nil {
		return nil, errors.New("could not base64 decode cert")
	}
	cert = bytes.TrimRight(cert, "\n")

	var version string
	versionRaw, ok := data[acmevaultVersion]
	if ok {
		version = versionRaw.(string)
	}

	var issuer []byte
	if version == "v1" {
		rawIssuer, ok := data[acmevaultKeyIssuer]
		if ok {
			ca, err := base64.StdEncoding.DecodeString(rawIssuer.(string))
			if err == nil {
				issuer = bytes.TrimRight(ca, "\n")
				// TODO: remove support in future, this is apparently a bug in acmevault
				issuer = bytes.TrimLeft(issuer, "\n")
				// TODO end
			}
		}
	} else {
		// TODO: remove support in the future
		rawIssuer, ok := data["dummyIssuer"]
		if ok {
			ca, err := base64.StdEncoding.DecodeString(rawIssuer.(string))
			if err == nil {
				issuer = bytes.TrimRight(ca, "\n")
			}
		}
	}

	return &pki.CertData{Certificate: cert, CaData: issuer}, nil
}

func (c *VaultAcmeClient) readAcmeSecret(ctx context.Context, commonName string) (*pki.CertData, error) {
	path := c.getAcmevaultDataPath(commonName, acmevaultKv2SecretNamePrivatekey)
	data, err := c.readKv2Secret(ctx, path)
	if err != nil {
		return nil, err
	}

	rawKey, ok := data[acmevaultKeyPrivateKey]
	if !ok {
		return nil, errors.New("read kv2 secret does not contain private key data")
	}

	privateKey, err := base64.StdEncoding.DecodeString(rawKey.(string))
	if err != nil {
		return nil, errors.New("could not base64 decode key")
	}

	privateKey = bytes.TrimRight(privateKey, "\n")
	return &pki.CertData{PrivateKey: privateKey}, nil
}
