package repositories

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"github.com/darlingson/Oort-Object-Storage/internal/storage/models"
)

type PostgresSignedURLRepository struct {
	db *sql.DB
}

func NewPostgresSignedURLRepository(
	db *sql.DB,
) *PostgresSignedURLRepository {

	return &PostgresSignedURLRepository{
		db: db,
	}
}

func (r *PostgresSignedURLRepository) Create(
	ctx context.Context,
	signedURL *models.SignedURL,
) error {

	query := `
	INSERT INTO signed_urls(id, bucket_name, object_key, token, operation, expires_at)
	VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		signedURL.ID,
		signedURL.BucketName,
		signedURL.ObjectKey,
		signedURL.Token,
		signedURL.Operation,
		signedURL.ExpiresAt,
	)

	return err
}

func (r *PostgresSignedURLRepository) FindByToken(
	ctx context.Context,
	token string,
) (*models.SignedURL, error) {

	query := `
	SELECT id, bucket_name, object_key, token, operation, expires_at, created_at
	FROM signed_urls
	WHERE token = $1
	`

	row := r.db.QueryRowContext(ctx, query, token)

	var su models.SignedURL
	err := row.Scan(
		&su.ID,
		&su.BucketName,
		&su.ObjectKey,
		&su.Token,
		&su.Operation,
		&su.ExpiresAt,
		&su.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &su, nil
}

func (r *PostgresSignedURLRepository) Delete(
	ctx context.Context,
	id uuid.UUID,
) error {

	query := `DELETE FROM signed_urls WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id)

	return err
}
