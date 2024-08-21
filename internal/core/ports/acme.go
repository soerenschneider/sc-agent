package ports

import (
	"context"

	"github.com/soerenschneider/sc-agent/internal/domain/x509"
)

type Acme interface {
	WatchCertificates(ctx context.Context)
	ReadAcme(ctx context.Context, config x509.ManagedCertificateConfig) error
	GetManagedCertificateConfig(id string) (x509.ManagedCertificateConfig, error)
	GetManagedCertificateConfigs() ([]x509.ManagedCertificateConfig, error)
}
