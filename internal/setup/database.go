package setup

import (
	"context"
	"fmt"

	"github.com/The127/ioc"
	"github.com/the127/dockyard/internal/config"
	"github.com/the127/dockyard/internal/database"
	"github.com/the127/dockyard/internal/database/inmemory"
	"github.com/the127/dockyard/internal/database/postgres"
	"github.com/the127/dockyard/internal/logging"
)

func Database(dc *ioc.DependencyCollection, c config.DatabaseConfig) database.Database {
	db := connectToDatabase(c)

	ioc.RegisterSingleton(dc, func(_ *ioc.DependencyProvider) database.Factory {
		return database.NewDbFactory(db)
	})

	ioc.RegisterScoped(dc, func(dp *ioc.DependencyProvider) database.Context {
		dbFactory := ioc.GetDependency[database.Factory](dp)
		dbContext, err := dbFactory.NewDbContext(context.TODO())
		if err != nil {
			logging.Logger.Panicf("failed to create database context: %s", err)
		}

		return dbContext
	})
	ioc.RegisterCloseHandler(dc, func(dbContext database.Context) error {
		err := dbContext.SaveChanges(context.TODO())
		if err != nil {
			return fmt.Errorf("failed to save changes: %w", err)
		}

		return nil
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
