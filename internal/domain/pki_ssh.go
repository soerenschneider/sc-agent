package domain

import (
	"errors"
	"fmt"
	"strings"

	"github.com/soerenschneider/vault-ssh-cli/pkg"
	"github.com/soerenschneider/vault-ssh-cli/pkg/ssh"
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

type SshSignatureRequest struct {
	Role            string            `yaml:"role"`
	Principals      []string          `validate:"required" yaml:"principals"`
	PublicKeyFile   string            `validate:"required,file" yaml:"public_key_file"`
	CertificateFile string            `validate:"omitempty,filepath" yaml:"certificate_file"`
	Ttl             string            `yaml:"ttl"`
	CertType        string            `validate:"required,oneof=user host" yaml:"cert_type"`
	CriticalOptions map[string]string `yaml:"critical_options"`
	Extensions      map[string]string `yaml:"extensions"`
}

func (r *SshSignatureRequest) GetCertificateFile() string {
	if len(r.CertificateFile) == 0 && len(r.PublicKeyFile) > 0 {
		auto := strings.Replace(r.PublicKeyFile, ".pub", "", 1)
		auto = pkg.GetExpandedFile(fmt.Sprintf("%s-cert.pub", auto))
		return auto
	}

	return r.CertificateFile
}

type SshSignatureResult struct {
	Action          string
	PublicKeyFile   string
	CertificateFile string
	CertData        *ssh.CertInfo
}
