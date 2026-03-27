// main.go
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

	jwtAdapter "Auth/adapters/auth"
	httpServer "Auth/adapters/http"
	"Auth/adapters/http/handlers"
	"Auth/adapters/repository"
	"Auth/core/services"
	"Auth/infrastructure/config"
	"Auth/infrastructure/db"
)

func main() {
	// 1. Inicializar dependencias base
	logger := setupLogger()
	cfg := config.Load()

	// 2. Conectar a la base de datos
	db := setupDatabase(cfg, logger)
	// ¡Ojo aquí! El defer se queda en el main para que la conexión
	// viva mientras el servidor esté encendido.
	defer db.Close()

	// 3. Ensamblar la aplicación y obtener el servidor HTTP
	srv := buildServer(cfg, db, logger)

	// 4. Arrancar el servidor y manejar el apagado seguro (Graceful Shutdown)
	runServer(srv, logger)
}

// setupDatabase intenta conectar a la BD y detiene el programa si falla.
func setupDatabase(cfg *config.Config, logger *slog.Logger) *database.DB {
	db, err := database.Connect(cfg.Database, logger)
	if err != nil {
		logger.Error("no se pudo conectar a la base de datos", "error", err)
		os.Exit(1) // Si no hay BD, no tiene sentido arrancar el servidor
	}

	logger.Info("conexión a la base de datos establecida exitosamente")
	return db
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
func buildServer(cfg *config.Config, db *database.DB, logger *slog.Logger) *http.Server {
	// -- Capa de Persistencia (Repositorios) --
	batchRepo := repository.NewBatchRepository(db)
	imageRepo := repository.NewImageRepository(db)
	nodeRepo := repository.NewNodeRepository(db)
	logRepo := repository.NewLogRepository(db)

	// -- Capa de Dominio (Servicios) --
	batchSvc := services.NewBatchService(batchRepo, imageRepo)
	imageSvc := services.NewImageService(imageRepo)
	nodeSvc := services.NewNodeService(nodeRepo)
	logSvc := services.NewLogService(logRepo)

	// -- Adaptadores de Autenticación --
	tokenSvc := jwtAdapter.NewJWTTokenService(cfg.JWT.SecretKey, cfg.JWT.ServiceSecret)

	// -- Capa de Presentación (Handlers) --
	batchHandler := handlers.NewBatchHandler(batchSvc, imageSvc)
	imageHandler := handlers.NewImageHandler(imageSvc)
	nodeHandler := handlers.NewNodeHandler(nodeSvc, logSvc)
	logHandler := handlers.NewLogHandler(logSvc)

	// -- Enrutador --
	router := httpServer.NewRouter(
		batchHandler, imageHandler, nodeHandler, logHandler,
		tokenSvc, logger,
	)

	// -- Configuración del Servidor HTTP --
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
