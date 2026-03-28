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
// Rutas públicas  (sin auth): /auth/register, /auth/login, /health
// Rutas protegidas (con auth): /auth/profile, /auth/validate, /auth/account
func NewRouter(
	authHandler *handlers.AuthHandler,
	tokenSvc    driving.TokenServicePort,
	logger      *slog.Logger,
) http.Handler {

	mux := http.NewServeMux()

	// Middleware de autenticación listo para aplicar a rutas protegidas
	authMw := middleware.Auth(tokenSvc)

	// ── Health check ─────────────────────────────────────────────────────────
	// Sin auth — usado por el orquestador para verificar que el servicio vive
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		middleware.WriteJSON(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"service": "auth-server",
		})
	})

	mux.HandleFunc("GET  /api", func (w http.ResponseWriter, r *http.Request)  {
		endpoints :=  []map[string]any{
			{
                "path":        "/health",
                "method":      "GET",
                "auth":        false,
                "description": "Verifica que el servicio esté vivo",
            },
            {
                "path":        "/api",
                "method":      "GET",
                "auth":        false,
                "description": "Lista y documenta todos los endpoints disponibles",
            },
            {
                "path":        "/auth/register",
                "method":      "POST",
                "auth":        false,
                "description": "Crea una cuenta nueva. Requiere username, password y role_id.",
            },
            {
                "path":        "/auth/login",
                "method":      "POST",
                "auth":        false,
                "description": "Autentica al usuario y devuelve un token JWT.",
            },
            {
                "path":        "/auth/profile",
                "method":      "GET",
                "auth":        true,
                "description": "Retorna el perfil del usuario autenticado.",
            },
            {
                "path":        "/auth/profile",
                "method":      "PUT",
                "auth":        true,
                "description": "Actualiza el username del usuario autenticado.",
            },
            {
                "path":        "/auth/account",
                "method":      "DELETE",
                "auth":        true,
                "description": "Desactiva la cuenta del usuario autenticado (soft delete).",
            },
            {
                "path":        "/auth/validate",
                "method":      "POST",
                "auth":        true,
                "description": "Valida un token proporcionado. Útil para comunicación inter-servicios.",
            },
		}

		middleware.WriteJSON(w,   http.StatusOK, map[string]any{
			"api_version": "v1",
			"service":  "Auth Server",
			"endpoints": endpoints,
		})
	})

	// ── Rutas públicas ────────────────────────────────────────────────────────
	// No requieren token — son el punto de entrada al sistema

	// POST /auth/register → crear cuenta nueva
	mux.HandleFunc("POST /auth/register", authHandler.Register)

	// POST /auth/login → autenticarse y recibir JWT
	mux.HandleFunc("POST /auth/login", authHandler.Login)

	// ── Rutas protegidas ──────────────────────────────────────────────────────
	// El middleware.Auth valida el Bearer token antes de llegar al handler.
	// Si el token es inválido → 401 automático, el handler nunca se ejecuta.

	// GET /auth/profile → ver perfil del usuario autenticado
	mux.Handle("GET /auth/profile",
		authMw(http.HandlerFunc(authHandler.Profile)))

	// PUT /auth/profile → actualizar username
	mux.Handle("PUT /auth/profile",
		authMw(http.HandlerFunc(authHandler.UpdateProfile)))

	// DELETE /auth/account → desactivar cuenta (soft delete)
	mux.Handle("DELETE /auth/account",
		authMw(http.HandlerFunc(authHandler.DeleteAccount)))

	// POST /auth/validate → verificar un token (para uso de otros servicios)
	// El token a validar viene en el body, pero también se valida el Bearer del caller
	mux.Handle("POST /auth/validate",
		authMw(http.HandlerFunc(authHandler.ValidateToken)))

	// ── Middlewares globales (se aplican a TODAS las rutas) ───────────────────
	// Orden de ejecución: CORS → Logger → mux
	// CORS va primero para responder OPTIONS sin llegar al logger
	return middleware.CORS(
		middleware.Logger(logger)(
			middleware.JSONContentType(mux),
		),
	)
}