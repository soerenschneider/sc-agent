package vault

import (
	"gopkg.in/yaml.v3"
)

const (
	defaultAuthMethod   = "approle"
	defaultApproleMount = "approle"
)

type Vault struct {
	Address                  string                    `yaml:"address" env:"VAULT_ADDR" validate:"omitempty,http_url"`
	AuthMethod               string                    `yaml:"auth_method" validate:"required,oneof=token approle"`
	Token                    string                    `yaml:"token" env:"VAULT_TOKEN" validate:"required_if=AuthMethod token"`
	RoleId                   string                    `yaml:"role_id" validate:"required_if=AuthMethod approle"`
	SecretIdFile             string                    `yaml:"secret_id_file" validate:"required_if=AuthMethod approle,file"`
	SecretIdFileUser         string                    `yaml:"secret_id_file_user" validate:"string"`
	MountApprole             string                    `yaml:"approle_mount" validate:"required_if=AuthMethod approle"`
	ApproleCidrTokenResolver *VaultApproleCidrResolver `yaml:"cidr_token_resolver"`
	ApproleCidrLoginResolver *VaultApproleCidrResolver `yaml:"cidr_login_resolver"`
}

type VaultApproleCidrResolver struct {
	Type string `yaml:"type" validate:"required,oneof=dynamic static"`
	Args any    `yaml:"args"`
}

func (conf *Vault) UnmarshalYAML(node *yaml.Node) error {
	type Alias Vault // Create an alias to avoid recursion during unmarshalling

	// Define conf temporary struct with default values
	tmp := &Alias{
		AuthMethod:   defaultAuthMethod,
		MountApprole: defaultApproleMount,
	}

	// Unmarshal the yaml data into the temporary struct
	if err := node.Decode(&tmp); err != nil {
		return err
	}

	// Assign the values from the temporary struct to the original struct
	*conf = Vault(*tmp)
	return nil
}
