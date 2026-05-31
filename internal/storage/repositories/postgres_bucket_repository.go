package repositories

import (
	"context"
	"database/sql"

	"github.com/darlingson/Oort-Object-Storage/internal/storage/models"
)

type PostgresBucketRepository struct {
	db *sql.DB
}

func NewPostgresBucketRepository(db *sql.DB) *PostgresBucketRepository {
	return &PostgresBucketRepository{
		db: db,
	}
}

func (r *PostgresBucketRepository) Create(
	ctx context.Context,
	bucket *models.Bucket,
) error {

	query := `
	INSERT INTO buckets(id, name)
	VALUES ($1, $2)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		bucket.ID,
		bucket.Name,
	)

	return err
}

func (r *PostgresBucketRepository) FindByName(
	ctx context.Context,
	name string,
) (*models.Bucket, error) {

	query := `
	SELECT id, name, created_at
	FROM buckets
	WHERE name = $1
	`

	var bucket models.Bucket

	err := r.db.QueryRowContext(
		ctx,
		query,
		name,
	).Scan(
		&bucket.ID,
		&bucket.Name,
		&bucket.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &bucket, nil
}

func (r *PostgresBucketRepository) List(
	ctx context.Context,
) ([]models.Bucket, error) {

	query := `
	SELECT id, name, created_at
	FROM buckets
	ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var buckets []models.Bucket

	for rows.Next() {

		var bucket models.Bucket

		err := rows.Scan(
			&bucket.ID,
			&bucket.Name,
			&bucket.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		buckets = append(buckets, bucket)
	}

	return buckets, nil
}