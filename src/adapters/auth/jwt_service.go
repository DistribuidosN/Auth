// Package auth implementa el adaptador de autenticación JWT.
// Es un adaptador secundario (driven) que implementa el puerto TokenService.
package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"Auth/core/ports"
)

// jwtClaims son los claims completos del JWT (incluyendo los estándar)
type jwtClaims struct {
	UserUUID string `json:"user_uuid"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type jwtTokenService struct {
	secretKey      []byte
	serviceSecret  []byte
	tokenDuration  time.Duration
}

// NewJWTTokenService crea el servicio de tokens JWT.
//   - secretKey: la misma clave que usa el Auth Server para firmar tokens de usuario
//   - serviceSecret: clave para tokens internos entre microservicios
func NewJWTTokenService(secretKey, serviceSecret string) ports.TokenService {
	return &jwtTokenService{
		secretKey:     []byte(secretKey),
		serviceSecret: []byte(serviceSecret),
		tokenDuration: 24 * time.Hour,
	}
}

// ValidateToken valida un JWT firmado por el Auth Server.
// El DB Server no genera tokens de usuario, solo los valida.
func (s *jwtTokenService) ValidateToken(tokenStr string) (*ports.TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwtClaims{}, func(t *jwt.Token) (interface{}, error) {
		// Verificar que el algoritmo es el esperado (HMAC)
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("algoritmo de firma inesperado: %v", t.Header["alg"])
		}
		return s.secretKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("token inválido: %w", err)
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("claims del token inválidos")
	}

	return &ports.TokenClaims{
		UserUUID: claims.UserUUID,
		Username: claims.Username,
		Role:     claims.Role,
	}, nil
}

// GenerateServiceToken genera un JWT de corta duración para comunicación
// interna entre el orquestador y este servidor DB.
func (s *jwtTokenService) GenerateServiceToken(serviceID string) (string, error) {
	claims := jwtClaims{
		UserUUID: serviceID,
		Username: serviceID,
		Role:     "SERVICE",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "db-server",
			Subject:   serviceID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.serviceSecret)
	if err != nil {
		return "", fmt.Errorf("error firmando token de servicio: %w", err)
	}
	return signed, nil
}
