// Package ports define los contratos (interfaces) del núcleo de la aplicación.
// En arquitectura hexagonal, estos son los "puertos" que el dominio expone
// y que los adaptadores deben implementar.
package ports

import (
	"context"

	"Auth/core/domain"
)

// ────────────────────────────────────────────────────────────
// Puerto de repositorio para Batches
// ────────────────────────────────────────────────────────────

type BatchRepository interface {
	// CreateBatch inserta un nuevo lote y devuelve el registro creado
	CreateBatch(ctx context.Context, batch *domain.Batch) (*domain.Batch, error)

	// GetBatchByID retorna un lote por su ID
	GetBatchByID(ctx context.Context, id int) (*domain.Batch, error)

	// GetBatchesByUser retorna todos los lotes de un usuario (paginado)
	GetBatchesByUser(ctx context.Context, userUUID string, limit, offset int) ([]domain.Batch, error)

	// UpdateBatchStatus actualiza el estado global de un lote
	UpdateBatchStatus(ctx context.Context, id int, status domain.BatchStatus) error

	// FinalizeBatch marca un lote como completado con su resultado
	FinalizeBatch(ctx context.Context, id int, status domain.BatchStatus) error
}

// ────────────────────────────────────────────────────────────
// Puerto de repositorio para Images
// ────────────────────────────────────────────────────────────

type ImageRepository interface {
	// CreateImages inserta múltiples imágenes en un batch (insert bulk)
	CreateImages(ctx context.Context, images []domain.Image) ([]domain.Image, error)

	// GetImageByID retorna una imagen por su ID
	GetImageByID(ctx context.Context, id int) (*domain.Image, error)

	// GetImagesByBatch retorna todas las imágenes de un lote
	GetImagesByBatch(ctx context.Context, batchID int) ([]domain.Image, error)

	// UpdateImageStatus actualiza el estado de una imagen
	UpdateImageStatus(ctx context.Context, id int, status string) error

	// FinalizeImage marca una imagen como procesada con su ruta de resultado
	FinalizeImage(ctx context.Context, id int, resultPath string, status string) error

	// AssignImageToNode asigna una imagen a un nodo de procesamiento
	AssignImageToNode(ctx context.Context, imageID, nodeID int) error
}

// ────────────────────────────────────────────────────────────
// Puerto de repositorio para Nodes
// ────────────────────────────────────────────────────────────

type NodeRepository interface {
	// RegisterNode registra un nuevo nodo o lo actualiza si ya existe
	RegisterNode(ctx context.Context, node *domain.Node) (*domain.Node, error)

	// GetAllNodes retorna todos los nodos registrados
	GetAllNodes(ctx context.Context) ([]domain.Node, error)

	// GetActiveNodes retorna solo los nodos activos
	GetActiveNodes(ctx context.Context) ([]domain.Node, error)

	// UpdateNodeStatus actualiza el estado de un nodo
	UpdateNodeStatus(ctx context.Context, nodeID string, status string) error
}

// ────────────────────────────────────────────────────────────
// Puerto de repositorio para Logs
// ────────────────────────────────────────────────────────────

type LogRepository interface {
	// CreateLog inserta un log de procesamiento
	CreateLog(ctx context.Context, log *domain.ProcessingLog) error

	// GetLogsByImage retorna los logs de una imagen específica
	GetLogsByImage(ctx context.Context, imageID int) ([]domain.ProcessingLog, error)

	// GetLogsByNode retorna los logs de un nodo específico
	GetLogsByNode(ctx context.Context, nodeID int) ([]domain.ProcessingLog, error)
}
