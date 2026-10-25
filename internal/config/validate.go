package config

import (
	"fmt"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/config/vault"
)

var (
	validate *validator.Validate
	once     sync.Once
)

func Validate(s any) error {
	once.Do(func() {
		validate = validator.New()
		if err := validate.RegisterValidation("duration", validateDuration); err != nil {
			log.Fatal().Err(err).Msg("could not build custom validation 'validateDuration'")
		}

		if err := validate.RegisterValidation("broker", validateBroker); err != nil {
			log.Fatal().Err(err).Msg("could not build custom validation 'validateBroker'")
		}

		validate.RegisterStructValidation(validateVault, vault.Vault{})
		validate.RegisterStructValidation(validateVaultClients, Config{})
	})

	return validate.Struct(s)
}

func validateDuration(fl validator.FieldLevel) bool {
	_, err := time.ParseDuration(fl.Field().String())
	return err == nil
}

// validateVault requires the secret_id file to exist, unless it can be bootstrapped, and rotation-only clients to use
// approle auth.
func validateVault(sl validator.StructLevel) {
	conf, ok := sl.Current().Interface().(vault.Vault)
	if !ok {
		return
	}
	if conf.RotationOnly && conf.AuthMethod != "approle" {
		sl.ReportError(conf.RotationOnly, "RotationOnly", "RotationOnly", "rotation_only_requires_approle", "")
	}
	if conf.AuthMethod != "approle" || conf.Bootstrap != nil || conf.SecretIdFile == "" {
		return
	}

	if info, err := os.Stat(conf.SecretIdFile); err != nil || info.IsDir() {
		sl.ReportError(conf.SecretIdFile, "SecretIdFile", "SecretIdFile", "file", "")
	}
}

// validateVaultClients makes sure components don't use rotation-only Vault clients and that no two approle clients
// share a secret_id file, as their rotators would overwrite each other's secret_id.
func validateVaultClients(sl validator.StructLevel) {
	conf, ok := sl.Current().Interface().(Config)
	if !ok {
		return
	}

	secretIdFiles := map[string]string{}
	for id, client := range conf.Vault {
		if client.AuthMethod != "approle" || client.SecretIdFile == "" {
			continue
		}
		if other, found := secretIdFiles[client.SecretIdFile]; found {
			sl.ReportError(client.SecretIdFile, fmt.Sprintf("Vault[%s].SecretIdFile", id), "SecretIdFile", "unique_secret_id_file", other)
		}
		secretIdFiles[client.SecretIdFile] = id
	}

	vaultIds := map[string]string{}
	if conf.Acme != nil {
		vaultIds["Acme"] = conf.Acme.VaultId
	}
	if conf.SecretsReplication != nil {
		vaultIds["SecretsReplication"] = conf.SecretsReplication.VaultId
	}
	if conf.SshSigner != nil {
		vaultIds["SshSigner"] = conf.SshSigner.VaultId
	}
	if conf.X509Pki != nil {
		vaultIds["X509Pki"] = conf.X509Pki.VaultId
	}
	for component, vaultId := range vaultIds {
		if client, found := conf.Vault[vaultId]; found && client.RotationOnly {
			sl.ReportError(vaultId, component+".VaultId", "VaultId", "not_rotation_only", "")
		}
	}
}

func validateBroker(fl validator.FieldLevel) bool {
	broker := fl.Field().String()
	return IsValidMqttUrl(broker)
}

func IsValidMqttUrl(input string) bool {
	_, err := url.ParseRequestURI(input)
	if err != nil {
		return false
	}

	u, err := url.Parse(input)
	if err != nil || u.Scheme == "" || u.Host == "" || u.Port() == "" {
		return false
	}

	return true
}
