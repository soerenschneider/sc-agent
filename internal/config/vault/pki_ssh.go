package vault

import (
	"github.com/soerenschneider/sc-agent/internal/domain"
	"gopkg.in/yaml.v3"
)

const (
	defaultSshMount = "ssh"
)

type SshPki struct {
	Enabled   bool                                  `yaml:"enabled"`
	VaultId   string                                `yaml:"vault"`
	MountPath string                                `yaml:"ssh_mount" validate:"required"`
	UserKeys  map[string]domain.SshSignatureRequest `yaml:"user_keys" validate:"omitempty,dive"`
	HostKeys  map[string]domain.SshSignatureRequest `yaml:"host_keys" validate:"omitempty,dive"`
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
