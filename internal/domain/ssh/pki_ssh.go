package ssh

import (
	"errors"
	"fmt"
	"math"
	"time"

	"golang.org/x/crypto/ssh"
)

const (
	ActionNewCertificate = "NewCertificate"
	ActionNotUpdate      = "NotUpdated"
)

var (
	ErrPkiSshConfigNotFound      = errors.New("config not found")
	ErrPkiSshPubkeyNotFound      = errors.New("public key not found")
	ErrPkiSshCertificateNotFound = errors.New("certificate not found")
	ErrPkiSshBadCertificate      = errors.New("malformed certificate data")
)

type ManagedCertificateConfig struct {
	CertificateConfig *CertificateConfig
	StorageConfig     *CertificateStorage
	Certificate       *Certificate
}

type CertificateConfig struct {
	Id              string
	Role            string
	Principals      []string
	Ttl             string
	CertType        string
	CriticalOptions map[string]string
	Extensions      map[string]string
}

type CertificateStorage struct {
	PublicKeyFile   string
	CertificateFile string
}

type Certificate struct {
	Type            string
	Serial          uint64
	ValidAfter      time.Time
	ValidBefore     time.Time
	Principals      []string
	Extensions      map[string]string
	CriticalOptions map[string]string
	Percentage      float32
}

type RequestCertificateResult struct {
	Action     string
	CertConfig *CertificateConfig
	CertData   *Certificate
}

func GetPercentage(from, to time.Time) float32 {
	total := to.Sub(from).Seconds()
	if total == 0 {
		return 0.
	}

	left := time.Until(to).Seconds()
	return float32(math.Max(0, left*100/total))
}

func (l *Certificate) GetPercentage() float32 {
	return GetPercentage(l.ValidAfter, l.ValidBefore)
}

func ParseCertData(pubKeyBytes []byte) (Certificate, error) {
	pubKey, _, _, _, err := ssh.ParseAuthorizedKey(pubKeyBytes)
	if err != nil {
		return Certificate{}, err
	}

	cert, ok := pubKey.(*ssh.Certificate)
	if !ok {
		return Certificate{}, fmt.Errorf("pub key is not a valid certificate: %w", err)
	}

	return Certificate{
		Type:            cert.Type(),
		Serial:          cert.Serial,
		Principals:      cert.ValidPrincipals,
		Extensions:      cert.Extensions,
		CriticalOptions: cert.CriticalOptions,
		ValidAfter:      time.Unix(int64(cert.ValidAfter), 0).UTC(),                                                             //#nosec:G115
		ValidBefore:     time.Unix(int64(cert.ValidBefore), 0).UTC(),                                                            //#nosec:G115
		Percentage:      GetPercentage(time.Unix(int64(cert.ValidAfter), 0).UTC(), time.Unix(int64(cert.ValidBefore), 0).UTC()), //#nosec:G115
	}, nil
}
