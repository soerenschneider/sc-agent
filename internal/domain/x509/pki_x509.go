package x509

import (
	"crypto/x509"
	"math"
	"time"

	"github.com/soerenschneider/sc-agent/internal/domain"
	"github.com/soerenschneider/sc-agent/pkg/pki"
)

type ManagedCertificateConfig struct {
	CertificateConfig *CertificateConfig
	StorageConfig     []CertificateStorage
	PostHooks         []domain.PostHook
	Certificate       *Certificate
}

type CertificateStorage struct {
	CaFile   string `validate:"omitempty" yaml:"ca_file"`
	CertFile string `validate:"omitempty" yaml:"cert_file"`
	KeyFile  string `validate:"omitempty" yaml:"key_file"`
}

type CertificateConfig struct {
	Id         string   `json:"id"`
	Role       string   `json:"role"`
	CommonName string   `json:"common_name"`
	Ttl        string   `json:"ttl"`
	AltNames   []string `json:"alt_names"`
	IpSans     []string `json:"ip_sans"`
}

type Certificate struct {
	Issuer         Issuer    `json:"id"`
	Subject        string    `json:"subject"`
	Serial         string    `json:"serial"`
	EmailAddresses []string  `json:"email_addresses"`
	NotBefore      time.Time `json:"not_before"`
	NotAfter       time.Time `json:"not_after"`
	Percentage     float32   `json:"percentage"`
}

func ParseX509Certificate(certificate x509.Certificate) Certificate {
	from := certificate.NotBefore
	expiry := certificate.NotAfter

	secondsTotal := expiry.Sub(from).Seconds()
	durationUntilExpiration := time.Until(expiry)

	percentage := math.Max(0, durationUntilExpiration.Seconds()*100./secondsTotal)

	return Certificate{
		Issuer: Issuer{
			SerialNumber: certificate.Issuer.SerialNumber,
			CommonName:   certificate.Issuer.CommonName,
		},
		Subject:        certificate.Subject.CommonName,
		Serial:         pki.FormatSerial(certificate.SerialNumber),
		EmailAddresses: certificate.EmailAddresses,
		NotBefore:      certificate.NotBefore,
		NotAfter:       certificate.NotAfter,
		Percentage:     float32(percentage),
	}
}

type Issuer struct {
	SerialNumber string `json:"serial_number"`
	CommonName   string `json:"common_name"`
}
