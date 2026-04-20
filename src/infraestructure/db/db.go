package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"Auth/infraestructure/config"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

// ConnectDB abre la conexión a Postgres y configura el pool.
// Retorna error en lugar de log.Fatalf para que main.go decida qué hacer.
func ConnectDB(cfg *config.DBConfig, logger *slog.Logger) (*sqlx.DB, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
	)
 
	db, err := sqlx.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("error abriendo conexión: %w", err)
	}

	//CONFIGURACIÓN DEL POOL
	db.SetMaxOpenConns(25)                 // máximo de conexiones abiertas
	db.SetMaxIdleConns(10)                 // conexiones en espera (reutilizables)
	db.SetConnMaxIdleTime(5 * time.Minute) // tiempo máximo idle
	db.SetConnMaxLifetime(1 * time.Hour)   // vida máxima de una conexión

	// Verificación con contexto
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("no se pudo conectar a postgres en %s:%s: %w",
			cfg.DBHost, cfg.DBPort, err) // ← retorna, no mata
	}

	logger.Info("conexión a postgres establecida con pool",
		"host", cfg.DBHost,
		"port", cfg.DBPort,
		"database", cfg.DBName,
	)
	return db, nil
}
