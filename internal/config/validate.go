package config

import (
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
	})

	return validate.Struct(s)
}

func validateDuration(fl validator.FieldLevel) bool {
	_, err := time.ParseDuration(fl.Field().String())
	return err == nil
}

// validateVault requires the secret_id file to exist, unless it can be bootstrapped.
func validateVault(sl validator.StructLevel) {
	conf, ok := sl.Current().Interface().(vault.Vault)
	if !ok || conf.AuthMethod != "approle" || conf.Bootstrap != nil || conf.SecretIdFile == "" {
		return
	}

	if info, err := os.Stat(conf.SecretIdFile); err != nil || info.IsDir() {
		sl.ReportError(conf.SecretIdFile, "SecretIdFile", "SecretIdFile", "file", "")
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
