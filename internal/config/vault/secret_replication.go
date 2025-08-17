package vault

import (
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	defaultSecretMount = "secret"
)

type SecretsReplication struct {
	Enabled             bool                            `yaml:"enabled"`
	VaultId             string                          `yaml:"vault"`
	Kv2Mount            string                          `yaml:"kv2_mount" validate:"required"`
	ReplicationRequests map[string]VaultReplicationItem `yaml:"replication_requests" validate:"dive"`
}

func (conf *SecretsReplication) UnmarshalYAML(node *yaml.Node) error {
	type Alias SecretsReplication // Create an alias to avoid recursion during unmarshalling

	// Define conf temporary struct with default values
	tmp := &Alias{
		Enabled:  true,
		Kv2Mount: defaultSecretMount,
	}

	// Unmarshal the yaml data into the temporary struct
	if err := node.Decode(&tmp); err != nil {
		return err
	}

	// Assign the values from the temporary struct to the original struct
	*conf = SecretsReplication(*tmp)
	return nil
}

type VaultReplicationItem struct {
	SecretPath    string         `yaml:"secret_path" validate:"required"`
	Formatter     string         `yaml:"formatter" validate:"required,oneof=yaml json env template"`
	FormatterArgs map[string]any `yaml:"formatter_args"`
	DestUri       string         `yaml:"dest" validate:"required"`
}

func (conf *VaultReplicationItem) UnmarshalYAML(node *yaml.Node) error {
	type Alias VaultReplicationItem // Create an alias to avoid recursion during unmarshalling

	// Define conf temporary struct with default values
	tmp := &Alias{}

	// Unmarshal the yaml data into the temporary struct
	if err := node.Decode(&tmp); err != nil {
		return err
	}

	var formatter string
	defaultExtension := filepath.Ext(tmp.DestUri)
	switch defaultExtension {
	case ".json":
		formatter = "json"
	case ".yaml":
		formatter = "yaml"
	}

	if tmp.Formatter == "" && formatter != "" {
		tmp.Formatter = formatter
	}

	// Assign the values from the temporary struct to the original struct
	*conf = VaultReplicationItem(*tmp)
	return nil
}
