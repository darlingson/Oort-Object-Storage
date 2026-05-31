package repositories

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"github.com/darlingson/Oort-Object-Storage/internal/storage/models"
)

type PostgresObjectRepository struct {
	db *sql.DB
}

func NewPostgresObjectRepository(
	db *sql.DB,
) *PostgresObjectRepository {
	return &PostgresObjectRepository{
		db: db,
	}
}

func (r *PostgresObjectRepository) Create(
	ctx context.Context,
	object *models.Object,
) error {

	query := `
	INSERT INTO objects (
		id,
		bucket_id,
		object_key,
		storage_path,
		content_type,
		size_bytes,
		checksum
	)
	VALUES ($1,$2,$3,$4,$5,$6,$7)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		object.ID,
		object.BucketID,
		object.ObjectKey,
		object.StoragePath,
		object.ContentType,
		object.SizeBytes,
		object.Checksum,
	)

	return err
}

func (r *PostgresObjectRepository) Upsert(
	ctx context.Context,
	object *models.Object,
) error {

	query := `
	INSERT INTO objects (
		id,
		bucket_id,
		object_key,
		storage_path,
		content_type,
		size_bytes,
		checksum
	)
	VALUES ($1,$2,$3,$4,$5,$6,$7)
	ON CONFLICT (bucket_id, object_key)
	DO UPDATE SET
		id = EXCLUDED.id,
		storage_path = EXCLUDED.storage_path,
		content_type = EXCLUDED.content_type,
		size_bytes = EXCLUDED.size_bytes,
		checksum = EXCLUDED.checksum
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		object.ID,
		object.BucketID,
		object.ObjectKey,
		object.StoragePath,
		object.ContentType,
		object.SizeBytes,
		object.Checksum,
	)

	return err
}

func (r *PostgresObjectRepository) FindByKey(
	ctx context.Context,
	bucketID uuid.UUID,
	key string,
) (*models.Object, error) {

	query := `
	SELECT
		id,
		bucket_id,
		object_key,
		storage_path,
		content_type,
		size_bytes,
		checksum,
		created_at
	FROM objects
	WHERE bucket_id = $1
	AND object_key = $2
	`

	var object models.Object

	err := r.db.QueryRowContext(
		ctx,
		query,
		bucketID,
		key,
	).Scan(
		&object.ID,
		&object.BucketID,
		&object.ObjectKey,
		&object.StoragePath,
		&object.ContentType,
		&object.SizeBytes,
		&object.Checksum,
		&object.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &object, nil
}

func (r *PostgresObjectRepository) Delete(
	ctx context.Context,
	id uuid.UUID,
) error {

	query := `
	DELETE FROM objects
	WHERE id = $1
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		id,
	)

	return err
}

func (r *PostgresObjectRepository) ListByBucket(
	ctx context.Context,
	bucketID uuid.UUID,
) ([]models.Object, error) {

	query := `
	SELECT
		id,
		bucket_id,
		object_key,
		storage_path,
		content_type,
		size_bytes,
		checksum,
		created_at
	FROM objects
	WHERE bucket_id = $1
	ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(
		ctx,
		query,
		bucketID,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var objects []models.Object

	for rows.Next() {

		var object models.Object

		err := rows.Scan(
			&object.ID,
			&object.BucketID,
			&object.ObjectKey,
			&object.StoragePath,
			&object.ContentType,
			&object.SizeBytes,
			&object.Checksum,
			&object.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		objects = append(objects, object)
	}

	return objects, nil
}

func (r *PostgresObjectRepository) DeleteByKey(
	ctx context.Context,
	bucketID uuid.UUID,
	key string,
) error {

	_, err := r.db.ExecContext(
		ctx,
		`DELETE FROM objects WHERE bucket_id = $1 AND object_key = $2`,
		bucketID,
		key,
	)

	return err
}