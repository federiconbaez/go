package ports

import (
	"context"
	"io"

	"github.com/google/uuid"
)

// FileStorageService define la interfaz para el servicio de almacenamiento de archivos
type FileStorageService interface {
	StoreFile(ctx context.Context, filename string, reader io.Reader, compress bool, compressionType string) (string, string, int64, error) // path, checksum, size, error
	RetrieveFile(ctx context.Context, path string) (io.ReadCloser, error)
	DeleteFile(ctx context.Context, path string) error
	CompressFile(data []byte, compressionType string) ([]byte, error)
	DecompressFile(data []byte, compressionType string) ([]byte, error)
}

// NotificationService define la interfaz para el servicio de notificaciones
type NotificationService interface {
	SendNotification(ctx context.Context, userID uuid.UUID, title, message, notificationType string, channels []string, metadata map[string]string) error
	SubscribeToNotifications(ctx context.Context, userID uuid.UUID, channels []string) (<-chan Notification, error)
	UnsubscribeFromNotifications(ctx context.Context, userID uuid.UUID) error
}

// Notification representa una notificación
type Notification struct {
	ID       uuid.UUID
	Title    string
	Message  string
	Type     string
	UserID   uuid.UUID
	Metadata map[string]string
}

// CompressionService define la interfaz para el servicio de compresión
type CompressionService interface {
	Compress(data []byte, compressionType string) ([]byte, error)
	Decompress(data []byte, compressionType string) ([]byte, error)
	GetCompressionRatio(originalSize, compressedSize int64) float32
}

// EventBus define la interfaz para el bus de eventos
type EventBus interface {
	Publish(ctx context.Context, event interface{}) error
	Subscribe(eventType string, handler EventHandler) error
}

// EventHandler define el manejador de eventos
type EventHandler func(ctx context.Context, event interface{}) error