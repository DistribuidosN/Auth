// Package handlers contiene los adaptadores HTTP primarios.
// Traducen HTTP ↔ servicio. No contienen lógica de negocio.
package handlers

import (
	"encoding/json"
	"net/http"

	"Auth/adapters/http/middleware"
	"Auth/core/ports/driving"
)

// AuthHandler maneja todos los endpoints de autenticación.
type AuthHandler struct {
	authSvc driving.AuthServicePort
}

// NewAuthHandler crea el handler inyectando la interfaz del servicio.
func NewAuthHandler(authSvc driving.AuthServicePort) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
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
		req.RoleID = 1 // rol por defecto: usuario normal
	}

	user, err := h.authSvc.Register(req.Username, req.Password, req.Email , req.RoleID)
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
	Username string `json:"username"`
	Password string `json:"password"`
}

// Login valida credenciales y devuelve un JWT.
//
//	POST /auth/login
//	Body: { "username": "christian", "password": "segura1234" }
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "body JSON inválido")
		return
	}

	token, user, err := h.authSvc.Login(req.Username, req.Password)
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

// GET /auth/profile 

// Profile retorna el perfil del usuario autenticado.
// Requiere middleware.Auth aplicado — extrae el UUID del token.
//
//	GET /auth/profile
//	Header: Authorization: Bearer <token>
func (h *AuthHandler) Profile(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	if claims == nil {
		middleware.WriteError(w, http.StatusUnauthorized, "no autenticado")
		return
	}

	user, err := h.authSvc.GetProfile(claims.UserUUID)
	if err != nil {
		middleware.WriteError(w, http.StatusNotFound, "usuario no encontrado")
		return
	}

	middleware.WriteJSON(w, http.StatusOK, user)
}

// PUT /auth/profile

// updateRequest es el body para actualizar el perfil.
type updateRequest struct {
	Username string `json:"username"`
}

// UpdateProfile actualiza el username del usuario autenticado.
//
//	PUT /auth/profile
//	Header: Authorization: Bearer <token>
//	Body: { "username": "nuevo_nombre" }
func (h *AuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	if claims == nil {
		middleware.WriteError(w, http.StatusUnauthorized, "no autenticado")
		return
	}

	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "body JSON inválido")
		return
	}

	user, err := h.authSvc.UpdateProfile(claims.UserUUID, req.Username)
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	middleware.WriteJSON(w, http.StatusOK, map[string]any{
		"message":  "perfil actualizado",
		"username": user.Username,
	})
}

// DELETE /auth/account

// DeleteAccount desactiva la cuenta del usuario autenticado (soft delete).
//
//	DELETE /auth/account
//	Header: Authorization: Bearer <token>
func (h *AuthHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	if claims == nil {
		middleware.WriteError(w, http.StatusUnauthorized, "no autenticado")
		return
	}

	if err := h.authSvc.DeleteAccount(claims.UserUUID); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "cuenta desactivada correctamente",
	})
}

// POST /auth/validate

// ValidateToken permite a otros servicios verificar un token sin llamar al Auth Server.
// Útil para el orquestador Java o el DB Server.
//
//	POST /auth/validate
//	Body: { "token": "eyJhbGci..." }
func (h *AuthHandler) ValidateToken(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Token == "" {
		middleware.WriteError(w, http.StatusBadRequest, "campo 'token' requerido")
		return
	}

	// Reutiliza los claims que ya inyectó el middleware si la ruta está protegida,
	// o valida manualmente si esta ruta es pública.
	claims := middleware.GetClaims(r)
	if claims == nil {
		middleware.WriteError(w, http.StatusUnauthorized, "token inválido")
		return
	}

	middleware.WriteJSON(w, http.StatusOK, map[string]any{
		"valid":     true,
		"user_uuid": claims.UserUUID,
		"username":  claims.Username,
		"role":      claims.Role,
	})
}