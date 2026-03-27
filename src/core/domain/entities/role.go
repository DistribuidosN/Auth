package entities

// Role representa el perfil de acceso
type Role struct {
	ID     int    `json:"id" db:"id"`
	Name   string `json:"name" db:"name"`
	Status int    `json:"status" db:"status"`
}

// Permission representa la ruta o acción permitida
type Permission struct {
	ID          int    `json:"id" db:"id"`
	Description string `json:"description" db:"description"`
	Route       string `json:"route" db:"route"`
}