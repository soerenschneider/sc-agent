package vault

import (
	"github.com/soerenschneider/sc-agent/internal/domain"
	"github.com/soerenschneider/sc-agent/internal/domain/x509"
	"gopkg.in/yaml.v3"
)

const (
	defaultKv2Mount = "kv2"
)

type Acme struct {
	Enabled      bool             `yaml:"enabled"`
	VaultId      string           `yaml:"vault"`
	MountPath    string           `yaml:"mount_path" validate:"required"`
	ManagedCerts []AcmeCertConfig `yaml:"managed_certs" validate:"omitempty,dive"`
}

// CertConfig configures a cert
type AcmeCertConfig struct {
	CommonName string            `validate:"required" yaml:"common_name"`
	Storage    []CertStorage     `yaml:"storage"`
	PostHooks  map[string]string `yaml:"post_hooks"`
}

func (c *AcmeCertConfig) ToDomainModel() x509.ManagedCertificateConfig {
	var storageConf []x509.CertificateStorage
	for _, conf := range c.Storage {
		storageConf = append(storageConf, conf.ToDomainModel())
	}

	var postHooks []domain.PostHook
	for key, val := range c.PostHooks {
		postHooks = append(postHooks, domain.PostHook{
			Name: key,
			Cmd:  val,
		})
	}

	return x509.ManagedCertificateConfig{
		CertificateConfig: &x509.CertificateConfig{
			CommonName: c.CommonName,
		},
		StorageConfig: storageConf,
		PostHooks:     postHooks,
	}
}

func (conf *Acme) UnmarshalYAML(node *yaml.Node) error {
	type Alias Acme // Create an alias to avoid recursion during unmarshalling

	// Define conf temporary struct with default values
	tmp := &Alias{
		Enabled:   true,
		MountPath: defaultKv2Mount,
	}

	// Unmarshal the yaml data into the temporary struct
	if err := node.Decode(&tmp); err != nil {
		return err
	}

	// Assign the values from the temporary struct to the original struct
	*conf = Acme(*tmp)
	return nil
}
