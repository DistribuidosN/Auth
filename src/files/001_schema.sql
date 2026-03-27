-- ============================================================
-- Schema: image_processing_db
-- DB Server — Sistema Distribuido de Procesamiento de Imágenes
-- ============================================================

-- Extensión para UUIDs (si se necesita en el futuro)
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ── Tabla de estados de lotes (catálogo) ─────────────────────
CREATE TABLE IF NOT EXISTS batch_status (
    id     SERIAL PRIMARY KEY,
    name   VARCHAR(50) UNIQUE NOT NULL  -- PENDING, PROCESSING, COMPLETED, FAILED
);

INSERT INTO batch_status (name) VALUES
    ('PENDING'), ('PROCESSING'), ('COMPLETED'), ('FAILED')
ON CONFLICT DO NOTHING;

-- ── Tabla de estados de imágenes (catálogo) ───────────────────
CREATE TABLE IF NOT EXISTS image_status (
    id   SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL
);

INSERT INTO image_status (name) VALUES
    ('PENDING'), ('PROCESSING'), ('COMPLETED'), ('FAILED')
ON CONFLICT DO NOTHING;

-- ── Tabla de niveles de log (catálogo) ───────────────────────
CREATE TABLE IF NOT EXISTS log_levels (
    id   SERIAL PRIMARY KEY,
    name VARCHAR(20) UNIQUE NOT NULL
);

INSERT INTO log_levels (name) VALUES
    ('INFO'), ('WARNING'), ('ERROR'), ('DEBUG')
ON CONFLICT DO NOTHING;

-- ── Tabla de tipos de transformación (catálogo extensible) ───
-- Al ser tabla y no ENUM, agregar un nuevo tipo es solo un INSERT
CREATE TABLE IF NOT EXISTS transformation_types (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(100) UNIQUE NOT NULL,
    description VARCHAR(255)
);

INSERT INTO transformation_types (name, description) VALUES
    ('RESIZE',     'Redimensionar imagen'),
    ('ROTATE',     'Rotar imagen en grados'),
    ('GRAYSCALE',  'Convertir a escala de grises'),
    ('BLUR',       'Aplicar desenfoque'),
    ('SHARPEN',    'Aumentar nitidez')
ON CONFLICT DO NOTHING;

-- ── Nodos de procesamiento ────────────────────────────────────
CREATE TABLE IF NOT EXISTS nodes (
    id           SERIAL PRIMARY KEY,
    node_id      VARCHAR(100) UNIQUE NOT NULL,  -- identificador único del nodo (ej: "node-1")
    host         VARCHAR(255)        NOT NULL,
    port         INT                 NOT NULL,
    status       VARCHAR(20)         NOT NULL DEFAULT 'ACTIVE',
    last_ping    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_nodes_status ON nodes(status);

-- ── Estado de salud de nodos (historial de monitoreo) ────────
CREATE TABLE IF NOT EXISTS node_status (
    id          SERIAL PRIMARY KEY,
    node_id     INT         NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    status      VARCHAR(20) NOT NULL,
    description VARCHAR(255),
    checked_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_node_status_node_id ON node_status(node_id);

-- ── Lotes de procesamiento ────────────────────────────────────
CREATE TABLE IF NOT EXISTS batches (
    id             SERIAL PRIMARY KEY,
    user_uuid      CHAR(36)    NOT NULL,          -- UUID del usuario (viene del Auth Server)
    request_time   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    overall_status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    CONSTRAINT fk_batch_status FOREIGN KEY (overall_status)
        REFERENCES batch_status(name) ON UPDATE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_batches_user_uuid     ON batches(user_uuid);
CREATE INDEX IF NOT EXISTS idx_batches_overall_status ON batches(overall_status);

-- ── Imágenes individuales dentro de un lote ───────────────────
CREATE TABLE IF NOT EXISTS images (
    id              SERIAL PRIMARY KEY,
    batch_id        INT          NOT NULL REFERENCES batches(id) ON DELETE CASCADE,
    original_name   VARCHAR(255) NOT NULL,
    result_path     VARCHAR(500),                 -- ruta donde el nodo guardó el resultado
    status          VARCHAR(20)  NOT NULL DEFAULT 'PENDING',
    node_id         INT          REFERENCES nodes(id) ON DELETE SET NULL,
    reception_time  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    conversion_time TIMESTAMP WITH TIME ZONE,     -- cuándo terminó el procesamiento
    CONSTRAINT fk_image_status FOREIGN KEY (status)
        REFERENCES image_status(name) ON UPDATE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_images_batch_id ON images(batch_id);
CREATE INDEX IF NOT EXISTS idx_images_status   ON images(status);
CREATE INDEX IF NOT EXISTS idx_images_node_id  ON images(node_id);

-- ── Transformaciones requeridas por imagen ────────────────────
CREATE TABLE IF NOT EXISTS image_transformations (
    id                  SERIAL PRIMARY KEY,
    image_id            INT         NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    transformation_type VARCHAR(100) NOT NULL REFERENCES transformation_types(name) ON UPDATE CASCADE,
    params              JSONB,                    -- parámetros específicos: {"degrees": 90}, {"width": 800}
    applied_at          TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_img_transform_image_id ON image_transformations(image_id);

-- ── Logs de procesamiento generados por los nodos ─────────────
CREATE TABLE IF NOT EXISTS processing_logs (
    id        SERIAL PRIMARY KEY,
    node_id   INT         NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    image_id  INT         NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    log_level VARCHAR(20) NOT NULL DEFAULT 'INFO',
    message   TEXT        NOT NULL,
    log_time  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_log_level FOREIGN KEY (log_level)
        REFERENCES log_levels(name) ON UPDATE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_proc_logs_image_id ON processing_logs(image_id);
CREATE INDEX IF NOT EXISTS idx_proc_logs_node_id  ON processing_logs(node_id);
CREATE INDEX IF NOT EXISTS idx_proc_logs_log_time ON processing_logs(log_time DESC);

-- ── Vista útil: estado completo de un lote ────────────────────
CREATE OR REPLACE VIEW v_batch_summary AS
SELECT
    b.id            AS batch_id,
    b.user_uuid,
    b.request_time,
    b.overall_status,
    COUNT(i.id)                                          AS total_images,
    COUNT(i.id) FILTER (WHERE i.status = 'COMPLETED')   AS completed,
    COUNT(i.id) FILTER (WHERE i.status = 'FAILED')      AS failed,
    COUNT(i.id) FILTER (WHERE i.status = 'PROCESSING')  AS processing,
    COUNT(i.id) FILTER (WHERE i.status = 'PENDING')     AS pending
FROM batches b
LEFT JOIN images i ON i.batch_id = b.id
GROUP BY b.id;
