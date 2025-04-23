// internal/utils/validation_errors.go
package utils

import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

func FormatValidationErrors(err error) map[string]string {
	errorsMap := make(map[string]string)
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fieldErr := range validationErrors {
			fieldName := fieldErr.Field()
			// Anda bisa membuat pesan error yang lebih spesifik berdasarkan `fieldErr.Tag()`
			errorsMap[fieldName] = fmt.Sprintf("Field validation for '%s' failed on the '%s' tag", fieldName, fieldErr.Tag())
		}
	} else {
		// Jika bukan error validasi spesifik, kembalikan pesan generik
		errorsMap["error"] = "Invalid input data"
	}
	return errorsMap
}
