# DB Server — Arquitectura Hexagonal en Go

Servidor de base de datos para el sistema distribuido de procesamiento de imágenes.
Expone una API REST **solo en localhost**, valida tokens JWT y tiene métodos predefinidos
para que ningún otro servicio toque la DB directamente.

---

## Arquitectura Hexagonal (Ports & Adapters)

```
┌─────────────────────────────────────────────────────────────────┐
│                        MUNDO EXTERIOR                           │
│   Orquestador (Java)          Nodos Python                      │
│        │ JWT token                  │ JWT token                 │
└────────┼───────────────────────────┼────────────────────────────┘
         │                           │
         ▼                           ▼
┌─────────────────────────────────────────────────────────────────┐
│              ADAPTADORES PRIMARIOS (Driving)                    │
│                                                                 │
│  internal/adapters/http/                                        │
│  ├── middleware/middleware.go   ← JWT Auth, Logger, CORS        │
│  ├── handlers/batch_handler.go ← HTTP → llama BatchService      │
│  ├── handlers/node_handler.go  ← HTTP → llama NodeService       │
│  └── server.go                 ← Router, une rutas y handlers   │
└──────────────────────┬──────────────────────────────────────────┘
                       │ llama a (interfaces/puertos)
                       ▼
┌─────────────────────────────────────────────────────────────────┐
│                    NÚCLEO / DOMINIO                             │
│                                                                 │
│  internal/core/                                                 │
│  ├── domain/                                                    │
│  │   └── batch.go    ← Entidades: Batch, Image, Node, Log      │
│  ├── ports/                                                     │
│  │   ├── repository_ports.go  ← Interfaces de repositorio      │
│  │   └── service_ports.go     ← Interfaces de servicio + JWT   │
│  └── services/                                                  │
│      ├── batch_service.go     ← Lógica de negocio pura         │
│      └── other_services.go   ← Image, Node, Log services       │
└──────────────────────┬──────────────────────────────────────────┘
                       │ implementado por (interfaces/puertos)
                       ▼
┌─────────────────────────────────────────────────────────────────┐
│              ADAPTADORES SECUNDARIOS (Driven)                   │
│                                                                 │
│  internal/adapters/                                             │
│  ├── repository/                                                │
│  │   ├── batch_repository.go  ← Postgres SQL (sqlx)            │
│  │   └── other_repositories.go ← Image, Node, Log repos        │
│  └── auth/                                                      │
│      └── jwt_service.go       ← Validación JWT (golang-jwt)    │
│                                                                 │
│  internal/infrastructure/                                       │
│  ├── config/config.go         ← Variables de entorno           │
│  └── database/postgres.go     ← Pool de conexiones Postgres    │
└─────────────────────────────────────────────────────────────────┘
```

---

## Estructura de carpetas

```
db-server/
├── cmd/
│   └── server/
│       └── main.go               ← Punto de entrada + inyección de dependencias
├── internal/
│   ├── core/                     ← Núcleo: no depende de nada externo
│   │   ├── domain/
│   │   │   └── batch.go          ← Entidades del dominio
│   │   ├── ports/
│   │   │   ├── repository_ports.go  ← Contratos de persistencia
│   │   │   └── service_ports.go     ← Contratos de servicios + JWT
│   │   └── services/
│   │       ├── batch_service.go     ← Lógica de negocio (batches)
│   │       └── other_services.go   ← Lógica (images, nodes, logs)
│   ├── adapters/                 ← Implementaciones concretas
│   │   ├── auth/
│   │   │   └── jwt_service.go    ← Implementa ports.TokenService
│   │   ├── repository/
│   │   │   ├── batch_repository.go  ← Implementa ports.BatchRepository
│   │   │   └── other_repositories.go
│   │   └── http/
│   │       ├── middleware/
│   │       │   └── middleware.go ← Auth JWT, Logger, CORS
│   │       ├── handlers/
│   │       │   ├── batch_handler.go
│   │       │   └── node_handler.go
│   │       └── server.go         ← Router principal
│   └── infrastructure/           ← Detalles técnicos de infraestructura
│       ├── config/config.go      ← Carga de env vars
│       └── database/postgres.go  ← Conexión + pool de conexiones
├── migrations/
│   └── 001_schema.sql            ← Schema completo de Postgres
├── .env.example                  ← Plantilla de variables de entorno
├── .gitignore
├── go.mod
├── Makefile
└── README.md
```

---

## Flujo de una request (ejemplo: POST /batches)

