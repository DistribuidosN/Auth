package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"Auth/adapters/http/middleware"
	"Auth/core/ports/driving"
)

type UserHandler struct {
	userSvc  driving.UserServicePort
	tokenSvc driving.TokenServicePort
}

func NewUserHandler(userSvc driving.UserServicePort, tokenSvc driving.TokenServicePort) *UserHandler {
	return &UserHandler{userSvc: userSvc, tokenSvc: tokenSvc}
}

func (h *UserHandler) extractClaims(w http.ResponseWriter, r *http.Request) *driving.TokenClaims {
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

// GET /user/profile

// Profile retorna el perfil del usuario autenticado.
func (h *UserHandler) Profile(w http.ResponseWriter, r *http.Request) {
	claims := h.extractClaims(w, r)
	if claims == nil {
		return
	}

	user, err := h.userSvc.GetProfile(claims)
	if err != nil {
		middleware.WriteError(w, http.StatusNotFound, "usuario no encontrado: "+err.Error())
		return
	}

	middleware.WriteJSON(w, http.StatusOK, user)
}

// PUT /user/profile

type updateRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	RoleID   int    `json:"role_id"`
	Status   int    `json:"status"`
}

// UpdateProfile actualiza el perfil del usuario autenticado.
func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	claims := h.extractClaims(w, r)
	if claims == nil {
		return
	}

	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "body JSON inválido")
		return
	}

	user, err := h.userSvc.UpdateProfile(claims, req.Username, req.Email, req.RoleID, req.Status)
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	middleware.WriteJSON(w, http.StatusOK, map[string]any{
		"message":  "perfil actualizado",
		"username": user.Username,
		"email":    user.Email,
		"role_id":  user.RoleID,
		"status":   user.Status,
		"valid":    true,
	})
}

// DELETE /user/account

// DeleteAccount desactiva la cuenta del usuario autenticado.
func (h *UserHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	claims := h.extractClaims(w, r)
	if claims == nil {
		return
	}

	if err := h.userSvc.DeleteAccount(claims); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "cuenta desactivada exitosamente",
	})
}

// GET /user/search?username=...

func (h *UserHandler) SearchUser(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		middleware.WriteError(w, http.StatusBadRequest, "parámetro username es requerido")
		return
	}

	user, err := h.userSvc.SearchUser(username)
	if err != nil {
		middleware.WriteError(w, http.StatusNotFound, err.Error())
		return
	}

	middleware.WriteJSON(w, http.StatusOK, user)
}

// GET /roles?role_id=...

func (h *UserHandler) GetRoleByID(w http.ResponseWriter, r *http.Request) {
	roleIDStr := r.URL.Query().Get("role_id")
	if roleIDStr == "" {
		middleware.WriteError(w, http.StatusBadRequest, "parámetro role_id es requerido")
		return
	}
	roleID, err := strconv.Atoi(roleIDStr)
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "role_id inválido")
		return
	}

	role, err := h.userSvc.GetRoleByID(roleID)
	if err != nil {
		middleware.WriteError(w, http.StatusNotFound, err.Error())
		return
	}
	middleware.WriteJSON(w, http.StatusOK, role)
}

// GET /permissions?role_id=...

func (h *UserHandler) GetPermissionsByRoleID(w http.ResponseWriter, r *http.Request) {
	roleIDStr := r.URL.Query().Get("role_id")
	if roleIDStr == "" {
		middleware.WriteError(w, http.StatusBadRequest, "parámetro role_id es requerido")
		return
	}
	roleID, err := strconv.Atoi(roleIDStr)
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "role_id inválido")
		return
	}

	perms, err := h.userSvc.GetPermissionsByRoleID(roleID)
	if err != nil {
		middleware.WriteError(w, http.StatusNotFound, err.Error())
		return
	}
	middleware.WriteJSON(w, http.StatusOK, perms)
}