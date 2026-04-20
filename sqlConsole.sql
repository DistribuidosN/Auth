CREATE DATABASE auth_db;
\c auth_db;

-- 1. Tabla de Roles
CREATE TABLE roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    status SMALLINT NOT NULL DEFAULT 1 
);
COMMENT ON TABLE roles IS 'Almacena los perfiles(roles) de acceso disponibles';
COMMENT ON COLUMN roles.id IS 'Identificador único incremental del rol';
COMMENT ON COLUMN roles.name IS 'Nombre descriptivo del perfil de usuario (ej. ADMIN, CLIENTE)';
COMMENT ON COLUMN roles.status IS 'Estado lógico del rol: 1 para Activo, 0 para Inactivo';

-- 2. Tabla de Permisos
CREATE TABLE permissions (
    id SERIAL PRIMARY KEY,
    description VARCHAR(100) NOT NULL UNIQUE,
    route VARCHAR(100) NOT NULL
);
COMMENT ON TABLE permissions IS 'Catálogo de rutas y acciones protegidas por el sistema';
COMMENT ON COLUMN permissions.id IS 'Identificador único del permiso o acción';
COMMENT ON COLUMN permissions.description IS 'Descripción de la funcionalidad protegida (ej. Procesar Lote)';
COMMENT ON COLUMN permissions.route IS 'Endpoint o ruta técnica de la API que se desea restringir';

-- 3. Tabla de Usuarios
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    user_uuid UUID NOT NULL UNIQUE,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(100) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role_id INT NOT NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (role_id) REFERENCES roles(id)
);
COMMENT ON TABLE users IS 'Entidad principal de usuarios con vinculación directa a un perfil de seguridad';
COMMENT ON COLUMN users.id IS 'ID interno del registro del usuario';
COMMENT ON COLUMN users.user_uuid IS 'Identificador universal compartido con la base de datos de procesamiento';
COMMENT ON COLUMN users.username IS 'Nombre único de acceso al sistema';
COMMENT ON COLUMN users.password_hash IS 'Contraseña cifrada mediante algoritmos de hashing';
COMMENT ON COLUMN users.role_id IS 'FK que define el rol único asignado al usuario (Relación 1:N)';
COMMENT ON COLUMN users.status IS 'Estado de la cuenta: 1 para Activo, 0 para Suspendido';
COMMENT ON COLUMN users.created_at IS 'Fecha y hora de registro del usuario';

-- 4. Tabla de Ruptura: Roles y Permisos
CREATE TABLE role_permissions (
    role_id INT,
    permission_id INT,
    PRIMARY KEY (role_id, permission_id),
    FOREIGN KEY (role_id) REFERENCES roles(id),
    FOREIGN KEY (permission_id) REFERENCES permissions(id)
);
COMMENT ON TABLE role_permissions IS 'Asociación de muchos a muchos para definir los permisos permitidos por cada perfil';