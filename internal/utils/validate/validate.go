package validate

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/the127/dockyard/internal/utils/apiError"
)

var validate = validator.New()

func Validate(s any) error {
	err := validate.Struct(s)
	if err != nil {
		return fmt.Errorf("invalid request: %s, %w", err.Error(), apiError.ErrApiBadRequest)
	}

	// TODO: make an api friendly error type

	return nil
}
