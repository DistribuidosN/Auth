// Package repository implementa los adaptadores de persistencia.
// Implementa ports.AuthRepositoryPort usando Postgres + sqlx.
package repository

import (
	"fmt"

	"github.com/jmoiron/sqlx"

	"Auth/core/domain/entities"
	"Auth/core/ports/driven"
)

type postgresAuthRepo struct {
	db *sqlx.DB
}

// NewAuthRepository crea el repositorio de autenticación.
func NewAuthRepository(db *sqlx.DB) driven.AuthRepositoryPort {
	return &postgresAuthRepo{db: db}
}

// CreateUser inserta un usuario nuevo y retorna el registro creado con su ID y timestamps.
func (r *postgresAuthRepo) CreateUser(user *entities.User) (*entities.User, error) {
	insertQuery := `
		INSERT INTO users (user_uuid, username, email, password_hash, role_id, status)
		VALUES (?, ?, ?, ?, ?, ?)`

	result, err := r.db.Exec(insertQuery,
		user.UserUUID,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.RoleID,
		user.Status,
	)
	if err != nil {
		return nil, fmt.Errorf("CreateUser exec: %w", err)
	}

	// 2. Obtenemos el ID autoincremental que MySQL le acaba de asignar
	lastID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("CreateUser LastInsertId: %w", err)
	}

	// 3. Hacemos un SELECT para obtener el registro completo (incluyendo el created_at)
	created := &entities.User{}
	selectQuery := `
		SELECT id, user_uuid, username, email, password_hash, role_id, status, created_at 
		FROM users 
		WHERE id = ?`

	// Usamos Get de sqlx que mapea automáticamente un solo resultado al struct
	err = r.db.Get(created, selectQuery, lastID)
	if err != nil {
		return nil, fmt.Errorf("CreateUser select: %w", err)
	}

	return created, nil
}

// GetUserByUsername busca un usuario por su nombre de usuario.
func (r *postgresAuthRepo) GetUserByUsername(username string) (*entities.User, error) {
	query := `
		SELECT id, user_uuid, username, password_hash, role_id, status, created_at
		FROM users
		WHERE username = $1 AND status = 1`

	user := &entities.User{}
	if err := r.db.Get(user, query, username); err != nil {
		return nil, fmt.Errorf("GetUserByUsername(%s): %w", username, err)
	}
	return user, nil
}

// GetUserByUUID busca un usuario por su UUID público.
func (r *postgresAuthRepo) GetUserByUUID(uuid string) (*entities.User, error) {
	query := `
		SELECT id, user_uuid, username, password_hash, role_id, status, created_at
		FROM users
		WHERE user_uuid = $1`

	user := &entities.User{}
	if err := r.db.Get(user, query, uuid); err != nil {
		return nil, fmt.Errorf("GetUserByUUID(%s): %w", uuid, err)
	}
	return user, nil
}

// GetRoleByID obtiene un rol por su ID.
func (r *postgresAuthRepo) GetRoleByID(roleID int) (*entities.Role, error) {
	query := `SELECT id, name, status FROM roles WHERE id = $1`
	role := &entities.Role{}
	if err := r.db.Get(role, query, roleID); err != nil {
		return nil, fmt.Errorf("GetRoleByID(%d): %w", roleID, err)
	}
	return role, nil
}
