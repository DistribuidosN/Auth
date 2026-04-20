package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"

	jwtAdapter "Auth/adapters/auth"
	httpServer "Auth/adapters/http"
	"Auth/adapters/http/handlers"
	"Auth/adapters/repository"
	"Auth/core/services"
	"Auth/infraestructure/config"
	infraDB "Auth/infraestructure/db"
)

func main() {
	logger := setupLogger()
	cfg := config.LoadConfig()

	// Conectar a la base de datos
	db := setupDatabase(cfg, logger)
	//El defer se queda en el main para que la conexión
	// viva mientras el servidor esté encendido.
	defer db.Close()

	// Ensamblar la aplicación y obtener el servidor HTTP
	srv := buildServer(cfg, db, logger)

	// Arrancar el servidor y manejar el apagado seguro (Graceful Shutdown)
	runServer(srv, logger)
}

// setupDatabase intenta conectar a la BD y detiene el programa si falla.
func setupDatabase(cfg *config.Config, logger *slog.Logger) *sqlx.DB {
	sqlxDB, err := infraDB.ConnectDB(&cfg.Database, logger)
	if err != nil {
		logger.Error("no se pudo conectar a la base de datos", "error", err)
		os.Exit(1) // Si no hay BD, no tiene sentido arrancar el servidor
	}

	logger.Info("conexión a la base de datos establecida exitosamente")
	return sqlxDB
}

// setupLogger configura y retorna el logger principal de la aplicación.
func setupLogger() *slog.Logger {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
	return logger
}

// buildServer se encarga de toda la Inyección de Dependencias.
// Recibe la configuración, la BD y el logger, y devuelve un servidor HTTP listo.
// Nota: Ajusta los tipos `*config.Config` y `*database.DB` según como los tengas definidos en tu código.
func buildServer(cfg *config.Config, sqlxDB *sqlx.DB, logger *slog.Logger) *http.Server {
	// Adaptador JWT (único objeto, implementa dos puertos)
	// driven.TokenGeneratorPort → lo usa el servicio para GENERAR tokens
	// driving.TokenServicePort  → lo usa el middleware para VALIDAR tokens
	jwtSvc := jwtAdapter.NewJWTService(cfg.JWT.SecretKey, cfg.JWT.ServiceSecret)

	// Adaptadores secundarios: repositorios Postgres 
	// Recibe *sqlx.DB — no sabe cómo se abrió la conexión
	authRepo := repository.NewAuthRepository(sqlxDB)
	userRepo := repository.NewUserRepository(sqlxDB)

	// Núcleo: servicios de dominio
	// jwtSvc satisface driven.TokenGeneratorPort automáticamente.
	authSvc := services.NewAuthService(authRepo, jwtSvc)
	userSvc := services.NewUserService(userRepo)


	// Adaptadores primarios: handlers HTTP 
	authHandler := handlers.NewAuthHandler(authSvc, jwtSvc)
	userHandler := handlers.NewUserHandler(userSvc, jwtSvc)
 

	// Router principal
	router := httpServer.NewRouter(authHandler, userHandler, jwtSvc, logger)
 

	// Configuración del Servidor HTTP
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	return &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// runServer encapsula la lógica de concurrencia y el Graceful Shutdown.
func runServer(srv *http.Server, logger *slog.Logger) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Ejecutamos el servidor en una goroutine para que no bloquee la ejecución
	go func() {
		logger.Info("DB Server iniciado",
			"address", srv.Addr,
			"note", "solo accesible desde localhost",
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("error en el servidor", "error", err)
			os.Exit(1)
		}
	}()

	// El hilo principal se bloquea aquí esperando una señal del sistema (Ctrl+C)
	<-quit
	logger.Info("apagando servidor...")

	// Damos 10 segundos para que las peticiones en curso terminen
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("error en shutdown", "error", err)
	}
	logger.Info("servidor apagado correctamente")
}
