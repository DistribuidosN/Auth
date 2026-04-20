// Package services contiene la lógica de negocio pura.
// Solo importa los puertos (interfaces) — nunca Postgres, JWT, ni HTTP directamente.
package services

import (
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"Auth/core/domain/entities"
	"Auth/core/ports/driven"  // lo que el servicio exige hacia afuera (repo, JWT generator)
	"Auth/core/ports/driving" // la interfaz que este servicio implementa
)

type authService struct {
	authRepo driven.AuthRepositoryPort
	tokenSvc driven.TokenGeneratorPort
}

// NewAuthService crea el servicio inyectando interfaces driven (no concreciones).
// Retorna la interfaz driving para que main.go la pase a los handlers.
func NewAuthService(
	repo driven.AuthRepositoryPort,
	tokenSvc driven.TokenGeneratorPort,
) driving.AuthServicePort {
	return &authService{
		authRepo: repo,
		tokenSvc: tokenSvc,
	}
}

// Register crea un nuevo usuario con contraseña hasheada.
func (s *authService) Register(username, password string, email string, roleID int) (*entities.User, error) {
	if username == "" || password == "" {
		return nil, fmt.Errorf("username y password son requeridos")
	}
	if len(password) < 8 {
		return nil, fmt.Errorf("la contraseña debe tener al menos 8 caracteres")
	}

	// Regla de negocio: username único
	existing, _ := s.authRepo.GetUserByUsername(username)
	if existing != nil {
		return nil, fmt.Errorf("el username '%s' ya está registrado", username)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, fmt.Errorf("error hasheando contraseña: %w", err)
	}

	user := &entities.User{
		UserUUID:     uuid.New().String(),
		Username:     username,
		Email:        email,
		PasswordHash: string(hash),
		RoleID:       roleID,
		Status:       1,
	}
	return s.authRepo.CreateUser(user)
}

// Login valida credenciales y retorna un JWT + el usuario.
func (s *authService) Login(username, password string) (string, *entities.User, error) {
	if username == "" || password == "" {
		return "", nil, fmt.Errorf("username y password son requeridos")
	}

	user, err := s.authRepo.GetUserByUsername(username)
	if err != nil {
		return "", nil, fmt.Errorf("credenciales inválidas") // no revelar si existe
	}
	if user.Status != 1 {
		return "", nil, fmt.Errorf("cuenta desactivada")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", nil, fmt.Errorf("credenciales inválidas")
	}

	token, err := s.tokenSvc.GenerateToken(user)
	if err != nil {
		return "", nil, fmt.Errorf("error generando token: %w", err)
	}
	return token, user, nil
}

// Logout no tiene efecto real con JWT sin estado, pero se incluye por completitud.
func (s *authService) Logout(tokenStr string) error {
	// Aquí se podría implementar una lista negra (Redis)
	return nil
}

// ForgetPassword permite restablecer directamente la contraseña proporcionando el email.
func (s *authService) ForgetPassword(email string, newPassword string) error {
	if email == "" {
		return fmt.Errorf("el email es requerido")
	}
	if len(newPassword) < 8 {
		return fmt.Errorf("la nueva contraseña debe tener al menos 8 caracteres")
	}

	user, err := s.authRepo.GetUserByEmail(email)
	if err != nil {
		return fmt.Errorf("no se encontró una cuenta activa con ese correo")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return fmt.Errorf("error hasheando contraseña: %w", err)
	}

	return s.authRepo.UpdatePassword(user.UserUUID, string(hash))
}

// ResetPassword cambia la contraseña del usuario identificado en los claims.
func (s *authService) ResetPassword(claims *driving.TokenClaims, newPassword string) error {
	if claims == nil || claims.UserUUID == "" {
		return fmt.Errorf("identidad de usuario no válida")
	}
	if len(newPassword) < 8 {
		return fmt.Errorf("la nueva contraseña debe tener al menos 8 caracteres")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return fmt.Errorf("error hasheando contraseña: %w", err)
	}

	return s.authRepo.UpdatePassword(claims.UserUUID, string(hash))
}

// Detokenize decodifica y valida un tokenJWT 
func (s *authService) Detokenize(tokenStr string) (*driving.TokenClaims, error) {
	// Idealmente, el generador también podría tener la firma de ValidateToken, o podemos castear
	// Como tokenSvc actual es driven.TokenGeneratorPort, necesitamos una forma de acceder al validador.
	// Asumimos que TokenGeneratorPort de infrastructure puede validar o llamamos a un cast si se injectó el de JWT
	// Para respetar la inversión, el JWT service implementa driving.TokenServicePort también.
	// Haremos fail-safe cast o definimos que se debe pasar.
	if validador, ok := s.tokenSvc.(driving.TokenServicePort); ok {
		return validador.ValidateToken(tokenStr)
	}
	return nil, fmt.Errorf("el servicio de tokens inyectado no soporta validación")
}
