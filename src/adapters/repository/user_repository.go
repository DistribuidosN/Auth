package repository

import (
	"fmt"

	"github.com/jmoiron/sqlx"

	"Auth/core/domain/entities"
	"Auth/core/ports/driven"
)

type postgresUserRepo struct {
	db *sqlx.DB
}

// NewUserRepository crea el repositorio de usuarios.
func NewUserRepository(db *sqlx.DB) driven.UserRepositoryPort {
	return &postgresUserRepo{db: db}
}

func (r *postgresUserRepo) GetUserByUUID(uuid string) (*entities.User, error) {
	query := `
		SELECT id, user_uuid, username, email, password_hash, role_id, status, created_at
		FROM users
		WHERE user_uuid = $1`

	user := &entities.User{}
	if err := r.db.Get(user, query, uuid); err != nil {
		return nil, fmt.Errorf("GetUserByUUID(%s): %w", uuid, err)
	}
	return user, nil
}

func (r *postgresUserRepo) UpdateUser(uuid, username, email string, roleId, status int) error {
	query := `UPDATE users SET username = $1, email = $2, role_id = $3, status = $4 WHERE user_uuid = $5`
	_, err := r.db.Exec(query, username, email, roleId, status, uuid)
	if err != nil {
		return fmt.Errorf("UpdateUser(%s): %w", uuid, err)
	}
	return nil
}

func (r *postgresUserRepo) UpdateUserStatus(uuid string, status int) error {
	query := `UPDATE users SET status = $1 WHERE user_uuid = $2`
	_, err := r.db.Exec(query, status, uuid)
	if err != nil {
		return fmt.Errorf("UpdateUserStatus(%s): %w", uuid, err)
	}
	return nil
}

func (r *postgresUserRepo) SearchUsers(username string) ([]*entities.User, error) {
	query := `
		SELECT id, user_uuid, username, email, password_hash, role_id, status, created_at
		FROM users
		WHERE username ILIKE $1`

	var users []*entities.User
	// agregamos comodines para búsqueda parcial
	searchTerm := "%" + username + "%"
	if err := r.db.Select(&users, query, searchTerm); err != nil {
		return nil, fmt.Errorf("SearchUsers(%s): %w", username, err)
	}
	return users, nil
}

func (r *postgresUserRepo) GetRoleByID(roleID int) (*entities.Role, error) {
	query := `SELECT id, name, status FROM roles WHERE id = $1`
	role := &entities.Role{}
	if err := r.db.Get(role, query, roleID); err != nil {
		return nil, fmt.Errorf("GetRoleByID(%d): %w", roleID, err)
	}
	return role, nil
}

func (r *postgresUserRepo) GetPermissionsByRoleID(roleID int) ([]*entities.Permission, error) {
	query := `
		SELECT p.id, p.description, p.route
		FROM permissions p
		INNER JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_id = $1`

	var perms []*entities.Permission
	if err := r.db.Select(&perms, query, roleID); err != nil {
		return nil, fmt.Errorf("GetPermissionsByRoleID(%d): %w", roleID, err)
	}
	return perms, nil
}
