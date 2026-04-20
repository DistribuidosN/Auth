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
	query := `
		INSERT INTO users (user_uuid, username, email, password_hash, role_id, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_uuid, username, email, password_hash, role_id, status, created_at`

	created := &entities.User{}
	err := r.db.Get(created, query,
		user.UserUUID,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.RoleID,
		user.Status,
	)
	if err != nil {
		return nil, fmt.Errorf("CreateUser: %w", err)
	}

	return created, nil
}

func (r *postgresAuthRepo) GetUserByUsername(username string) (*entities.User, error) {
	query := `
		SELECT id, user_uuid, username, email, password_hash, role_id, status, created_at
		FROM users
		WHERE username = $1 AND status = 1`

	user := &entities.User{}
	if err := r.db.Get(user, query, username); err != nil {
		return nil, fmt.Errorf("GetUserByUsername(%s): %w", username, err)
	}
	return user, nil
}

// GetUserByEmail busca un usuario por su correo electrónico.
func (r *postgresAuthRepo) GetUserByEmail(email string) (*entities.User, error) {
	query := `
		SELECT id, user_uuid, username, email, password_hash, role_id, status, created_at
		FROM users
		WHERE email = $1 AND status = 1`

	user := &entities.User{}
	if err := r.db.Get(user, query, email); err != nil {
		return nil, fmt.Errorf("GetUserByEmail(%s): %w", email, err)
	}
	return user, nil
}

// UpdatePassword updates the user's password hash.
func (r *postgresAuthRepo) UpdatePassword(userUUID string, newPasswordHash string) error {
	query := `UPDATE users SET password_hash = $1 WHERE user_uuid = $2`
	_, err := r.db.Exec(query, newPasswordHash, userUUID)
	if err != nil {
		return fmt.Errorf("UpdatePassword(%s): %w", userUUID, err)
	}
	return nil
}
