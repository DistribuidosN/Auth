// Package auth implementa el adaptador JWT.
// Es un adaptador secundario (driven) que implementa dos puertos:
//   - driven.TokenGeneratorPort  → lo usa el servicio de auth para GENERAR tokens
//   - driving.TokenServicePort   → lo usa el middleware para VALIDAR tokens
package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"Auth/core/domain/entities"
	"Auth/core/ports/driven"
	"Auth/core/ports/driving"
)

type jwtClaims struct {
	UserUUID string `json:"user_uuid"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type jwtTokenService struct {
	secretKey     []byte
	serviceSecret []byte
	tokenDuration time.Duration
}

// jwtTokenService implementa driven.TokenGeneratorPort (generación)
var _ driven.TokenGeneratorPort = (*jwtTokenService)(nil)

// jwtTokenService implementa driving.TokenServicePort (validación)
var _ driving.TokenServicePort = (*jwtTokenService)(nil)

// NewJWTService crea el adaptador JWT.
// Retorna interface{} compuesta — main.go la convierte al puerto que necesite.
// En la práctica, main.go lo pasa como driven.TokenGeneratorPort al servicio
// y como driving.TokenServicePort al middleware.
func NewJWTService(secretKey, serviceSecret string) *jwtTokenService {
	return &jwtTokenService{
		secretKey:     []byte(secretKey),
		serviceSecret: []byte(serviceSecret),
		tokenDuration: 24 * time.Hour,
	}
}

// ── driven.TokenGeneratorPort ─────────────────────────────────────────────────

// GenerateToken crea un JWT firmado para un usuario autenticado.
func (s *jwtTokenService) GenerateToken(user *entities.User) (string, error) {
	claims := jwtClaims{
		UserUUID: user.UserUUID,
		Username: user.Username,
		Role:     fmt.Sprintf("%d", user.RoleID),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "auth-server",
			Subject:   user.UserUUID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.tokenDuration)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secretKey)
	if err != nil {
		return "", fmt.Errorf("error firmando token: %w", err)
	}
	return signed, nil
}

// GenerateServiceToken crea un JWT de corta vida para comunicación inter-servicios.
func (s *jwtTokenService) GenerateServiceToken(serviceID string) (string, error) {
	claims := jwtClaims{
		UserUUID: serviceID,
		Username: serviceID,
		Role:     "SERVICE",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "auth-server",
			Subject:   serviceID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.serviceSecret)
	if err != nil {
		return "", fmt.Errorf("error firmando service token: %w", err)
	}
	return signed, nil
}

// ── driving.TokenServicePort ──────────────────────────────────────────────────

// ValidateToken verifica la firma y expiración del JWT.
// Lo llama el middleware Auth antes de cada request protegida.
func (s *jwtTokenService) ValidateToken(tokenStr string) (*driving.TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwtClaims{}, func(t *jwt.Token) (interface{}, error) {
		// Protección contra ataque "alg:none"
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("algoritmo inesperado: %v", t.Header["alg"])
		}
		return s.secretKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("token inválido: %w", err)
	}
	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("claims inválidos")
	}
	return &driving.TokenClaims{
		UserUUID: claims.UserUUID,
		Username: claims.Username,
		Role:     claims.Role,
	}, nil
}