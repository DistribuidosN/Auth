package services

import (
	"fmt"

	"Auth/core/domain/entities"
	"Auth/core/ports/driven"
	"Auth/core/ports/driving"
)

type userService struct {
	userRepo driven.UserRepositoryPort
}

// NewUserService crea el servicio de usuario.
func NewUserService(repo driven.UserRepositoryPort) driving.UserServicePort {
	return &userService{
		userRepo: repo,
	}
}

func (s *userService) GetProfile(claims *driving.TokenClaims) (*entities.User, error) {
	if claims == nil || claims.UserUUID == "" {
		return nil, fmt.Errorf("claims inválidos o falta UUID")
	}
	return s.userRepo.GetUserByUUID(claims.UserUUID)
}

func (s *userService) UpdateProfile(claims *driving.TokenClaims, username, email string, roleId, status int) (*entities.User, error) {
	if claims == nil || claims.UserUUID == "" {
		return nil, fmt.Errorf("claims inválidos")
	}
	if username == "" || email == "" {
		return nil, fmt.Errorf("username y email son obligatorios")
	}
	
	err := s.userRepo.UpdateUser(claims.UserUUID, username, email, roleId, status)
	if err != nil {
		return nil, fmt.Errorf("error actualizando perfil: %w", err)
	}

	return s.userRepo.GetUserByUUID(claims.UserUUID)
}

func (s *userService) DeleteAccount(claims *driving.TokenClaims) error {
	if claims == nil || claims.UserUUID == "" {
		return fmt.Errorf("claims inválidos")
	}
	
	// Soft delete: status = 0
	return s.userRepo.UpdateUserStatus(claims.UserUUID, 0)
}

func (s *userService) SearchUser(username string) (*entities.User, error) {
	if username == "" {
		return nil, fmt.Errorf("se requiere un username para buscar")
	}
	// Busca coincidencias parciales pero aquí simplificamos a la primera coincidencia
	users, err := s.userRepo.SearchUsers(username)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("usuario no encontrado")
	}
	return users[0], nil
}

func (s *userService) GetRoleByID(roleID int) (*entities.Role, error) {
	return s.userRepo.GetRoleByID(roleID)
}

func (s *userService) GetPermissionsByRoleID(roleID int) ([]*entities.Permission, error) {
	return s.userRepo.GetPermissionsByRoleID(roleID)
}
