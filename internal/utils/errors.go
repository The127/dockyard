package utils

import (
	"fmt"

	"github.com/the127/dockyard/internal/logging"
)

func PanicOnError(f func() error, message string) {
	err := f()
	if err != nil {
		logging.Logger.Panic(fmt.Errorf("%s: %w", message, err))
	}
}
