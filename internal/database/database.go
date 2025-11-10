package database

type Database interface {
	Migrate() error
	Tx() (Transaction, error)
}
