package driven

import "Auth/core/domain/entities"

// TokenGeneratorPort define lo que el servicio de auth necesita del JWT.
// Es un puerto driven: el núcleo lo exige, el adaptador JWT lo implementa.
//
// Solo necesita GENERAR tokens — la validación la hace el middleware
// a través de driving.TokenServicePort.
//
// Lo implementa: adapters/auth/jwt_service.go
type TokenGeneratorPort interface {
	GenerateToken(user *entities.User) (string, error)
	GenerateServiceToken(serviceID string) (string, error)
}