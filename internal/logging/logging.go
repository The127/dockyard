package logging

import (
	"fmt"

	"github.com/the127/dockyard/internal/args"

	"go.uber.org/zap"
)

var Logger *zap.SugaredLogger

func Init() {
	if args.IsProduction() {
		logger, err := zap.NewProduction()
		if err != nil {
			panic(fmt.Errorf("failed to initialize production logger: %w", err))
		}
		Logger = logger.Sugar()
	} else {
		logger, err := zap.NewDevelopment()
		if err != nil {
			panic(fmt.Errorf("failed to initialize development logger: %w", err))
		}
		Logger = logger.Sugar()
	}
}
