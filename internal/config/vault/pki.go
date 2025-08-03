package vault

import (
	"github.com/soerenschneider/sc-agent/internal/domain"
	"github.com/soerenschneider/sc-agent/internal/domain/x509"
	"gopkg.in/yaml.v3"
)

const (
	defaultPkiMount = "pki"
)

type X509Pki struct {
	Enabled      bool         `yaml:"enabled"`
	VaultId      string       `yaml:"vault"`
	MountPath    string       `yaml:"mount_path" validate:"required"`
	ManagedCerts []CertConfig `yaml:"managed_certs" validate:"omitempty,dive"`
}

// CertConfig configures a cert
type CertConfig struct {
	Id         string        `yaml:"id" validate:"required"`
	Role       string        `yaml:"role"`
	CommonName string        `validate:"required" yaml:"common_name"`
	Storage    []CertStorage `yaml:"storage"`
	Ttl        string        `yaml:"ttl"`
	AltNames   []string      `yaml:"alt_names"`
	IpSans     []string      `yaml:"ip_sans"`

	PostHooks map[string]string `yaml:"post_hooks"`
}

func (c *CertConfig) ToDomainModel() x509.ManagedCertificateConfig {
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
			Id:         c.Id,
			Role:       c.Role,
			CommonName: c.CommonName,
			Ttl:        c.Ttl,
			AltNames:   c.AltNames,
			IpSans:     c.IpSans,
		},
		StorageConfig: storageConf,
		PostHooks:     postHooks,
	}
}

type CertStorage struct {
	CaChainFile string `validate:"omitempty" yaml:"ca_chain_file"`
	CaFile      string `validate:"omitempty" yaml:"ca_file"`
	CertFile    string `validate:"omitempty" yaml:"cert_file"`
	KeyFile     string `validate:"omitempty" yaml:"key_file"`
}

func (c *CertStorage) ToDomainModel() x509.CertificateStorage {
	return x509.CertificateStorage{
		CaChainFile: c.CaChainFile,
		CaFile:      c.CaFile,
		CertFile:    c.CertFile,
		KeyFile:     c.KeyFile,
	}
}

func (conf *X509Pki) UnmarshalYAML(node *yaml.Node) error {
	type Alias X509Pki // Create an alias to avoid recursion during unmarshalling

	// Define conf temporary struct with default values
	tmp := &Alias{
		Enabled:   true,
		MountPath: defaultPkiMount,
	}

	// Unmarshal the yaml data into the temporary struct
	if err := node.Decode(&tmp); err != nil {
		return err
	}

	// Assign the values from the temporary struct to the original struct
	*conf = X509Pki(*tmp)
	return nil
}
