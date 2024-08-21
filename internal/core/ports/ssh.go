package ports

import (
	"context"

	"github.com/soerenschneider/sc-agent/internal/domain/ssh"
)

type SshPki interface {
	SignAndUpdateCert(ctx context.Context, req ssh.ManagedCertificateConfig, force bool) (*ssh.RequestCertificateResult, error)
	WatchCertificates(ctx context.Context)

	GetManagedCertificateConfig(id string) (ssh.ManagedCertificateConfig, error)
	GetManagedCertificatesConfigs() []ssh.ManagedCertificateConfig
}
