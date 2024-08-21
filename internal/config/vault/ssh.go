package vault

import (
	"fmt"
	"strings"

	"github.com/soerenschneider/sc-agent/internal/domain/ssh"
	"github.com/soerenschneider/sc-agent/pkg"
	"gopkg.in/yaml.v3"
)

const (
	defaultSshMount = "ssh"
)

type SshPki struct {
	Enabled     bool                       `yaml:"enabled"`
	VaultId     string                     `yaml:"vault"`
	MountPath   string                     `yaml:"mount_path" validate:"required"`
	ManagedKeys []ManagedCertificateConfig `yaml:"managed_keys" validate:"omitempty,dive"`
}

func (conf *SshPki) UnmarshalYAML(node *yaml.Node) error {
	type Alias SshPki // Create an alias to avoid recursion during unmarshalling

	// Define conf temporary struct with default values
	tmp := &Alias{
		Enabled:   true,
		MountPath: defaultSshMount,
	}

	// Unmarshal the yaml data into the temporary struct
	if err := node.Decode(&tmp); err != nil {
		return err
	}

	// Assign the values from the temporary struct to the original struct
	*conf = SshPki(*tmp)
	return nil
}

type ManagedCertificateConfig struct {
	Id              string            `yaml:"id" validate:"required"`
	Role            string            `yaml:"role"`
	Principals      []string          `yaml:"principals"`
	PublicKeyFile   string            `yaml:"public_key_file" validate:"required,file"`
	CertificateFile string            `yaml:"certificate_file" validate:"omitempty,filepath"`
	Ttl             string            `yaml:"ttl"`
	CertType        string            `yaml:"cert_type" validate:"required,oneof=user host"`
	CriticalOptions map[string]string `yaml:"critical_options"`
	Extensions      map[string]string `yaml:"extensions"`
}

func (c ManagedCertificateConfig) ToDomainModel() ssh.ManagedCertificateConfig {
	return ssh.ManagedCertificateConfig{
		CertificateConfig: &ssh.CertificateConfig{
			Id:              c.Id,
			Role:            c.Role,
			Principals:      c.Principals,
			Ttl:             c.Ttl,
			CertType:        c.CertType,
			CriticalOptions: c.CriticalOptions,
			Extensions:      c.Extensions,
		},
		StorageConfig: &ssh.CertificateStorage{
			PublicKeyFile:   c.PublicKeyFile,
			CertificateFile: c.getCertificateFile(),
		},
	}
}

func (c ManagedCertificateConfig) getCertificateFile() string {
	if len(c.CertificateFile) == 0 && len(c.PublicKeyFile) > 0 {
		auto := strings.Replace(c.PublicKeyFile, ".pub", "", 1)
		auto = pkg.GetExpandedFile(fmt.Sprintf("%s-cert.pub", auto))
		return auto
	}

	return c.CertificateFile
}
