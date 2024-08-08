package ports

import (
	"context"

	"github.com/soerenschneider/sc-agent/internal/domain"
)

type SshPki interface {
	SignAndUpdateCert(ctx context.Context, req domain.SshSignatureRequest, force bool) (*domain.SshSignatureResult, error)
	GetHostRequest(id string) (domain.SshSignatureRequest, error)
	GetUserRequest(id string) (domain.SshSignatureRequest, error)
	WatchCertificates(ctx context.Context)
}

type X509Pki interface {
	Issue(ctx context.Context, certConf domain.CertConfig) error
	ReadCa(ctx context.Context) ([]byte, error)
	Start(ctx context.Context)
	GetManagedCertConfig(id string) (domain.CertConfig, error)
	GetManagedCert(id string) (*domain.X509CertInfo, error)
}
