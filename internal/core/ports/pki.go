package ports

import (
	"context"

	"github.com/soerenschneider/sc-agent/internal/domain/x509"
)

type X509Pki interface {
	Issue(ctx context.Context, certConf x509.ManagedCertificateConfig) error
	ReadCa(ctx context.Context) ([]byte, error)
	WatchCertificates(ctx context.Context)

	GetManagedCertificateConfig(id string) (x509.ManagedCertificateConfig, error)
	GetManagedCertificatesConfigs() []x509.ManagedCertificateConfig
}
