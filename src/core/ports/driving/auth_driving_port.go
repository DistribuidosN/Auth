// Package driving contiene los puertos PRIMARIOS (driving ports).
// Son las interfaces que los adaptadores externos (HTTP handlers, middleware)
// usan para llamar al núcleo de la aplicación.
// El núcleo EXPONE estas interfaces — el mundo exterior las llama.
package driving

import "Auth/core/domain/entities"

// AuthServicePort define lo que el handler HTTP puede pedirle al servicio.
// Lo implementa: core/services/auth_service.go
type AuthServicePort interface {
	Register(username, password string, email string, roleID int) (*entities.User, error)
	Login(identifier, password string) (token string, user *entities.User, err error)
	Logout(tokenStr string) error
	ForgetPassword(email string, newPassword string) error
	ResetPassword(claims *TokenClaims, newPassword string) error
	Detokenize(tokenStr string) (*TokenClaims, error)
}