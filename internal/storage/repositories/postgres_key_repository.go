package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/darlingson/Oort-Object-Storage/internal/storage/models"
)

type PostgresKeyRepository struct {
	db *sql.DB
}

func NewPostgresKeyRepository(db *sql.DB) *PostgresKeyRepository {
	return &PostgresKeyRepository{
		db: db,
	}
}

func (r *PostgresKeyRepository) Create(
	ctx context.Context,
	key *models.APIKey,
) error {

	query := `
	INSERT INTO api_keys (
		id,
		name,
		key_hash,
		buckets,
		expires_at
	)
	VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		key.ID,
		key.Name,
		key.KeyHash,
		pq.Array(key.Buckets),
		key.ExpiresAt,
	)

	return err
}

func (r *PostgresKeyRepository) FindByHash(
	ctx context.Context,
	hash string,
) (*models.APIKey, error) {

	query := `
	SELECT
		id,
		name,
		key_hash,
		buckets,
		expires_at,
		last_used_at,
		created_at
	FROM api_keys
	WHERE key_hash = $1
	`

	var key models.APIKey

	err := r.db.QueryRowContext(
		ctx,
		query,
		hash,
	).Scan(
		&key.ID,
		&key.Name,
		&key.KeyHash,
		pq.Array(&key.Buckets),
		&key.ExpiresAt,
		&key.LastUsedAt,
		&key.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &key, nil
}

func (r *PostgresKeyRepository) List(
	ctx context.Context,
) ([]models.APIKey, error) {

	query := `
	SELECT
		id,
		name,
		key_hash,
		buckets,
		expires_at,
		last_used_at,
		created_at
	FROM api_keys
	ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var keys []models.APIKey

	for rows.Next() {

		var key models.APIKey

		err := rows.Scan(
			&key.ID,
			&key.Name,
			&key.KeyHash,
			pq.Array(&key.Buckets),
			&key.ExpiresAt,
			&key.LastUsedAt,
			&key.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		keys = append(keys, key)
	}

	return keys, nil
}

func (r *PostgresKeyRepository) Delete(
	ctx context.Context,
	id uuid.UUID,
) error {

	query := `
	DELETE FROM api_keys
	WHERE id = $1
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		id,
	)

	return err
}

func (r *PostgresKeyRepository) UpdateLastUsed(
	ctx context.Context,
	id uuid.UUID,
	at time.Time,
) error {

	query := `
	UPDATE api_keys
	SET last_used_at = $1
	WHERE id = $2
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		at,
		id,
	)

	return err
}
