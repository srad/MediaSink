package app

import (
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/astaxie/beego/validation"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// MarkErrors logs error logs
func MarkErrors(errors []*validation.Error) {
	for _, err := range errors {
		log.Errorln(err.Key, err.Message)
	}
}

// BindAndValid binds and validates data
func BindAndValid(c *gin.Context, form interface{}) int {
	err := c.Bind(form)
	if err != nil {
		return http.StatusBadRequest
	}

	valid := validation.Validation{}
	check, err := valid.Valid(form)
	if err != nil {
		return http.StatusInternalServerError
	}
	if !check {
		MarkErrors(valid.Errors)
		return http.StatusBadRequest
	}

	return http.StatusOK
}

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
