// Package handlers contiene los adaptadores HTTP primarios.
// Traducen HTTP ↔ servicio. No contienen lógica de negocio.
package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"Auth/adapters/http/middleware"
	"Auth/core/ports/driving"
)

// AuthHandler maneja todos los endpoints de autenticación.
type AuthHandler struct {
	authSvc  driving.AuthServicePort
	tokenSvc driving.TokenServicePort
}

// NewAuthHandler crea el handler inyectando la interfaz del servicio y el validador de tokens.
func NewAuthHandler(authSvc driving.AuthServicePort, tokenSvc driving.TokenServicePort) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, tokenSvc: tokenSvc}
}

func (h *AuthHandler) extractClaims(w http.ResponseWriter, r *http.Request) *driving.TokenClaims {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		middleware.WriteError(w, http.StatusUnauthorized, "header Authorization requerido")
		return nil
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		middleware.WriteError(w, http.StatusUnauthorized, "formato inválido — use: Bearer <token>")
		return nil
	}
	claims, err := h.tokenSvc.ValidateToken(parts[1])
	if err != nil {
		middleware.WriteError(w, http.StatusUnauthorized, "token inválido o expirado")
		return nil
	}
	return claims
}

// POST /auth/register

// registerRequest es el body esperado para registrar un usuario.
type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	RoleID   int    `json:"role_id"`
}

// Register crea una nueva cuenta de usuario.
//
//	POST /auth/register
//	Body: { "username": "christian", "password": "segura1234", "role_id": 1 }
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "body JSON inválido")
		return
	}

	if req.RoleID == 0 {
		req.RoleID = 2 // rol por defecto: usuario normal
	}

	user, err := h.authSvc.Register(req.Username, req.Password, req.Email, req.RoleID)
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	middleware.WriteJSON(w, http.StatusCreated, map[string]any{
		"message":   "usuario creado exitosamente",
		"user_uuid": user.UserUUID,
		"username":  user.Username,
	})
}

//  POST /auth/login

// loginRequest es el body para autenticarse.
type loginRequest struct {
	Identity string `json:"identity"` // Acepta username o email
	Password string `json:"password"`
}

// Login valida credenciales y devuelve un JWT.
//
//	POST /auth/login
//	Body: { "identity": "christian@mail.com", "password": "..." }
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "body JSON inválido")
		return
	}

	token, user, err := h.authSvc.Login(req.Identity, req.Password)
	if err != nil {
		// 401 para credenciales inválidas — no revelar si el usuario existe
		middleware.WriteError(w, http.StatusUnauthorized, err.Error())
		return
	}

	middleware.WriteJSON(w, http.StatusOK, map[string]any{
		"token":     token,
		"user_uuid": user.UserUUID,
		"username":  user.Username,
		"role_id":   user.RoleID,
	})
}

// DELETE /auth/logout

// Logout maneja el cierre de sesión simulado.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Solo devolvemos ok, el frontend debe borrar su token
	middleware.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "sesión cerrada exitosamente",
	})
}

// POST /auth/forget-password

type forgetRequest struct {
	Email       string `json:"email"`
	NewPassword string `json:"new_password"`
}

// ForgetPassword cambia la contraseña usando el email brindado
// POST /auth/forget-password
// Body: { "email": "test@test.com", "new_password": "..." }
func (h *AuthHandler) ForgetPassword(w http.ResponseWriter, r *http.Request) {
	var req forgetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "body JSON inválido")
		return
	}
	
	err := h.authSvc.ForgetPassword(req.Email, req.NewPassword)
	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "contraseña actualizada exitosamente",
	})
}

// ValidateToken permite a otros servicios verificar un token. El token DEBE viajar en el header Authorization.
func (h *AuthHandler) ValidateToken(w http.ResponseWriter, r *http.Request) {
	claims := h.extractClaims(w, r)
	if claims == nil {
		return
	}

	middleware.WriteJSON(w, http.StatusOK, map[string]any{
		"valid":     true,
		"user_uuid": claims.UserUUID,
		"username":  claims.Username,
		"role":      claims.Role,
	})
}

// POST /auth/reset-password

type resetRequest struct {
	NewPassword string `json:"new_password"`
}

// ResetPassword cambia la contraseña. El token de autorización DEBE viajar en el header.
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	claims := h.extractClaims(w, r)
	if claims == nil {
		return
	}

	var req resetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "body JSON inválido")
		return
	}

	if req.NewPassword == "" {
		middleware.WriteError(w, http.StatusBadRequest, "new_password es requerido")
		return
	}

	err := h.authSvc.ResetPassword(claims, req.NewPassword)
	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "contraseña actualizada exitosamente",
	})
}