package usecases

import (
	"context"
	"io"

	"github.com/fbaez/grpc-go-android/server-go/internal/domain/entities"
	"github.com/fbaez/grpc-go-android/server-go/internal/domain/ports"
	"github.com/google/uuid"
)

// FileUseCases contiene los casos de uso para archivos
type FileUseCases struct {
	fileRepo        ports.FileRepository
	storageService  ports.FileStorageService
	eventBus        ports.EventBus
}

// NewFileUseCases crea una nueva instancia de FileUseCases
func NewFileUseCases(fileRepo ports.FileRepository, storageService ports.FileStorageService, eventBus ports.EventBus) *FileUseCases {
	return &FileUseCases{
		fileRepo:       fileRepo,
		storageService: storageService,
		eventBus:       eventBus,
	}
}

// UploadFile sube un archivo al sistema
func (uc *FileUseCases) UploadFile(ctx context.Context, filename, contentType string, reader io.Reader, userID uuid.UUID, compress bool, compressionType string) (*entities.FileInfo, error) {
	// Almacenar el archivo físicamente
	path, checksum, size, err := uc.storageService.StoreFile(ctx, filename, reader, compress, compressionType)
	if err != nil {
		return nil, err
	}
	
	// Crear la entidad de archivo
	fileInfo := entities.NewFileInfo(filename, contentType, checksum, path, size, userID, compress, compressionType)
	
	if err := fileInfo.Validate(); err != nil {
		// Si falla la validación, eliminar el archivo físico
		uc.storageService.DeleteFile(ctx, path)
		return nil, err
	}
	
	// Guardar la información en la base de datos
	if err := uc.fileRepo.Create(ctx, fileInfo); err != nil {
		// Si falla la creación en BD, eliminar el archivo físico
		uc.storageService.DeleteFile(ctx, path)
		return nil, err
	}
	
	// Publicar evento de archivo subido
	if uc.eventBus != nil {
		event := &FileUploadedEvent{
			FileID:   fileInfo.ID,
			UserID:   userID,
			Filename: filename,
			Size:     size,
		}
		uc.eventBus.Publish(ctx, event)
	}
	
	return fileInfo, nil
}

// DownloadFile descarga un archivo del sistema
func (uc *FileUseCases) DownloadFile(ctx context.Context, fileID, userID uuid.UUID) (*entities.FileInfo, io.ReadCloser, error) {
	// Obtener información del archivo
	fileInfo, err := uc.fileRepo.GetByID(ctx, fileID)
	if err != nil {
		return nil, nil, err
	}
	
	if !fileInfo.IsOwnedBy(userID) {
		return nil, nil, entities.ErrFileUnauthorized
	}
	
	// Obtener el archivo físico
	reader, err := uc.storageService.RetrieveFile(ctx, fileInfo.Path)
	if err != nil {
		return nil, nil, err
	}
	
	// Publicar evento de archivo descargado
	if uc.eventBus != nil {
		event := &FileDownloadedEvent{
			FileID:   fileInfo.ID,
			UserID:   userID,
			Filename: fileInfo.Filename,
		}
		uc.eventBus.Publish(ctx, event)
	}
	
	return fileInfo, reader, nil
}

// DeleteFile elimina un archivo del sistema
func (uc *FileUseCases) DeleteFile(ctx context.Context, fileID, userID uuid.UUID) error {
	// Obtener información del archivo
	fileInfo, err := uc.fileRepo.GetByID(ctx, fileID)
	if err != nil {
		return err
	}
	
	if !fileInfo.IsOwnedBy(userID) {
		return entities.ErrFileUnauthorized
	}
	
	// Eliminar de la base de datos
	if err := uc.fileRepo.Delete(ctx, fileID); err != nil {
		return err
	}
	
	// Eliminar el archivo físico
	if err := uc.storageService.DeleteFile(ctx, fileInfo.Path); err != nil {
		// Log del error pero no fallar la operación
		// ya que el registro ya fue eliminado de la BD
	}
	
	// Publicar evento de archivo eliminado
	if uc.eventBus != nil {
		event := &FileDeletedEvent{
			FileID:   fileID,
			UserID:   userID,
			Filename: fileInfo.Filename,
		}
		uc.eventBus.Publish(ctx, event)
	}
	
	return nil
}

// ListFiles lista los archivos de un usuario
func (uc *FileUseCases) ListFiles(ctx context.Context, userID uuid.UUID, filters ports.FileFilters) ([]*entities.FileInfo, int, error) {
	return uc.fileRepo.GetByUserID(ctx, userID, filters)
}

// GetFileInfo obtiene la información de un archivo
func (uc *FileUseCases) GetFileInfo(ctx context.Context, fileID, userID uuid.UUID) (*entities.FileInfo, error) {
	fileInfo, err := uc.fileRepo.GetByID(ctx, fileID)
	if err != nil {
		return nil, err
	}
	
	if !fileInfo.IsOwnedBy(userID) {
		return nil, entities.ErrFileUnauthorized
	}
	
	return fileInfo, nil
}

// Events
type FileUploadedEvent struct {
	FileID   uuid.UUID
	UserID   uuid.UUID
	Filename string
	Size     int64
}

type FileDownloadedEvent struct {
	FileID   uuid.UUID
	UserID   uuid.UUID
	Filename string
}

type FileDeletedEvent struct {
	FileID   uuid.UUID
	UserID   uuid.UUID
	Filename string
}