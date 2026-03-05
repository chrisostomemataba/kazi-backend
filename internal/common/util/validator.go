package util

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func ValidateStruct(data interface{}) error {
	if err := validate.Struct(data); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		var errorMessages []string
		for _, fieldError := range validationErrors {
			errorMessages = append(errorMessages, fmt.Sprintf("%s is %s", fieldError.Field(), fieldError.Tag()))
		}
		return errors.New(strings.Join(errorMessages, ", "))

	}
	return nil
}

func FormatPhoneNumber(phone string) string {
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.TrimPrefix(phone, "+")
	if strings.HasPrefix(phone, "0") {
		phone = "255" + phone[1:]
	}
	if !strings.HasPrefix(phone, "255") {
		phone = "255" + phone
	}
	return phone
}