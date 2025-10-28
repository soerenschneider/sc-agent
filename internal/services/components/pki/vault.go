package pki

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	vault "github.com/hashicorp/vault/api"
	"github.com/soerenschneider/sc-agent/internal/domain"
	"github.com/soerenschneider/sc-agent/internal/domain/x509"
	"github.com/soerenschneider/sc-agent/pkg/pki"
	"go.uber.org/multierr"
)

const (
	defaultTtl       = "8h"
	defaultMountPath = "pki"
)

type VaultClient interface {
	WriteWithContext(ctx context.Context, path string, reqData map[string]any) (*vault.Secret, error)
	ReadWithContext(ctx context.Context, path string) (*vault.Secret, error)
	ReadRawWithContext(ctx context.Context, path string) (*vault.Response, error)
}

type VaultX509Client struct {
	client    VaultClient
	mountPath string
}

type VaultClientOpts func(v *VaultX509Client) error

func NewVaultClient(client VaultClient, opts ...VaultClientOpts) (*VaultX509Client, error) {
	if client == nil {
		return nil, errors.New("empty x509 vault client passed")
	}

	ret := &VaultX509Client{
		client:    client,
		mountPath: defaultMountPath,
	}

	var errs error
	for _, opt := range opts {
		if err := opt(ret); err != nil {
			errs = multierr.Append(errs, err)
		}
	}

	return ret, errs
}

func (v *VaultX509Client) Issue(ctx context.Context, req x509.CertificateConfig) (*pki.CertData, error) {
	reqData := getVaultIssueRequest(req)
	path := fmt.Sprintf("%s/issue/%s", v.mountPath, req.Role)
	resp, err := v.client.WriteWithContext(ctx, path, reqData)
	if err != nil {
		return nil, fmt.Errorf("could not issue certficate: %w", err)
	}

	if resp == nil || resp.Data == nil {
		return nil, fmt.Errorf("empty response: %w", domain.ErrVaultInvalidResponse)
	}

	privKeyData, found := resp.Data["private_key"]
	if !found {
		return nil, fmt.Errorf("response is missing 'private_key' data: %w", domain.ErrVaultInvalidResponse)
	}

	privKey, conversionOk := privKeyData.(string)
	if !conversionOk {
		return nil, fmt.Errorf("could not convert 'private_key' data to string: %w", domain.ErrVaultInvalidResponse)
	}

	certData, found := resp.Data["certificate"]
	if !found {
		return nil, fmt.Errorf("response is missing 'certificate' data: %w", domain.ErrVaultInvalidResponse)
	}

	cert, conversionOk := certData.(string)
	if !conversionOk {
		return nil, fmt.Errorf("could not convert 'certificate' data to string: %w", domain.ErrVaultInvalidResponse)
	}

	var caChain []string
	caChainData, found := resp.Data["ca_chain"]
	if found {
		switch v := caChainData.(type) {
		case []string:
			caChain = v
		case []any:
			var result []string
			for _, item := range v {
				s, ok := item.(string)
				if !ok {
					return nil, fmt.Errorf("element %v is not a string", item)
				}
				result = append(result, s)
			}
			caChain = result
		default:
			return nil, fmt.Errorf("unsupported type for 'ca_chain': %T", caChainData)
		}
	}

	caData, found := resp.Data["issuing_ca"]
	if !found {
		return nil, fmt.Errorf("response is missing 'issuing_ca' data: %w", domain.ErrVaultInvalidResponse)
	}

	ca, conversionOk := caData.(string)
	if !conversionOk {
		return nil, fmt.Errorf("could not convert 'issuing_ca' data to string: %w", domain.ErrVaultInvalidResponse)
	}

	return &pki.CertData{
		PrivateKey:  []byte(privKey),
		Certificate: []byte(cert),
		CaData:      []byte(ca),
		CaChain:     []byte(strings.Join(caChain, "\n")),
	}, nil
}

func getVaultIssueRequest(req x509.CertificateConfig) map[string]any {
	return map[string]any{
		"common_name": req.CommonName,
		"ttl":         cmp.Or(req.Ttl, defaultTtl),
		"format":      "pem",
		"ip_sans":     strings.Join(req.IpSans, ","),
		"alt_names":   strings.Join(req.AltNames, ","),
	}
}

func (c *VaultX509Client) ReadCa(ctx context.Context, binary bool) ([]byte, error) {
	path := fmt.Sprintf("%s/ca", c.mountPath)
	if !binary {
		path = path + "/pem"
	}

	return c.readRaw(ctx, path)
}

func (c *VaultX509Client) ReadCaChain(ctx context.Context) ([]byte, error) {
	path := fmt.Sprintf("/%s/ca_chain", c.mountPath)
	return c.readRaw(ctx, path)
}

func (c *VaultX509Client) ReadCrl(ctx context.Context, binary bool) ([]byte, error) {
	path := fmt.Sprintf("%s/crl", c.mountPath)
	if !binary {
		path += "/pem"
	}

	return c.readRaw(ctx, path)
}

func (c *VaultX509Client) readRaw(ctx context.Context, path string) ([]byte, error) {
	secret, err := c.client.ReadRawWithContext(ctx, path)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = secret.Body.Close()
	}()

	return io.ReadAll(secret.Body)
}
