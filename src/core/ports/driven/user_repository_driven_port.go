package driven

import "Auth/core/domain/entities"

// UserRepositoryPort define lo que el servicio de usuarios necesita de la base de datos.
type UserRepositoryPort interface {
	// GetUserByUUID busca un usuario por su UUID público.
	GetUserByUUID(uuid string) (*entities.User, error)

	// UpdateUser actualiza múltiples campos de un usuario.
	UpdateUser(uuid, username, email string, roleId, status int) error

	// UpdateUserStatus desactiva o activa a un usuario (0 inactivo, 1 activo).
	UpdateUserStatus(uuid string, status int) error

	// SearchUsers busca usuarios por username (coincidencia parcial o exacta).
	SearchUsers(username string) ([]*entities.User, error)

	// GetRoleByID obtiene un rol por su ID.
	GetRoleByID(roleID int) (*entities.Role, error)

	// GetPermissionsByRoleID lista todos los permisos asociados a un rol específico.
	GetPermissionsByRoleID(roleID int) ([]*entities.Permission, error)
}
