package services

import (
	"fmt"

	"github.com/the127/dockyard/internal/database"
)

type DbService interface {
	GetTransaction() (database.Transaction, error)
	Close() error
}

type dbService struct {
	db          database.Database
	transaction database.Transaction
}

func NewDbService(db database.Database) DbService {
	return &dbService{
		db:          db,
		transaction: nil,
	}
}

func (s *dbService) GetTransaction() (database.Transaction, error) {
	if s.transaction == nil {
		tx, err := s.db.Tx()
		if err != nil {
			return nil, fmt.Errorf("failed to get transaction: %w", err)
		}
		s.transaction = tx
	}
	return s.transaction, nil
}

func (s *dbService) Close() error {
	if s.transaction != nil {
		err := s.transaction.Commit()

		if err != nil {
			err = s.transaction.Rollback()
			if err != nil {
				return fmt.Errorf("failed to rollback transaction: %w", err)
			}

			return fmt.Errorf("failed to commit transaction: %w", err)
		}
	}

	return nil
}
