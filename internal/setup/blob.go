package setup

import (
	"fmt"

	"github.com/The127/ioc"
	"github.com/the127/dockyard/internal/config"
	"github.com/the127/dockyard/internal/services/blobStorage"
	"github.com/the127/dockyard/internal/storageBackends/directory"
	"github.com/the127/dockyard/internal/storageBackends/inmemory"
)

func Blob(dc *ioc.DependencyCollection, c config.BlobStorageConfig) {
	ioc.RegisterSingleton(dc, func(_ *ioc.DependencyProvider) blobStorage.Service {
		switch c.Mode {
		case config.BlobStorageModeInMemory:
			return blobStorage.NewBlobStorageService(inmemory.New())

		case config.BlobStorageModeDirectory:
			storageBackend, err := directory.New(c.Directory)
			if err != nil {
				panic(fmt.Errorf("initializing directory blob storage: %w", err))
			}

			return blobStorage.NewBlobStorageService(storageBackend)

		default:
			panic(fmt.Errorf("unsupported blob storage mode: %s", c.Mode))
		}
	})
}
