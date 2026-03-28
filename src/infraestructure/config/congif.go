package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DBConfig
	JWT      JWTConfig
}
type ServerConfig struct {
	Host string
	Port string
}
type DBConfig struct {
	DBUser     string
	DBPassword string
	DBHost     string
	DBPort     string
	DBName     string
}

type JWTConfig struct {
	SecretKey     string // clave compartida con todos los servicios que validen tokens
	ServiceSecret string // clave para tokens inter-servicios
}

// Load carga configuración desde .env o variables del sistema.
// Hace panic si faltan variables críticas — mejor fallar al inicio.
func LoadConfig() *Config {
	_ = godotenv.Load("../.env")

	return &Config{
		Server: ServerConfig{
			Host: mustEnv("SERVER_HOST"),
			Port: mustEnv("SERVER_PORT"),
		},
		Database: DBConfig{
			DBUser:     mustEnv("DB_USER"),
			DBPassword: mustEnv("DB_PASSWORD"),
			DBHost:     mustEnv("DB_HOST"),
			DBPort:     mustEnv("DB_PORT"),
			DBName:     mustEnv("DB_NAME"),
		},
		JWT: JWTConfig{
			SecretKey:     mustEnv("JWT_SECRET"),
			ServiceSecret: mustEnv("JWT_SERVICE_SECRET"),
		},
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("variable de entorno requerida no definida: %s", key))
	}
	return v
}
