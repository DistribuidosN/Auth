package driving

import "Auth/core/domain/entities"

// TokenServicePort define las operaciones JWT que el middleware y handlers usan.
// Lo implementa: adapters/auth/jwt_service.go
//
// Es un puerto driving porque el adaptador HTTP (middleware) lo LLAMA
// para validar tokens — el flujo va de fuera hacia adentro.
type TokenServicePort interface {
	// ValidateToken verifica firma y expiración. Lo usa el middleware Auth.
	ValidateToken(tokenStr string) (*TokenClaims, error)

	// GenerateToken crea un JWT para un usuario autenticado. Lo usa el servicio de auth.
	GenerateToken(user *entities.User) (string, error)

	// GenerateServiceToken crea un JWT inter-servicios (orquestador → auth).
	GenerateServiceToken(serviceID string) (string, error)
}

// TokenClaims son los datos que se extraen de un JWT válido.
type TokenClaims struct {
	UserUUID string `json:"user_uuid"`
	Username string `json:"username"`
	Role     string `json:"role"`
}