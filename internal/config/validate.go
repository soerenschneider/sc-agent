package config

import (
	"sync"

	"github.com/go-playground/validator/v10"
)

var (
	validate *validator.Validate
	once     sync.Once
)

func Validate(s any) error {
	once.Do(func() {
		validate = validator.New()
	})

	return validate.Struct(s)
}
