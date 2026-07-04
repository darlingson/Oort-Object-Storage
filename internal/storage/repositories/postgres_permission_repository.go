package repositories

import (
	"context"
	"database/sql"

	"github.com/darlingson/Oort-Object-Storage/internal/storage/models"
)

type PostgresPermissionRepository struct {
	db *sql.DB
}

func NewPostgresPermissionRepository(db *sql.DB) *PostgresPermissionRepository {
	return &PostgresPermissionRepository{
		db: db,
	}
}

func (r *PostgresPermissionRepository) Create(
	ctx context.Context,
	permission *models.Permission,
) error {

	query := `
	INSERT INTO permissions(id, name, description)
	VALUES ($1, $2, $3)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		permission.ID,
		permission.Name,
		permission.Description,
	)

	return err
}

func (r *PostgresPermissionRepository) FindByName(
	ctx context.Context,
	name string,
) (*models.Permission, error) {

	query := `
	SELECT id, name, description, created_at
	FROM permissions
	WHERE name = $1
	`

	var p models.Permission

	err := r.db.QueryRowContext(
		ctx,
		query,
		name,
	).Scan(
		&p.ID,
		&p.Name,
		&p.Description,
		&p.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (r *PostgresPermissionRepository) List(
	ctx context.Context,
) ([]models.Permission, error) {

	query := `
	SELECT id, name, description, created_at
	FROM permissions
	ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var permissions []models.Permission

	for rows.Next() {

		var p models.Permission

		err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.Description,
			&p.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		permissions = append(permissions, p)
	}

	return permissions, nil
}
