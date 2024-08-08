package vault

import (
	"github.com/soerenschneider/sc-agent/internal/domain"
	"gopkg.in/yaml.v3"
)

const (
	defaultPkiMount = "pki"
)

type X509Pki struct {
	Enabled      bool                  `yaml:"enabled"`
	VaultId      string                `yaml:"vault"`
	MountPath    string                `yaml:"mount" validate:"required"`
	ManagedCerts map[string]CertConfig `yaml:"managed_certs" validate:"omitempty,dive"`
}

// CertConfig configures a cert
type CertConfig struct {
	Role       string        `yaml:"role"`
	CommonName string        `validate:"required" yaml:"common_name"`
	Storage    []CertStorage `yaml:"storage"`
	Ttl        string        `yaml:"ttl"`
	AltNames   []string
	IpSans     []string
}

func (c *X509Pki) GetManagedCerts() map[string]domain.CertConfig {
	ret := map[string]domain.CertConfig{}
	for key, c := range c.ManagedCerts {
		ret[key] = domain.CertConfig{
			Id:         key,
			Role:       c.Role,
			CommonName: c.CommonName,
			Ttl:        c.Ttl,
			AltNames:   c.AltNames,
			IpSans:     c.IpSans,
		}
	}
	return ret
}

type CertStorage struct {
	CaFile   string `validate:"omitempty" yaml:"ca_file"`
	CertFile string `validate:"omitempty" yaml:"cert_file"`
	KeyFile  string `validate:"omitempty" yaml:"key_file"`
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
