package app

import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

// ValidateRequest validates a request struct using struct tags
func (g *Gin) ValidateRequest(form interface{}) error {
	validate := validator.New()
	err := validate.Struct(form)
	if err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			var errMsg string
			for i, fieldError := range validationErrors {
				if i > 0 {
					errMsg += "; "
				}
				errMsg += fmt.Sprintf("%s: %s", fieldError.Field(), fieldError.Tag())
			}
			return fmt.Errorf("validation failed: %s", errMsg)
		}
		return fmt.Errorf("validation error: %w", err)
	}
	return nil
}
