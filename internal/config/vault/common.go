package vault

import (
	"gopkg.in/yaml.v3"
)

const (
	defaultAuthMethod   = "approle"
	defaultApproleMount = "approle"
)

type Vault struct {
	Address    string `yaml:"address" env:"VAULT_ADDR" validate:"omitempty,http_url"`
	AuthMethod string `yaml:"auth_method" validate:"required,oneof=token approle"`
	Token      string `yaml:"token" env:"VAULT_TOKEN" validate:"required_if=AuthMethod token"`
	RoleId     string `yaml:"role_id" validate:"required_if=AuthMethod approle"`
	// RoleName is the name of the approle role, it defaults to the role_id.
	RoleName                 string                    `yaml:"role_name"`
	SecretIdFile             string                    `yaml:"secret_id_file" validate:"required_if=AuthMethod approle"`
	SecretIdFileUser         string                    `yaml:"secret_id_file_user"`
	MountApprole             string                    `yaml:"approle_mount" validate:"required_if=AuthMethod approle"`
	ApproleCidrTokenResolver *VaultApproleCidrResolver `yaml:"cidr_token_resolver"`
	ApproleCidrLoginResolver *VaultApproleCidrResolver `yaml:"cidr_login_resolver"`
	Bootstrap                *VaultBootstrap           `yaml:"bootstrap"`
	// RotationOnly clients only rotate their secret_id: they log in for each rotation check instead of keeping a token
	// and can not be used by other components.
	RotationOnly bool `yaml:"rotation_only"`
}

// VaultBootstrap configures fetching a wrapped secret_id from approle-courier if the secret_id file is missing or the
// secret_id is no longer valid.
type VaultBootstrap struct {
	Address string `yaml:"address" validate:"required,http_url"`
	// MachineId defaults to the hostname.
	MachineId string `yaml:"machine_id"`
	// RoleName defaults to the role_name of the Vault client.
	RoleName        string `yaml:"role_name"`
	AgeIdentityFile string `yaml:"age_identity_file" validate:"required,file"`
	WrapTtl         string `yaml:"wrap_ttl" validate:"omitempty,duration"`
}

type VaultApproleCidrResolver struct {
	Type string `yaml:"type" validate:"required,oneof=dynamic static"`
	Args any    `yaml:"args"`
}

// ApproleRoleName returns the name of the approle role, which defaults to the role_id.
func (conf Vault) ApproleRoleName() string {
	if conf.RoleName != "" {
		return conf.RoleName
	}
	return conf.RoleId
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
