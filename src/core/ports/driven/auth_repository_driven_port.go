// Package driven contiene los puertos SECUNDARIOS (driven ports).
// Son las interfaces que el núcleo de la aplicación define y que los
// adaptadores externos (Postgres, JWT) deben implementar.
// El núcleo EXIGE estas interfaces — lo implementa el mundo exterior.
package driven

import "Auth/core/domain/entities"

// AuthRepositoryPort define lo que el servicio de auth necesita de la base de datos.
// Lo implementa: adapters/repository/auth_repository.go
//
// Es un puerto driven porque el núcleo lo LLAMA hacia afuera
// (hacia Postgres), no al revés.
type AuthRepositoryPort interface {
	// CreateUser inserta un usuario y retorna el registro completo (con id, created_at).
	CreateUser(user *entities.User) (*entities.User, error)

	// GetUserByUsername busca un usuario activo por su nombre de usuario.
	GetUserByUsername(username string) (*entities.User, error)

	// GetUserByEmail busca un usuario por su correo electrónico.
	GetUserByEmail(email string) (*entities.User, error)

	// UpdatePassword updates the user's password.
	UpdatePassword(userUUID string, newPasswordHash string) error
}