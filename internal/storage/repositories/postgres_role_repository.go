package repositories

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"github.com/darlingson/Oort-Object-Storage/internal/storage/models"
)

type PostgresRoleRepository struct {
	db *sql.DB
}

func NewPostgresRoleRepository(db *sql.DB) *PostgresRoleRepository {
	return &PostgresRoleRepository{
		db: db,
	}
}

func (r *PostgresRoleRepository) Create(
	ctx context.Context,
	role *models.Role,
) error {

	query := `
	INSERT INTO roles(id, name)
	VALUES ($1, $2)
	`

	_, err := r.db.ExecContext(ctx, query, role.ID, role.Name)

	return err
}

func (r *PostgresRoleRepository) FindByName(
	ctx context.Context,
	name string,
) (*models.Role, error) {

	query := `
	SELECT id, name, created_at
	FROM roles
	WHERE name = $1
	`

	var role models.Role

	err := r.db.QueryRowContext(
		ctx,
		query,
		name,
	).Scan(
		&role.ID,
		&role.Name,
		&role.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &role, nil
}

func (r *PostgresRoleRepository) FindByUserID(
	ctx context.Context,
	userID uuid.UUID,
) ([]models.Role, error) {

	query := `
	SELECT r.id, r.name, r.created_at
	FROM roles r
	JOIN user_roles ur ON ur.role_id = r.id
	WHERE ur.user_id = $1
	`

	rows, err := r.db.QueryContext(ctx, query, userID)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var roles []models.Role

	for rows.Next() {

		var role models.Role

		err := rows.Scan(
			&role.ID,
			&role.Name,
			&role.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		roles = append(roles, role)
	}

	return roles, nil
}

func (r *PostgresRoleRepository) AssignPermission(
	ctx context.Context,
	roleID, permissionID uuid.UUID,
) error {

	query := `
	INSERT INTO role_permissions(role_id, permission_id)
	VALUES ($1, $2)
	`

	_, err := r.db.ExecContext(ctx, query, roleID, permissionID)

	return err
}

func (r *PostgresRoleRepository) ListPermissions(
	ctx context.Context,
	roleID uuid.UUID,
) ([]models.Permission, error) {

	query := `
	SELECT p.id, p.name, p.description, p.created_at
	FROM permissions p
	JOIN role_permissions rp ON rp.permission_id = p.id
	WHERE rp.role_id = $1
	ORDER BY p.name
	`

	rows, err := r.db.QueryContext(ctx, query, roleID)

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

func (r *PostgresRoleRepository) AssignToUser(
	ctx context.Context,
	userID, roleID uuid.UUID,
) error {

	query := `
	INSERT INTO user_roles(user_id, role_id)
	VALUES ($1, $2)
	`

	_, err := r.db.ExecContext(ctx, query, userID, roleID)

	return err
}
