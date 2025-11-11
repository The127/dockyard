package setup

import (
	"fmt"

	"github.com/The127/ioc"
	"github.com/the127/dockyard/internal/config"
	"github.com/the127/dockyard/internal/services/blob"
)

func Blob(dc *ioc.DependencyCollection, c config.BlobStorageConfig) {
	ioc.RegisterSingleton(dc, func(_ *ioc.DependencyProvider) blob.Service {
		switch c.Mode {
		case config.BlobStorageModeInMemory:
			return blob.NewInMemoryService()

		default:
			panic(fmt.Errorf("unsupported blob storage mode: %s", c.Mode))
		}
	})
}
