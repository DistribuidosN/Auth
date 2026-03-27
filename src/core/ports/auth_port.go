package ports

import "Auth/core/domain"

// AuthRepository define las operaciones de base de datos que el servicio necesitará
type AuthRepositoryPort interface {
	CreateUser(user *domain.User) error
	GetUserByUsername(username string) (*domain.User, error)
	GetUserByUUID(uuid string) (*domain.User, error)
}