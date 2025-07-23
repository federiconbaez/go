package ports

import (
	"context"

	https://github.com/federiconbaez/gogrpc-go-android/server-go/internal/domain/entities"
	"github.com/google/uuid"
)

// IdeaRepository define la interfaz para el repositorio de ideas
type IdeaRepository interface {
	Create(ctx context.Context, idea *entities.Idea) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Idea, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, filters IdeaFilters) ([]*entities.Idea, int, error)
	Update(ctx context.Context, idea *entities.Idea) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ReminderRepository define la interfaz para el repositorio de recordatorios
type ReminderRepository interface {
	Create(ctx context.Context, reminder *entities.Reminder) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Reminder, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, filters ReminderFilters) ([]*entities.Reminder, int, error)
	Update(ctx context.Context, reminder *entities.Reminder) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetOverdueReminders(ctx context.Context) ([]*entities.Reminder, error)
}

// FileRepository define la interfaz para el repositorio de archivos
type FileRepository interface {
	Create(ctx context.Context, fileInfo *entities.FileInfo) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.FileInfo, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, filters FileFilters) ([]*entities.FileInfo, int, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// ProgressRepository define la interfaz para el repositorio de progreso
type ProgressRepository interface {
	Create(ctx context.Context, progress *entities.Progress) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Progress, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.Progress, error)
	Update(ctx context.Context, progress *entities.Progress) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// Filtros para consultas

// IdeaFilters contiene los filtros para buscar ideas
type IdeaFilters struct {
	Category entities.IdeaCategory
	Status   entities.IdeaStatus
	Tags     []string
	Page     int
	PageSize int
	SortBy   string
	SortDesc bool
}

// ReminderFilters contiene los filtros para buscar recordatorios
type ReminderFilters struct {
	Type     entities.ReminderType
	Status   entities.ReminderStatus
	FromDate *string // ISO 8601 format
	ToDate   *string // ISO 8601 format
	Page     int
	PageSize int
}

// FileFilters contiene los filtros para buscar archivos
type FileFilters struct {
	ContentTypeFilter string
	Page              int
	PageSize          int
	SortBy            string
	SortDesc          bool
}