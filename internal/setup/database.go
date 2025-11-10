package setup

import (
	"fmt"

	"github.com/The127/ioc"
	"github.com/the127/dockyard/internal/config"
	"github.com/the127/dockyard/internal/database"
	"github.com/the127/dockyard/internal/database/inmemory"
	"github.com/the127/dockyard/internal/services"
)

func Database(dc *ioc.DependencyCollection, c config.DatabaseConfig) database.Database {
	db := connectToDatabase(c)

	ioc.RegisterScoped(dc, func(_ *ioc.DependencyProvider) services.DbService {
		return services.NewDbService(db)
	})
	ioc.RegisterCloseHandler(dc, func(dbService services.DbService) error {
		return dbService.Close()
	})

	return db
}

func connectToDatabase(c config.DatabaseConfig) database.Database {
	var db database.Database
	var err error

	switch c.Mode {
	case config.DatabaseModeInMemory:
		db, err = inmemory.NewInMemoryDatabase()

	default:
		panic(fmt.Errorf("unsupported database mode: %s", c.Mode))
	}

	if err != nil {
		panic(fmt.Errorf("failed to connect to database: %w", err))
	}

	return db
}
