package setup

import (
	"fmt"

	"github.com/The127/ioc"
	"github.com/the127/dockyard/internal/config"
	"github.com/the127/dockyard/internal/database"
	"github.com/the127/dockyard/internal/database/inmemory"
	"github.com/the127/dockyard/internal/database/postgres"
)

func Database(dc *ioc.DependencyCollection, c config.DatabaseConfig) database.Database {
	db := connectToDatabase(c)

	ioc.RegisterScoped(dc, func(_ *ioc.DependencyProvider) database.Factory {
		return database.NewDbFactory(db)
	})

	return db
}

func connectToDatabase(c config.DatabaseConfig) database.Database {
	var db database.Database
	var err error

	switch c.Mode {
	case config.DatabaseModeInMemory:
		db, err = inmemory.NewInMemoryDatabase()

	case config.DatabaseModePostgres:
		db, err = postgres.NewPostgresDatabase(c.Postgres)

	default:
		panic(fmt.Errorf("unsupported database mode: %s", c.Mode))
	}

	if err != nil {
		panic(fmt.Errorf("failed to connect to database: %w", err))
	}

	return db
}
