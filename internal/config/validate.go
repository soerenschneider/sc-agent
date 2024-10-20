package config

import (
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
			log.Error().Err(err).Msg("could not register validation")
		}
	})

	return validate.Struct(s)
}

func validateDuration(fl validator.FieldLevel) bool {
	_, err := time.ParseDuration(fl.Field().String())
	return err == nil
}
