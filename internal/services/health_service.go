package services

import (
	"context"
	"database/sql"
	"os"
)

type HealthService struct {
	db        *sql.DB
	storageRoot string
}

func NewHealthService(db *sql.DB, storageRoot string) *HealthService {
	return &HealthService{
		db:          db,
		storageRoot: storageRoot,
	}
}

func (s *HealthService) CheckDB(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *HealthService) CheckStorage() error {
	info, err := os.Stat(s.storageRoot)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return os.ErrInvalid
	}
	return nil
}
