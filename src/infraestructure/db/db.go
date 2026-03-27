package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"Auth/infraestructure/config"

	"github.com/jmoiron/sqlx"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func ConnectDB(cfg *config.Config) *sqlx.DB {
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
		log.Fatalf("Error abriendo conexión: %v", err)
	}

	//CONFIGURACIÓN DEL POOL
	db.SetMaxOpenConns(25)              // máximo de conexiones abiertas
	db.SetMaxIdleConns(10)              // conexiones en espera (reutilizables)
	db.SetConnMaxIdleTime(5 * time.Minute) // tiempo máximo idle
	db.SetConnMaxLifetime(1 * time.Hour)   // vida máxima de una conexión

	// Verificación con contexto
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Error conectando a PostgreSQL: %v", err)
	}

	log.Println("✅ Conexión a PostgreSQL con pool exitosa")
	return db
}