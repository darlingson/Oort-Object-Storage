package database

import (
	"database/sql"
	"fmt"

	"github.com/darlingson/Oort-Object-Storage/internal/config"

	_ "github.com/lib/pq"
)

func NewPostgres(cfg *config.Config) (*sql.DB, error) {

	psqlInfo := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
	)

	db, err := sql.Open("postgres", psqlInfo)

	if err != nil {
		return nil, err
	}

	err = db.Ping()

	if err != nil {
		return nil, err
	}

	return db, nil
}