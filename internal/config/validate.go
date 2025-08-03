package config

import (
	"net/url"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
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
	})

	return validate.Struct(s)
}

func validateDuration(fl validator.FieldLevel) bool {
	_, err := time.ParseDuration(fl.Field().String())
	return err == nil
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
