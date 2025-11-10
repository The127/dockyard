package setup

import (
	"fmt"

	"github.com/The127/ioc"
	"github.com/the127/dockyard/internal/config"
	"github.com/the127/dockyard/internal/services/kv"
)

func Kv(dc *ioc.DependencyCollection, kvConfig config.KvConfig) {
	ioc.RegisterSingleton(dc, func(_ *ioc.DependencyProvider) kv.Store {
		switch kvConfig.Mode {
		case config.KvModeInMemory:
			return kv.NewMemoryStore()

		case config.KvModeRedis:
			return kv.NewRedisStore(kvConfig)

		default:
			panic(fmt.Errorf("unsupported kv mode: %s", kvConfig.Mode))
		}
	})
}
