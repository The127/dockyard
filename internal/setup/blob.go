package setup

import (
	"fmt"

	"github.com/The127/ioc"
	"github.com/the127/dockyard/internal/config"
	"github.com/the127/dockyard/internal/services/blobStorage"
	"github.com/the127/dockyard/internal/storageBackends/inmemory"
)

func Blob(dc *ioc.DependencyCollection, c config.BlobStorageConfig) {
	ioc.RegisterSingleton(dc, func(_ *ioc.DependencyProvider) blobStorage.Service {
		switch c.Mode {
		case config.BlobStorageModeInMemory:
			return blobStorage.NewBlobStorageService(inmemory.New())

		default:
			panic(fmt.Errorf("unsupported blob storage mode: %s", c.Mode))
		}
	})
}
