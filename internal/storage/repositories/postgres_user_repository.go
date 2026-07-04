package repositories

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"github.com/darlingson/Oort-Object-Storage/internal/storage/models"
)

type PostgresUserRepository struct {
	db *sql.DB
}

func NewPostgresUserRepository(db *sql.DB) *PostgresUserRepository {
	return &PostgresUserRepository{
		db: db,
	}
}

func (r *PostgresUserRepository) Create(
	ctx context.Context,
	user *models.User,
) error {

	query := `
	INSERT INTO users(id, email, password_hash)
	VALUES ($1, $2, $3)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		user.ID,
		user.Email,
		user.PasswordHash,
	)

	return err
}

func (r *PostgresUserRepository) FindByEmail(
	ctx context.Context,
	email string,
) (*models.User, error) {

	query := `
	SELECT id, email, password_hash, created_at
	FROM users
	WHERE email = $1
	`

	var user models.User

	err := r.db.QueryRowContext(
		ctx,
		query,
		email,
	).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *PostgresUserRepository) FindByID(
	ctx context.Context,
	id uuid.UUID,
) (*models.User, error) {

	query := `
	SELECT id, email, password_hash, created_at
	FROM users
	WHERE id = $1
	`

	var user models.User

	err := r.db.QueryRowContext(
		ctx,
		query,
		id,
	).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *PostgresUserRepository) AssignPermission(
	ctx context.Context,
	userID, permissionID uuid.UUID,
) error {

	query := `
	INSERT INTO user_permissions(user_id, permission_id)
	VALUES ($1, $2)
	`

	_, err := r.db.ExecContext(ctx, query, userID, permissionID)

	return err
}

func (r *PostgresUserRepository) GetEffectivePermissions(
	ctx context.Context,
	userID uuid.UUID,
) ([]models.Permission, error) {

	query := `
	SELECT p.id, p.name, p.description, p.created_at
	FROM permissions p
	JOIN user_permissions up ON up.permission_id = p.id
	WHERE up.user_id = $1

	UNION

	SELECT p.id, p.name, p.description, p.created_at
	FROM permissions p
	JOIN role_permissions rp ON rp.permission_id = p.id
	JOIN user_roles ur ON ur.role_id = rp.role_id
	WHERE ur.user_id = $1

	ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query, userID)

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
