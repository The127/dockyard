package setup

import (
	"fmt"

	"github.com/The127/go-clock"
	"github.com/The127/ioc"
	"github.com/The127/signr"
	signrMemory "github.com/The127/signr/backends/memory"
	"github.com/the127/dockyard/internal/config"
)

func Kms(dc *ioc.DependencyCollection, c config.KmsConfig) {
	switch c.Mode {
	case config.KmsModeMemory:
		ioc.RegisterSingleton(dc, func(dp *ioc.DependencyProvider) signr.KeyManager {
			keyManager, err := signr.New(signr.Config{
				Backend: signrMemory.Config{
					Clock: ioc.GetDependency[clock.Service](dp),
				},
			})
			if err != nil {
				panic(fmt.Errorf("failed to create kms: %w", err))
			}

			return keyManager
		})

	default:
		panic(fmt.Errorf("unsupported kms mode: %s", c.Mode))
	}
}
