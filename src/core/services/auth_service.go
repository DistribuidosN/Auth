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

// GetProfile retorna el perfil del usuario por UUID.
func (s *authService) GetProfile(userUUID string) (*entities.User, error) {
	if userUUID == "" {
		return nil, fmt.Errorf("userUUID requerido")
	}
	return s.authRepo.GetUserByUUID(userUUID)
}

// UpdateProfile actualiza el username. Solo cambia en memoria aquí — el repo
// debería tener un UpdateUser; por ahora retorna el usuario modificado.
func (s *authService) UpdateProfile(userUUID, newUsername string) (*entities.User, error) {
	if newUsername == "" {
		return nil, fmt.Errorf("el nuevo username no puede estar vacío")
	}
	existing, _ := s.authRepo.GetUserByUsername(newUsername)
	if existing != nil && existing.UserUUID != userUUID {
		return nil, fmt.Errorf("el username '%s' ya está en uso", newUsername)
	}
	user, err := s.authRepo.GetUserByUUID(userUUID)
	if err != nil {
		return nil, fmt.Errorf("usuario no encontrado")
	}
	user.Username = newUsername
	return user, nil
}

// DeleteAccount desactiva la cuenta (soft delete).
func (s *authService) DeleteAccount(userUUID string) error {
	_, err := s.authRepo.GetUserByUUID(userUUID)
	if err != nil {
		return fmt.Errorf("usuario no encontrado")
	}
	// Aquí iría un repo.UpdateUserStatus(userUUID, 0)
	// Por ahora el puerto driven necesita ese método — se añade después.
	return nil
}
