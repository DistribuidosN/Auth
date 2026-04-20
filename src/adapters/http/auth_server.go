// Package http configura el router y conecta rutas con handlers y middlewares.
package http

import (
	"log/slog"
	"net/http"

	"Auth/adapters/http/handlers"
	"Auth/adapters/http/middleware"
	"Auth/core/ports/driving"
)

// NewRouter construye el ServeMux de Go 1.22 con todas las rutas.
//
// Rutas (sin middleware de auth, gestionado por Java):
// /auth/register, /auth/login, /auth/validate, /auth/logout, /auth/forget-password, /auth/reset-password
// /user/profile, /user/account, /user/search, /roles, /permissions
func NewRouter(
	authHandler *handlers.AuthHandler,
	userHandler *handlers.UserHandler,
	tokenSvc    driving.TokenServicePort,
	logger      *slog.Logger,
) http.Handler {

	mux := http.NewServeMux()

	// ── Health check ─────────────────────────────────────────────────────────
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		middleware.WriteJSON(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"service": "auth-server",
		})
	})

	mux.HandleFunc("GET /api", func(w http.ResponseWriter, r *http.Request) {
		endpoints := []map[string]any{
			{"path": "/health", "method": "GET", "auth": false, "description": "Verifica que el servicio esté vivo"},
			{"path": "/api", "method": "GET", "auth": false, "description": "Lista de endpoints"},
			{"path": "/auth/register", "method": "POST", "auth": false, "description": "Crear cuenta"},
			{"path": "/auth/login", "method": "POST", "auth": false, "description": "Iniciar sesión"},
			{"path": "/auth/validate", "method": "POST", "auth": false, "description": "Valida un token JWT"},
			{"path": "/auth/logout", "method": "POST", "auth": false, "description": "Cerrar sesión"},
			{"path": "/auth/forget-password", "method": "POST", "auth": false, "description": "Reseteo directo de clave vía email"},
			{"path": "/auth/reset-password", "method": "POST", "auth": false, "description": "Resetear clave (requiere token en header)"},

			{"path": "/user/profile", "method": "GET", "auth": false, "description": "Ver perfil (requiere token en header)"},
			{"path": "/user/profile", "method": "PUT", "auth": false, "description": "Actualizar perfil (requiere token en header)"},
			{"path": "/user/account", "method": "DELETE", "auth": false, "description": "Desactivar perfil (requiere token en header)"},
			{"path": "/user/search", "method": "GET", "auth": false, "description": "Buscar usuario"},
			{"path": "/roles", "method": "GET", "auth": false, "description": "Obtener rol por ID"},
			{"path": "/permissions", "method": "GET", "auth": false, "description": "Obtener permisos por rol ID"},
		}

		middleware.WriteJSON(w, http.StatusOK, map[string]any{
			"api_version": "v1",
			"service":     "Auth Server",
			"endpoints":   endpoints,
		})
	})

	// ── Rutas de Autenticación ──────────────────────────────────────────────────
	mux.HandleFunc("POST /auth/register", authHandler.Register)
	mux.HandleFunc("POST /auth/login", authHandler.Login)
	mux.HandleFunc("POST /auth/validate", authHandler.ValidateToken) // Expuesto din middleware para auth inter-servicio
	mux.HandleFunc("POST /auth/logout", authHandler.Logout)
	mux.HandleFunc("POST /auth/forget-password", authHandler.ForgetPassword)
	mux.HandleFunc("POST /auth/reset-password", authHandler.ResetPassword)

	// ── Rutas de Usuario ────────────────────────────────────────────────────────
	// El middleware ya no protege estas rutas en el enrutamiento.
	// La comprobación del token la hace el Handler manualmente invocando `extractClaims`
	mux.HandleFunc("GET /user/profile", userHandler.Profile)
	mux.HandleFunc("PUT /user/profile", userHandler.UpdateProfile)
	mux.HandleFunc("DELETE /user/account", userHandler.DeleteAccount)
	mux.HandleFunc("GET /user/search", userHandler.SearchUser)
	mux.HandleFunc("GET /roles", userHandler.GetRoleByID)
	mux.HandleFunc("GET /permissions", userHandler.GetPermissionsByRoleID)


	// ── Middlewares globales (se aplican a TODAS las rutas) ───────────────────
	// Orden de ejecución: CORS → Logger → mux
	return middleware.CORS(
		middleware.Logger(logger)(
			middleware.JSONContentType(mux),
		),
	)
}