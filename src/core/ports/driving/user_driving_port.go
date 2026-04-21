// Package driving contiene los puertos PRIMARIOS (driving ports).
// Son las interfaces que los adaptadores externos (HTTP handlers, middleware)
// usan para llamar al núcleo de la aplicación.
// El núcleo EXPONE estas interfaces — el mundo exterior las llama.
package driving

import "Auth/core/domain/entities"

// UserServicePort define lo que el handler HTTP puede pedirle al servicio de usuarios.
// Lo implementa: core/services/user_service.go
type UserServicePort interface {
	GetProfile(claims *TokenClaims) (*entities.User, error)
	UpdateProfile(claims *TokenClaims, username, email string, roleId, status int) (*entities.User, error)
	DeleteAccount(claims *TokenClaims) error
	SearchUser(username string) (*entities.User, error)
	GetRoleByID(roleID int) (*entities.Role, error)
	GetPermissionsByRoleID(roleID int) ([]*entities.Permission, error)
}