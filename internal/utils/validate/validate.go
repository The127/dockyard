package validate

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/the127/dockyard/internal/utils/apiError"
)

var validate = func() *validator.Validate {
	v := validator.New()
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "" || name == "-" {
			return fld.Name
		}
		return name
	})
	return v
}()

func Validate(s any) error {
	err := validate.Struct(s)
	if err != nil {
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			msgs := make([]string, 0, len(validationErrors))
			for _, fe := range validationErrors {
				msgs = append(msgs, fmt.Sprintf("field '%s' failed validation '%s'", fe.Field(), fe.Tag()))
			}
			return fmt.Errorf("invalid request: %s: %w", strings.Join(msgs, "; "), apiError.ErrApiBadRequest)
		}
		return fmt.Errorf("invalid request: %s: %w", err.Error(), apiError.ErrApiBadRequest)
	}

	return nil
}
