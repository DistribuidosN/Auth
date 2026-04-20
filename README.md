# Auth Server — Servicio de Autenticación

Microservicio de autenticación construido en **Go 1.22** siguiendo **Arquitectura Hexagonal (Ports & Adapters)**.
Expone una API REST en `localhost`, gestiona usuarios con contraseñas hasheadas en bcrypt y protege rutas mediante JWT.

---

## Tabla de contenido

1. [¿Qué hace este servicio?](#qué-hace-este-servicio)
2. [Arquitectura Hexagonal](#arquitectura-hexagonal)
3. [Separación: Driving vs Driven](#separación-driving-vs-driven)
4. [Estructura de carpetas](#estructura-de-carpetas)
5. [Flujo de una request](#flujo-de-una-request)
6. [Endpoints](#endpoints)
7. [Cómo correr el servidor](#cómo-correr-el-servidor)
8. [Variables de entorno](#variables-de-entorno)
9. [Schema de base de datos](#schema-de-base-de-datos)
10. [Librerías utilizadas](#librerías-utilizadas)

---

## ¿Qué hace este servicio?

Este servidor es el **guardián de identidad** del sistema distribuido de procesamiento de imágenes.
Ningún otro servicio (orquestador Java, nodos Python, DB Server) procesa imágenes sin pasar primero
por aquí. El flujo es:

```
Cliente / Orquestador
        │
        ▼
  ┌─────────────┐     POST /auth/login      ┌──────────────┐
  │  Auth Server│ ◄───────────────────────► │  PostgreSQL  │
  │  (este svc) │     JWT firmado           │  auth_db     │
  └─────────────┘                           └──────────────┘
        │
        │  JWT (válido 24h)
        ▼
  Orquestador Java / DB Server
  (validan el JWT con la misma clave secreta)
```

---

## Arquitectura Hexagonal

La arquitectura hexagonal (también llamada *Ports & Adapters*) tiene una regla fundamental:

> **El núcleo de la aplicación no sabe nada del mundo exterior.**
> No sabe que existe Postgres. No sabe que existe JWT. No sabe que existe HTTP.
> Solo conoce interfaces (puertos).

```
╔══════════════════════════════════════════════════════════════════╗
║                      MUNDO EXTERIOR                              ║
║                                                                  ║
║   HTTP Request         PostgreSQL          JWT Library           ║
║       │                    │                    │                ║
╠═══════╪════════════════════╪════════════════════╪════════════════╣
║       │   ADAPTADORES      │                    │                ║
║       ▼   (driving)        ▼   (driven)         ▼   (driven)     ║
║  ┌──────────┐        ┌──────────┐        ┌──────────────┐        ║
║  │  HTTP    │        │Postgres  │        │ JWT Adapter  │        ║
║  │ handlers │        │  repo    │        │ jwt_service  │        ║
║  │middleware│        │auth_repo │        │   .go        │        ║
║  └────┬─────┘        └────┬─────┘        └──────┬───────┘        ║
║       │                   │                     │                ║
╠═══════╪═══════════════════╪═════════════════════╪════════════════╣
║       │              PUERTOS                    │                ║
║       │   driving/              driven/         │                ║
║       ▼                                         ▼                ║
║  ┌────────────────┐       ╔══════════════════════════════╗       ║
║  │AuthServicePort │       ║        NÚCLEO               ║       ║
║  │TokenServicePort│◄─────►║   core/domain/entities      ║       ║
║  └────────────────┘       ║   core/services/auth_service ║       ║
║                           ║                              ║       ║
║                           ║   (solo conoce interfaces)   ║       ║
║  ┌─────────────────┐      ╚══════════════════════════════╝       ║
║  │AuthRepositoryPort│◄────────────────────────────────────────   ║
║  │TokenGeneratorPort│                                            ║
║  └─────────────────┘                                            ║
╚══════════════════════════════════════════════════════════════════╝
```

### La regla de oro de los imports

```
✓  adaptadores  →  puertos  →  (nunca hacia afuera)
✓  servicios    →  puertos  →  (nunca hacia afuera)
✗  core/services NO puede importar adapters/
✗  core/services NO puede importar infrastructure/
✗  core/domain   NO puede importar NADA externo
```

Si en algún momento un archivo dentro de `core/` tiene un import de
`github.com/jmoiron/sqlx` o `github.com/golang-jwt/jwt`, la arquitectura está rota.

---

## Separación: Driving vs Driven

Este es el punto más importante y el que más se confunde.

```
                    ┌─────────────────────────────────────────┐
                    │             NÚCLEO                       │
                    │                                          │
  MUNDO             │  ports/driving/          ports/driven/   │  MUNDO
  EXTERIOR  ──────► │  (el core EXPONE)  vs   (el core EXIGE) │ ◄──────  EXTERIOR
  llama al           │                                          │           implementa
  núcleo             └─────────────────────────────────────────┘           esto
```

### `ports/driving/` — Puertos primarios

El mundo exterior LLAMA al núcleo a través de estos puertos.
Los adaptadores primarios (HTTP handlers, middleware) los usan.

```
ports/driving/
├── auth_service_port.go    ← AuthServicePort
│     Register(...)
│     Login(...)
│     GetProfile(...)
│     UpdateProfile(...)
│     DeleteAccount(...)
│
└── token_service_port.go   ← TokenServicePort
      ValidateToken(...)     ← lo usa el middleware para validar requests
      GenerateToken(...)
      GenerateServiceToken(...)
```

**¿Quién los implementa?**
- `AuthServicePort` → `core/services/auth_service.go`
- `TokenServicePort` → `adapters/auth/jwt_service.go`

### `ports/driven/` — Puertos secundarios

El núcleo EXIGE al mundo exterior que implemente estos puertos.
El núcleo los llama para persistir datos o generar tokens.

```
ports/driven/
├── auth_repository_port.go  ← AuthRepositoryPort
│     CreateUser(...)         ← el core llama esto, Postgres lo hace
│     GetUserByUsername(...)
│     GetUserByUUID(...)
│     GetRoleByID(...)
│
└── token_generator_port.go  ← TokenGeneratorPort
      GenerateToken(...)       ← el core llama esto, JWT lo hace
      GenerateServiceToken(...)
```

**¿Quién los implementa?**
- `AuthRepositoryPort` → `adapters/repository/auth_repository.go`
- `TokenGeneratorPort` → `adapters/auth/jwt_service.go`

### El caso especial del JWT Adapter

`jwt_service.go` implementa **dos puertos distintos** con un solo struct:

```
adapters/auth/jwt_service.go
         │
         ├── implementa driven.TokenGeneratorPort
         │   → el servicio de auth lo usa para GENERAR tokens al hacer login
         │
         └── implementa driving.TokenServicePort
             → el middleware lo usa para VALIDAR tokens en cada request
```

Go resuelve esto automáticamente por duck typing. El mismo objeto,
dos roles distintos según quién lo recibe.

```go
jwtSvc := auth.NewJWTService(secretKey, serviceSecret)
//                               │
//     ┌───────────────────────┴────────────────────────┐
//     ▼                                                ▼
// services.NewAuthService(authRepo, jwtSvc)    router.NewRouter(..., jwtSvc, ...)
//   jwtSvc como driven.TokenGeneratorPort        jwtSvc como driving.TokenServicePort
```

---

## Estructura de carpetas

```
Auth/
│
├── main.go                          ← Punto de entrada + Inyección de Dependencias
├── go.mod
├── go.sum
├── .env.example
├── Makefile
├── migrations/
│   └── 001_schema.sql               ← Schema de PostgreSQL
│
├── core/                            ← NÚCLEO (no depende de nada externo)
│   ├── domain/
│   │   └── entities/
│   │       ├── user.go              ← struct User
│   │       └── role.go              ← struct Role, Permission
│   │
│   ├── ports/
│   │   ├── driving/                 ← Puertos primarios (el mundo llama al core)
│   │   │   ├── auth_service_port.go ← interface AuthServicePort
│   │   │   └── token_service_port.go← interface TokenServicePort
│   │   │
│   │   └── driven/                  ← Puertos secundarios (el core exige al mundo)
│   │       ├── auth_repository_port.go  ← interface AuthRepositoryPort
│   │       └── token_generator_port.go  ← interface TokenGeneratorPort
│   │
│   └── services/
│       └── auth_service.go          ← Lógica de negocio pura
│                                      Solo importa ports/driving y ports/driven
│
├── adapters/                        ← IMPLEMENTACIONES CONCRETAS
│   ├── auth/
│   │   └── jwt_service.go           ← Implementa TokenGeneratorPort + TokenServicePort
│   │
│   ├── repository/
│   │   └── auth_repository.go       ← Implementa AuthRepositoryPort con Postgres
│   │
│   └── http/
│       ├── handlers/
│       │   └── auth_handler.go      ← HTTP → llama AuthServicePort
│       ├── middleware/
│       │   └── auth_middleware.go   ← Auth JWT, Logger, CORS, JSON
│       └── auth_server.go           ← Router: mapea rutas + aplica middlewares
│
└── infrastructure/                  ← FONTANERÍA TÉCNICA (sin puerto, sin negocio)
    ├── config/
    │   └── config.go                ← Carga .env → struct Config
    └── db/
        └── db.go                    ← Abre conexión Postgres + configura pool
```

### ¿Por qué `infrastructure/` es distinto de `adapters/`?

```
adapters/repository/auth_repository.go
  └─ TIENE un puerto: implementa driven.AuthRepositoryPort
  └─ El núcleo sabe que existe algo que hace esto (por la interfaz)

infrastructure/db/db.go
  └─ NO TIENE un puerto: nadie en core/ pide "abre una conexión"
  └─ Solo existe para que auth_repository.go tenga un *sqlx.DB con qué trabajar
  └─ Es fontanería: sin él nada funciona, pero el dominio no lo conoce
```

---

## Flujo de una request

### POST /auth/login (ruta pública)

```
Cliente
  │
  │  POST /auth/login
  │  Body: { "username": "christian", "password": "1234abcd" }
  │
  ▼
middleware.CORS
  │  Agrega headers de CORS
  ▼
middleware.Logger
  │  Registra método, path, inicio del timer
  ▼
middleware.JSONContentType
  │  Setea Content-Type: application/json
  ▼
mux (router Go 1.22)
  │  Encuentra "POST /auth/login"
  ▼
AuthHandler.Login()
  │  Parsea body JSON
  │  Llama authSvc.Login("christian", "1234abcd")
  ▼
authService.Login()                         [NÚCLEO]
  │  Valida que username y password no estén vacíos
  │  Llama authRepo.GetUserByUsername("christian")
  ▼
postgresAuthRepo.GetUserByUsername()        [ADAPTADOR DRIVEN]
  │  SELECT ... FROM users WHERE username = $1
  ▼
PostgreSQL
  │  Retorna el registro
  ▼
authService.Login() continúa
  │  bcrypt.CompareHashAndPassword(hash, password)
  │  Si coincide: llama tokenSvc.GenerateToken(user)
  ▼
jwtTokenService.GenerateToken()             [ADAPTADOR DRIVEN]
  │  Crea jwtClaims con UserUUID, Username, RoleID
  │  Firma con HS256 y secretKey
  │  Retorna "eyJhbGci..."
  ▼
AuthHandler.Login() recibe (token, user, nil)
  │
  ▼
Respuesta HTTP 200
{
  "token":     "eyJhbGci...",
  "user_uuid": "a1b2-c3d4-...",
  "username":  "christian",
  "role_id":   1
}
```

### GET /user/profile (identidad local, validación perimetral por Java)

```text
Cliente → Orquestador Java (Valida perimetralmente)
  │
  ├─ Java reenvía a Go: GET /user/profile
  │  Authorization: Bearer eyJhbGci...
  ▼
middleware.CORS → Logger → JSONContentType
  ▼
mux → encuentra "GET /user/profile"
  ▼
UserHandler.Profile()                       [ADAPTADOR PRIMARIO]
  │  Llama a extractClaims() leyendo el Header directamente sin middleware bloqueante
  │  Detokeniza a través del adapter JWT
  ▼
jwtTokenService.ValidateToken()             [ADAPTADOR DRIVEN]
  │  jwt.ParseWithClaims(...)
  │  Verifica firma y expiración
  │  Retorna &TokenClaims{UserUUID: "a1b2-...", ...}
  ▼
UserHandler.Profile()
  │  Llama userSvc.GetProfile(claims)        (nunca expone raw User ID a la web)
  ▼
userService.GetProfile()                    [NÚCLEO USER]
  │  Llama userRepo.GetUserByUUID(claims.UserUUID)
  ▼
postgresUserRepo.GetUserByUUID()            [ADAPTADOR DRIVEN]
  │  SELECT ... FROM users WHERE user_uuid = $1
  ▼
Respuesta HTTP 200 — User struct (sin password_hash por json:"-")
```

---

## Endpoints

| Método   | Ruta               | Auth    | Descripción                              |
|----------|--------------------|---------|------------------------------------------|
| `GET`    | `/health`          | No      | Estado del servidor                      |
| `POST`   | `/auth/register`   | No      | Crear cuenta nueva                       |
| `POST`   | `/auth/login`      | No      | Autenticarse y recibir JWT               |
| `POST`   | `/auth/logout`     | No      | Cerrar sesión (Frontend drops token)     |
| `POST`   | `/auth/forget-password`| No  | Generar enlace virtual mágico            |
| `POST`   | `/auth/reset-password` | Token | Cambiar clave usando token dinámico    |
| `POST`   | `/auth/validate`   | Local   | Verificar JWT (para orquestador Java)    |
| `GET`    | `/user/profile`    | Passthru| Ver perfil (Java envía Token en Header)  |
| `PUT`    | `/user/profile`    | Passthru| Actualizar username                      |
| `DELETE` | `/user/account`    | Passthru| Desactivar cuenta (soft delete)          |
| `GET`    | `/user/search`     | Passthru| Buscar usuario por similitud de nombre   |
| `GET`    | `/roles`           | Passthru| Obtener rol por ID (`?role_id=X`)        |
| `GET`    | `/permissions`     | Passthru| Obtener permisos del rol (`?role_id=X`)  |

> **Sobre la Autorización (Auth):** La protección estricta 401 (Middleware Auth global) fue removida de este componente porque el API Gateway de Java asume la responsabilidad perimetral. Los endpoints de usuario (`/user/*`) reciben el token originario transparente (`Passthru`) desde Java y extraen la identidad para aplicar la acción localmente destokenizando.

### Ejemplos de request

```bash
# Validar token a voluntad (Java Orchestrator -> Go)
curl -X POST http://localhost:8080/auth/validate \
  -H "Content-Type: application/json" \
  -d '{"token": "eyJhbGci..."}'

# User profile update
curl -X PUT http://localhost:8080/user/profile \
  -H "Authorization: Bearer eyJhbGci..." \
  -H "Content-Type: application/json" \
  -d '{"username": "christian_new"}'
```

---

## Cómo correr el servidor

### 1. Prerrequisitos

- Go 1.22+
- PostgreSQL corriendo en localhost

### 2. Configurar variables de entorno

```bash
cp .env.example .env
# Editar .env con tus credenciales reales
```

### 3. Crear la base de datos y aplicar el schema

```bash
make createdb    # crea la base de datos
make migrate     # aplica migrations/001_schema.sql
```

### 4. Descargar dependencias

```bash
make tidy        # go mod tidy
```

### 5. Correr el servidor

```bash
make run         # arranca en http://127.0.0.1:8080
```

### 6. Compilar binario

```bash
make build       # genera bin/auth-server
```

---

## Variables de entorno

| Variable            | Default       | Requerida | Descripción                                    |
|---------------------|---------------|-----------|------------------------------------------------|
| `SERVER_HOST`       | `127.0.0.1`   | No        | Host del servidor. `127.0.0.1` = solo localhost |
| `SERVER_PORT`       | `8080`        | No        | Puerto donde escucha                            |
| `DB_USER`           | `postgres`    | No        | Usuario de PostgreSQL                           |
| `DB_PASSWORD`       | —             | **Sí**    | Contraseña de PostgreSQL                        |
| `DB_HOST`           | `localhost`   | No        | Host de PostgreSQL                              |
| `DB_PORT`           | `5432`        | No        | Puerto de PostgreSQL                            |
| `DB_NAME`           | `auth_db`     | No        | Nombre de la base de datos                      |
| `JWT_SECRET`        | —             | **Sí**    | Clave para firmar tokens. Compartida con todos los servicios que validen JWT |
| `JWT_SERVICE_SECRET`| = JWT_SECRET  | No        | Clave para tokens inter-servicios               |

> **Importante:** `JWT_SECRET` debe ser **idéntica** en todos los microservicios
> (Auth Server, DB Server, Orquestador) para que puedan validar tokens entre sí.

---

## Schema de base de datos

```sql
-- Roles del sistema
CREATE TABLE roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    status SMALLINT NOT NULL DEFAULT 1 
);
COMMENT ON TABLE roles IS 'Almacena los perfiles(roles) de acceso disponibles';

-- Permisos por ruta
CREATE TABLE permissions (
    id SERIAL PRIMARY KEY,
    description VARCHAR(100) NOT NULL UNIQUE,
    route VARCHAR(100) NOT NULL
);
COMMENT ON TABLE permissions IS 'Catálogo de rutas y acciones protegidas por el sistema';

-- Relación rol ↔ permiso (RBAC)
CREATE TABLE role_permissions (
    role_id INT REFERENCES roles(id),
    permission_id INT REFERENCES permissions(id),
    PRIMARY KEY (role_id, permission_id)
);
COMMENT ON TABLE role_permissions IS 'Asociación de muchos a muchos para definir permisos por perfil';

-- Usuarios
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    user_uuid UUID NOT NULL UNIQUE,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(100) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role_id INT NOT NULL REFERENCES roles(id),
    status SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
COMMENT ON TABLE users IS 'Entidad principal asignando perfiles 1:N';
```

### Relaciones

```
roles ──< role_permissions >── permissions
  │
  └── users (role_id FK)
```

El campo `user_uuid` (no el `id` interno) es el que viaja en el JWT y se comparte
entre microservicios. Así el `id` interno de Postgres nunca se expone.

---

## Librerías utilizadas

| Librería | Versión | Propósito |
|---|---|---|
| `github.com/golang-jwt/jwt/v5` | v5.3.1 | Generación y validación de JWT con HMAC-SHA256 |
| `github.com/jackc/pgx/v5` | v5.9.1 | Driver PostgreSQL de alto rendimiento |
| `github.com/jmoiron/sqlx` | v1.4.0 | Mapeo de rows SQL a structs Go (`StructScan`, `Get`, `Select`) |
| `github.com/joho/godotenv` | v1.5.1 | Carga de archivo `.env` en desarrollo |
| `github.com/google/uuid` | v1.6.0 | Generación de UUIDs v4 para `user_uuid` |
| `golang.org/x/crypto` | v0.22.0 | BCrypt para hashear contraseñas |
| `log/slog` | stdlib Go 1.21 | Logging estructurado JSON (sin dependencias externas) |
| `net/http` | stdlib Go 1.22 | Servidor HTTP + router con path params y métodos |

> **Cero frameworks HTTP.** Go 1.22 soporta `GET /ruta/{id}` y discriminación por
> método en el `ServeMux` nativo. No se necesita Gin, Echo ni Chi.

---

## Pool de conexiones

El `db.go` configura el pool de conexiones de `sqlx` para manejar goroutines
concurrentes de manera eficiente:

```
db.SetMaxOpenConns(25)          → máximo de conexiones simultáneas a Postgres
db.SetMaxIdleConns(10)          → conexiones en espera listas para reutilizar
db.SetConnMaxIdleTime(5 min)    → tiempo máximo que una conexión puede estar idle
db.SetConnMaxLifetime(1 hora)   → reconexión automática para evitar conexiones obsoletas
```

Sin configurar el pool, una carga alta de goroutines puede saturar el
`max_connections` de PostgreSQL (default: 100).

---

## Seguridad

- Las contraseñas se hashean con **bcrypt costo 12** — nunca se almacenan en texto plano.
- El campo `password_hash` tiene `json:"-"` — nunca aparece en respuestas JSON.
- El JWT usa **HMAC-SHA256**. Se verifica que el algoritmo sea exactamente ese
  (protección contra el ataque `alg:none`).
- El servidor escucha en `127.0.0.1` por defecto — no expuesto a la red externa.
- Los errores de login no revelan si el usuario existe o no
  (siempre responden "credenciales inválidas").
- El `user_uuid` viaja en el JWT, nunca el `id` interno de la base de datos.