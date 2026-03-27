package ports

import (
	"context"

	"Auth/core/domain"
)

// ────────────────────────────────────────────────────────────
// Puerto de servicio para Batches (lo que el handler llama)
// ────────────────────────────────────────────────────────────

type BatchService interface {
	CreateBatch(ctx context.Context, userUUID string, imageNames []string) (*domain.Batch, []domain.Image, error)
	GetBatch(ctx context.Context, id int) (*domain.Batch, error)
	GetUserBatches(ctx context.Context, userUUID string, limit, offset int) ([]domain.Batch, error)
	UpdateBatchStatus(ctx context.Context, id int, status domain.BatchStatus) error
	FinalizeBatch(ctx context.Context, id int, status domain.BatchStatus) error
}

// ────────────────────────────────────────────────────────────
// Puerto de servicio para Images
// ────────────────────────────────────────────────────────────

type ImageService interface {
	GetImage(ctx context.Context, id int) (*domain.Image, error)
	GetBatchImages(ctx context.Context, batchID int) ([]domain.Image, error)
	FinalizeImage(ctx context.Context, id int, resultPath string, status string) error
	AssignToNode(ctx context.Context, imageID, nodeID int) error
}

// ────────────────────────────────────────────────────────────
// Puerto de servicio para Nodes
// ────────────────────────────────────────────────────────────

type NodeService interface {
	RegisterNode(ctx context.Context, node *domain.Node) (*domain.Node, error)
	GetAllNodes(ctx context.Context) ([]domain.Node, error)
	GetActiveNodes(ctx context.Context) ([]domain.Node, error)
	UpdateNodeStatus(ctx context.Context, nodeID string, status string) error
}

// ────────────────────────────────────────────────────────────
// Puerto de servicio para Logs
// ────────────────────────────────────────────────────────────

type LogService interface {
	CreateLog(ctx context.Context, log *domain.ProcessingLog) error
	GetImageLogs(ctx context.Context, imageID int) ([]domain.ProcessingLog, error)
	GetNodeLogs(ctx context.Context, nodeID int) ([]domain.ProcessingLog, error)
}

// ────────────────────────────────────────────────────────────
// Puerto de autenticación (para validar tokens JWT)
// ────────────────────────────────────────────────────────────

type TokenService interface {
	// ValidateToken valida un JWT y retorna los claims si es válido
	ValidateToken(tokenStr string) (*TokenClaims, error)

	// GenerateToken genera un JWT para uso interno entre servicios
	GenerateServiceToken(serviceID string) (string, error)
}

// TokenClaims son los datos extraídos de un JWT válido
type TokenClaims struct {
	UserUUID string `json:"user_uuid"`
	Username string `json:"username"`
	Role     string `json:"role"`
}
