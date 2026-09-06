package validator

import (
	"github.com/go-playground/validator/v10"
)

type CustomValidator struct {
	validator *validator.Validate
}

func New() *CustomValidator {
	v := validator.New()
	return &CustomValidator{validator: v}
}

func (cv *CustomValidator) Validate(i any) error {
	if cv == nil || cv.validator == nil {
		return nil
	}
	return cv.validator.Struct(i)
}
