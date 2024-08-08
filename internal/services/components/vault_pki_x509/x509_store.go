package vault_x509

import (
	"crypto/x509"

	"github.com/soerenschneider/sc-agent/pkg/pki"
)

type X509CertStore interface {
	WriteCert(cert *pki.CertData) error
	ReadCert() (*x509.Certificate, error)
}

type X509CrlStore interface {
	WriteCrl(crlData []byte) error
}

type X509CsrStore interface {
	ReadCsr() ([]byte, error)
	WriteSignature(cert *pki.Signature) error
}

type X509CaStore interface {
	WriteCa(certData []byte) error
}
