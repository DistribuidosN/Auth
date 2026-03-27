// cmd/server/main.go — Punto de entrada del DB Server
// Aquí se ensamblan todas las capas de la arquitectura hexagonal:
//
//   Config → DB → Repositories → Services → Handlers → Router → HTTP Server
//
// Cada capa solo conoce interfaces, no implementaciones concretas.
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

	jwtAdapter    "github.com/christian/db-server/internal/adapters/auth"
	"github.com/christian/db-server/internal/adapters/http/handlers"
	httpServer     "github.com/christian/db-server/internal/adapters/http"
	"github.com/christian/db-server/internal/adapters/repository"
	"github.com/christian/db-server/internal/core/services"
	"github.com/christian/db-server/internal/infrastructure/config"
	"github.com/christian/db-server/internal/infrastructure/database"
)

func main() {
	// ── Logger estructurado ───────────────────────────────────
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// ── Configuración ─────────────────────────────────────────
	cfg := config.Load()

	// ── Base de datos ─────────────────────────────────────────
	db, err := database.Connect(cfg.Database, logger)
	if err != nil {
		logger.Error("no se pudo conectar a la base de datos", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// ── Adaptadores de repositorio (capa de persistencia) ─────
	batchRepo := repository.NewBatchRepository(db)
	imageRepo := repository.NewImageRepository(db)
	nodeRepo  := repository.NewNodeRepository(db)
	logRepo   := repository.NewLogRepository(db)

	// ── Servicios del dominio (lógica de negocio) ─────────────
	// Los servicios reciben interfaces (puertos), no concreciones
	batchSvc := services.NewBatchService(batchRepo, imageRepo)
	imageSvc := services.NewImageService(imageRepo)
	nodeSvc  := services.NewNodeService(nodeRepo)
	logSvc   := services.NewLogService(logRepo)

	// ── Adaptador de autenticación JWT ────────────────────────
	tokenSvc := jwtAdapter.NewJWTTokenService(cfg.JWT.SecretKey, cfg.JWT.ServiceSecret)

	// ── Handlers HTTP (adaptadores primarios) ─────────────────
	batchHandler := handlers.NewBatchHandler(batchSvc, imageSvc)
	imageHandler := handlers.NewImageHandler(imageSvc)
	nodeHandler  := handlers.NewNodeHandler(nodeSvc, logSvc)
	logHandler   := handlers.NewLogHandler(logSvc)

	// ── Router ────────────────────────────────────────────────
	router := httpServer.NewRouter(
		batchHandler, imageHandler, nodeHandler, logHandler,
		tokenSvc, logger,
	)

	// ── Servidor HTTP ─────────────────────────────────────────
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ── Graceful shutdown ─────────────────────────────────────
	// El servidor espera que terminen las requests en vuelo antes de cerrar
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("DB Server iniciado",
			"address", addr,
			"note", "solo accesible desde localhost",
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("error en el servidor", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	logger.Info("apagando servidor...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("error en shutdown", "error", err)
	}
	logger.Info("servidor apagado correctamente")
}