```
1. Orquestador Java envía:
   POST http://localhost:8081/batches
   Authorization: Bearer eyJhbGci...

2. middleware.Logger  → registra la request entrante

3. middleware.Auth    → extrae el Bearer token
                     → llama a jwtService.ValidateToken()
                     → inyecta TokenClaims en el contexto

4. BatchHandler.Create → lee el body JSON
                       → extrae claims del contexto (user_uuid)
                       → llama a batchService.CreateBatch()

5. batchService      → valida reglas de negocio
                     → llama a batchRepo.CreateBatch()
                     → llama a imageRepo.CreateImages() (bulk insert)
                     → retorna Batch + []Image

6. BatchHandler      → serializa a JSON
                     → responde 201 Created
```

---

## Librerías utilizadas

| Librería | Propósito |
|---|---|
| `github.com/golang-jwt/jwt/v5` | Validar tokens JWT firmados por el Auth Server |
| `github.com/jmoiron/sqlx` | Queries SQL con mapeo a structs (sobre database/sql) |
| `github.com/jackc/pgx/v5` | Driver Postgres de alta performance |
| `golang.org/x/crypto` | BCrypt para contraseñas (si el Auth Server también es Go) |
| `log/slog` | Logging estructurado JSON (stdlib Go 1.21+) |
| `net/http` | Servidor HTTP + router (stdlib Go 1.22 con path params) |

> **Sin frameworks HTTP externos.** Go 1.22 ya soporta `{pathParam}` y métodos
> (`GET /ruta`, `POST /ruta`) en el ServeMux nativo. Cero dependencias innecesarias.

---

## Cómo correr el servidor

### 1. Prerrequisitos
- Go 1.22+
- Postgres corriendo en localhost

### 2. Configurar variables de entorno
```bash
cp .env.example .env
# Editar .env con tus valores reales
```

### 3. Crear la base de datos y aplicar el schema
```bash
make createdb
make migrate
```

### 4. Descargar dependencias
```bash
make tidy
```

### 5. Correr el servidor
```bash
make run
# El servidor arranca en http://127.0.0.1:8081
```

---

## Endpoints disponibles

### Batches
| Método | Ruta | Descripción |
|---|---|---|
| `POST` | `/batches` | Crear lote con lista de imágenes |
| `GET` | `/batches` | Listar lotes del usuario autenticado |
| `GET` | `/batches/{id}` | Obtener lote por ID |
| `PUT` | `/batches/{id}/status` | Actualizar estado del lote |
| `GET` | `/batches/{id}/images` | Listar imágenes de un lote |

### Images
| Método | Ruta | Descripción |
|---|---|---|
| `GET` | `/images/{id}` | Obtener imagen por ID |
| `PUT` | `/images/{id}/finalize` | Marcar imagen como procesada |
| `PUT` | `/images/{id}/assign` | Asignar imagen a un nodo |

### Nodes
| Método | Ruta | Descripción |
|---|---|---|
| `POST` | `/nodes` | Registrar o actualizar nodo |
| `GET` | `/nodes` | Listar nodos (`?active=true` solo activos) |
| `PUT` | `/nodes/{nodeId}/status` | Actualizar estado del nodo |
| `GET` | `/nodes/{id}/logs` | Obtener logs de un nodo |

### Logs
| Método | Ruta | Descripción |
|---|---|---|
| `POST` | `/logs` | Registrar log de procesamiento |
| `GET` | `/logs/image/{id}` | Logs de una imagen específica |

### Health
| Método | Ruta | Descripción |
|---|---|---|
| `GET` | `/health` | Estado del servidor (sin auth) |

---

## Ejemplo de request

```bash
# Crear un lote (el token viene del Auth Server)
curl -X POST http://localhost:8081/batches \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json" \
  -d '{"image_names": ["foto1.jpg", "foto2.png", "foto3.jpg"]}'

# Respuesta:
{
  "batch": {
    "id": 1,
    "user_uuid": "a1b2c3d4-...",
    "request_time": "2026-01-15T10:30:00Z",
    "overall_status": "PENDING"
  },
  "images": [
    {"id": 1, "batch_id": 1, "original_name": "foto1.jpg", "status": "PENDING", ...},
    {"id": 2, "batch_id": 1, "original_name": "foto2.png", "status": "PENDING", ...}
  ]
}
```

---

## Por qué arquitectura hexagonal aquí

El dominio (`core/`) **no importa nada de Go externo**. Solo conoce interfaces.
Si mañana cambias de Postgres a MongoDB, solo reescribes `adapters/repository/` —
el dominio y los servicios no cambian ni una línea.

Lo mismo aplica para JWT: si cambias el algoritmo de firma, solo modificas
`adapters/auth/jwt_service.go`.
