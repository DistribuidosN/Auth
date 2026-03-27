package entities

import "time"

// User representa la entidad principal de usuarios
type User struct {
	ID           int       `json:"id" db:"id"`
	UserUUID     string    `json:"user_uuid" db:"user_uuid"`
	Username     string    `json:"username" db:"username"`
	PasswordHash string    `json:"-" db:"password_hash"` // El "-" oculta la contraseña cuando se devuelva en un JSON
	RoleID       int       `json:"role_id" db:"role_id"`
	Status       int       `json:"status" db:"status"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}