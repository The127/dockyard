package postgres

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/the127/dockyard/internal/config"
	db "github.com/the127/dockyard/internal/database"
	"github.com/the127/dockyard/internal/logging"

	_ "github.com/lib/pq"
	"github.com/rubenv/sql-migrate"
)

//go:embed migrations/*
var migrations embed.FS

type database struct {
	db *sql.DB
}

func NewPostgresDatabase(pc config.PostgresConfig) (db.Database, error) {
	dbConnection, err := ConnectToDatabase(pc)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %v", err)
	}

	return &database{
		db: dbConnection,
	}, nil
}

func ConnectToDatabase(pc config.PostgresConfig) (*sql.DB, error) {
	logging.Logger.Infof("Connecting to database %s via %s:%d",
		pc.Database,
		pc.Host,
		pc.Port)

	connectionString := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		pc.Host,
		pc.Port,
		pc.Database,
		pc.Username,
		pc.Password,
		pc.SslMode)

	dbConnection, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("opening database connection: %v", err)
	}

	return dbConnection, nil
}

func (d *database) Migrate() error {
	migrations := migrate.EmbedFileSystemMigrationSource{
		FileSystem: migrations,
		Root:       "migrations",
	}

	logging.Logger.Infof("Applying migrations...")

	n, err := migrate.Exec(d.db, "postgres", migrations, migrate.Up)
	if err != nil {
		return fmt.Errorf("failed to apply migrations: %v", err)
	}

	logging.Logger.Infof("Applied %d migrations", n)
	return nil
}

func (d *database) Tx() (db.Transaction, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}

	return newTransaction(tx), nil
}
